package main

import (
    "log"
    "fmt"
    "strings"
    "strconv"
    _ "github.com/mattn/go-sqlite3"
    dg "github.com/bwmarrin/discordgo"
)

func challengeCommand(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    msgOptions := getOptionsOrRespond(s, i, NewMsg)
    if msgOptions == nil { return }

    challengeeID := msgOptions[0]
    challengee, err := s.User(fmt.Sprintf("%v", challengeeID.Value))
    if err != nil { log.Panic(err) }

    interactionUID := getInteractUID(i)

    challengeeUser := getUser(db, challengee.ID)
    if challengeeUser.interactable == 0 {
        addInteractionResponse(s, i, NewMsg, "Användaren tillåter inte utmaningar.")
        return
    }

    if strconv.Itoa(challengeeUser.uid) == interactionUID {
        addInteractionResponse(s, i, NewMsg, "Du kan inte utmana dig själv.")
        return
    }

    alreadyChallenged := getChallenge(db, "challengeeUID=? AND status!=? AND status!=? AND status!=?", challengee.ID, Unhandled, Declined, Forfeited ).id != -1
    if alreadyChallenged {
        addInteractionResponse(s, i, NewMsg, "Du kan inte utmana samma spelare flera gånger.")
        return
    }

    options := getCurrentMatchesAsOptions(db, challengee.ID)
    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Vilken match vill du utmana om?",
                    CustomID: "challSelectWinner",
                    Options: *options,
                },
            },
        },
    }

    msg := fmt.Sprintf("Utmana användare %v om hur följande match kommer sluta.", challengee.Username)
    addCompInteractionResponse(s, i, NewMsg, msg, components)
}

func challSelectWinner(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }
    value := values[0]

    splitted := strings.Split(value, "_")

    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchID := splitted[1]

    m := getMatch(db, "id=?", matchID)

    val := challengee.ID + "_" + matchID + "_"

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "challSelectPoints",
                    Placeholder: "Välj ett lag...",
                    Options: []dg.SelectMenuOption{
                        {
                            Label: m.homeTeam,
                            Value: val + "homeTeam",
                        },
                        {
                            Label: m.awayTeam,
                            Value: val + "awayTeam",
                        },
                    },
                },
            },
        },
    }

    msg := "Vem tror du vinner?"
    addCompInteractionResponse(s, i, UpdateMsg, msg, components)
}

func challSelectPoints(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }

    splitted := strings.Split(values[0], "_")
    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    interactionUID := getInteractUID(i)

    var (
        challengerPoints = getUser(db, interactionUID).seasonPoints
        challengeePoints = getUser(db, challengee.ID).seasonPoints
        maxPoints = -1
    )

    if challengerPoints < challengeePoints {
        maxPoints = challengerPoints
    } else {
        maxPoints = challengeePoints
    }

    if maxPoints == 0 {
        addCompInteractionResponse(s, i, UpdateMsg, "Inga poäng att satsa med.", []dg.MessageComponent{})
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "challAcceptDiscard",
                    Placeholder: "Poäng att satsa.",
                    Options: *getPointsAsOptions(values[0], maxPoints),
                },
            },
        },
    }

    msg := "Hur mycket poäng vill du satsa?\n"
    msg += "Du kan som mest satsa lika många poäng som motståndern har."
    addCompInteractionResponse(s, i, UpdateMsg, msg, components)
}

func challAcceptDiscard(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }
    splitted := strings.Split(values[0], "_")

    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchID := splitted[1]
    winnerTeam := splitted[2]
    points := splitted[3]

    teamName := ""
    err = db.QueryRow("SELECT " + winnerTeam + " FROM matches WHERE id=?", matchID).Scan(&teamName)
    if err != nil { log.Panic(err) }

    options := []dg.SelectMenuOption{
        {
            Label: "Skicka",
            Value: values[0],
        },
        {
            Label: "Släng",
            Value: "discard",
        },
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "challAcceptDiscardDo",
                    Placeholder: "Välj ett alternativ...",
                    Options: options,
                },
            },
        },
    }

    msg := fmt.Sprintf("\nDu vill utmana **%v** om att **%v** vinner för **%v** poäng.\n\n", challengee.Username, teamName, points)
    msg += "Är du säker? En utmaning kan bara tas bort om den du utmanar accepterar borttagningen och" +
           " poängen kommer finnas hos vadhållaren tills dess att utmaningen är klar eller den du utmanat" +
           " har nekat vadet."
    addCompInteractionResponse(s, i, UpdateMsg, msg, components)
}

func challAcceptDiscardDo(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }
    value := values[0]

    splitted := strings.Split(value, "_")

    if value == "discard" {
        addCompInteractionResponse(s, i, UpdateMsg, "Utmaningen är slängd", []dg.MessageComponent{})
        return
    }

    interactionUID := getInteractUID(i)

    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchID := splitted[1]
    winnerTeam := splitted[2]
    points := splitted[3]

    _, err = db.Exec("UPDATE users SET seasonPoints=seasonPoints - ? WHERE uid=?", points, interactionUID)
    if err != nil { log.Panic(err) }

    res, err := db.Exec("INSERT INTO challenges (challengerUID, challengeeUID, type, matchID, points, condition) VALUES (?, ?, ?, ?, ?, ?)",
                         interactionUID, challengee.ID, 0, matchID, points, winnerTeam)
    if err != nil { log.Panic(err) }

    msg := "Utmaning mottagen och poängen har plockats från ditt konto.\n"
    msg += "Nu måste motståndaren acceptera/neka utmaningen.\n\n"
    msg += "Om motståndaren inte accepterar/nekar utmaningen innan matchstart kommer poängen tillbaka till ditt konto."

    components := []dg.MessageComponent {}
    addCompInteractionResponse(s, i, UpdateMsg, msg, components)

    insertedCID, err := res.LastInsertId()
    if err != nil { log.Fatal(err) }

    sendChallenge(s, interactionUID, challengee.ID, int(insertedCID), points)
}

func sendChallenge(s *dg.Session, challengerID string, challengeeID string, cid int, points string) {
    db := connectDB()
    defer db.Close()

    m := getMatch(db, "id=?", cid)

    _, err := db.Exec("UPDATE challenges SET status=? WHERE id=?", Sent, challengerID)
    if err != nil { log.Panic(err) }

    options := []dg.SelectMenuOption{
        {
            Label: "Acceptera",
            Value: fmt.Sprintf("accept_%v", cid),
        },
        {
            Label: "Neka",
            Value: fmt.Sprintf("decline_%v", cid),
        },
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "challAnswer",
                    Placeholder: "Välj ett alternativ...",
                    Options: options,
                },
            },
        },
    }

    msg := fmt.Sprintf("Hej!\n")
    msg += fmt.Sprintf("%v har utmanat dig till följande utmaning...\n", challengerID)
    msg += fmt.Sprintf("**%v** vinner på hemmaplan mot **%v** för **%v** poäng.\n\n", m.homeTeam, m.awayTeam, points)
    msg += fmt.Sprintf("Vill du satsa emot?\n\n")
    msg += fmt.Sprintf("*du kan stänga av utmaningar via **/inställningar** kommandot*")

    dmcid, _ := s.UserChannelCreate(challengeeID)
    s.ChannelMessageSendComplex(dmcid.ID, &dg.MessageSend{
        Content: msg,
        Components: components,
    })
}

func challAnswer(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }
    value := values[0]

    splitted := strings.Split(value, "_")
    answ := splitted[0]
    cid := splitted[1]

    msgChallenged := ""
    msgChallengee := ""
    userUID := 0
    status := 1
    plusOrMinus := ""

    c := getChallenge(db, "id=?", cid)

    if answ == "accept" {
        status = 2
        msgChallenged = fmt.Sprintf("Din utmaning har blivit accepterad.")
        msgChallengee = fmt.Sprintf("Skickar bekräftelse till utmanaren.")
        userUID = c.challengeeUID
        plusOrMinus = "-"
    } else if answ == "decline" {
        status = 3
        msgChallenged = fmt.Sprintf("Din utmaning har blivit nekad.")
        msgChallengee = fmt.Sprintf("Du har nekat utmaningen.")
        userUID = c.challengerUID
        plusOrMinus = "+"
    }

    _, err := db.Exec("UPDATE users SET seasonPoints=seasonPoints " + plusOrMinus + " ? WHERE uid=?", c.points, userUID)
    if err != nil { log.Panic(err) }

    _, err = db.Exec("UPDATE challenges SET status=? WHERE id=?", status, cid)
    if err != nil { log.Panic(err) }

    addCompInteractionResponse(s, i, UpdateMsg, msgChallengee, []dg.MessageComponent{})

    dmcid, _ := s.UserChannelCreate(fmt.Sprintf("%v", c.challengerUID))
    s.ChannelMessageSend(dmcid.ID, msgChallenged)
}

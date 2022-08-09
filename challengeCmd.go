package main

import (
    "log"
    "fmt"
    "strings"
    "strconv"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    dg "github.com/bwmarrin/discordgo"
)

func challengeCommand(s *dg.Session, i *dg.InteractionCreate) {
    betsDB, err := sql.Open(DB_TYPE, BETS_DB)
    defer betsDB.Close()
    if err != nil { log.Fatal(err) }

    if len(i.Interaction.ApplicationCommandData().Options) != 2 {
        log.Panic("Not enough argumenst given...")
    }

    option := i.Interaction.ApplicationCommandData().Options[0]
    challengee, err := s.User(fmt.Sprintf("%v", option.Value))
    if err != nil { log.Panic(err) }

    uid := getInteractUID(i)

    // Check if user allows challenges
    u := getUser(challengee.ID)
    if u.interactable == 0 {
        msgStdInteractionResponse(s, i, "Användaren tillåter inte utmaningar.")
        return
    }

    if strconv.Itoa(u.uid) == uid {
        msgStdInteractionResponse(s, i, "Du kan inte utmana dig själv.")
        return
    }

    oldBet := 0
    err = betsDB.QueryRow("SELECT id FROM challenges WHERE challengedUID=? AND status!=? AND status!=? AND status!=?", challengee.ID, Unhandled, Declined, Forfeited).Scan(&oldBet)
    if err != nil {
        if err != sql.ErrNoRows { log.Panic(err) }
    }

    if oldBet != 0 {
        msgStdInteractionResponse(s, i, "Du kan inte utmana samma spelare flera gånger")
        return
    }

    msg := fmt.Sprintf("Utmana användare %v om hur följande match kommer sluta.", challengee.Username)

    options := getRoundMatchesAsOptions(challengee.ID)

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

    compInteractionResponse(s, i, dg.InteractionResponseChannelMessageWithSource, msg, components)
}

func challSelectWinner(s *dg.Session, i *dg.InteractionCreate) {
    svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
    defer svffDB.Close()
    if err != nil { log.Fatal(err) }

    betsDB, err := sql.Open(DB_TYPE, BETS_DB)
    defer betsDB.Close()
    if err != nil { log.Fatal(err) }

    // Parsing values
    vals := i.MessageComponentData().Values
    if len(vals) == 0 { log.Panic(err) }

    splitted := strings.Split(vals[0], "_")

    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchID := splitted[1]

    _, err = s.User(getInteractUID(i))
    if err != nil { log.Panic(err) }

    // Do things
    var m match

    err = svffDB.QueryRow("SELECT id, homeTeam, awayTeam, date FROM matches WHERE id=?", matchID).Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date)
    if err != nil { log.Panic(err) }

    val := challengee.ID + "_" + matchID + "_"

    msg := "Vem tror du vinner?"

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

    compInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, msg, components)
}

// Value strings comes in as challengeeID_matchID_winnerTeam
func challSelectPoints(s *dg.Session, i *dg.InteractionCreate) {
    betsDB, err := sql.Open(DB_TYPE, BETS_DB)
    defer betsDB.Close()
    if err != nil { log.Fatal(err) }

    // Parsing values
    vals := i.MessageComponentData().Values
    if len(vals) == 0 { log.Panic("Not enough values given.") }

    splitted := strings.Split(vals[0], "_")

    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    uid := getInteractUID(i)

    challengerPoints, challengeePoints, maxPoints := 0, 0, 0
    err = betsDB.QueryRow("SELECT season FROM points WHERE uid=?", uid).Scan(&challengerPoints)
    if err != nil { log.Panic(err) }

    err = betsDB.QueryRow("SELECT season FROM points WHERE uid=?", challengee.ID).Scan(&challengeePoints)
    if err != nil { log.Panic(err) }

    if challengerPoints < challengeePoints {
        maxPoints = challengerPoints
    } else {
        maxPoints = challengeePoints
    }

    msg := "Hur mycket poäng vill du satsa?\n"
    msg += "Du kan som mest satsa lika många poäng som motståndern har."

    components := []dg.MessageComponent {}

    if maxPoints == 0 {
        components = []dg.MessageComponent {
            dg.ActionsRow {
                Components: []dg.MessageComponent {
                    dg.SelectMenu {
                        CustomID: "challAcceptDiscard",
                        Placeholder: "Inga poäng att satsa med.",
                        Disabled: true,
                        Options: *getPointsOptions(vals[0], maxPoints),
                    },
                },
            },
        }
    } else {
        components = []dg.MessageComponent {
            dg.ActionsRow {
                Components: []dg.MessageComponent {
                    dg.SelectMenu {
                        CustomID: "challAcceptDiscard",
                        Placeholder: "Poäng att satsa.",
                        Options: *getPointsOptions(vals[0], maxPoints),
                    },
                },
            },
        }
    }

    compInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, msg, components)
}

func challAcceptDiscard(s *dg.Session, i *dg.InteractionCreate) {
    svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
    defer svffDB.Close()
    if err != nil { log.Fatal(err) }

    // Parsing values
    vals := i.MessageComponentData().Values
    if len(vals) == 0 { log.Panic("Not enough values given.") }
    splitted := strings.Split(vals[0], "_")

    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchID := splitted[1]
    winnerTeam := splitted[2]
    points := splitted[3]

    teamName := ""

    if winnerTeam == "homeTeam" {
        err = svffDB.QueryRow("SELECT homeTeam FROM matches WHERE id=?", matchID).Scan(&teamName)
        if err != nil { log.Panic(err) }
    } else {
        err = svffDB.QueryRow("SELECT awayTeam FROM matches WHERE id=?", matchID).Scan(&teamName)
        if err != nil { log.Panic(err) }
    }

    msg := fmt.Sprintf("\nDu vill utmana **%v** om att **%v** vinner för **%v** poäng.\n\n",
                        challengee.Username, teamName, points)
    msg += "Är du säker? En utmaning kan bara tas bort om den du utmanar accepterar borttagningen och" +
           " poängen kommer finnas hos vadhållaren tills dess att utmaningen är klar eller den du utmanat" +
           " har nekat vadet."


    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "challAcceptDiscardDo",
                    Placeholder: "Välj ett alternativ...",
                    Options: []dg.SelectMenuOption{
                        {
                            Label: "Skicka",
                            Value: vals[0],
                        },
                        {
                            Label: "Släng",
                            Value: "discard",
                        },
                    },
                },
            },
        },
    }

    compInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, msg, components)
}

func challAcceptDiscardDo(s *dg.Session, i *dg.InteractionCreate) {
    betsDB, err := sql.Open(DB_TYPE, BETS_DB)
    defer betsDB.Close()
    if err != nil { log.Fatal(err) }

    // Parsing values
    vals := i.MessageComponentData().Values
    if len(vals) == 0 { log.Panic("Not enough values given.") }

    if vals[0] == "discard" {
        components := []dg.MessageComponent {}
        compInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, "Utmaningen är slängd", components)
        return
    }

    uid := getInteractUID(i)

    // split
    splitted := strings.Split(vals[0], "_")
    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchID := splitted[1]
    winnerTeam := splitted[2]
    points := splitted[3]

    _, err = betsDB.Exec("UPDATE points SET season=season - ? WHERE uid=?", points, uid)
    if err != nil { log.Fatal(err) }

    res, err := betsDB.Exec("INSERT INTO challenges (challengerUID, challengedUID, type, matchID, points, condition) VALUES (?, ?, ?, ?, ?, ?)",
                         uid, challengee.ID, 0, matchID, points, winnerTeam)
    if err != nil { log.Fatal(err) }

    msg := "Utmaning mottagen och poängen har plockats från ditt konto.\n"
    msg += "Nu måste motståndaren acceptera/neka utmaningen.\n\n"
    msg += "Om motståndaren inte accepterar/nekar utmaningen innan matchstart kommer poängen tillbaka till ditt konto."

    components := []dg.MessageComponent {}
    compInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, msg, components)

    cid, _ := res.LastInsertId()
    sendChallenge(s, uid, challengee.ID, int(cid), points)
}

func sendChallenge(s *dg.Session, challengerID string, challengeeID string, cid int, points string) {
    svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
    defer svffDB.Close()
    if err != nil { log.Fatal(err) }

    betsDB, err := sql.Open(DB_TYPE, BETS_DB)
    defer betsDB.Close()
    if err != nil { log.Fatal(err) }

    var m match
    err = svffDB.QueryRow("SELECT homeTeam, awayTeam FROM matches WHERE id=?", cid).Scan(&m.homeTeam, &m.awayTeam)
    if err != nil { log.Panic(err) }

    msg := fmt.Sprintf("Hej!\n")
    msg += fmt.Sprintf("%v har utmanat dig till följande utmaning...\n", challengerID)
    msg += fmt.Sprintf("**%v** vinner på hemmaplan mot **%v** för **%v** poäng.\n\n", m.homeTeam, m.awayTeam, points)
    msg += fmt.Sprintf("Vill du satsa emot?\n\n")
    msg += fmt.Sprintf("*du kan stänga av utmaningar via **/inställningar** kommandot*")

    _, err = betsDB.Exec("UPDATE challenges SET status=? WHERE id=?", 1, challengerID)
    if err != nil { log.Panic(err) }

    dmcid, _ := s.UserChannelCreate(challengeeID)

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "challAnswer",
                    Placeholder: "Välj ett alternativ...",
                    Options: []dg.SelectMenuOption{
                        {
                            Label: "Acceptera",
                            Value: fmt.Sprintf("accept_%v", cid),
                        },
                        {
                            Label: "Neka",
                            Value: fmt.Sprintf("decline_%v", cid),
                        },
                    },
                },
            },
        },
    }

    s.ChannelMessageSendComplex(dmcid.ID, &dg.MessageSend{
        Content: msg,
        Components: components,
    })
}

func challAnswer(s *dg.Session, i *dg.InteractionCreate) {
    betsDB, err := sql.Open(DB_TYPE, BETS_DB)
    defer betsDB.Close()
    if err != nil { log.Fatal(err) }

    vals := i.MessageComponentData().Values

    splitted := strings.Split(vals[0], "_")
    answ := splitted[0]
    cid := splitted[1]

    msgChallenged := ""
    msgChallengee := ""
    status := 1

    challenger, challengee := 0, 0
    points := 0
    err = betsDB.QueryRow("SELECT challengerUID, challengedUID, points FROM challenges WHERE id=?", cid).Scan(&challenger, &challengee, &points)

    if answ == "accept" {
        status = 2
        msgChallenged = fmt.Sprintf("Din utmaning har blivit accepterad.")
        msgChallengee = fmt.Sprintf("Skickar bekräftelse till utmanaren.")
        _, err = betsDB.Exec("UPDATE points SET season=season - ? WHERE uid=?", points, challengee)
    } else if answ == "decline" {
        status = 3
        msgChallenged = fmt.Sprintf("Din utmaning har blivit nekad.")
        msgChallengee = fmt.Sprintf("Du har nekat utmaningen.")
        _, err = betsDB.Exec("UPDATE points SET season=season + ? WHERE uid=?", points, challenger)
        if err != nil { log.Panic(err) }
    }

    _, err = betsDB.Exec("UPDATE challenges SET status=? WHERE id=?", status, cid)
    if err != nil { log.Panic(err) }

    components := []dg.MessageComponent {}
    compInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, msgChallengee, components)

    dmcid, _ := s.UserChannelCreate(fmt.Sprintf("%v", challenger))
    s.ChannelMessageSend(dmcid.ID, msgChallenged)
}

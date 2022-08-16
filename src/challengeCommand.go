/*
   This command allows the user to select a match from this round, choose which team wins the match
   and use this information as grounds for a challenge to another user.

   It's divided into the following steps...

   0. The command is performed with options that are the user to challenge and the type of challenge to send.

   The challenger:
   1. Choose match (do security checks to give user information early).
   2. Choose winner.
   3. Choose points to bet.
   4. Choose to actually send or discard the challenge.
   5. Send or discard the challenge (do security checks again so that any value passed along from step 1
   hasn't changed).

   The challengee:
   0. Is sent a notification about the challenge and can choose to accept/discard it.
   1. Send response about discard or accept (do security checks once again).
*/

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

    challenges := *getChallenges(db, "challengeeUID=? AND (status=? OR status=? OR status=? OR status=?)", challengee.ID, Unhandled, Sent, Accepted, RequestForfeit)
    if len(challenges) >= 25 {
        addInteractionResponse(s, i, NewMsg, "Du kan inte ha mer än 25 utmaningar.")
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
    loserTeam := ""
    points := splitted[3]

    var m match
    err = db.QueryRow("SELECT homeTeam, awayTeam FROM matches WHERE id=?", matchID).Scan(&m.homeTeam, &m.awayTeam)
    if err != nil { log.Panic(err) }

    if winnerTeam == "homeTeam" {
        winnerTeam = m.homeTeam
        loserTeam = m.awayTeam
    } else {
        winnerTeam = m.awayTeam
        loserTeam = m.homeTeam
    }

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

    msg := fmt.Sprintf("\nDu vill utmana **%v** om att **%v** vinner mot **%v** för **%v** poäng.\n\n", challengee.Username, winnerTeam, loserTeam, points)
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

    challengerUser := getUserFromInteraction(db, i)
    interactionUID := getInteractUID(i)

    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchID := splitted[1]
    winnerTeam := splitted[2]
    points := splitted[3]

    // Security checks
    challengeeUser := getUser(db, challengee.ID)
    if challengeeUser.interactable == 0 {
        addInteractionResponse(s, i, UpdateMsg, "Användaren tillåter inte utmaningar.")
        return
    }

    if strconv.Itoa(challengeeUser.uid) == interactionUID {
        addInteractionResponse(s, i, UpdateMsg, "Du kan inte utmana dig själv.")
        return
    }

    alreadyChallenged := getChallenge(db, "challengeeUID=? AND status!=? AND status!=? AND status!=?", challengee.ID, Unhandled, Declined, Forfeited ).id != -1
    if alreadyChallenged {
        addInteractionResponse(s, i, UpdateMsg, "Du kan inte utmana samma spelare flera gånger.")
        return
    }

    challenges := *getChallenges(db, "challengeeUID=? AND (status=? OR status=? OR status=? OR status=?)", challengee.ID, Unhandled, Sent, Accepted, RequestForfeit)
    if len(challenges) >= 25 {
        addInteractionResponse(s, i, UpdateMsg, "Du kan inte ha mer än 25 utmaningar.")
        return
    }

    m := getMatch(db, "id=?", matchID)
    if matchHasBegun(s, i, m) {
        addInteractionResponse(s, i, UpdateMsg, "Du kan inte utmana om en match som redan startat.")
        return
    }

    pointsInt, err := strconv.Atoi(points)
    if err != nil { log.Panic(err) }
    if challengeeUser.seasonPoints < pointsInt || challengerUser.seasonPoints < pointsInt {
        addInteractionResponse(s, i, UpdateMsg, "Du eller den du utmanar har inte nog med poäng för att anta utmaningen.")
        return
    }

    // Write to database
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

    var m match
    var c challenge
    err := db.QueryRow("SELECT m.homeTeam, m.awayTeam, m.date, c.condition FROM challenges AS c " +
                       "JOIN matches AS m ON m.id = c.matchID " +
                       "WHERE c.id=?", cid).Scan(&m.homeTeam, &m.awayTeam, &m.date, &c.condition)
    if err != nil { log.Panic(err) }

    _, err = db.Exec("UPDATE challenges SET status=? WHERE id=?", Sent, challengerID)
    if err != nil { log.Panic(err) }

    options := []dg.SelectMenuOption{
        {
            Label: "Anta utmaningen",
            Value: fmt.Sprintf("accept_%v", cid),
        },
        {
            Label: "Neka utmaningen",
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

    winner, loser := m.homeTeam, m.awayTeam

    if c.condition == "awayTeam" {
        winner = m.awayTeam
        loser = m.homeTeam
    }

    challenger, _ := s.User(challengerID)

    msg := fmt.Sprintf("Du har blivit utmanad!\n")
    msg += fmt.Sprintf("**%v** tror att **%v** vinner mot **%v** den **%v** för **%v** poäng.\n\n", challenger.Username, winner, loser, m.date, points)
    msg += fmt.Sprintf("Vill du satsa emot?\n\n")
    msg += fmt.Sprintf("*du kan stänga av utmaningar via **/inställningar** kommandot*")

    dmcid, _ := s.UserChannelCreate(challengeeID)
    s.ChannelMessageSendComplex(dmcid.ID, &dg.MessageSend{
        Content: msg,
        Components: components,
    })
}

// KOLLA: Att poängen finns, att man är den man påstår, att man inte har 25 utmaningar redan (lägg vilande isf),
// att utmaningen inte redan existerar, att matchen inte har startat
// FIXA ÄVEN: att man fattar VILKEN utmaning som blivit accpeterad/declinad
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

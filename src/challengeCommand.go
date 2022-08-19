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
    "fmt"
    "log"
    "strconv"
    "strings"
    "time"
    dg "github.com/bwmarrin/discordgo"
    _ "github.com/lib/pq"
)

func newChallenge(b *botHolder) *Challenge {
    cmd := new(Challenge)
    cmd.bot = b
    cmd.name = HelpCommand
    cmd.description = "testar"
    cmd.addComponents()
    return cmd
}

func (cmd *Challenge) addComponents() {
    cmd.bot.addComponent("challSelectWinner", cmd.challSelectWinner)
    cmd.bot.addComponent("challSelectPoints", cmd.challSelectPoints)
    cmd.bot.addComponent("challAcceptDiscard", cmd.challAcceptDiscard)
    cmd.bot.addComponent("challAcceptDiscardDo", cmd.challAcceptDiscardDo)
    cmd.bot.addComponent("challAnswer", cmd.challAnswer)
}

func (cmd *Challenge) run(i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()
    s := cmd.bot.session

    msgOptions := getOptionsOrRespond(s, i, NewMsg)
    if msgOptions == nil { return }

    challengeeID := msgOptions[0]
    challengee, err := s.User(fmt.Sprintf("%v", challengeeID.Value))
    if err != nil { log.Panic(err) }

    interactionUID := getInteractUID(i)

    challengeeUser := getUser(db, challengee.ID)
    if !challengeeUser.interactable {
        addInteractionResponse(s, i, NewMsg, "Användaren tillåter inte utmaningar.")
        return
    }

    if !getUserFromInteraction(db, i).interactable {
        msg := "Du måste själv tillåta utmaningar för att kunna utmana andra. "
        msg += "Se /inställingar för valet."
        addInteractionResponse(s, i, NewMsg, msg)
        return
    }

    if strconv.Itoa(challengeeUser.uid) == interactionUID {
        addInteractionResponse(s, i, NewMsg, "Du kan inte utmana dig själv.")
        return
    }

    challenges := *getChallenges(db, "(challengeeid=$1 OR challengerid=$2) AND (status=$3 OR status=$4 OR status=$5 OR status=$6)",
                                      interactionUID, interactionUID, ChallengeStatusUnhandled, ChallengeStatusSent, ChallengeStatusAccepted, ChallengeStatusRequestForfeit)
    if len(challenges) >= 25 {
        addInteractionResponse(s, i, NewMsg, "Du kan inte ha mer än 25 utmaningar.")
        return
    }

    // Get the options
    options := *getCurrentMatchesAsOptions(db, challengee.ID)

    if len(options) == 0 {
        addInteractionResponse(s, i, NewMsg, "Inga matcher du kan utmana spelaren om.")
        return
    }

    // Remove matches where a challenge already exists
    realOptions := []dg.SelectMenuOption{}
    for _, option := range options {
        matchid := strings.Split(option.Value, "_")[1]

        existingChallenge := getChallenge(db, "((challengerid=$1 AND challengeeid=$2) OR " +
                                              "(challengeeid=$3 AND challengerid=$4)) " +
                                              "AND matchid=$5 " +
                                              "AND status!=$6 AND status!=$7",
                                              interactionUID, challengee.ID,
                                              interactionUID, challengee.ID,
                                              matchid,
                                              ChallengeStatusDeclined, ChallengeStatusForfeited)
        if existingChallenge.id == -1 {
            realOptions = append(realOptions, option)
        }
    }

    // Put it all together and send
    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Vilken match vill du utmana om?",
                    CustomID: "challSelectWinner",
                    Options: realOptions,
                },
            },
        },
    }

    msg := fmt.Sprintf("Utmana användare %v om hur följande match kommer sluta.", challengee.Username)
    addCompInteractionResponse(s, i, NewMsg, msg, components)
}

func (cmd *Challenge) challSelectWinner(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }
    value := values[0]

    splitted := strings.Split(value, "_")

    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchid := splitted[1]

    m := getMatch(db, "id=$1", matchid)

    val := challengee.ID + "_" + matchid + "_"

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "challSelectPoints",
                    Placeholder: "Välj ett lag...",
                    Options: []dg.SelectMenuOption{
                        {
                            Label: m.hometeam,
                            Value: val + "hometeam",
                        },
                        {
                            Label: m.awayteam,
                            Value: val + "awayteam",
                        },
                    },
                },
            },
        },
    }

    msg := "Vem tror du vinner?"
    addCompInteractionResponse(s, i, UpdateMsg, msg, components)
}

func (cmd *Challenge) challSelectPoints(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }

    splitted := strings.Split(values[0], "_")
    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    interactionUID := getInteractUID(i)

    var (
        challengerPoints = getUser(db, interactionUID).points
        challengeePoints = getUser(db, challengee.ID).points
        maxPoints = -1
    )

    if challengerPoints < challengeePoints {
        maxPoints = challengerPoints
    } else {
        maxPoints = challengeePoints
    }

    if maxPoints == 0 {
        addCompInteractionResponse(s, i, UpdateMsg, "Inga poäng att satsa med.", []dg.MessageComponent{})
        return
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

func (cmd *Challenge) challAcceptDiscard(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }
    splitted := strings.Split(values[0], "_")

    challengee, err := s.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchid := splitted[1]
    winnerTeam := splitted[2]
    loserTeam := ""
    points := splitted[3]

    var m match
    err = db.QueryRow("SELECT hometeam, awayteam FROM matches WHERE id=$1", matchid).Scan(&m.hometeam, &m.awayteam)
    if err != nil { log.Panic(err) }

    if winnerTeam == "hometeam" {
        winnerTeam = m.hometeam
        loserTeam = m.awayteam
    } else {
        winnerTeam = m.awayteam
        loserTeam = m.hometeam
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

func (cmd *Challenge) challAcceptDiscardDo(s *dg.Session, i *dg.InteractionCreate) {
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

    matchid := splitted[1]
    winnerTeam := splitted[2]
    points := splitted[3]
    challCondition := ChallengeConditionWinnerHome

    if winnerTeam == "awayteam" {
        challCondition = ChallengeConditionWinnerAway
    }

    // Security checks
    challengeeUser := getUser(db, challengee.ID)
    if !challengeeUser.interactable {
        addCompInteractionResponse(s, i, UpdateMsg, "Användaren tillåter inte utmaningar.", []dg.MessageComponent{})
        return
    }

    if strconv.Itoa(challengeeUser.uid) == interactionUID {
        addCompInteractionResponse(s, i, UpdateMsg, "Du kan inte utmana dig själv.", []dg.MessageComponent{})
        return
    }

    existingChallenge := getChallenge(db, "((challengerid=$1 AND challengeeid=$2) OR " +
                                          "(challengeeid=$3 AND challengerid=$4)) " +
                                          "AND matchid=$5 " +
                                          "AND status!=$6 AND status!=$7",
                                           interactionUID, challengee.ID,
                                           interactionUID, challengee.ID,
                                           matchid,
                                           ChallengeStatusDeclined, ChallengeStatusForfeited)
    if existingChallenge.id != -1 {
        addCompInteractionResponse(s, i, UpdateMsg, "Du kan inte utmana samma spelare flera gånger.", []dg.MessageComponent{})
        return
    }

    challenges := *getChallenges(db, "challengeeid=$1 AND (status=$2 OR status=$3 OR status=$4 OR status=$5)", challengee.ID, ChallengeStatusUnhandled, ChallengeStatusSent, ChallengeStatusAccepted, ChallengeStatusRequestForfeit)
    if len(challenges) >= 25 {
        addCompInteractionResponse(s, i, UpdateMsg, "Du kan inte ha mer än 25 utmaningar.", []dg.MessageComponent{})
        return
    }

    m := getMatch(db, "id=$1", matchid)
    if matchHasBegun(s, i, m) {
        addCompInteractionResponse(s, i, UpdateMsg, "Du kan inte utmana om en match som redan startat.", []dg.MessageComponent{})
        return
    }

    pointsInt, err := strconv.Atoi(points)
    if err != nil { log.Panic(err) }
    if challengeeUser.points < pointsInt || challengerUser.points < pointsInt {
        addCompInteractionResponse(s, i, UpdateMsg, "Du eller den du utmanar har inte nog med poäng för att anta utmaningen.", []dg.MessageComponent{})
        return
    }

    // Write to database
    _, err = db.Exec("UPDATE users SET points=points-$1 WHERE uid=$2", points, interactionUID)
    if err != nil { log.Panic(err) }

    insertedCID := 0
    err = db.QueryRow("INSERT INTO challenges (challengerid, challengeeid, type, matchid, points, condition) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
                     interactionUID, challengee.ID, ChallengeTypeWinner, matchid, points, challCondition).Scan(&insertedCID)
    if err != nil { log.Panic(err) }

    msg := "Utmaning har skickats och poängen har plockats från ditt konto. "
    msg += "Nu är det upp till motståndaren att acceptera/neka utmaningen.\n\n"
    msg += "Om motståndaren inte accepterar/nekar utmaningen innan matchstart kommer poängen tillbaka till ditt konto."

    components := []dg.MessageComponent {}
    addCompInteractionResponse(s, i, UpdateMsg, msg, components)

    cmd.sendChallenge(s, interactionUID, challengee.ID, int(insertedCID), points)
}

func (cmd *Challenge) sendChallenge(s *dg.Session, challengerID string, challengeeid string, cid int, points string) {
    db := connectDB()
    defer db.Close()

    var m match
    var c challenge
    err := db.QueryRow("SELECT m.hometeam, m.awayteam, m.date, c.id, c.condition FROM challenges AS c " +
                       "JOIN matches AS m ON m.id = c.matchid " +
                       "WHERE c.id=$1", cid).Scan(&m.hometeam, &m.awayteam, &m.date, &c.id, &c.condition)
    if err != nil { log.Panic(err) }

    _, err = db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusSent, c.id)
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

    winner, loser := m.hometeam, m.awayteam

    if c.condition == ChallengeConditionWinnerAway {
        winner = m.awayteam
        loser = m.hometeam
    }

    challenger, _ := s.User(challengerID)
    datetime, _ := time.Parse(DB_TIME_LAYOUT, m.date)

    msg := fmt.Sprintf("Du har blivit utmanad!\n")
    msg += fmt.Sprintf("**%v** tror att **%v** vinner mot **%v** den **%v** för **%v** poäng.\n\n",
                       challenger.Username, winner, loser, datetime.Format(MSG_TIME_LAYOUT), points)
    msg += fmt.Sprintf("Vill du satsa emot?\n\n")
    msg += fmt.Sprintf("*du kan stänga av utmaningar via **/inställningar** kommandot*")

    dmcid, _ := s.UserChannelCreate(challengeeid)
    s.ChannelMessageSendComplex(dmcid.ID, &dg.MessageSend{
        Content: msg,
        Components: components,
    })
}

func (cmd *Challenge) challAnswer(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }
    value := values[0]

    splitted := strings.Split(value, "_")
    answ := splitted[0]
    cid := splitted[1]

    msgChallenger := ""
    msgChallengee := ""
    userUID := 0
    status := ChallengeStatusUnhandled
    plusOrMinus := ""

    c := getChallenge(db, "id=$1", cid)
    m := getMatch(db, "id=$1", c.matchid)
    challengee := getUser(db, strconv.Itoa(c.challengeeid))

    // Security checks
    if c.status != ChallengeStatusSent {
        addCompInteractionResponse(s, i, UpdateMsg, "Utmaningen har redan blivit hanterad.", []dg.MessageComponent{})
        return
    }

    challenges := *getChallenges(db, "challengeeid=$1 AND (status=$2 OR status=$3 OR status=$4 OR status=$5)", challengee.uid, ChallengeStatusUnhandled, ChallengeStatusSent, ChallengeStatusAccepted, ChallengeStatusRequestForfeit)
    if len(challenges) >= 25 {
        addInteractionResponse(s, i, UpdateMsg, "Du kan inte ha mer än 25 utmaningar, ta bort en för att kunna acceptera denna.")
        return
    }

    if strconv.Itoa(challengee.uid) != getInteractUID(i) {
        addInteractionResponse(s, i, UpdateMsg, "Du är inte den utmanade.")
        return
    }

    if challengee.points < c.points {
        msgChallenger = "Motståndandaren har inte tillräckligt med poäng för att acceptera utmaningen.\n"
        msgChallengee = "Du har inte tillräckligt med poäng för att acceptera utmaningen.\n"
        answ = "decline"
    }

    existingChallenge := getChallenge(db, "((challengerid=$1 AND challengeeid=$2) OR " +
                                          "(challengeeid=$3 AND challengerid=$4)) " +
                                          "AND matchid=$5 AND id!=$6 " +
                                          "AND status!=$7 AND status!=$8",
                                           c.challengerid, challengee.uid,
                                           c.challengerid, challengee.uid,
                                           c.matchid, c.id,
                                           ChallengeStatusDeclined, ChallengeStatusForfeited)
    if existingChallenge.id != -1 {
        log.Printf("chall: %v", existingChallenge)
        msgChallenger = "Du har redan en aktiv utmaning med användaren.\n"
        msgChallengee = "Du har redan en aktiv utmaning med användaren.\n"
        answ = "decline"
    }

    if matchHasBegun(s, i, m) {
        msgChallenger = "Motståndaren svarade försent.\n"
        msgChallengee = "Matchen har redan startat.\n"
        answ = "decline"
    }

    challengeeUsername, _ := s.User(fmt.Sprint(challengee.uid))

    // Do stuff
    if answ == "accept" {
        status = ChallengeStatusAccepted
        msgChallenger += fmt.Sprintf("Din utmaning har blivit accepterad.\n")
        msgChallenger += fmt.Sprintf("**%v** vs **%v** för **%v** poäng mot **%v**",
                                     m.hometeam, m.awayteam, c.points, challengeeUsername)
        msgChallengee += fmt.Sprintf("Skickar bekräftelse till utmanaren.")
        userUID = c.challengeeid
        plusOrMinus = "-"
    } else {
        status = ChallengeStatusDeclined
        msgChallenger += fmt.Sprintf("Din utmaning har blivit nekad.\n")
        msgChallenger += fmt.Sprintf("**%v** vs **%v** för **%v** poäng mot **%v**",
                                     m.hometeam, m.awayteam, c.points, challengeeUsername)
        msgChallengee += fmt.Sprintf("Du har nekat utmaningen.")
        userUID = c.challengerid
        plusOrMinus = "+"
    }

    _, err := db.Exec("UPDATE users SET points=points " + plusOrMinus + " $1 WHERE uid=$2", c.points, userUID)
    if err != nil { log.Panic(err) }

    _, err = db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", status, cid)
    if err != nil { log.Panic(err) }

    addCompInteractionResponse(s, i, UpdateMsg, msgChallengee, []dg.MessageComponent{})

    dmcid, _ := s.UserChannelCreate(fmt.Sprintf("%v", c.challengerid))
    s.ChannelMessageSend(dmcid.ID, msgChallenger)
}

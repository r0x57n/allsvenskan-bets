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

func (b *botHolder) challengeCommand(i *dg.InteractionCreate) {
    msgOptions := getOptionsOrRespond(b.session, i, NewMsg)
    if msgOptions == nil { return }

    challengeeID := msgOptions[0]
    challengee, err := b.session.User(fmt.Sprintf("%v", challengeeID.Value))
    if err != nil { log.Panic(err) }

    challengeeUser := getUser(b.db, challengee.ID)
    interactionUID := getInteractUID(i)
    options := *getCurrentMatchesAsOptions(b.db, challengee.ID)

    currentChallenges := *getChallenges(b.db, "(challengeeid=$1 OR challengerid=$2) AND (status=$3 OR status=$4 OR status=$5 OR status=$6)",
                                        interactionUID, interactionUID, ChallengeStatusUnhandled, ChallengeStatusSent, ChallengeStatusAccepted, ChallengeStatusRequestForfeit)

    // Security checks
    errors := []CommandError{}
    if !challengeeUser.interactable {
        errors = append(errors, ErrorOtherNotInteractable)
    }

    if !getUserFromInteraction(b.db, i).interactable {
        errors = append(errors, ErrorSelfNotInteractable)
    }

    if strconv.Itoa(challengeeUser.uid) == interactionUID {
        errors = append(errors, ErrorInteractingWithSelf)
    }

    if len(currentChallenges) >= 25 {
        errors = append(errors, ErrorMaxChallenges)
    }

    if len(options) == 0 {
        errors = append(errors, ErrorNoMatches)
    }

    if len(errors) != 0 {
        addErrorsResponse(b.session, i, NewMsg, errors, "Det går inte att utmana användaren.")
        return
    }

    // Remove matches where a challenge already exists
    realOptions := []dg.SelectMenuOption{}
    for _, option := range options {
        matchid := strings.Split(option.Value, "_")[1]

        existingChallenge := getChallenge(b.db, "((challengerid=$1 AND challengeeid=$2) OR " +
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
                    CustomID: ChallSelectWinner,
                    Options: realOptions,
                },
            },
        },
    }

    msg := fmt.Sprintf("Utmana användare %v om hur följande match kommer sluta.", challengee.Username)
    addCompInteractionResponse(b.session, i, NewMsg, msg, components)
}

func (b *botHolder) challSelectWinner(i *dg.InteractionCreate) {
    values := getValuesOrRespond(b.session, i, UpdateMsg)
    if values == nil { return }
    value := values[0]

    splitted := strings.Split(value, "_")

    challengee, err := b.session.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchid := splitted[1]

    m := getMatch(b.db, "id=$1", matchid)

    val := challengee.ID + "_" + matchid + "_"

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: ChallSelectPoints,
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
    addCompInteractionResponse(b.session, i, UpdateMsg, msg, components)
}

func (b *botHolder) challSelectPoints(i *dg.InteractionCreate) {
    values := getValuesOrRespond(b.session, i, UpdateMsg)
    if values == nil { return }

    splitted := strings.Split(values[0], "_")
    challengee, err := b.session.User(splitted[0])
    if err != nil { log.Panic(err) }

    interactionUID := getInteractUID(i)

    var (
        challengerPoints = getUser(b.db, interactionUID).points
        challengeePoints = getUser(b.db, challengee.ID).points
        maxPoints = -1
    )

    if challengerPoints < challengeePoints {
        maxPoints = challengerPoints
    } else {
        maxPoints = challengeePoints
    }

    if maxPoints == 0 {
        addCompInteractionResponse(b.session, i, UpdateMsg, "Inga poäng att satsa med.", []dg.MessageComponent{})
        return
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: ChallAcceptDiscard,
                    Placeholder: "Poäng att satsa.",
                    Options: *getPointsAsOptions(values[0], maxPoints),
                },
            },
        },
    }

    msg := "Hur mycket poäng vill du satsa?\n"
    msg += "Du kan som mest satsa lika många poäng som motståndern har."
    addCompInteractionResponse(b.session, i, UpdateMsg, msg, components)
}

func (b *botHolder) challAcceptDiscard(i *dg.InteractionCreate) {
    values := getValuesOrRespond(b.session, i, UpdateMsg)
    if values == nil { return }
    splitted := strings.Split(values[0], "_")

    challengee, err := b.session.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchid := splitted[1]
    winnerTeam := splitted[2]
    loserTeam := ""
    points := splitted[3]

    var m match
    err = b.db.QueryRow("SELECT hometeam, awayteam FROM matches WHERE id=$1", matchid).Scan(&m.hometeam, &m.awayteam)
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
                    CustomID: ChallAcceptDiscardDo,
                    Placeholder: "Välj ett alternativ...",
                    Options: options,
                },
            },
        },
    }

    msg := fmt.Sprintf("\nDu vill utmana **%v** om att **%v** vinner mot **%v** för **%v** poäng.\n\n", challengee.Username, winnerTeam, loserTeam, points)
    msg += "Är du säker? En utmaning kan bara tas bort om den du utmanar accepterar borttagningen och" +
           " poängen kommer finnas hos vadhållaren tills dess att utmaningen är klar eller den du utmanat" +
           " har nekat vadet. Slutar matchen lika får båda parter tillbaka sina poäng."
    addCompInteractionResponse(b.session, i, UpdateMsg, msg, components)
}

func (b *botHolder) challAcceptDiscardDo(i *dg.InteractionCreate) {
    values := getValuesOrRespond(b.session, i, UpdateMsg)
    if values == nil { return }
    value := values[0]

    splitted := strings.Split(value, "_")

    if value == "discard" {
        addCompInteractionResponse(b.session, i, UpdateMsg, "Utmaningen är slängd", []dg.MessageComponent{})
        return
    }

    challengerUser := getUserFromInteraction(b.db, i)
    interactionUID := getInteractUID(i)

    challengee, err := b.session.User(splitted[0])
    if err != nil { log.Panic(err) }

    matchid := splitted[1]
    winnerTeam := splitted[2]
    points := splitted[3]
    challCondition := ChallengeConditionWinnerHome

    if winnerTeam == "awayteam" {
        challCondition = ChallengeConditionWinnerAway
    }

    m := getMatch(b.db, "id=$1", matchid)
    challengeeUser := getUser(b.db, challengee.ID)
    challenges := *getChallenges(b.db, "challengeeid=$1 AND (status=$2 OR status=$3 OR status=$4 OR status=$5)", challengee.ID, ChallengeStatusUnhandled, ChallengeStatusSent, ChallengeStatusAccepted, ChallengeStatusRequestForfeit)
    existingChallenge := getChallenge(b.db, "((challengerid=$1 AND challengeeid=$2) OR " +
                                            "(challengeeid=$3 AND challengerid=$4)) " +
                                            "AND matchid=$5 " +
                                            "AND status!=$6 AND status!=$7",
                                             interactionUID, challengee.ID,
                                             interactionUID, challengee.ID,
                                             matchid,
                                             ChallengeStatusDeclined, ChallengeStatusForfeited)
    pointsInt, err := strconv.Atoi(points)
    if err != nil { log.Panic(err) }

    // Security checks
    errors := []CommandError{}
    if !challengeeUser.interactable {
        errors = append(errors, ErrorOtherNotInteractable)
    }

    if strconv.Itoa(challengeeUser.uid) == interactionUID {
        errors = append(errors, ErrorInteractingWithSelf)
    }

    if existingChallenge.id != -1 {
        errors = append(errors, ErrorIdenticalChallenge)
    }

    if len(challenges) >= 25 {
        errors = append(errors, ErrorMaxChallenges)
    }

    if matchHasBegun(b.session, i, m) {
        errors = append(errors, ErrorMatchStarted)
    }

    if challengeeUser.points < pointsInt || challengerUser.points < pointsInt {
        errors = append(errors, ErrorNotEnoughPoints)
    }

    if len(errors) != 0 {
        addErrorsResponse(b.session, i, UpdateMsg, errors, "Det går inte att acceptera/avfärda utmaningen.")
        return
    }

    // Do stuff
    _, err = b.db.Exec("UPDATE users SET points=points-$1 WHERE uid=$2", points, interactionUID)
    if err != nil { log.Panic(err) }

    insertedCID := 0
    err = b.db.QueryRow("INSERT INTO challenges (challengerid, challengeeid, type, matchid, points, condition) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
                        interactionUID, challengee.ID, ChallengeTypeWinner, matchid, points, challCondition).Scan(&insertedCID)
    if err != nil { log.Panic(err) }

    msg := "Utmaning har skickats och poängen har plockats från ditt konto. "
    msg += "Nu är det upp till motståndaren att acceptera/neka utmaningen.\n\n"
    msg += "Om motståndaren inte accepterar/nekar utmaningen innan matchstart kommer poängen tillbaka till ditt konto."

    components := []dg.MessageComponent {}
    addCompInteractionResponse(b.session, i, UpdateMsg, msg, components)

    b.sendChallenge(interactionUID, challengee.ID, int(insertedCID), points)
}

func (b *botHolder) sendChallenge(challengerID string, challengeeid string, cid int, points string) {
    var m match
    var c challenge
    err := b.db.QueryRow("SELECT m.hometeam, m.awayteam, m.date, c.id, c.condition FROM challenges AS c " +
                         "JOIN matches AS m ON m.id = c.matchid " +
                         "WHERE c.id=$1", cid).Scan(&m.hometeam, &m.awayteam, &m.date, &c.id, &c.condition)
    if err != nil { log.Panic(err) }

    _, err = b.db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusSent, c.id)
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
                    CustomID: ChallAnswer,
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

    challenger, _ := b.session.User(challengerID)
    datetime, _ := time.Parse(DB_TIME_LAYOUT, m.date)

    msg := fmt.Sprintf("Du har blivit utmanad!\n")
    msg += fmt.Sprintf("**%v** tror att **%v** vinner mot **%v** den **%v** för **%v** poäng.\n\n",
                       challenger.Username, winner, loser, datetime.Format(MSG_TIME_LAYOUT), points)
    msg += fmt.Sprintf("Vill du satsa emot?\n\n")
    msg += fmt.Sprintf("*du kan stänga av utmaningar via **/inställningar** kommandot*")

    dmcid, _ := b.session.UserChannelCreate(challengeeid)
    b.session.ChannelMessageSendComplex(dmcid.ID, &dg.MessageSend{
        Content: msg,
        Components: components,
    })
}

func (b *botHolder) challAnswer(i *dg.InteractionCreate) {
    values := getValuesOrRespond(b.session, i, UpdateMsg)
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

    c := getChallenge(b.db, "id=$1", cid)
    m := getMatch(b.db, "id=$1", c.matchid)
    challengee := getUser(b.db, strconv.Itoa(c.challengeeid))
    challengeeUsername, _ := b.session.User(fmt.Sprint(challengee.uid))
    challenges := *getChallenges(b.db, "challengeeid=$1 AND (status=$2 OR status=$3 OR status=$4 OR status=$5)", challengee.uid, ChallengeStatusUnhandled, ChallengeStatusSent, ChallengeStatusAccepted, ChallengeStatusRequestForfeit)
    existingChallenge := getChallenge(b.db, "((challengerid=$1 AND challengeeid=$2) OR " +
                                            "(challengeeid=$3 AND challengerid=$4)) " +
                                            "AND matchid=$5 AND id!=$6 " +
                                            "AND status!=$7 AND status!=$8",
                                             c.challengerid, challengee.uid,
                                             c.challengerid, challengee.uid,
                                             c.matchid, c.id,
                                             ChallengeStatusDeclined, ChallengeStatusForfeited)

    // Security checks
    errors := []CommandError{}
    if c.status != ChallengeStatusSent {
        errors = append(errors, ErrorChallengeHandled)
    }

    if len(challenges) >= 25 {
        errors = append(errors, ErrorMaxChallenges)
    }

    if strconv.Itoa(challengee.uid) != getInteractUID(i) {
        errors = append(errors, ErrorNoRights)
    }

    if len(errors) != 0 {
        addErrorsResponse(b.session, i, UpdateMsg, errors, "Det går inte att acceptera/avfärda utmaningen.")
        return
    }

    if challengee.points < c.points {
        msgChallenger += "- Motståndandaren har inte tillräckligt med poäng för att acceptera utmaningen.\n"
        msgChallengee += "- Du har inte tillräckligt med poäng för att acceptera utmaningen.\n"
        answ = "decline"
    }

    if existingChallenge.id != -1 {
        msgChallenger += "- Du har redan en aktiv utmaning med användaren.\n"
        msgChallengee += "- Du har redan en aktiv utmaning med användaren.\n"
        answ = "decline"
    }

    if matchHasBegun(b.session, i, m) {
        msgChallenger += "- Motståndaren svarade försent.\n"
        msgChallengee += "- Matchen har redan startat.\n"
        answ = "decline"
    }

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

    _, err := b.db.Exec("UPDATE users SET points=points " + plusOrMinus + " $1 WHERE uid=$2", c.points, userUID)
    if err != nil { log.Panic(err) }

    _, err = b.db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", status, cid)
    if err != nil { log.Panic(err) }

    addCompInteractionResponse(b.session, i, UpdateMsg, msgChallengee, []dg.MessageComponent{})

    dmcid, _ := b.session.UserChannelCreate(fmt.Sprintf("%v", c.challengerid))
    b.session.ChannelMessageSend(dmcid.ID, msgChallenger)
}

package main

import (
    "log"
    "fmt"
    "time"
    "strings"
    "strconv"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func (b *botHolder) chickenCommand(i *dg.InteractionCreate) {
    uid := getInteractUID(i)

    now := time.Now().Format(DB_TIME_LAYOUT)
    rows, err := b.db.Query("SELECT c.id, c.challengerid, c.challengeeid, c.type, c.matchid, c.points, c.condition, c.status " +
                            "FROM challenges AS c " +
                            "JOIN matches AS m ON m.id=c.matchid " +
                            "WHERE m.date>=$1 AND " +
                            "((c.challengerid=$2 OR c.challengeeid=$3) " +
                            "OR (c.challengeeid=$4 OR c.challengerid=$5)) " +
                            "AND (c.status=$6 OR c.status=$7)",
                            now, uid, uid, uid, uid, ChallengeStatusSent, ChallengeStatusAccepted)
    if err != nil { log.Panic(err) }

    var challenges []challenge
    for rows.Next() {
        var c challenge

        err := rows.Scan(&c.id, &c.challengerid, &c.challengeeid, &c.typ, &c.matchid, &c.points, &c.condition, &c.status)
        if err != nil { log.Panic(err) }

        challenges = append(challenges, c)
    }

    if len(challenges) == 0 {
        addInteractionResponse(b.session, i, NewMsg, "Inga utmaningar gjorda!")
        return
    }

    options := []dg.SelectMenuOption{}
    for _, c := range challenges {
        msg := ""
        desc := ""
        hometeam := ""
        awayteam := ""
        username := ""
        m := getMatch(b.db, "id=$1", c.matchid)

        if strconv.Itoa(c.challengerid) == uid {
            challengee, _ := b.session.User(strconv.Itoa(c.challengeeid))
            username = challengee.Username

            if c.condition == ChallengeConditionWinnerHome {
                hometeam = "[" + m.hometeam + "]"
                awayteam = m.awayteam
            } else {
                hometeam = m.hometeam
                awayteam = "[" + m.awayteam + "]"
            }
        } else {
            challenger, _ := b.session.User(strconv.Itoa(c.challengerid))
            username = challenger.Username

            if c.condition == ChallengeConditionWinnerAway {
                hometeam = "[" + m.hometeam + "]"
                awayteam = m.awayteam
            } else {
                hometeam = m.hometeam
                awayteam = "[" + m.awayteam + "]"
            }
        }

        msg = fmt.Sprintf("%v vs %v för %v poäng", hometeam, awayteam, c.points)
        desc = fmt.Sprintf("%v - %v", username, m.date)

        options = append(options, dg.SelectMenuOption{
            Label: msg,
            Description: desc,
            Value: fmt.Sprintf("%v", c.id),
        })
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en utmaning",
                    CustomID: ChickenSelected,
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(b.session, i, NewMsg, "Dina utmaningar.", components)
}

func (b *botHolder) chickenChallengeSelected(i *dg.InteractionCreate) {
    interactionUID, _ := strconv.Atoi(getInteractUID(i))
    vals := getValuesOrRespond(b.session, i, UpdateMsg)
    if vals == nil { return }

    cid := vals[0]
    c := getChallenge(b.db, "id=$1", cid)
    m := getMatch(b.db, "id=$1", c.matchid)

    contactID := c.challengerid
    requesterIsChallenger := interactionUID == c.challengerid;
    if requesterIsChallenger {
        contactID = c.challengeeid
    }

    // Security checks
    errors := []CommandError{}

    if (c.challengerid != interactionUID && c.challengeeid != interactionUID) {
        errors = append(errors, ErrorNoRights)
    }

    if m.date <= time.Now().Format(DB_TIME_LAYOUT) {
        errors = append(errors, ErrorMatchStarted)
    }

    if len(errors) != 0 {
        addErrorsResponse(b.session, i, UpdateMsg, errors, "Kunde inte fega ur för utmaningen.")
        return
    }

    // Creates a DM channel (or fetches the existing one)
    channelID, _ := b.session.UserChannelCreate(strconv.Itoa(contactID))

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: ChickenAnswer,
                    Placeholder: "Välj ett alternativ...",
                    Options: []dg.SelectMenuOption{
                        {
                            Label: "Acceptera",
                            Value: fmt.Sprintf("accept_%v", c.id),
                        },
                        {
                            Label: "Neka",
                            Value: fmt.Sprintf("decline_%v", c.id),
                        },
                    },
                },
            },
        },
    }

    user, _ := b.session.User(getInteractUID(i))
    msg := fmt.Sprintf("**%v** vill avbryta om **%v** vs **%v** för **%v** poäng, vad vill du göra?",
                        user.Username, m.hometeam, m.awayteam, c.points)

    if c.status == ChallengeStatusSent {
        _, err := b.db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusDeclined, c.id)
        if err != nil { log.Panic(err) }

        _, err = b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, c.challengerid)
        if err != nil { log.Panic(err) }

        addCompInteractionResponse(b.session, i, UpdateMsg, "Utmaningen borttagen.", []dg.MessageComponent{})

        msg = fmt.Sprintf("**%v** avbröt utmaningen om **%v** vs **%v** för **%v** poäng.",
                            user.Username, m.hometeam, m.awayteam, c.points)

        b.session.ChannelMessageSendComplex(channelID.ID, &dg.MessageSend{
            Content: msg,
            Components: []dg.MessageComponent{},
        })

        return
    }

    _, err := b.db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusRequestForfeit, c.id)
    if err != nil { log.Panic(err) }

    addCompInteractionResponse(b.session, i, UpdateMsg, "Skickat förfrågan om att avbryta utmaningen.", []dg.MessageComponent{})

    b.session.ChannelMessageSendComplex(channelID.ID, &dg.MessageSend{
        Content: msg,
        Components: components,
    })
}

func (b *botHolder) chickenAnswer(i *dg.InteractionCreate) {
    vals := i.MessageComponentData().Values

    splitted := strings.Split(vals[0], "_")
    answer := splitted[0]
    cid := splitted[1]

    c := getChallenge(b.db, "id=$1", cid)
    m := getMatch(b.db, "id=$1", c.matchid)

    // Security checks
    errors := []CommandError{}

    interactionUID, _ := strconv.Atoi(getInteractUID(i))
    if (c.challengerid != interactionUID && c.challengeeid != interactionUID) {
        errors = append(errors, ErrorNoRights)
    }

    if m.date <= time.Now().Format(DB_TIME_LAYOUT) {
        errors = append(errors, ErrorMatchStarted)
    }

    if len(errors) != 0 {
        addErrorsResponse(b.session, i, UpdateMsg, errors, "Kunde slutföra förfrågan.")
        return
    }

    // Do things
    msgChicken := ""
    msgAcceptor := ""
    if answer == "accept" {
        msgChicken = fmt.Sprintf("Din förfrågan om att fega ur har blivit accepterad.")
        msgAcceptor = fmt.Sprintf("Skickar bekräftelse till fegisen.")

        _, err := b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, c.challengerid)
        if err != nil { log.Panic(err) }
        _, err = b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, c.challengeeid)
        if err != nil { log.Panic(err) }
        _, err = b.db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusForfeited, cid)
        if err != nil { log.Panic(err) }
    } else if answer == "decline" {
        msgChicken = fmt.Sprintf("Din förfrågan om att fega ur har blivit nekad.")
        msgAcceptor = fmt.Sprintf("Du har nekat fegisens förfrågan.")
        _, err := b.db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusAccepted, cid)
        if err != nil { log.Panic(err) }
    }

    chickenID, acceptorID := 0, getInteractUID(i)
    if acceptorID == strconv.Itoa(c.challengerid) {
        chickenID = c.challengeeid
    } else {
        chickenID = c.challengerid
    }

    channelID, _ := b.session.UserChannelCreate(strconv.Itoa(chickenID))
    b.session.ChannelMessageSend(channelID.ID, msgChicken)

    addCompInteractionResponse(b.session, i, UpdateMsg, msgAcceptor, []dg.MessageComponent{})
}

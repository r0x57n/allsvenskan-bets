package main

import (
    "log"
    "fmt"
    "strings"
    "strconv"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func (b *botHolder) chickenCommand(i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()
    s := b.session

    uid := getInteractUID(i)
    challenges := *getChallenges(db, "(challengerid=$1 OR challengeeid=$2) " +
                                     "AND (status=$3 OR status=$4)",
                                     uid, uid, ChallengeStatusSent, ChallengeStatusAccepted)

    if len(challenges) == 0 {
        addInteractionResponse(s, i, NewMsg, "Inga utmaningar gjorda!")
        return
    }

    options := []dg.SelectMenuOption{}
    for _, c := range challenges {
        msg := ""
        desc := ""
        hometeam := ""
        awayteam := ""
        username := ""
        m := getMatch(db, "id=$1", c.matchid)

        if strconv.Itoa(c.challengerid) == uid {
            challengee, _ := s.User(strconv.Itoa(c.challengeeid))
            username = challengee.Username

            if c.condition == ChallengeConditionWinnerHome {
                hometeam = "[" + m.hometeam + "]"
                awayteam = m.awayteam
            } else {
                hometeam = m.hometeam
                awayteam = "[" + m.awayteam + "]"
            }
        } else {
            challenger, _ := s.User(strconv.Itoa(c.challengerid))
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
                    Placeholder: "Välj en match",
                    CustomID: "chickenSelected", // component handler
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(s, i, NewMsg, "Dina utmaningar.", components)
}

func chickenSelected(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    interactionUID, _ := strconv.Atoi(getInteractUID(i))
    vals := getValuesOrRespond(s, i, UpdateMsg)
    if vals == nil { return }

    cid := vals[0]
    c := getChallenge(db, "id=$1", cid)

    contactID := c.challengerid
    requesterIsChallenger := interactionUID == c.challengerid;
    if requesterIsChallenger {
        contactID = c.challengeeid
    }

    // Creates a DM channel (or fetches the one existing)
    channelID, _ := s.UserChannelCreate(strconv.Itoa(contactID))

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "chickenAnswer",
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

    user, _ := s.User(getInteractUID(i))
    m := getMatch(db, "id=$1", c.matchid)
    msg := fmt.Sprintf("**%v** vill avbryta om **%v** vs **%v** för **%v** poäng, vad vill du göra?",
                        user.Username, m.hometeam, m.awayteam, c.points)

    if c.status == ChallengeStatusSent {
        _, err := db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusDeclined, c.id)
        if err != nil { log.Panic(err) }

        _, err = db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, c.challengerid)
        if err != nil { log.Panic(err) }

        addCompInteractionResponse(s, i, UpdateMsg, "Utmaningen borttagen.", []dg.MessageComponent{})

        msg = fmt.Sprintf("**%v** avbröt utmaningen om **%v** vs **%v** för **%v** poäng.",
                            user.Username, m.hometeam, m.awayteam, c.points)

        s.ChannelMessageSendComplex(channelID.ID, &dg.MessageSend{
            Content: msg,
            Components: []dg.MessageComponent{},
        })

        return
    }

    _, err := db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusRequestForfeit, c.id)
    if err != nil { log.Panic(err) }

    addCompInteractionResponse(s, i, UpdateMsg, "Skickat förfrågan om att avbryta utmaningen.", []dg.MessageComponent{})

    s.ChannelMessageSendComplex(channelID.ID, &dg.MessageSend{
        Content: msg,
        Components: components,
    })
}

func chickenAnswer(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    vals := i.MessageComponentData().Values

    splitted := strings.Split(vals[0], "_")
    answer := splitted[0]
    cid := splitted[1]

    c := getChallenge(db, "id=$1", cid)

    msgChicken := ""
    msgAcceptor := ""
    if answer == "accept" {
        msgChicken = fmt.Sprintf("Din förfrågan om att fega ur har blivit accepterad.")
        msgAcceptor = fmt.Sprintf("Skickar bekräftelse till fegisen.")

        _, err := db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, c.challengerid)
        if err != nil { log.Panic(err) }
        _, err = db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, c.challengeeid)
        if err != nil { log.Panic(err) }
        _, err = db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusForfeited, cid)
        if err != nil { log.Panic(err) }
    } else if answer == "decline" {
        msgChicken = fmt.Sprintf("Din förfrågan om att fega ur har blivit nekad.")
        msgAcceptor = fmt.Sprintf("Du har nekat fegisens förfrågan.")
        _, err := db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusAccepted, cid)
        if err != nil { log.Panic(err) }
    }

    chickenID, acceptorID := 0, getInteractUID(i)
    if acceptorID == strconv.Itoa(c.challengerid) {
        chickenID = c.challengeeid
    } else {
        chickenID = c.challengerid
    }

    channelID, _ := s.UserChannelCreate(strconv.Itoa(chickenID))
    s.ChannelMessageSend(channelID.ID, msgChicken)

    addCompInteractionResponse(s, i, UpdateMsg, msgAcceptor, []dg.MessageComponent{})
}

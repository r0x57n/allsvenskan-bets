package main

import (
    "log"
    "fmt"
    "strings"
    "strconv"
    _ "github.com/mattn/go-sqlite3"
    dg "github.com/bwmarrin/discordgo"
)

func chickenCommand(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    uid := getInteractUID(i)
    challenges := *getChallenges(db, "(challengerUID=? OR challengeeUID=?) " +
                                     "AND (status=? OR status=?)",
                                     uid, uid, Sent, Accepted)

    if len(challenges) == 0 {
        addInteractionResponse(s, i, NewMsg, "Inga utmaningar gjorda!")
        return
    }

    options := []dg.SelectMenuOption{}
    for _, c := range challenges {
        msg := ""
        desc := ""
        homeTeam := ""
        awayTeam := ""
        username := ""
        m := getMatch(db, "id=?", c.matchID)

        if strconv.Itoa(c.challengerUID) == uid {
            challengee, _ := s.User(strconv.Itoa(c.challengeeUID))
            username = challengee.Username

            if c.condition == "homeTeam" {
                homeTeam = "[" + m.homeTeam + "]"
                awayTeam = m.awayTeam
            } else {
                homeTeam = m.homeTeam
                awayTeam = "[" + m.awayTeam + "]"
            }
        } else {
            challenger, _ := s.User(strconv.Itoa(c.challengerUID))
            username = challenger.Username

            if c.condition == "awayTeam" {
                homeTeam = "[" + m.homeTeam + "]"
                awayTeam = m.awayTeam
            } else {
                homeTeam = m.homeTeam
                awayTeam = "[" + m.awayTeam + "]"
            }
        }

        msg = fmt.Sprintf("%v vs %v för %v poäng", homeTeam, awayTeam, c.points)
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
    c := getChallenge(db, "id=?", cid)

    contactID := c.challengerUID
    requesterIsChallenger := interactionUID == c.challengerUID;
    if requesterIsChallenger {
        contactID = c.challengeeUID
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
    m := getMatch(db, "id=?", c.matchID)
    msg := fmt.Sprintf("**%v** vill avbryta om **%v** vs **%v** för **%v** poäng, vad vill du göra?",
                        user.Username, m.homeTeam, m.awayTeam, c.points)

    if c.status == Sent {
        _, err := db.Exec("UPDATE challenges SET status=? WHERE id=?", Declined, c.id)
        if err != nil { log.Panic(err) }

        _, err = db.Exec("UPDATE users SET seasonPoints=seasonPoints + ? WHERE uid=?", c.points, c.challengerUID)
        if err != nil { log.Panic(err) }

        addCompInteractionResponse(s, i, UpdateMsg, "Utmaningen borttagen.", []dg.MessageComponent{})

        msg = fmt.Sprintf("**%v** avbröt utmaningen om **%v** vs **%v** för **%v** poäng.",
                            user.Username, m.homeTeam, m.awayTeam, c.points)

        s.ChannelMessageSendComplex(channelID.ID, &dg.MessageSend{
            Content: msg,
            Components: []dg.MessageComponent{},
        })

        return
    }

    _, err := db.Exec("UPDATE challenges SET status=? WHERE id=?", RequestForfeit, c.id)
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

    c := getChallenge(db, "id=?", cid)

    msgChicken := ""
    msgAcceptor := ""
    if answer == "accept" {
        msgChicken = fmt.Sprintf("Din förfrågan om att fega ur har blivit accepterad.")
        msgAcceptor = fmt.Sprintf("Skickar bekräftelse till fegisen.")

        _, err := db.Exec("UPDATE users SET seasonPoints=seasonPoints + ? WHERE uid=?", c.points, c.challengerUID)
        if err != nil { log.Panic(err) }
        _, err = db.Exec("UPDATE users SET seasonPoints=seasonPoints + ? WHERE uid=?", c.points, c.challengeeUID)
        if err != nil { log.Panic(err) }
        _, err = db.Exec("UPDATE challenges SET status=? WHERE id=?", Forfeited, cid)
        if err != nil { log.Panic(err) }
    } else if answer == "decline" {
        msgChicken = fmt.Sprintf("Din förfrågan om att fega ur har blivit nekad.")
        msgAcceptor = fmt.Sprintf("Du har nekat fegisens förfrågan.")
        _, err := db.Exec("UPDATE challenges SET status=? WHERE id=?", Accepted, cid)
        if err != nil { log.Panic(err) }
    }

    chickenID, acceptorID := 0, getInteractUID(i)
    if acceptorID == strconv.Itoa(c.challengerUID) {
        chickenID = c.challengeeUID
    } else {
        chickenID = c.challengerUID
    }

    channelID, _ := s.UserChannelCreate(strconv.Itoa(chickenID))
    s.ChannelMessageSend(channelID.ID, msgChicken)

    addCompInteractionResponse(s, i, UpdateMsg, msgAcceptor, []dg.MessageComponent{})
}

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
    challenges := *getChallenges(db, fmt.Sprintf("(challengerUID=%v OR challengeeUID=%v) AND status=%v", uid, uid, Accepted))

    if len(challenges) == 0 {
        addInteractionResponse(s, i, NewMsg, "Inga utmaningar gjorda!")
        return
    }

    options := []dg.SelectMenuOption{}
    for _, c := range challenges {
        msg := fmt.Sprintf("%v, matchvinnare %v för %v poäng", c.challengerUID, c.matchID, c.points)

        options = append(options, dg.SelectMenuOption{
            Label: msg,
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

    addCompInteractionResponse(s, i, UpdateMsg, "Skickat förfrågan om att avbryta utmaningen.", []dg.MessageComponent{})

    interactionUID, _ := strconv.Atoi(getInteractUID(i))
    cid := getValuesOrRespond(s, i, UpdateMsg)
    if cid == nil { return }

    c := getChallenge(db, fmt.Sprintf("id=%v", cid[0]))

    _, err := db.Exec("UPDATE challenges SET status=? WHERE id=?", RequestForfeit, c.id)
    if err != nil { log.Panic(err) }

    contactID := c.challengerUID

    requesterIsChallenger := interactionUID == c.challengerUID;
    if requesterIsChallenger {
        contactID = c.challengeeUID
    }

    dmcid, _ := s.UserChannelCreate(strconv.Itoa(contactID))

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

    msg := "Vill avbryta utmaning, vad vill du göra?"

    s.ChannelMessageSendComplex(dmcid.ID, &dg.MessageSend{
        Content: msg,
        Components: components,
    })
}

func chickenAnswer(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    vals := i.MessageComponentData().Values

    splitted := strings.Split(vals[0], "_")
    answ := splitted[0]
    cid := splitted[1]

    c := getChallenge(db, fmt.Sprintf("id=%v", cid))

    msgChicken := ""
    msgAcceptor := ""
    if answ == "accept" {
        msgChicken = fmt.Sprintf("Din förfrågan om att fega ur har blivit accepterad.")
        msgAcceptor = fmt.Sprintf("Skickar bekräftelse till fegisen.")

        _, err := db.Exec("UPDATE users SET seasonPoints=seasonPoints + ? WHERE uid=?", c.points, c.challengerUID)
        if err != nil { log.Panic(err) }
        _, err = db.Exec("UPDATE users SET seasonPoints=seasonPoints + ? WHERE uid=?", c.points, c.challengeeUID)
        if err != nil { log.Panic(err) }
        _, err = db.Exec("UPDATE challenges SET status=? WHERE id=?", Forfeited, cid)
        if err != nil { log.Panic(err) }
    } else if answ == "decline" {
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

    dmcid, _ := s.UserChannelCreate(strconv.Itoa(chickenID))
    s.ChannelMessageSend(dmcid.ID, msgChicken)

    addCompInteractionResponse(s, i, UpdateMsg, msgAcceptor, []dg.MessageComponent{})
}

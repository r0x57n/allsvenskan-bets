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

func chickenCommand(s *dg.Session, i *dg.InteractionCreate) {
    db, err := sql.Open(DB_TYPE, DB)
    defer db.Close()
    if err != nil { log.Fatal(err) }

    uid := getInteractUID(i)
    challenges := *getUserChallenges(db, uid)

    if len(challenges) == 0 {
        addInteractionResponse(s, i, NewMsg, "Inga utmaningar gjorda!")
        return
    }

    options := []dg.SelectMenuOption {}

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

    addCompInteractionResponse(s, i, dg.InteractionResponseChannelMessageWithSource, "Dina utmaningar.", components)
}

func chickenSelected(s *dg.Session, i *dg.InteractionCreate) {
    db, err := sql.Open(DB_TYPE, DB)
    defer db.Close()
    if err != nil { log.Fatal(err) }

    components := []dg.MessageComponent {}
    addCompInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, "Skickat förfrågan om att avbryta utmaningen.", components)

    cid := i.Interaction.MessageComponentData().Values[0]
    uid := getInteractUID(i)

    var c challenge
    err = db.QueryRow("SELECT id, challengerUID, challengeeUID, type, matchID, points, condition, status FROM challenges WHERE id=?", cid).
                 Scan(&c.id, &c.challengerUID, &c.challengeeUID, &c.typ, &c.matchID, &c.points, &c.condition, &c.status)
    if err != nil { log.Panic(err) }

    _, err = db.Exec("UPDATE challenges SET status=? WHERE id=?", RequestForfeit, c.id)

    contactID := 0
    if id,_ := strconv.Atoi(uid); id == c.challengerUID {
        contactID = c.challengeeUID
    } else {
        contactID = c.challengerUID
    }

    dmcid, _ := s.UserChannelCreate(strconv.Itoa(contactID))

    components = []dg.MessageComponent {
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
    db, err := sql.Open(DB_TYPE, DB)
    defer db.Close()
    if err != nil { log.Fatal(err) }

    vals := i.MessageComponentData().Values

    splitted := strings.Split(vals[0], "_")
    answ := splitted[0]
    cid := splitted[1]

    challenger, challengee := 0, 0
    points := 0
    err = db.QueryRow("SELECT challengerUID, challengeeUID, points FROM challenges WHERE id=?", cid).Scan(&challenger, &challengee, &points)

    msgChicken := ""
    msgAcceptor := ""
    if answ == "accept" {
        msgChicken = fmt.Sprintf("Din förfrågan om att fega ur har blivit accepterad.")
        msgAcceptor = fmt.Sprintf("Skickar bekräftelse till fegisen.")

        _, err = db.Exec("UPDATE users SET seasonPoints=seasonPoints + ? WHERE uid=?", points, challenger)
        if err != nil { log.Panic(err) }
        _, err = db.Exec("UPDATE users SET seasonPoints=seasonPoints + ? WHERE uid=?", points, challengee)
        if err != nil { log.Panic(err) }
        _, err = db.Exec("UPDATE challenges SET status=? WHERE id=?", Forfeited, cid)
        if err != nil { log.Panic(err) }
    } else if answ == "decline" {
        msgChicken = fmt.Sprintf("Din förfrågan om att fega ur har blivit nekad.")
        msgAcceptor = fmt.Sprintf("Du har nekat fegisens förfrågan.")
        _, err = db.Exec("UPDATE challenges SET status=? WHERE id=?", Accepted, cid)
        if err != nil { log.Panic(err) }
    }

    chickenID, acceptorID := 0, getInteractUID(i)
    if acceptorID == strconv.Itoa(challenger) {
        chickenID = challengee
    } else {
        chickenID = challenger
    }

    dmcid, _ := s.UserChannelCreate(strconv.Itoa(chickenID))
    s.ChannelMessageSend(dmcid.ID, msgChicken)

    components := []dg.MessageComponent {}
    addCompInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, msgAcceptor, components)
}

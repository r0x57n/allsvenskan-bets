package main

import (
    "log"
    _ "github.com/mattn/go-sqlite3"
    dg "github.com/bwmarrin/discordgo"
)

// Command: inställningar
func settingsCommand(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    uid := getInteractUID(i)
    u := getUser(db, uid)

    defOption := true
    if u.viewable == 0 {
        defOption = false
    }

    visibilityOptions := getAcceptDiscardOptions("Ja", "Nej", defOption)

    defOption = true
    if u.interactable == 0 {
        defOption = false
    }

    interactableOptions := getAcceptDiscardOptions("Ja", "Nej", defOption)

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "settingsVisibilityLabel",
                    Placeholder: "Låt andra kunna se dina tidigare bet",
                    Disabled: true,
                    Options: []dg.SelectMenuOption{
                        {
                            Label: "1",
                            Value: "1",
                        },
                    },
                },
            },
        },
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "settingsVisibility",
                    Options: visibilityOptions,
                },
            },
        },
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "settingsChallLabel",
                    Placeholder: "Låt andra kunna utmana dig",
                    Disabled: true,
                    Options: []dg.SelectMenuOption{
                        {
                            Label: "1",
                            Value: "1",
                        },
                    },
                },
            },
        },
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "settingsChall",
                    Options: interactableOptions,
                },
            },
        },
    }

    msg := "Inställningar för ditt konto."
    addCompInteractionResponse(s, i, NewMsg, msg, components)
}

func settingsVisibility(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    vals := getValuesOrRespond(s, i, UpdateMsg)
    if vals == nil { return }

    u := getUserFromInteraction(db, i)

    _, err := db.Exec("UPDATE users SET viewable=? WHERE uid=?", vals[0], u.uid)
    if err != nil { log.Panic(err) }

    ignoreInteraction(s, i)
}

func settingsChall(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    vals := getValuesOrRespond(s, i, UpdateMsg)
    if vals == nil { return }

    u := getUserFromInteraction(db, i)

    _, err := db.Exec("UPDATE users SET interactable=? WHERE uid=?", vals[0], u.uid)
    if err != nil { log.Panic(err) }

    ignoreInteraction(s, i)
}

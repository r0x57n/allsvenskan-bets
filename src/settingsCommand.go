package main

import (
    "log"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) settingsCommand(i *dg.InteractionCreate) {
    uid := getInteractUID(i)
    u := getUser(b.db, uid)

    defOption := true
    if !u.Viewable {
        defOption = false
    }

    visibilityOptions := []dg.SelectMenuOption{
        {
            Label: "Ja",
            Value: "1",
            Default: defOption,
        },
        {
            Label: "Nej",
            Value: "0",
            Default: !defOption,
        },
    }

    defOption = true
    if !u.Interactable {
        defOption = false
    }

    interactableOptions := []dg.SelectMenuOption{
        {
            Label: "Ja",
            Value: "1",
            Default: defOption,
        },
        {
            Label: "Nej",
            Value: "0",
            Default: !defOption,
        },
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "settingsVisibilityLabel",
                    Placeholder: "Låt andra kunna se dina tidigare vadslagningar",
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
                    CustomID: SettingsVisibility,
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
                    CustomID: SettingsChall,
                    Options: interactableOptions,
                },
            },
        },
    }

    msg := "Inställningar för ditt konto."
    addCompInteractionResponse(b.session, i, NewMsg, msg, components)
}

func (b *Bot) settingsVisibility(i *dg.InteractionCreate) {
    vals := getValuesOrRespond(b.session, i, UpdateMsg)
    if vals == nil { return }

    uid := getInteractUID(i)

    _, err := b.db.Exec("UPDATE users SET viewable=$1 WHERE uid=$2", vals[0], uid)
    if err != nil { log.Panic(err) }

    addNoInteractionResponse(b.session, i)
}

func (b *Bot) settingsChall(i *dg.InteractionCreate) {
    vals := getValuesOrRespond(b.session, i, UpdateMsg)
    if vals == nil { return }

    uid := getInteractUID(i)

    _, err := b.db.Exec("UPDATE users SET interactable=$1 WHERE uid=$2", vals[0], uid)
    if err != nil { log.Panic(err) }

    addNoInteractionResponse(b.session, i)
}

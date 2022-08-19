package main

import (
    "log"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func newSettings(b *botHolder) *Settings {
    cmd := new(Settings)
    cmd.bot = b
    cmd.name = HelpCommand
    cmd.description = "testar"
    cmd.addComponents()
    return cmd
}

func (cmd *Settings) addComponents() {
    cmd.bot.addComponent("settingsVisibility", cmd.settingsVisibility)
    cmd.bot.addComponent("settingsChall", cmd.settingsChall)
}

func (cmd *Settings) run(i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()
    s := cmd.bot.session

    uid := getInteractUID(i)
    u := getUser(db, uid)

    defOption := true
    if !u.viewable {
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
    if !u.interactable {
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

func (cmd *Settings) settingsVisibility(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    vals := getValuesOrRespond(s, i, UpdateMsg)
    if vals == nil { return }

    uid := getInteractUID(i)

    _, err := db.Exec("UPDATE users SET viewable=$1 WHERE uid=$2", vals[0], uid)
    if err != nil { log.Panic(err) }

    addNoInteractionResponse(s, i)
}

func (cmd *Settings) settingsChall(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    vals := getValuesOrRespond(s, i, UpdateMsg)
    if vals == nil { return }

    uid := getInteractUID(i)

    _, err := db.Exec("UPDATE users SET interactable=$1 WHERE uid=$2", vals[0], uid)
    if err != nil { log.Panic(err) }

    addNoInteractionResponse(s, i)
}

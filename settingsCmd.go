package main

import (
    "fmt"
    "log"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

// Command: inställningar
func settingsCommand(s *dg.Session, i *dg.InteractionCreate) {
    msg := "Inställningar för ditt konto."

    u := getUser(i.Interaction.Member.User.ID)

    visibOptions := []dg.SelectMenuOption{
                    {
                        Label: "Ja",
                        Value: "1",
                        Default: true,
                    },
                    {
                        Label: "Nej",
                        Value: "0",
                    },
    }

    if u.viewable == 0 {
        visibOptions = []dg.SelectMenuOption{
                        {
                            Label: "Ja",
                            Value: "1",
                        },
                        {
                            Label: "Nej",
                            Value: "0",
                            Default: true,
                        },
        }
    }

    interOptions := []dg.SelectMenuOption{
                    {
                        Label: "Ja",
                        Value: "1",
                        Default: true,
                    },
                    {
                        Label: "Nej",
                        Value: "0",
                    },
    }

    if u.interactable == 0 {
        interOptions = []dg.SelectMenuOption{
                        {
                            Label: "Ja",
                            Value: "1",
                        },
                        {
                            Label: "Nej",
                            Value: "0",
                            Default: true,
                        },
        }
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "settingsVisibilityLabel",
                    Placeholder: "Låt andra kunna se dina tidigare bet. \\/",
                    Options: []dg.SelectMenuOption{
                        {
                            Label: "Låt andra kunna se dina tidigare bet. \\/",
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
                    Options: visibOptions,
                },
            },
        },
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "settingsChallLabel",
                    Placeholder: "Låt andra kunna utmana dig. \\/",
                    Options: []dg.SelectMenuOption{
                        {
                            Label: "Låt andra kunna utmana dig. \\/",
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
                    Options: interOptions,
                },
            },
        },
    }

    compInteractionResponse(s, i, dg.InteractionResponseChannelMessageWithSource, msg, components)
}

func settingsVisibility(s *dg.Session, i *dg.InteractionCreate) {
	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

    vals := i.Interaction.MessageComponentData().Values
    if len(vals) == 0 { log.Panic("No options passed...") }

    uID := fmt.Sprint(i.Interaction.Member.User.ID)

    u := getUser(uID)

    _, err = betsDB.Exec("UPDATE points SET viewable=? WHERE uid=?", vals[0], u.uid)

    addInteractionResponse(s, i, dg.InteractionResponseDeferredMessageUpdate, "")
}

func settingsChall(s *dg.Session, i *dg.InteractionCreate) {
	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

    vals := i.Interaction.MessageComponentData().Values
    if len(vals) == 0 { log.Panic("No options passed...") }

    u := getUser(fmt.Sprint(i.Interaction.Member.User.ID))

    _, err = betsDB.Exec("UPDATE points SET interactable=? WHERE uid=?", vals[0], u.uid)

    addInteractionResponse(s, i, dg.InteractionResponseDeferredMessageUpdate, "")
}

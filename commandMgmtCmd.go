package main

import (
	"log"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

// Command: refresh
func updateCommand(s *dg.Session, i *dg.InteractionCreate) {
    if isOwner(i) {
        options := []dg.SelectMenuOption{
            {
                Label: "Alla",
                Value: "all",
            },
        }
        cmds, _ := s.ApplicationCommands(*APP_ID, *GUILD_ID)

        for _, cmd := range cmds {
            options = append(options, dg.SelectMenuOption{
                Label: cmd.Name,
                Value: cmd.Name,
                Description: "",
            })
        }

        components := []dg.MessageComponent {
            dg.ActionsRow {
                Components: []dg.MessageComponent {
                    dg.SelectMenu {
                        Placeholder: "Välj ett kommando att uppdatera",
                        CustomID: "updateCommandDo",
                        Options: options,
                    },
                },
            },
        }

        compInteractionResponse(s, i, dg.InteractionResponseChannelMessageWithSource, "Välj kommando att uppdatera:", components)
	} else {
        msgStdInteractionResponse(s, i, "Du har inte rättigheter att köra detta kommando...")
	}
}

func updateCommandDo(s *dg.Session, i *dg.InteractionCreate, COMMANDS *[]dg.ApplicationCommand) {
	if isOwner(i) {
        addInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, "Updating commands...")

        cmdName := i.Interaction.MessageComponentData().Values[0]

        log.Printf("Updating commands...")

        // Update one or all commands
        for _, cmd := range *COMMANDS {
            if cmd.Name == cmdName {
                _, err := s.ApplicationCommandCreate(*APP_ID, *GUILD_ID, &cmd)
                if err != nil {
                    log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
                }
                log.Printf("Updating only: %v", cmdName)
            } else if cmdName == "all" {
                _, err := s.ApplicationCommandCreate(*APP_ID, *GUILD_ID, &cmd)
                if err != nil {
                    log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
                }
                log.Printf("Updating: %v", cmd.Name)
            }
        }

        log.Println("Finished updating!")
	} else {
        msgStdInteractionResponse(s, i, "Du har inte rättigheter att köra detta kommando...")
	}
}

// Command: delete
func deleteCommand(s *dg.Session, i *dg.InteractionCreate) {
    if isOwner(i) {
        options := []dg.SelectMenuOption{}
        cmds, _ := s.ApplicationCommands(*APP_ID, *GUILD_ID)

        for _, cmd := range cmds {
            options = append(options, dg.SelectMenuOption{
                Label: cmd.Name,
                Value: cmd.ID,
                Description: "",
            })
        }

        components := []dg.MessageComponent {
            dg.ActionsRow {
                Components: []dg.MessageComponent {
                    dg.SelectMenu {
                        Placeholder: "Välj ett kommando att radera",
                        CustomID: "deleteCommandDo",
                        Options: options,
                    },
                },
            },
        }

        compInteractionResponse(s, i, dg.InteractionResponseChannelMessageWithSource, "Välj kommando att radera:", components)
	} else {
        msgStdInteractionResponse(s, i, "Du har inte rättigheter att köra detta kommando...")
	}
}

func deleteCommandDo(s *dg.Session, i *dg.InteractionCreate) {
	if isOwner(i) {
        val := i.MessageComponentData().Values[0]

        msgStdInteractionResponse(s, i, "Deleting: " + val)
        log.Printf("Deleting: %v", val)

        s.ApplicationCommandDelete(*APP_ID, *GUILD_ID, val)
	} else {
        msgStdInteractionResponse(s, i, "Du har inte rättigheter att köra detta kommando...")
	}
}

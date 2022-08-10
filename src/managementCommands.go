package main

import (
    "log"
    "fmt"
    _ "github.com/mattn/go-sqlite3"
    dg "github.com/bwmarrin/discordgo"
)


// Command: refresh
func updateCommand(s *dg.Session, i *dg.InteractionCreate) {
	if notOwner(s, i) { return }

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

    addCompInteractionResponse(s, i, dg.InteractionResponseChannelMessageWithSource, "Välj kommando att uppdatera:", components)
}

func updateCommandDo(s *dg.Session, i *dg.InteractionCreate, COMMANDS *[]dg.ApplicationCommand) {
    if notOwner(s, i) { return }

    components := []dg.MessageComponent {}
    addCompInteractionResponse(s, i, dg.InteractionResponseChannelMessageWithSource, "Uppdaterar kommandon...", components)

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
}

// Command: delete
func deleteCommand(s *dg.Session, i *dg.InteractionCreate) {
    if notOwner(s, i) { return }

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

    addCompInteractionResponse(s, i, dg.InteractionResponseChannelMessageWithSource, "Välj kommando att radera:", components)
}

func deleteCommandDo(s *dg.Session, i *dg.InteractionCreate) {
    if notOwner(s, i) { return }

    val := i.MessageComponentData().Values[0]

    s.ApplicationCommandDelete(*APP_ID, *GUILD_ID, val)
    addInteractionResponse(s, i, NewMsg, fmt.Sprintf("Deleted: %v", val))
    log.Printf("Deleted: %v", val)
}

package main

import (
	"fmt"
	"log"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

// Command: refresh
func updateCommand(s *dg.Session, i *dg.InteractionCreate, COMMANDS *[]dg.ApplicationCommand) {
	if isOwner(i) {
        log.Println("Update commands...")

        msgStdInteractionResponse(s, i, "Updating commands...")

        cID := ""

        if len(i.Interaction.ApplicationCommandData().Options) == 1 {
            cID = fmt.Sprintf("%v", i.Interaction.ApplicationCommandData().Options[0].Value)
        }

        // Initialize commands
        cmdIDs := make(map[string]string, len(*COMMANDS))

        for _, cmd := range *COMMANDS {
            if cmd.Name == cID {
                log.Printf("Adding: %v", cmd.Name)

                rcmd, err := s.ApplicationCommandCreate(*APP_ID, *GUILD_ID, &cmd)
                if err != nil {
                    log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
                }

                cmdIDs[rcmd.ID] = rcmd.Name
            } else if cID == "" {
                log.Printf("Adding: %v", cmd.Name)

                rcmd, err := s.ApplicationCommandCreate(*APP_ID, *GUILD_ID, &cmd)
                if err != nil {
                    log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
                }

                cmdIDs[rcmd.ID] = rcmd.Name
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
        if len(i.Interaction.ApplicationCommandData().Options) == 0 {
            log.Panic("No command given...")
        }

        cmdName := fmt.Sprintf("%v", i.Interaction.ApplicationCommandData().Options[0].Name)
        cmdID := fmt.Sprintf("%v", i.Interaction.ApplicationCommandData().Options[0].Value)

        msgStdInteractionResponse(s, i, "Deleting: " + cmdName)

        s.ApplicationCommandDelete(*APP_ID, *GUILD_ID, cmdID)
	} else {
        msgStdInteractionResponse(s, i, "Du har inte rättigheter att köra detta kommando...")
	}
}

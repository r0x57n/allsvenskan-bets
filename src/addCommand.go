package main

import (
    "log"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) addCommand(i *dg.InteractionCreate) {
    if b.notOwnerRespond(i) { return }

    options := []dg.SelectMenuOption{
        {
            Label: "Alla",
            Value: "all",
        },
    }

    cmds, _ := b.session.ApplicationCommands(b.appID, b.guildID)

    for _, cmd := range cmds {
        options = append(options, dg.SelectMenuOption{
            Label: cmd.Name,
            Value: cmd.Name,
        })
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj ett kommando att uppdatera",
                    CustomID: AddCommandDo,
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(b.session, i, dg.InteractionResponseChannelMessageWithSource, "Välj kommando att uppdatera:", components)
}

func (b *Bot) addCommandDo(i *dg.InteractionCreate) {
    if b.notOwnerRespond(i) { return }

    components := []dg.MessageComponent {}
    addCompInteractionResponse(b.session, i, dg.InteractionResponseChannelMessageWithSource, "Uppdaterar kommandon...", components)

    cmdName := i.Interaction.MessageComponentData().Values[0]

    log.Printf("Updating commands...")

    // Update one or all commands
    for _, c := range b.commands {
        cmd := dg.ApplicationCommand {
            Name: c.Name,
            Description: c.Description,
            Options: c.Options,
        }

        if cmd.Name == cmdName {
            _, err := b.session.ApplicationCommandCreate(b.appID, b.guildID, &cmd)
            if err != nil {
                log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
            }
            log.Printf("Updating only: %v", cmdName)
        } else if cmdName == "all" {
            _, err := b.session.ApplicationCommandCreate(b.appID, b.guildID, &cmd)
            if err != nil {
                log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
            }
            log.Printf("Updating: %v", cmd.Name)
        }
    }

    b.messageOwner("Klar med uppdatering av kommandon!")
    log.Println("Finished updating!")
}

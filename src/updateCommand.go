package main

import (
    "log"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func newUpdate(b *botHolder) *Update {
    cmd := new(Update)
    cmd.bot = b
    cmd.name = UpdateCommand
    cmd.description = "testar"
    cmd.addComponents()
    return cmd
}

func (cmd *Update) addComponents() {
    cmd.bot.addComponent("updateCommandDo", cmd.updateCommandDo)
}

func (cmd *Update) run(i *dg.InteractionCreate) {
    b := cmd.bot

    if b.notOwner(getInteractUID(i)) { return }

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

    addCompInteractionResponse(b.session, i, dg.InteractionResponseChannelMessageWithSource, "Välj kommando att uppdatera:", components)
}

func (cmd *Update) updateCommandDo(s *dg.Session, i *dg.InteractionCreate) {
    b := cmd.bot

    if b.notOwner(getInteractUID(i)) { return }

    components := []dg.MessageComponent {}
    addCompInteractionResponse(b.session, i, dg.InteractionResponseChannelMessageWithSource, "Uppdaterar kommandon...", components)

    cmdName := i.Interaction.MessageComponentData().Values[0]

    log.Printf("Updating commands...")

    // Update one or all commands
    for _, c := range COMMANDS {
        cmd := dg.ApplicationCommand {
            Name: c.name,
            Description: c.description,
            Options: c.options,
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

    log.Println("Finished updating!")
}

package main

import (
    "log"
    "fmt"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) removeCommand(i *dg.InteractionCreate) {
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
            Value: cmd.ID,
        })
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj ett kommando att radera",
                    CustomID: RemoveCommandDo,
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(b.session, i, dg.InteractionResponseChannelMessageWithSource, "Välj kommando att radera:", components)
}

func (b *Bot) removeCommandDo(i *dg.InteractionCreate) {
    if b.notOwnerRespond(i) { return }

    val := i.MessageComponentData().Values[0]

    if val == "all" {
        cmds, _ := b.session.ApplicationCommands(b.appID, b.guildID)
        for _, cmd := range cmds {
            b.session.ApplicationCommandDelete(b.appID, b.guildID, cmd.ID)
            log.Printf("Deleted: %v", cmd.ID)
        }

    } else {
        b.session.ApplicationCommandDelete(b.appID, b.guildID, val)
        addInteractionResponse(b.session, i, NewMsg, fmt.Sprintf("Deleted: %v", val))
        log.Printf("Deleted: %v", val)
    }
}

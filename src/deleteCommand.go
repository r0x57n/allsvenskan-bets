package main

import (
    "log"
    "fmt"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func (b *botHolder) deleteCommand(i *dg.InteractionCreate) {
    if b.notOwner(getInteractUID(i)) { return }

    options := []dg.SelectMenuOption{}
    cmds, _ := b.session.ApplicationCommands(b.appID, b.guildID)

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
                    CustomID: DeleteCommandDo,
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(b.session, i, dg.InteractionResponseChannelMessageWithSource, "Välj kommando att radera:", components)
}

func (b *botHolder) deleteCommandDo(s *dg.Session, i *dg.InteractionCreate) {
    if b.notOwner(getInteractUID(i)) { return }

    val := i.MessageComponentData().Values[0]

    s.ApplicationCommandDelete(b.appID, b.guildID, val)
    addInteractionResponse(s, i, NewMsg, fmt.Sprintf("Deleted: %v", val))
    log.Printf("Deleted: %v", val)
}

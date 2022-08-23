package main

import (
    "fmt"
    "log"
    dg "github.com/bwmarrin/discordgo"
    _ "github.com/lib/pq"
)

func (b *Bot) matchCommand(i *dg.InteractionCreate) {
    round := getCurrentRound(b.db)
    cmdOptions := i.Interaction.ApplicationCommandData().Options
    if len(cmdOptions) != 0 {
        round = int(cmdOptions[0].IntValue())

        if round == 0 {
            round = getCurrentRound(b.db)
        } else if round < 0 || round > 30 {
            addInteractionResponse(b.session, i, NewMsg, "Välj en omgång mellan 0-30.")
            return
        }
    }

    rows, err := b.db.Query("SELECT m.id, m.hometeam, m.awayteam, m.date, m.homescore, m.awayscore, m.finished " +
                            "FROM matches AS m " +
                            "WHERE round=$1", round)
    if err != nil { log.Panic(err) }

    options := *getOptionsOutOfRows(rows)

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en match",
                    CustomID: MatchSummarySend, // component handler
                    Options: options,
                },
            },
        },
    }

    title := fmt.Sprintf("Matcher för omgång %v", round)
    addCompInteractionResponse(b.session, i, NewMsg, title, components)
}

func (b *Bot) matchSummarySend(i *dg.InteractionCreate) {
    vals := getValuesOrRespond(b.session, i, NewMsg)
    if vals == nil { return }

    mid := vals[0]
    summary := b.getMatchSummary(mid)

    if err := b.session.InteractionRespond(i.Interaction, &dg.InteractionResponse {
        Type: NewMsg,
        Data: &dg.InteractionResponseData {
            Flags: 1 << 6, // Ephemeral
            Components: []dg.MessageComponent{},
            Embeds: []*dg.MessageEmbed {
                {
                    Title: "Gissningar",
                    Description: summary.Info,
                    Fields: summary.Fields,
                },
            },

        },
    }); err != nil { log.Panic(err) }
}

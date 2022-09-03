package main

import (
    "fmt"
    "log"
    "strings"
    "github.com/olekukonko/tablewriter"
    dg "github.com/bwmarrin/discordgo"
    _ "github.com/lib/pq"
)

func (b *Bot) tableCommand(i *dg.InteractionCreate) {
    rows, err := b.db.Query("SELECT placement, teamname, matches, wins, ties, losses, plusminus, points FROM leaguetable")
    if err != nil { log.Panic(err) }
    defer rows.Close()

    tableString := &strings.Builder{}
    table := tablewriter.NewWriter(tableString)
    table.SetHeader([]string{"#", "Namn (v/o/f)", "M", "Mål/Insl.", "Poäng"})

    for rows.Next() {
        var e TableEntry
        rows.Scan(&e.Placement, &e.Teamname, &e.Matches, &e.Wins, &e.Ties, &e.Losses, &e.Plusminus, &e.Points)
        table.Append([]string{
            fmt.Sprint(e.Placement),
            fmt.Sprintf("%v (%v/%v/%v)", e.Teamname, e.Wins, e.Ties, e.Losses),
            fmt.Sprint(e.Matches),
            e.Plusminus,
            fmt.Sprint(e.Points),
        })
    }

    table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
    table.SetCenterSeparator("|")
    table.Render()

	if err := b.session.InteractionRespond(i.Interaction, &dg.InteractionResponse {
		Type: NewMsg,
		Data: &dg.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
            Embeds: []*dg.MessageEmbed {
                {
                    Title: "Tabell",
                    Description: "`" + tableString.String() + "`",
                },
            },

		},
	}); err != nil { log.Panic(err) }
}

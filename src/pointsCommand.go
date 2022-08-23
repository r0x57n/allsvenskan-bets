package main

import (
	"fmt"
	"strconv"
	"log"
    _ "github.com/lib/pq"
	dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) pointsCommand(i *dg.InteractionCreate) {
	rows, err := b.db.Query("SELECT uid, points FROM users ORDER BY points DESC LIMIT 10")
	defer rows.Close()
	if err != nil { log.Panic(err) }

	top10 := ""
	pos := 0

	for rows.Next() {
		var (
			uid int
			season int
		)

        username := ""
		rows.Scan(&uid, &season)
		user, err := b.session.User(strconv.Itoa(uid))
        if err != nil {
            username = strconv.Itoa(uid)
        } else {
            username = user.Username
        }

        pos++

		top10 += fmt.Sprintf("#%v **%v** med %v poäng\n", pos, username, season)
	}

    if top10 == "" {
        top10 += "Inga spelare ännu!"
    }

    user := getUserFromInteraction(b.db, i)
	userPoints := fmt.Sprintf("Du har samlat ihop **%v** poäng i år!", user.Points)

    fields := []*dg.MessageEmbedField {
        {
            Name: "Top 10",
            Value: top10,
        },
    }

    addEmbeddedInteractionResponse(b.session, i, NewMsg, fields, "Poäng", userPoints)
}

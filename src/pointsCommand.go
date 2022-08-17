package main

import (
	"fmt"
	"strconv"
	"log"
    _ "github.com/lib/pq"
	dg "github.com/bwmarrin/discordgo"
)


// Command: poäng
func pointsCommand(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
	defer db.Close()

	rows, err := db.Query("SELECT uid, points FROM users ORDER BY points DESC LIMIT 10")
	defer rows.Close()
	if err != nil { log.Panic(err) }

	top10 := ""
	pos := 0

	for rows.Next() {
		var (
			uid int
			season int
		)

		rows.Scan(&uid, &season)
		user, _ := s.User(strconv.Itoa(uid))
        pos++

		top10 += fmt.Sprintf("#%v **%v** med %v poäng\n", pos, user.Username, season)
	}

    if top10 == "" {
        top10 += "Inga spelare ännu!"
    }

    user := getUserFromInteraction(db, i)
	userPoints := fmt.Sprintf("Du har samlat ihop **%v** poäng i år!", user.points)

    fields := []*dg.MessageEmbedField {
        {
            Name: "Top 10",
            Value: top10,
        },
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Poäng", userPoints)
}

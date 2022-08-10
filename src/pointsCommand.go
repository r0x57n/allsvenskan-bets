package main

import (
	"fmt"
	"strconv"
	"log"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)


// Command: poäng
func pointsCommand(s *dg.Session, i *dg.InteractionCreate) {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	rows, err := db.Query("SELECT uid, season FROM points ORDER BY season DESC LIMIT 10")
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

	uPoints := 0
	if err := db.QueryRow("SELECT season FROM points WHERE uid=?", getInteractUID(i)).Scan(&uPoints); err != nil {
		if err == sql.ErrNoRows {
			// skip
		} else {
			log.Panic(err)
		}
	}

	userPoints := fmt.Sprintf("Du har samlat ihop **%v** poäng i år!", uPoints)

    fields := []*dg.MessageEmbedField {
        {
            Name: "Top 10",
            Value: top10,
        },
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Poäng", userPoints)
}

package main

import (
	"math"
	"fmt"
	"log"
	"time"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

// Command: kommande
func upcomingCommand(s *dg.Session, i *dg.InteractionCreate) {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	uID := getInteractUID(i)

	bets, _ := db.Query("SELECT id, uid, matchid, homeScore, awayScore FROM bets WHERE uid=? AND handled=0", uID)
	defer bets.Close()

	var b bet

    type temp struct {
        homeTeam string
        awayTeam string
        homeScore int
        awayScore int
    }

    betsCount := 0
	userBets := ""
    matches := make(map[float64][]temp)

	for bets.Next() {
		bets.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore)
		matchRow := db.QueryRow("SELECT homeTeam, awayTeam, date FROM matches WHERE id=?", b.matchid)

		var m match
		matchRow.Scan(&m.homeTeam, &m.awayTeam, &m.date)

		date, _ := time.Parse(TIME_LAYOUT, m.date)
		daysUntil := math.Round(time.Until(date).Hours() / 24)

		//userBets = userBets + fmt.Sprintf("%v (**%v**) - %v (**%v**), spelas om %v dagar.\n", m.homeTeam, b.homeScore, m.awayTeam, b.awayScore, daysUntil)
		var t temp
        t.homeTeam = m.homeTeam
        t.awayTeam = m.awayTeam
        t.homeScore = b.homeScore
        t.awayScore = b.awayScore

        matches[daysUntil] = append(matches[daysUntil], t)

        betsCount++
	}

    fields := []*dg.MessageEmbedField {}

    for k, v := range matches {
        str := ""
        name := ""

        for _, e := range v {
            str += fmt.Sprintf("%v (**%v**) vs %v (**%v**)\n", e.homeTeam, e.homeScore, e.awayTeam, e.awayScore)
        }

        if k == -0 {
            name = fmt.Sprintf("Spelas nu")
        } else {
            name = fmt.Sprintf("%v dagar kvar", k)
        }

        fields = append(fields, &dg.MessageEmbedField{
            Name: name,
            Value: str,
        })
    }

	if betsCount == 0 {
		userBets = "Inga vadslagningar ännu!"
	}

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Kommande vad", userBets)
}

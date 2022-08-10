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

	uid := getInteractUID(i)

	betRows, err := db.Query("SELECT id, uid, matchid, homeScore, awayScore FROM bets WHERE uid=? AND handled=0", uid)
    if err != nil { log.Panic(err) }
	defer betRows.Close()

    type temp struct {
        homeTeam string
        awayTeam string
        homeScore int
        awayScore int
    }

    betsCount := 0
	userBets := ""
    matches := make(map[float64][]temp)

	for betRows.Next() {
        var b bet
		betRows.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore)

		var m match
		err := db.QueryRow("SELECT homeTeam, awayTeam, date FROM matches WHERE id=?", b.matchid).Scan(&m.homeTeam, &m.awayTeam, &m.date)
        if err != nil { log.Panic(err) }

		date, _ := time.Parse(DB_TIME_LAYOUT, m.date)
		daysUntil := math.Round(time.Until(date).Hours() / 24)

		var t temp
        t.homeTeam = m.homeTeam
        t.awayTeam = m.awayTeam
        t.homeScore = b.homeScore
        t.awayScore = b.awayScore

        matches[daysUntil] = append(matches[daysUntil], t)

        betsCount++
	}

    fields := []*dg.MessageEmbedField {}

    for days, matches := range matches {
        categoryMsg := ""
        matchesMsg := ""

        for _, m := range matches {
            matchesMsg += fmt.Sprintf("%v (**%v**) vs %v (**%v**)\n", m.homeTeam, m.homeScore, m.awayTeam, m.awayScore)
        }

        if days == -0 {
            categoryMsg = fmt.Sprintf("Spelas nu")
        } else {
            categoryMsg = fmt.Sprintf("%v dagar kvar", days)
        }

        fields = append(fields, &dg.MessageEmbedField{
            Name: categoryMsg,
            Value: matchesMsg,
        })
    }

	if betsCount == 0 {
		userBets = "Inga vadslagningar ännu!"
	}

    challengesMsg := ""
    challenges := *getChallenges(db, fmt.Sprintf("(challengerUID=%v OR challengeeUID=%v) AND status=%v", uid, uid, Accepted))

    if len(challenges) != 0 {
        for _, c := range challenges {
            challengesMsg += fmt.Sprintf("%v vs %v för %v poäng\n", c.challengerUID, c.challengeeUID, c.points)
        }

        fields = append(fields, &dg.MessageEmbedField{
            Name: "Utmaningar",
            Value: challengesMsg,
        })
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Kommande vad", userBets)
}

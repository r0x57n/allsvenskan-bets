package main

import (
    "sort"
	"fmt"
	"strconv"
	"log"
	"time"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

// Command: sammanfatta
func summaryCommand(s *dg.Session, i *dg.InteractionCreate) {
	if notOwner(s, i) { return }

	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	today := time.Now().Format("2006-01-02")

    round := -1
    err = db.QueryRow("SELECT round FROM matches WHERE date(date)>=? AND finished='0' ORDER BY date", today).Scan(&round)
    if err != nil { log.Panic(err) }

    var matches []match
    matchesRows, err := db.Query("SELECT id, homeTeam, awayTeam, date, scoreHome, scoreAway, finished FROM matches WHERE round=?", round)
    if err != nil { log.Panic(err) }

    won, lost := 0, 0
    err = db.QueryRow("SELECT COUNT(id) FROM bets WHERE round=? AND won=1 AND handled=1", round).Scan(&lost)
    if err != nil { log.Panic(err) }
    err = db.QueryRow("SELECT COUNT(id) FROM bets WHERE round=? AND won=0 AND handled=1", round).Scan(&won)
    if err != nil { log.Panic(err) }

    var bets []bet
    wins := make(map[int]int)

    for matchesRows.Next() {
        var m match
        matchesRows.Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date, &m.scoreHome, &m.scoreAway, &m.finished)
        matches = append(matches, m)

        betsRows, err := db.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE matchid=?", m.id)
        if err != nil { log.Panic(err) }

        for betsRows.Next() {
            var b bet
            betsRows.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore, &b.won)
            bets = append(bets, b)

            if b.won == 1 {
                wins[b.uid]++
            }
        }
    }

    // Top three wins
    topThree := "Dom med flest vinster är:\n"
    keys := make([]int, 0, len(wins))
    for k := range wins {
        keys = append(keys, k)
    }

    sort.Ints(keys)

    for i, k := range keys {
        if i <= 3 {
            username, _ := s.User(strconv.Itoa(k))
            topThree += fmt.Sprintf("#%v - %v med %v vinster\n", i + 1, username.Username, wins[k])
        }
    }

    msg := fmt.Sprintf("Denna omgång spelades **%v** matcher och **%v** vadslagningar las.\n\n", len(matches), len(bets))
    msg += fmt.Sprintf("**%v**:st vann sina vad medans **%v**:st förlorade.\n\n", won, lost)
    msg += topThree

    addInteractionResponse(s, i, NewMsg, msg)
}

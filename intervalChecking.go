package main

import (
	"log"
	"time"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func handleTodaysBets() {
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	log.Printf("Handling bets for %v...", yesterday)

	// Fetch all match IDs for today
	// date(date) tells SQLite to turn date='2006-01-02T00:00:00.00' to '2006-01-02'
	rows, err := svffDB.Query("SELECT id FROM matches WHERE date(date)=? AND finished=1", yesterday)
	if err != nil { log.Panic(err) }

	var matchIDs []int

	for rows.Next() {
		var matchID int
		rows.Scan(&matchID)
		matchIDs = append(matchIDs, matchID)
	}

	rows.Close()

	if len(matchIDs) == 0 {
		log.Print("No matches to handle!")
	} else {
		log.Printf("%v matches to handle...", len(matchIDs))

		// Handle bets for each match individually
		for _, mID := range matchIDs {
			var (
				rHomeScore int
				rAwayScore int
			)

			row := svffDB.QueryRow("SELECT scoreHome, scoreAway FROM matches WHERE id=?", mID)
			if err := row.Scan(&rHomeScore, &rAwayScore); err != nil { log.Panic(err) }

			betRows, err := betsDB.Query("SELECT id, uid, homeScore, awayScore FROM bets WHERE matchid=? and handled=0", mID)
			defer betRows.Close()
			if err != nil { log.Panic(err) }

			var bets []bet

			for betRows.Next() {
				var bet bet

				betRows.Scan(&bet.id, &bet.uid, &bet.homeScore, &bet.awayScore)

				bets = append(bets, bet)
			}

			betRows.Close()


			for _, bet := range bets {
				if rHomeScore == bet.homeScore && rAwayScore == bet.awayScore {
					addPoints(bet, 1)
				} else {
					addPoints(bet, 0)
				}
			}

			log.Printf("%v bets handled!", len(bets))
		}
	}
}

func addPoints(b bet, points int) {
	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

	row := betsDB.QueryRow("SELECT season FROM points WHERE uid=?", b.uid)
	if err != nil { log.Panic(err) }

	var currPoints int
	if err := row.Scan(&currPoints); err != nil {
		if err == sql.ErrNoRows {
			if _, err := betsDB.Exec("INSERT INTO points (uid, season) VALUES (?, ?)", b.uid, points); err != nil { log.Panic(err) }
		} else {
			log.Panic(err)
		}
	} else {
		if _, err := betsDB.Exec("UPDATE points SET season=season + ? WHERE uid=?", points, b.uid); err != nil { log.Panic(err) }
	}

	if _, err := betsDB.Exec("UPDATE bets SET handled=1 WHERE id=?", b.id); err != nil { log.Panic(err) }
}

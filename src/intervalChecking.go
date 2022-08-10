package main

import (
	"log"
	"time"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func checkUnhandledBets() {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	today := time.Now().Format("2006-01-02")

	log.Printf("Handling bets for %v...", today)

	// Fetch all match IDs for today
	// date(date) tells SQLite to turn date='2006-01-02T00:00:00.00' to '2006-01-02'
	rows, err := db.Query("SELECT id FROM matches WHERE date(date)=? AND finished=1", today)
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
		log.Printf("%v matches today...", len(matchIDs))

		// Handle bets for each match individually
		for _, mID := range matchIDs {
			var (
				rHomeScore int
				rAwayScore int
			)

			row := db.QueryRow("SELECT scoreHome, scoreAway FROM matches WHERE id=?", mID)
			if err := row.Scan(&rHomeScore, &rAwayScore); err != nil { log.Panic(err) }

			betRows, err := db.Query("SELECT id, uid, homeScore, awayScore FROM bets WHERE matchid=? and handled=0", mID)
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

			log.Printf("%v bets handled for match %v!", len(bets), mID)
		}
	}
}

func checkUnhandledChallenges() {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

    log.Printf("Checking for unhandled challenges...")

    challRows, err := db.Query("SELECT id, challengerUID, challengeeUID, type, matchID, points, condition, status FROM challenges WHERE status=?", Unhandled)
    defer challRows.Close()
	if err != nil { log.Panic(err) }

    var challs []challenge
    for challRows.Next() {
        var c challenge

        challRows.Scan(&c.id, &c.challengerUID, &c.challengeeUID, &c.typ, &c.matchID, &c.points, &c.condition, &c.status)
        challs = append(challs, c)
    }

    if len(challs) == 0 {
        return
    }

    //for _, c := range challs {

    //}

    log.Printf("Handled all challenges...")
}

func addPoints(b bet, points int) {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

    log.Printf("Awarding %v point to %v", points, b.uid)

	row := db.QueryRow("SELECT seasonPoints FROM users WHERE uid=?", b.uid)
	if err != nil { log.Panic(err) }

	var currPoints int
	if err := row.Scan(&currPoints); err != nil {
		if err == sql.ErrNoRows {
			if _, err := db.Exec("INSERT INTO users (uid, seasonPoints) VALUES (?, ?)", b.uid, points); err != nil { log.Panic(err) }
		} else {
			log.Panic(err)
		}
	} else {
		if _, err := db.Exec("UPDATE users SET seasonPoints=seasonPoints + ? WHERE uid=?", points, b.uid); err != nil { log.Panic(err) }
	}

	if _, err := db.Exec("UPDATE bets SET handled=1, won=? WHERE id=?", points, b.id); err != nil { log.Panic(err) }
}

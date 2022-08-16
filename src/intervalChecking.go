package main

import (
	"log"
	"time"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func checkUnhandledBets() {
    db := connectDB()
	defer db.Close()

	today := time.Now().Format("2006-01-02")

	log.Printf("Checking unhandled bets...")

    // Fetch all unhandled bets for matches in the past
    rows, err := db.Query("SELECT b.id, b.uid, b.matchid, b.homeScore, b.awayScore, b.handled, b.won, b.round, m.scoreHome, m.scoreAway " +
                          "FROM bets AS b " +
                          "JOIN matches AS m ON m.id=b.matchid " +
                          "WHERE date(m.date)<=? AND b.handled=? AND m.finished=?", today, 0, 1)
    defer rows.Close()
	if err != nil { log.Panic(err) }

    type betAndMatch struct {
        b bet
        m match
    }

    var bets []betAndMatch
	for rows.Next() {
        var b bet
        var m match
        rows.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore, &b.handled, &b.won, &b.round, &m.scoreHome, &m.scoreAway)
        bets = append(bets, betAndMatch{b: b, m: m})
	}

	if len(bets) == 0 {
		log.Print("No bets to handle!")
	} else {
		log.Printf("%v bets today...", len(bets))

		// Handle bets for each match individually
		for _, bam := range bets {
            if bam.m.scoreHome == bam.b.homeScore && bam.m.scoreAway == bam.b.awayScore {
                addPoints(bam.b, 1)
            } else {
                addPoints(bam.b, 0)
            }

		}

		log.Printf("%v bets handled!", len(bets))
	}
}

func checkUnhandledChallenges() {
    db := connectDB()
	defer db.Close()

	today := time.Now().Format("2006-01-02")

	log.Printf("Checking unhandled challenges...")

    // Fetch all unhandled bets for matches in the past
    rows, err := db.Query("SELECT c.id, c.challengerUID, c.challengeeUID, c.type, c.matchID, c.points, c.condition, c.status, c.round " +
                          "FROM challenges AS c " +
                          "JOIN matches AS m ON m.id=c.matchID " +
                          "WHERE date(m.date)<=? AND c.status=? AND m.finished=?", today, ChallengeStatusAccepted, 1)
    defer rows.Close()
	if err != nil { log.Panic(err) }

    type challengeAndMatch struct {
        c challenge
        m match
    }

    var challenges []challengeAndMatch
	for rows.Next() {
        var c challenge
        var m match
        rows.Scan(&c.id, &c.challengerUID, &c.challengeeUID, &c.typ, &c.matchID, &c.points, &c.condition, &c.status, &c.round, &m.scoreHome, &m.scoreAway)
        challenges = append(challenges, challengeAndMatch{c: c, m: m})
	}

	if len(challenges) == 0 {
		log.Print("No challenges to handle!")
	} else {
		log.Printf("%v challenges today...", len(challenges))

		// Handle bets for each match individually
		for _, cam := range challenges {
            if cam.c.typ == ChallengeTypeWinner {
                winnerUID := 0
                homeWon := cam.m.scoreHome > cam.m.scoreAway

                if cam.m.scoreHome == cam.m.scoreAway {
                    winnerUID = -1
                } else if homeWon {
                    if cam.c.condition == "homeTeam" {
                        winnerUID = cam.c.challengerUID
                    } else {
                        winnerUID = cam.c.challengeeUID
                    }
                } else {
                    if cam.c.condition == "awayTeam" {
                        winnerUID = cam.c.challengerUID
                    } else {
                        winnerUID = cam.c.challengeeUID
                    }
                }

                addPointsChallenge(winnerUID, cam.c)
            }
		}

		log.Printf("%v challenges handled!", len(challenges))
	}
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

func addPointsChallenge(winner int, c challenge) {
    db := connectDB()
	defer db.Close()

    log.Printf("Awarding %v point to %v", c.points, winner)

	row := db.QueryRow("SELECT seasonPoints FROM users WHERE uid=?", winner)

	var currPoints int
	if err := row.Scan(&currPoints); err != nil {
		if err == sql.ErrNoRows {
			if _, err := db.Exec("INSERT INTO users (uid, seasonPoints) VALUES (?, ?)", winner, c.points); err != nil { log.Panic(err) }
		} else {
			log.Panic(err)
		}
	} else {
		if _, err := db.Exec("UPDATE users SET seasonPoints=seasonPoints + ? WHERE uid=?", c.points, winner); err != nil { log.Panic(err) }
	}

	if _, err := db.Exec("UPDATE challenges SET status=? WHERE id=?", ChallengeStatusHandled, c.id); err != nil { log.Panic(err) }
}

package main

import (
	"log"
	"time"
	"database/sql"
    _ "github.com/lib/pq"
)

func (b *botHolder) checkUnhandledBets() {
	today := time.Now().Format("2006-01-02")

	log.Printf("Checking unhandled bets...")

    // Fetch all unhandled bets for matches in the past
    rows, err := b.db.Query("SELECT b.id, b.uid, b.matchid, b.homescore, b.awayscore, b.status, m.homescore, m.awayscore " +
                            "FROM bets AS b " +
                            "JOIN matches AS m ON m.id=b.matchid " +
                            "WHERE date(m.date)<=$1 AND b.status=$2", today, BetStatusUnhandled)
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
        rows.Scan(&b.id, &b.uid, &b.matchid, &b.homescore, &b.awayscore, &b.status, &m.homescore, &m.awayscore)
        bets = append(bets, betAndMatch{b: b, m: m})
	}

	if len(bets) == 0 {
		log.Print("No bets to handle!")
	} else {
		log.Printf("%v bets today...", len(bets))

		// Handle bets for each match individually
		for _, bam := range bets {
            if bam.m.homescore == bam.b.homescore && bam.m.awayscore == bam.b.awayscore {
                b.addPoints(bam.b, 1, BetStatusWon)
            } else {
                b.addPoints(bam.b, 0, BetStatusLost)
            }

		}

		log.Printf("%v bets handled!", len(bets))
	}
}

func (b *botHolder) checkUnhandledChallenges() {
	today := time.Now().Format("2006-01-02")

	log.Printf("Checking unhandled challenges...")

    // Fetch all unhandled bets for matches in the past
    rows, err := b.db.Query("SELECT c.id, c.challengerid, c.challengeeid, c.type, c.matchid, c.points, c.condition, c.status " +
                            "FROM challenges AS c " +
                            "JOIN matches AS m ON m.id=c.matchid " +
                            "WHERE date(m.date)<=$1 AND c.status=$2", today, ChallengeStatusAccepted)
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
        rows.Scan(&c.id, &c.challengerid, &c.challengeeid, &c.typ, &c.matchid, &c.points, &c.condition, &c.status, &m.homescore, &m.awayscore)
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
                homeWon := cam.m.homescore > cam.m.awayscore

                if cam.m.homescore == cam.m.awayscore {
                    winnerUID = -1
                } else if homeWon {
                    if cam.c.condition == ChallengeConditionWinnerHome {
                        winnerUID = cam.c.challengerid
                    } else {
                        winnerUID = cam.c.challengeeid
                    }
                } else {
                    if cam.c.condition == ChallengeConditionWinnerAway {
                        winnerUID = cam.c.challengerid
                    } else {
                        winnerUID = cam.c.challengeeid
                    }
                }

                b.addPointsChallenge(winnerUID, cam.c)
            }
		}

		log.Printf("%v challenges handled!", len(challenges))
	}
}

func (b *botHolder) addPoints(bet bet, points int, status BetStatus) {
    log.Printf("Awarding %v point to %v", points, bet.uid)

	row := b.db.QueryRow("SELECT points FROM users WHERE uid=$1", bet.uid)

	var currPoints int
	if err := row.Scan(&currPoints); err != nil {
		if err == sql.ErrNoRows {
			if _, err := b.db.Exec("INSERT INTO users (uid, points) VALUES ($1, $2)", bet.uid, points); err != nil { log.Panic(err) }
		} else {
			log.Panic(err)
		}
	} else {
		if _, err := b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", points, bet.uid); err != nil { log.Panic(err) }
	}

	if _, err := b.db.Exec("UPDATE bets SET status=$1 WHERE id=$2", status, bet.id); err != nil { log.Panic(err) }
}

func (b *botHolder) addPointsChallenge(winner int, c challenge) {
    log.Printf("Awarding %v point to %v", c.points, winner)

	row := b.db.QueryRow("SELECT points FROM users WHERE uid=$1", winner)

	var currPoints int
	if err := row.Scan(&currPoints); err != nil {
		if err == sql.ErrNoRows {
			if _, err := b.db.Exec("INSERT INTO users (uid, points) VALUES ($1, $2)", winner, c.points); err != nil { log.Panic(err) }
		} else {
			log.Panic(err)
		}
	} else {
		if _, err := b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, winner); err != nil { log.Panic(err) }
	}

	if _, err := b.db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusHandled, c.id); err != nil { log.Panic(err) }
}

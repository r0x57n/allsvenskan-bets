package main

import (
	"fmt"
	"log"
	"time"
    "math"
	_ "github.com/lib/pq"
)

func (b *botHolder) checkUnhandledBets(interactive bool) {
	today := time.Now().Format(DB_TIME_LAYOUT)

	log.Printf("Checking unhandled bets...")

    rows, err := b.db.Query("SELECT b.id, b.uid, b.matchid, b.homescore, b.awayscore, b.status, m.homescore, m.awayscore " +
                            "FROM bets AS b " +
                            "JOIN matches AS m ON m.id=b.matchid " +
                            "WHERE m.date<=$1 AND b.status=$2 AND m.finished=$3", today, BetStatusUnhandled, true)
    defer rows.Close()
	if err != nil { log.Panic(err) }

    bets := make(map[bet]match)
	for rows.Next() {
        var b bet
        var m match
        rows.Scan(&b.id, &b.uid, &b.matchid, &b.homescore, &b.awayscore, &b.status, &m.homescore, &m.awayscore)
        bets[b] = m
	}

	if len(bets) == 0 {
		log.Print("No bets to handle!")

        if interactive {
            b.messageOwner("No bets to handle!")
        }
	} else {
		log.Printf("%v bets today...", len(bets))

		// Handle bets for each match individually
		for bet, m := range bets {
            homeDiff := math.Abs(float64(bet.homescore - m.homescore))
            awayDiff := math.Abs(float64(bet.awayscore - m.awayscore))

            if homeDiff == 0 && awayDiff == 0 {
                b.addPoints(bet, 3, BetStatusWon, interactive)
            } else if (homeDiff == 1 && awayDiff == 0) || (homeDiff == 0 && awayDiff == 1) {
                b.addPoints(bet, 1, BetStatusAlmostWon, interactive)
            } else {
                b.addPoints(bet, 0, BetStatusLost, interactive)
            }

		}

		log.Printf("%v bets handled!", len(bets))
        if interactive {
            b.messageOwner(fmt.Sprintf("%v bets handled!", len(bets)))
        }
	}
}

func (b *botHolder) addPoints(bet bet, points int, status BetStatus, interactive bool) {
    log.Printf("Awarding %v point to %v", points, bet.uid)

    if interactive {
        b.messageOwner(fmt.Sprintf("Awarding %v points to %v", points, bet.uid))
    }

    _, err := b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", points, bet.uid)
    if err != nil { log.Panic(err) }

	_, err = b.db.Exec("UPDATE bets SET status=$1 WHERE id=$2", status, bet.id)
    if err != nil { log.Panic(err) }
}

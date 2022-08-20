package main

import (
	"fmt"
	"log"
	"time"
	_ "github.com/lib/pq"
)

func (b *botHolder) checkUnhandledChallenges() {
	today := time.Now().Format(DB_TIME_LAYOUT)

	log.Printf("Checking unhandled challenges...")

    // Fetch all unhandled bets for matches in the past
    rows, err := b.db.Query("SELECT c.id, c.challengerid, c.challengeeid, c.type, c.matchid, c.points, c.condition, c.status, " +
                            "m.homescore, m.awayscore " +
                            "FROM challenges AS c " +
                            "JOIN matches AS m ON m.id=c.matchid " +
                            "WHERE m.date<=$1 AND c.status=$2 AND m.finished=$3", today, ChallengeStatusAccepted, true)
	if err != nil { log.Panic(err) }
    defer rows.Close()

    challenges := make(map[challenge]match)
	for rows.Next() {
        var c challenge
        var m match
        rows.Scan(&c.id, &c.challengerid, &c.challengeeid, &c.typ, &c.matchid, &c.points, &c.condition, &c.status,
                  &m.homescore, &m.awayscore)
        challenges[c] = m
	}

	if len(challenges) == 0 {
		log.Print("No challenges to handle!")
        b.messageOwner("No challenges to handle!")
	} else {
		log.Printf("%v challenges today...", len(challenges))

		// Handle bets for each match individually
		for c, m := range challenges {
            if c.typ == ChallengeTypeWinner {
                winnerUID := 0
                homeWon := m.homescore > m.awayscore

                if m.homescore == m.awayscore {
                    winnerUID = -1
                } else if homeWon {
                    if c.condition == ChallengeConditionWinnerHome {
                        winnerUID = c.challengerid
                    } else {
                        winnerUID = c.challengeeid
                    }
                } else {
                    if c.condition == ChallengeConditionWinnerAway {
                        winnerUID = c.challengerid
                    } else {
                        winnerUID = c.challengeeid
                    }
                }

                b.addPointsChallenge(winnerUID, c)
            }
		}

        b.messageOwner(fmt.Sprintf("%v challenges handled!", len(challenges)))
		log.Printf("%v challenges handled!", len(challenges))
	}
}

func (b *botHolder) addPointsChallenge(winner int, c challenge) {
    if winner == -1 {
        log.Printf("Game ended in a tie, giving back points...")

        _, err := b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, c.challengerid)
        if err != nil { log.Panic(err) }
        _, err = b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, c.challengeeid)
        if err != nil { log.Panic(err) }
    } else {
        log.Printf("Awarding %v point to %v", c.points, winner)

        _, err := b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, winner)
        if err != nil { log.Panic(err) }
    }

	_, err := b.db.Exec("UPDATE challenges SET status=$1 WHERE id=$2", ChallengeStatusHandled, c.id)
    if err != nil { log.Panic(err) }
}

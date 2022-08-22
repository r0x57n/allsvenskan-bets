package main

import (
	"fmt"
	"log"
	"time"
	_ "github.com/lib/pq"
)

func (b *botHolder) checkUnhandledChallenges(interactive bool) {
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

        if interactive {
            b.messageOwner("No challenges to handle!")
        }
	} else {
		log.Printf("%v challenges today...", len(challenges))

		// Handle bets for each match individually
		for c, m := range challenges {
            if c.typ == ChallengeTypeWinner {
                homeWon := m.homescore > m.awayscore

                if m.homescore == m.awayscore {
                    c.winner = ChallengeWinnerNone
                } else if homeWon {
                    if c.condition == ChallengeConditionWinnerHome {
                        c.winner = ChallengeWinnerChallenger
                    } else {
                        c.winner = ChallengeWinnerChallengee
                    }
                } else {
                    if c.condition == ChallengeConditionWinnerAway {
                        c.winner = ChallengeWinnerChallenger
                    } else {
                        c.winner = ChallengeWinnerChallengee
                    }
                }

                b.addPointsChallenge(c, interactive)
            }
		}

		log.Printf("%v challenges handled!", len(challenges))
        if interactive {
            b.messageOwner(fmt.Sprintf("%v challenges handled!", len(challenges)))
        }
	}
}

func (b *botHolder) addPointsChallenge(c challenge, interactive bool) {
    winnerUID := 0

    if c.winner == ChallengeWinnerNone {
        log.Printf("Game ended (%v - %v) in a tie, giving back points...", c.challengerid, c.challengeeid)
        if interactive {
            b.messageOwner(fmt.Sprintf("Game ended (%v - %v) in a tie, giving back points...", c.challengerid, c.challengeeid))
        }

        _, err := b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, c.challengerid)
        if err != nil { log.Panic(err) }
        _, err = b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points, c.challengeeid)
        if err != nil { log.Panic(err) }
    } else {
        if c.winner == ChallengeWinnerChallenger {
            winnerUID = c.challengerid
        } else if c.winner == ChallengeWinnerChallengee {
            winnerUID = c.challengeeid
        } else {
            log.Fatalf("Got %v as winnerUID when checking challenges.", winnerUID)
        }

        log.Printf("Awarding %v point to %v", c.points, winnerUID)
        if interactive {
            b.messageOwner(fmt.Sprintf("Awarding %v points to %v", c.points, winnerUID))
        }

        _, err := b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.points*2, winnerUID)
        if err != nil { log.Panic(err) }
    }

	_, err := b.db.Exec("UPDATE challenges SET status=$1, winner=$2 WHERE id=$3", ChallengeStatusHandled, c.winner, c.id)
    if err != nil { log.Panic(err) }
}

package main

import (
	"fmt"
	"log"
	"time"
	_ "github.com/lib/pq"
)

func (b *Bot) handleChallenges(interactive bool) {
	today := time.Now().Format(DB_TIME_LAYOUT)

	log.Printf("Checking unhandled challenges...")

    // Fetch all unhandled bets for matches in the past
    rows, err := b.db.Query("SELECT c.id, c.challengerid, c.challengeeid, c.type, c.matchid, c.points, c.condition, c.status, " +
                            "m.id, m.homescore, m.awayscore " +
                            "FROM challenges AS c " +
                            "JOIN matches AS m ON m.id=c.matchid " +
                            "WHERE m.date<=$1 AND c.status=$2 AND m.finished=$3", today, ChallengeStatusAccepted, true)
	if err != nil { log.Panic(err) }
    defer rows.Close()

    challenges := make(map[Challenge]Match)
    matchesChecked := make(map[Match]bool)
	for rows.Next() {
        var c Challenge
        var m Match
        rows.Scan(&c.ID, &c.ChallengerID, &c.ChallengeeID, &c.Type, &c.MatchID, &c.Points, &c.Condition, &c.Status,
                  &m.ID, &m.HomeScore, &m.AwayScore)
        challenges[c] = m
        matchesChecked[m] = false
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
            if c.Type == ChallengeTypeWinner {
                homeWon := m.HomeScore > m.AwayScore

                if m.HomeScore == m.AwayScore {
                    c.Winner = ChallengeWinnerNone
                } else if homeWon {
                    if c.Condition == ChallengeConditionWinnerHome {
                        c.Winner = ChallengeWinnerChallenger
                    } else {
                        c.Winner = ChallengeWinnerChallengee
                    }
                } else {
                    if c.Condition == ChallengeConditionWinnerAway {
                        c.Winner = ChallengeWinnerChallenger
                    } else {
                        c.Winner = ChallengeWinnerChallengee
                    }
                }

                b.addPointsChallenge(c, interactive)
            }
		}

        for m := range matchesChecked {
            b.db.Exec("UPDATE matches SET checkedchallenges=$1 WHERE id=$2", true, m.ID)
        }

		log.Printf("%v challenges handled!", len(challenges))
        if interactive {
            b.messageOwner(fmt.Sprintf("%v challenges handled!", len(challenges)))
        }
	}
}

func (b *Bot) addPointsChallenge(c Challenge, interactive bool) {
    winnerUID := 0

    if c.Winner == ChallengeWinnerNone {
        log.Printf("Game ended (%v - %v) in a tie, giving back points...", c.ChallengerID, c.ChallengeeID)
        if interactive {
            b.messageOwner(fmt.Sprintf("Game ended (%v - %v) in a tie, giving back points...", c.ChallengerID, c.ChallengeeID))
        }

        _, err := b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.Points, c.ChallengerID)
        if err != nil { log.Panic(err) }
        _, err = b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.Points, c.ChallengeeID)
        if err != nil { log.Panic(err) }
    } else {
        if c.Winner == ChallengeWinnerChallenger {
            winnerUID = c.ChallengerID
        } else if c.Winner == ChallengeWinnerChallengee {
            winnerUID = c.ChallengeeID
        } else {
            log.Fatalf("Got %v as winnerUID when checking challenges.", winnerUID)
        }

        log.Printf("Awarding %v point to %v", c.Points, winnerUID)
        if interactive {
            b.messageOwner(fmt.Sprintf("Awarding %v points to %v", c.Points, winnerUID))
        }

        _, err := b.db.Exec("UPDATE users SET points=points+$1 WHERE uid=$2", c.Points*2, winnerUID)
        if err != nil { log.Panic(err) }
    }

	_, err := b.db.Exec("UPDATE challenges SET status=$1, winner=$2 WHERE id=$3", ChallengeStatusHandled, c.Winner, c.ID)
    if err != nil { log.Panic(err) }
}

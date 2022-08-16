package main

import (
	"math"
	"fmt"
	"log"
	"time"
    "strconv"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

// Command: kommande
func upcomingCommand(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
	defer db.Close()

	uid := getInteractUID(i)

	betRows, err := db.Query("SELECT b.homeScore, b.awayScore, m.homeTeam, m.awayTeam, m.date " +
                             "FROM bets AS b " +
                             "JOIN matches AS m ON b.matchid=m.id " +
                             "WHERE b.uid=? AND b.handled=0", uid)
	defer betRows.Close()
    if err != nil { log.Panic(err) }

    type betAndMatch struct {
        b bet
        m match
    }

    betsCount := 0
	userBets := ""
    matches := make(map[float64][]betAndMatch)

	for betRows.Next() {
        var b bet
		var m match
		betRows.Scan(&b.homeScore, &b.awayScore,
                     &m.homeTeam, &m.awayTeam, &m.date)

		date, _ := time.Parse(DB_TIME_LAYOUT, m.date)
		daysUntil := math.Round(time.Until(date).Hours() / 24)

		betAMatch := betAndMatch{ b: b, m: m }

        matches[daysUntil] = append(matches[daysUntil], betAMatch)

        betsCount++
	}

    fields := []*dg.MessageEmbedField {}

    for daysUntil, matches := range matches {
        categoryMsg := ""
        matchesMsg := ""

        for _, m := range matches {
            matchesMsg += fmt.Sprintf("%v (**%v**) vs %v (**%v**)\n", m.m.homeTeam, m.b.homeScore,
                                                                      m.m.awayTeam, m.b.awayScore)
        }

        if math.Signbit(daysUntil) {
            categoryMsg = fmt.Sprintf("Spelas nu")
        } else if daysUntil == 0 {
            categoryMsg = fmt.Sprintf("Spelas idag")
        } else {
            categoryMsg = fmt.Sprintf("%v dagar kvar", daysUntil)
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
    rows, err := db.Query("SELECT c.challengerUID, c.challengeeUID, c.points, c.condition, " +
                          "m.homeTeam, m.awayTeam " +
                          "FROM challenges AS c " +
                          "JOIN matches AS m ON c.matchID=m.id " +
                          "WHERE (c.challengerUID=? OR c.challengeeUID=?) AND c.status=?", uid, uid, ChallengeStatusAccepted)
    defer rows.Close()
    if err != nil { log.Panic(err) }

    type challAndMatch struct {
        c challenge
        m match
    }

    var challenges []challAndMatch

    for rows.Next() {
        var c challenge
        var m match

        rows.Scan(&c.challengerUID, &c.challengeeUID, &c.points, &c.condition,
                  &m.homeTeam, &m.awayTeam)

        challenges = append(challenges, challAndMatch{c: c, m: m})
    }

    if len(challenges) != 0 {
        for _, c := range challenges {
            challenger, err := s.User(strconv.Itoa(c.c.challengerUID))
            if err != nil { log.Panic(err) }
            challengee, err := s.User(strconv.Itoa(c.c.challengeeUID))
            if err != nil { log.Panic(err) }

            winOrLose := "vinner"
            if c.c.condition == "awayTeam" {
                winOrLose = "förlorar"
            }

            challengesMsg += fmt.Sprintf("**%v** utmanar **%v** om att **%v** %v mot **%v** för **%v** poäng\n",
                                        challenger.Username, challengee.Username, c.m.homeTeam, winOrLose, c.m.awayTeam, c.c.points)
        }

        fields = append(fields, &dg.MessageEmbedField{
            Name: "Utmaningar",
            Value: challengesMsg,
        })
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Kommande vadslag", userBets)
}

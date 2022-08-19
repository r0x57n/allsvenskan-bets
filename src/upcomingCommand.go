package main

import (
	"math"
	"fmt"
	"log"
	"time"
    "strconv"
    _ "github.com/lib/pq"
	dg "github.com/bwmarrin/discordgo"
)

func newUpcoming(b *botHolder) *Upcoming {
    cmd := new(Upcoming)
    cmd.bot = b
    cmd.name = UpcomingCommand
    cmd.description = "testar"
    return cmd
}

func (cmd *Upcoming) run(i *dg.InteractionCreate) {
    s := cmd.bot.session
    db := connectDB()
	defer db.Close()

	uid := getInteractUID(i)

	betRows, err := db.Query("SELECT b.homescore, b.awayscore, m.hometeam, m.awayteam, m.date " +
                             "FROM bets AS b " +
                             "JOIN matches AS m ON b.matchid=m.id " +
                             "WHERE b.uid=$1 AND b.status=$2", uid, BetStatusUnhandled)
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
		betRows.Scan(&b.homescore, &b.awayscore,
                     &m.hometeam, &m.awayteam, &m.date)

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
            matchesMsg += fmt.Sprintf("%v (**%v**) vs %v (**%v**)\n", m.m.hometeam, m.b.homescore,
                                                                      m.m.awayteam, m.b.awayscore)
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
    rows, err := db.Query("SELECT c.challengerid, c.challengeeid, c.points, c.condition, " +
                          "m.hometeam, m.awayteam " +
                          "FROM challenges AS c " +
                          "JOIN matches AS m ON c.matchid=m.id " +
                          "WHERE (c.challengerid=$1 OR c.challengeeid=$2) AND c.status=$3", uid, uid, ChallengeStatusAccepted)
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

        rows.Scan(&c.challengerid, &c.challengeeid, &c.points, &c.condition,
                  &m.hometeam, &m.awayteam)

        challenges = append(challenges, challAndMatch{c: c, m: m})
    }

    if len(challenges) != 0 {
        for _, c := range challenges {
            challenger, err := s.User(strconv.Itoa(c.c.challengerid))
            if err != nil { log.Panic(err) }
            challengee, err := s.User(strconv.Itoa(c.c.challengeeid))
            if err != nil { log.Panic(err) }

            winOrLose := "vinner"
            if c.c.condition == ChallengeConditionWinnerAway {
                winOrLose = "förlorar"
            }

            challengesMsg += fmt.Sprintf("**%v** utmanar **%v** om att **%v** %v mot **%v** för **%v** poäng\n",
                                        challenger.Username, challengee.Username, c.m.hometeam, winOrLose, c.m.awayteam, c.c.points)
        }

        fields = append(fields, &dg.MessageEmbedField{
            Name: "Utmaningar",
            Value: challengesMsg,
        })
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Kommande vadslag", userBets)
}

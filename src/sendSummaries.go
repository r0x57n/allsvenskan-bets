package main

import (
    "fmt"
    "log"
    "strconv"
    "time"
    "database/sql"
    dg "github.com/bwmarrin/discordgo"
    _ "github.com/lib/pq"
)

func (b *botHolder) sendSummaries() {
	today := time.Now().Format(DB_TIME_LAYOUT)
    rows, err := b.db.Query("SELECT id FROM matches " +
                            "WHERE date<=$1 AND finished=$2 AND summarised=$3", today, true, false)
    if err != nil {
        if err != sql.ErrNoRows {
            log.Panic(err)
        } else {
            return
        }
    }

    var ids []int
    for rows.Next() {
        var mid int
        rows.Scan(&mid)
        ids = append(ids, mid)
    }

    for _, mid := range ids {
        m := getMatch(b.db, "id=$1", mid)

        matchInfo := ""
        if m.finished {
            matchInfo = fmt.Sprintf("%v - %v\n", m.hometeam, m.awayteam)
            matchInfo += fmt.Sprintf("Resultat: %v - %v\n\n", m.homescore, m.awayscore)
        }

        // all bets
        betRows, err := b.db.Query("SELECT b.id, b.uid, b.matchid, b.homescore, b.awayscore, b.status, b.round " +
                                "FROM bets AS b " +
                                "JOIN users AS u ON u.uid=b.uid " +
                                "WHERE b.matchid=$1 AND u.viewable=$2", m.id, true)
        if err != nil { log.Panic(err) }
        bets := *getBetsFromRows(betRows)

        msgBets := ""
        if len(bets) == 0 {
            msgBets = "-"
        }

        for _, bet := range bets {
            user, _ := b.session.User(strconv.Itoa(bet.uid))
            username := ""
            if user == nil {
                username = strconv.Itoa(bet.uid)
            } else {
                username = user.Username
            }

            if m.finished {
                won := bet.homescore == m.homescore && bet.awayscore == m.awayscore

                if won {
                    msgBets += fmt.Sprintf("**%v gissade på %v - %v**\n", username, bet.homescore, bet.awayscore)
                } else {
                    msgBets += fmt.Sprintf("%v gissade på %v - %v\n", username, bet.homescore, bet.awayscore)
                }
            } else {
                msgBets += fmt.Sprintf("%v gissar på %v - %v\n", username, bet.homescore, bet.awayscore)
            }
        }

        // all challenges
        challenges := *getChallenges(b.db, "matchid=$1 AND (status=$2 OR status=$3)",
                                    mid, ChallengeStatusHandled, ChallengeStatusAccepted)

        msgChalls := ""
        if len(challenges) == 0 {
            msgChalls = "-"
        }

        for _, c := range challenges {
            userChallenger, _ := b.session.User(strconv.Itoa(c.challengerid))
            userChallengee, _ := b.session.User(strconv.Itoa(c.challengeeid))
            usernameChallenger := ""
            usernameChallengee := ""
            if userChallenger == nil {
                usernameChallenger = strconv.Itoa(c.challengerid)
            } else {
                usernameChallenger = userChallenger.Username
            }

            if userChallengee == nil {
                usernameChallengee = strconv.Itoa(c.challengeeid)
            } else {
                usernameChallengee = userChallengee.Username
            }

            winner := ""
            if c.condition == ChallengeConditionWinnerHome {
                winner = m.hometeam
            } else {
                winner = m.awayteam
            }

            if m.finished {
                msgChalls += fmt.Sprintf("%v utmanade %v om att %v skulle vinna för %v poäng\n",
                                        usernameChallenger, usernameChallengee, winner, c.points)
            } else {
                msgChalls += fmt.Sprintf("%v utmanar %v om att %v ska vinna för %v poäng\n",
                                        usernameChallenger, usernameChallengee, winner, c.points)
            }
        }

        fields := []*dg.MessageEmbedField {
            {
                Name: "Gissningar",
                Value: msgBets,
            },
            {
                Name: "Utmaningar",
                Value: msgChalls,
            },
            {
                Name: "-",
                Value: "*använd /hjälp kommandot för att lära dig tippa*",
            },
        }

        channelID := "-1"
        channels, _ := b.session.GuildChannels(b.allsvenskanGuildID)
        for _, ch := range channels {
            if ch.Name == "bets" {
                channelID = ch.ID
            }
        }
        b.session.ChannelMessageSendComplex(channelID, &dg.MessageSend{
            Embeds: []*dg.MessageEmbed {
                {
                    Title: "Match färdigspelad!",
                    Description: matchInfo,
                    Fields: fields,
                },
            },
        })

        b.db.Exec("UPDATE matches SET summarised=$1 WHERE id=$2", true, mid)
    }
}

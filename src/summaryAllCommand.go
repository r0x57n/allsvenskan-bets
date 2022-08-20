package main

import (
    "fmt"
    "log"
    "strconv"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func (b *botHolder) summaryAllCommand(i *dg.InteractionCreate) {
    if b.notOwnerRespond(i) { return }

    round := getCurrentRound(b.db) - 1
    matches := *getMatches(b.db, "round=$1", round)

    addInteractionResponse(b.session, i, NewMsg, "Sammanfattar...")

    var (
        totalWon = 0
        totalLost = 0
    )

    var allBets []bet
    userWins := make(map[string]int)
    userLost := make(map[string]int)

    // Fetch info for all played matches
    for _, m := range matches {
        matchBets := *getBets(b.db, "matchid=$1", m.id)

        for _, bet := range matchBets {
            allBets = append(allBets, bet)

            if bet.status == BetStatusWon {
                totalWon++

                user, _ := b.session.User(strconv.Itoa(bet.uid))
                userWins[user.Username]++
            } else if bet.status == BetStatusLost {
                totalLost++

                user, _ := b.session.User(strconv.Itoa(bet.uid))
                userLost[user.Username]++
            }
        }
    }

    rows, err := b.db.Query("SELECT uid, count(uid) AS c FROM bets WHERE round=$1 AND status=$2 GROUP BY uid ORDER BY c DESC limit 10", round, BetStatusWon)
    if err != nil { log.Panic(err) }

    topFive := ""
    placement := 1
    for rows.Next() {
        uid, count := 0, 0
        rows.Scan(&uid, &count)
        user, _ := b.session.User(strconv.Itoa(uid))
        topFive += fmt.Sprintf("#%v - %v med **%v** vinster\n", placement, user.Username, count)
        placement++
    }

    if topFive == "" {
        topFive = "Ingen vann något denna omgång."
    }

    // Bottom 5 list
    rows, err = b.db.Query("SELECT uid, count(uid) AS c FROM bets WHERE round=$1 AND status=$2 GROUP BY uid ORDER BY c DESC limit 10", round, BetStatusLost)
    if err != nil { log.Panic(err) }

    bottomFive := ""
    placement = 1
    for rows.Next() {
        uid, count := 0, 0
        rows.Scan(&uid, &count)
        user, _ := b.session.User(strconv.Itoa(uid))
        bottomFive += fmt.Sprintf("#%v - %v med **%v** förluster\n", placement, user.Username, count)
        placement++
    }

    if bottomFive == "" {
        bottomFive = "Ingen förlorade något denna omgång."
    }

    // Add it all together
    title := fmt.Sprintf("Sammanfattning av omgång %v", round)
    msg := fmt.Sprintf("**%v** matcher spelades och **%v** vadslagningar (v: %v, f: %v) las. ", len(matches), len(allBets), totalWon, totalLost)

    fields := []*dg.MessageEmbedField {
        {
            Name: "Topp 5",
            Value: topFive,
            Inline: false,
        },
        {
            Name: "Bott 5",
            Value: bottomFive,
            Inline: false,
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
                Title: title,
                Description: msg,
                Fields: fields,
            },
        },
    })
}
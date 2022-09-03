package main

import (
    "log"
    "fmt"
    "time"
    "strconv"
    "encoding/json"
    dg "github.com/bwmarrin/discordgo"
    _ "github.com/lib/pq"
)

func (b *Bot) sendSummaries() {
    rows, err := b.db.Query("SELECT id FROM matches " +
                            "WHERE round=$1 AND finished=$2 AND summarised=$3", b.getInfo().CurrentRound, true, false)
    if err != nil { log.Panic(err) }

    var ids []int
    for rows.Next() {
        var mid int
        rows.Scan(&mid)
        ids = append(ids, mid)
    }

    // are all matches summarised? then summarise round
    if len(ids) == 0 {
        b.sendRoundSummary()
        return
    }

    for _, mid := range ids {
        summary := b.getMatchSummary(strconv.Itoa(mid))
        summary.Fields = append(summary.Fields, &dg.MessageEmbedField{
            Name: "-",
            Value: "*använd /hjälp kommandot för att lära dig tippa*",
        })

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
                    Description: summary.Info,
                    Fields: summary.Fields,
                },
            },
        })

        _, err := b.db.Exec("UPDATE matches SET summarised=$1 WHERE id=$2", true, mid)
        if err != nil { log.Panic(err) }
    }
}

func (b *Bot) sendRoundSummary() {
    info := b.getInfo()
    round := info.CurrentRound

    count := 0
    err := b.db.QueryRow("SELECT count(id) FROM matches WHERE finished=$1 AND round=$2", false, round).Scan(&count)
    if err != nil { log.Panic(err) }

    if (count != 0) {
        return
    }

    count = 0
    err = b.db.QueryRow("SELECT count(id) FROM summaries WHERE round=$1 AND year=$2", round, time.Now().Year()).Scan(&count)
    if err != nil { log.Panic(err) }

    if count == 0 {
        b.createRoundSummary(fmt.Sprint(round))
    }

    var id int
    var jsonD []byte
    var sent bool
    err = b.db.QueryRow("SELECT id, data, sent FROM summaries WHERE round=$1 AND year=$2", round, time.Now().Year()).Scan(&id, &jsonD, &sent)
    if err != nil { log.Panic(err) }

    if sent {
        return
    }

    var roundData Round
    json.Unmarshal(jsonD, &roundData)

    title := fmt.Sprintf("Sammanfattning av omgång %v", roundData.Num)

    msg := fmt.Sprintf("**%v** matcher spelades och **%v** vadslagningar las.\n",
                       roundData.NumMatches, roundData.NumBets)
    msg += fmt.Sprintf("Av dessa var **%v** vinster och **%v** förluster.",
                       roundData.NumWins, roundData.NumLoss)

    fields := []*dg.MessageEmbedField {
        {
            Name: "Flest korrekta gissningar",
            Value: roundData.TopFive,
        },
        {
            Name: "Flest nära gissningar",
            Value: roundData.CloseFive,
        },
        {
            Name: "Flest felaktiga gissningar",
            Value: roundData.BotFive,
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
                Title: title,
                Description: msg,
                Fields: fields,
            },
        },
    })

    _, err = b.db.Exec("UPDATE summaries SET sent=$1 WHERE id=$2", true, id)
    if err != nil { log.Panic(err) }
    _, err = b.db.Exec("UPDATE info SET currentround=currentround+1 WHERE id=0")
    if err != nil { log.Panic(err) }
}

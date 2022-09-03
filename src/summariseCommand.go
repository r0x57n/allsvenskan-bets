package main

import (
    "encoding/json"
    "fmt"
    "log"
    "time"
    dg "github.com/bwmarrin/discordgo"
    _ "github.com/lib/pq"
)

func (b *Bot) summariseCommand(i *dg.InteractionCreate) {
    if b.notOwnerRespond(i) { return }

    iOptions := getOptionsOrRespond(b.session, i, NewMsg)
    if iOptions == nil { return }

    title := ""

    options := []dg.SelectMenuOption{}
    if iOptions[0].Value == "0" { // omgång
        if len(iOptions) == 2 {
            round := fmt.Sprint(iOptions[1].Value)

            if round == "0" {
                round = fmt.Sprint(getCurrentRound(b.db))
            } else if round == "-1" {
                round = fmt.Sprint(getCurrentRound(b.db) - 1)
            }

            b.summariseRoundDo(i, round)
            return
        } else {
            addInteractionResponse(b.session, i, NewMsg, "välj en omgång å")
            return
        }
    } else { // match
        round := fmt.Sprint(getCurrentRound(b.db))

        if len(iOptions) == 2 {
            round = fmt.Sprint(iOptions[1].Value)

            if round == "-1" {
                round = fmt.Sprint(getCurrentRound(b.db) - 1)
            }
        }
        rows, err := b.db.Query("SELECT id, hometeam, awayteam, date, homescore, awayscore, finished " +
                                "FROM matches " +
                                "WHERE round=$1", round)
        if err != nil { log.Panic(err) }
        defer rows.Close()
        options = *getOptionsOutOfRows(rows)
        title = "Sammanfatta en match."
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en match",
                    CustomID: SummariseMatchSend, // component handler
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(b.session, i, NewMsg, title, components)
}

func (b *Bot) summariseRoundDo(i *dg.InteractionCreate, round string) {
    count := 0
    err := b.db.QueryRow("SELECT count(id) FROM summaries WHERE round=$1 AND year=$2", round, time.Now().Year()).Scan(&count)
    if err != nil { log.Panic(err) }

    if count == 0 {
        addInteractionResponse(b.session, i, NewMsg, "Sammanfattar...")
        b.createRoundSummary(round)
    } else {
        addInteractionResponse(b.session, i, NewMsg, "Skickar...")
    }

    b.summaryRoundSend(round)
}

func (b *Bot) summaryRoundSend(round string) {
    var jsonD []byte
    err := b.db.QueryRow("SELECT data FROM summaries WHERE round=$1 AND year=$2", round, time.Now().Year()).Scan(&jsonD)
    if err != nil { log.Panic(err) }

    var roundData Round
    json.Unmarshal(jsonD, &roundData)

    title := fmt.Sprintf("Sammanfattning av omgång %v", roundData.Num)

    msg := fmt.Sprintf("**%v** matcher spelades och **%v** vadslagningar las.\n",
                       roundData.NumMatches, roundData.NumBets)
    msg += fmt.Sprintf("Av dessa var **%v** vinster och **%v** förluster.",
                       roundData.NumWins, roundData.NumLoss)

    fields := []*dg.MessageEmbedField {
        {
            Name: "Topp 5",
            Value: roundData.TopFive,
        },
        {
            Name: "Bott 5",
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
}

func (b *Bot) summariseMatchSend(i *dg.InteractionCreate) {
    vals := getValuesOrRespond(b.session, i, NewMsg)
    if vals == nil { return }

    mid := vals[0]
    m := getMatch(b.db, "id=$1", mid)

    if !m.Finished {
        addCompInteractionResponse(b.session , i, UpdateMsg, "Matchen inte spelad...", []dg.MessageComponent{})
        return
    } else {
        addCompInteractionResponse(b.session , i, UpdateMsg, "Sammanfattar matchen...", []dg.MessageComponent{})
    }

    summary := b.getMatchSummary(mid)
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

    _, err := b.db.Exec("UPDATE matches SET summarised=$1 WHERE id=$2", true, m.ID)
    if err != nil { log.Panic(err) }
}

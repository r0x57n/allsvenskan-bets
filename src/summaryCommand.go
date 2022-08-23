package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "strconv"
    "time"
    dg "github.com/bwmarrin/discordgo"
    _ "github.com/lib/pq"
)

func (b *Bot) summaryCommand(i *dg.InteractionCreate) {
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

            b.summaryRoundDo(i, round)
            return
        } else {
            addInteractionResponse(b.session, i, NewMsg, "välj en omgång å")
            return
        }
    } else { // match
        rows, err := b.db.Query("SELECT id, hometeam, awayteam, date, homescore, awayscore, finished " +
                                "FROM matches " +
                                "WHERE round=$1", getCurrentRound(b.db))
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
                    CustomID: SummaryMatchDo, // component handler
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(b.session, i, NewMsg, title, components)
}

func (b *Bot) summaryRoundDo(i *dg.InteractionCreate, round string) {
    matches := *getMatches(b.db, "round=$1", round)

    summarised := true
    var earlier []byte
    err := b.db.QueryRow("SELECT data FROM summaries WHERE round=$1 AND year=$2", round, time.Now().Year()).Scan(&earlier)
    if err != nil {
        if err != sql.ErrNoRows {
            log.Panic(err)
        } else {
            summarised = false
            addInteractionResponse(b.session, i, NewMsg, "Sammanfattar...")
        }
    }

    if summarised {
        b.summaryRoundSend(round)
        return
    }

    var (
        totalWon = 0
        totalLost = 0
    )

    var allBets []Bet
    userWins := make(map[string]int)
    userLost := make(map[string]int)

    // Fetch info for all played matches
    for _, m := range matches {
        matchBets := *getBets(b.db, "matchid=$1", m.ID)

        for _, bet := range matchBets {
            allBets = append(allBets, bet)

            if bet.Status == BetStatusWon {
                totalWon++

                user, _ := b.session.User(strconv.Itoa(bet.UserID))
                userWins[user.Username]++
            } else if bet.Status == BetStatusLost {
                totalLost++

                user, _ := b.session.User(strconv.Itoa(bet.UserID))
                userLost[user.Username]++
            }
        }
    }

    rows, err := b.db.Query("SELECT uid, count(uid) AS c FROM bets WHERE round=$1 AND status=$2 GROUP BY uid ORDER BY c DESC limit 10", round, BetStatusWon)
    if err != nil { log.Panic(err) }
    defer rows.Close()

    topFive := make(map[string]int)
    for rows.Next() {
        uid, count := 0, 0
        rows.Scan(&uid, &count)
        user, _ := b.session.User(strconv.Itoa(uid))
        topFive[user.Username] = count
    }

    // Bottom 5 list
    rows, err = b.db.Query("SELECT uid, count(uid) AS c FROM bets WHERE round=$1 AND status=$2 GROUP BY uid ORDER BY c DESC limit 10", round, BetStatusLost)
    if err != nil { log.Panic(err) }
    defer rows.Close()

    bottomFive := make(map[string]int)
    for rows.Next() {
        uid, count := 0, 0
        rows.Scan(&uid, &count)
        user, _ := b.session.User(strconv.Itoa(uid))
        bottomFive[user.Username] = count
    }

    roundJson := Round{
        Num: round,
        NumMatches: len(matches),
        NumBets: len(allBets),
        NumWins: totalWon,
        NumLoss: totalLost,
        TopFive: topFive,
        BotFive: bottomFive,
    }

    json, _ := json.Marshal(roundJson)

    _, err = b.db.Exec("INSERT INTO summaries(data, round, year) VALUES ($1, $2, $3)", json, round, time.Now().Year())
    if err != nil { log.Panic(err) }

    b.summaryRoundSend(round)
}

func (b *Bot) summaryRoundSend(round string) {
    var jsonD []byte
    err := b.db.QueryRow("SELECT data FROM summaries WHERE round=$1 AND year=$2", round, time.Now().Year()).Scan(&jsonD)
    if err != nil { log.Panic(err) }

    var roundData Round
    json.Unmarshal(jsonD, &roundData)

    topFive := ""
    placement := 1
    for user, wins := range roundData.TopFive {
        topFive += fmt.Sprintf("#%v - %v med **%v** vinster\n", placement, user, wins)
        placement++
    }

    bottomFive := ""
    placement = 1
    for user, losses := range roundData.BotFive {
        bottomFive += fmt.Sprintf("#%v - %v med **%v** förluster\n", placement, user, losses)
        placement++
    }

    if topFive == "" {
        topFive = "-"
    }

    if bottomFive == "" {
        bottomFive = "-"
    }

    title := fmt.Sprintf("Sammanfattning av omgång %v", roundData.Num)

    msg := fmt.Sprintf("**%v** matcher spelades och **%v** vadslagningar las.\n",
                       roundData.NumMatches, roundData.NumBets)
    msg += fmt.Sprintf("Av dessa var **%v** vinster och **%v** förluster.",
                       roundData.NumWins, roundData.NumLoss)

    fields := []*dg.MessageEmbedField {
        {
            Name: "Topp 5",
            Value: topFive,
        },
        {
            Name: "Bott 5",
            Value: bottomFive,
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

func (b *Bot) summaryMatchDo(i *dg.InteractionCreate) {
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

    matchInfo := ""
    if m.Finished {
        matchInfo = fmt.Sprintf("%v - %v\n", m.HomeTeam, m.AwayTeam)
        matchInfo += fmt.Sprintf("Resultat: %v - %v\n\n", m.HomeScore, m.AwayScore)
    }

    // all bets
    betRows, err := b.db.Query("SELECT b.id, b.uid, b.matchid, b.homescore, b.awayscore, b.status, b.round " +
                               "FROM bets AS b " +
                               "JOIN users AS u ON u.uid=b.uid " +
                               "WHERE b.matchid=$1 AND u.viewable=$2", m.ID, true)
    if err != nil { log.Panic(err) }
    bets := *getBetsFromRows(betRows)

    msgBets := ""
    if len(bets) == 0 {
        msgBets = "-"
    }

    for _, bet := range bets {
        user, _ := b.session.User(strconv.Itoa(bet.UserID))
        username := ""
        if user == nil {
            username = strconv.Itoa(bet.UserID)
        } else {
            username = user.Username
        }

        if m.Finished {
            won := bet.HomeScore == m.HomeScore && bet.AwayScore == m.AwayScore

            if won {
                msgBets += fmt.Sprintf("**%v gissade på %v - %v**\n", username, bet.HomeScore, bet.AwayScore)
            } else {
                msgBets += fmt.Sprintf("%v gissade på %v - %v\n", username, bet.HomeScore, bet.AwayScore)
            }
        } else {
            msgBets += fmt.Sprintf("%v gissar på %v - %v\n", username, bet.HomeScore, bet.AwayScore)
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
        userChallenger, _ := b.session.User(strconv.Itoa(c.ChallengerID))
        userChallengee, _ := b.session.User(strconv.Itoa(c.ChallengeeID))
        usernameChallenger := ""
        usernameChallengee := ""
        if userChallenger == nil {
            usernameChallenger = strconv.Itoa(c.ChallengerID)
        } else {
            usernameChallenger = userChallenger.Username
        }

        if userChallengee == nil {
            usernameChallengee = strconv.Itoa(c.ChallengeeID)
        } else {
            usernameChallengee = userChallengee.Username
        }

        winner := ""
        if c.Condition == ChallengeConditionWinnerHome {
            winner = m.HomeTeam
        } else {
            winner = m.AwayTeam
        }

        if m.Finished {
            msgChalls += fmt.Sprintf("%v utmanade %v om att %v skulle vinna för %v poäng\n",
                                    usernameChallenger, usernameChallengee, winner, c.Points)
        } else {
            msgChalls += fmt.Sprintf("%v utmanar %v om att %v ska vinna för %v poäng\n",
                                    usernameChallenger, usernameChallengee, winner, c.Points)
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

    _, err = b.db.Exec("UPDATE matches SET summarised=$1 WHERE id=$2", true, m.ID)
    if err != nil { log.Panic(err) }
}

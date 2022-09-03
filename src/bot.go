package main

import (
    "log"
    "fmt"
    "time"
    "strconv"
    "encoding/json"
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) Init() {
    log.Print("Initializing...")

    // Login bot to get the active session
    s, err := dg.New("Bot " + b.token)
    if err != nil {
        log.Fatalf("Invalid bot parameters: %v", err)
    }

    b.session = s
    b.addCommands()

    s.AddHandler(func(s *dg.Session, i *dg.InteractionCreate) {
        switch i.Type {
            case dg.InteractionApplicationCommand:
                if h, ok := b.commandHandlers[i.ApplicationCommandData().Name]; ok { h(s, i) }
            case dg.InteractionMessageComponent:
                if h, ok := b.componentHandlers[i.MessageComponentData().CustomID]; ok { h(s, i) }
        }
    })

    // Handler to tell us when we logged in
    s.AddHandler(func(s *dg.Session, r *dg.Ready) {
        log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
        g, _ := s.Guild(s.State.Guilds[0].ID)
        log.Printf("In the following guild: %v", g.Name)
        b.allsvenskanGuildID = s.State.Guilds[0].ID

        if *ADD_COMMANDS {
            log.Printf("Adding commands...")
            for _, c := range b.commands {
                cmd := dg.ApplicationCommand {
                    Name: c.Name,
                    Description: c.Description,
                    Options: c.Options,
                }

                log.Printf("Adding: %v", c.Name)
                _, err := b.session.ApplicationCommandCreate(b.appID, b.guildID, &cmd)
                if err != nil {
                    log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
                }
            }
            log.Printf("Added all commands!")
        }
    })
}

func (b *Bot) checkStuff(interactive bool) {
    b.handleBets(interactive)
    b.handleChallenges(interactive)
    b.sendSummaries()
}

func (b *Bot) Start() {
    err := b.session.Open()
    if err != nil {
        log.Panicf("Cannot open the session: %v", err)
    }
}

func (b *Bot) Close() {
    b.session.Close()
    b.db.Close()
}

func (b *Bot) notOwnerRespond(i *dg.InteractionCreate) bool {
    if b.owner != getInteractUID(i) {
        addInteractionResponse(b.session, i, NewMsg, "Saknar behörighet.")
        return true
    }
    return false
}

func (b *Bot) messageOwner(msg string) {
    channelID, _ := b.session.UserChannelCreate(b.owner)
    b.session.ChannelMessageSend(channelID.ID, msg)
}

func (b *Bot) getInfo() *BotInfo {
    var i BotInfo
    err := b.db.QueryRow("SELECT currentround, lastupdate FROM info WHERE id=0").Scan(&i.CurrentRound, &i.LastUpdate)
    if err != nil { log.Panic(err) }
    return &i
}

func (b *Bot) getMatchSummary(mid string) *MatchSummary {
    m := getMatch(b.db, "id=$1", mid)

    datetime, _ := time.Parse(DB_TIME_LAYOUT, m.Date)

    matchInfo := ""
    if m.Finished {
        matchInfo = fmt.Sprintf("%v - %v\n", m.HomeTeam, m.AwayTeam)
        matchInfo += fmt.Sprintf("Resultat: %v - %v\n", m.HomeScore, m.AwayScore)
    } else {
        matchInfo = fmt.Sprintf("%v - %v, spelas %v", m.HomeTeam, m.AwayTeam, datetime.Format(MSG_TIME_LAYOUT))
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
    }

    return &MatchSummary{
        Info: matchInfo,
        Fields: fields,
    }
}

func (b *Bot) createRoundSummary(round string) {
    matches := *getMatches(b.db, "round=$1", round)

    var allBets []Bet
    userWins := make(map[string]int)
    userLost := make(map[string]int)
    totalWon, totalLost := 0, 0

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

    // Top 5 list
    rows, err := b.db.Query("SELECT uid, count(uid) AS c FROM bets WHERE round=$1 AND status=$2 GROUP BY uid ORDER BY c DESC limit 5", round, BetStatusWon)
    if err != nil { log.Panic(err) }
    defer rows.Close()

    placement := 1
    topFive := "-"
    for rows.Next() {
        uid, count := 0, 0
        rows.Scan(&uid, &count)
        user, _ := b.session.User(strconv.Itoa(uid))

        if topFive == "-" { topFive = "" }
        topFive += fmt.Sprintf("#%v - %v med **%v** vinster\n", placement, user, count)
        placement++
    }

    // Bottom 5 list
    rows, err = b.db.Query("SELECT uid, count(uid) AS c FROM bets WHERE round=$1 AND status=$2 GROUP BY uid ORDER BY c DESC limit 5", round, BetStatusLost)
    if err != nil { log.Panic(err) }
    defer rows.Close()

    placement = 1
    bottomFive := "-"
    for rows.Next() {
        uid, count := 0, 0
        rows.Scan(&uid, &count)
        user, _ := b.session.User(strconv.Itoa(uid))

        if bottomFive == "-" { bottomFive = "" }
        bottomFive += fmt.Sprintf("#%v - %v med **%v** förluster\n", placement, user, count)
        placement++
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
}

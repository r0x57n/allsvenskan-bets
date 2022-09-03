package main

import (
    "fmt"
    "strconv"
    "strings"
    "log"
    "time"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) guessCommand(i *dg.InteractionCreate) {
    options := *getCurrentMatchesAsOptions(b.db)
    if len(options) == 0 {
        addInteractionResponse(b.session, i, NewMsg, "Inga matcher tillgängliga! :(")
        return
    }

    uid := getInteractUID(i)

    // map match id:s to bets so we can easier build strings later
    matchidToBet := make(map[string]Bet)
    betsRows, err := b.db.Query("SELECT matchid, homescore, awayscore FROM bets " +
                                "WHERE status=$1 AND uid=$2 ", BetStatusUnhandled, uid)
    if err != nil { log.Panic(err) }

    for betsRows.Next() {
        var bet Bet
        betsRows.Scan(&bet.MatchID, &bet.HomeScore, &bet.AwayScore)
        matchidToBet[strconv.Itoa(bet.MatchID)] = bet
    }

    if len(matchidToBet) > 0 {
        for i, option := range options {
            if bet, ok := matchidToBet[option.Value]; ok {
                options[i].Label += fmt.Sprintf(" [%v-%v]", bet.HomeScore, bet.AwayScore)
            }
        }
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en match",
                    CustomID: GuessSelectScore, // component handler
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(b.session, i, NewMsg, "Kommande omgångens matcher.", components)
}

func (b *Bot) guessSelectGoals(i *dg.InteractionCreate) {
    values := getValuesOrRespond(b.session, i, UpdateMsg)
    if values == nil { return }

    mid := values[0]
    uid := getInteractUID(i)

    earlierBetScore := [2]int {-1, -1}
    earlierBet := getBet(b.db, "uid=$1 AND matchid=$2", uid, mid)
    if earlierBet.ID != -1 {
        earlierBetScore[0], earlierBetScore[1] = earlierBet.HomeScore, earlierBet.AwayScore
    }

    m := getMatch(b.db, "id=$1", mid)
    if m.ID == -1 {
        addErrorResponse(b.session, i, UpdateMsg)
        return
    }

    datetime, err := time.Parse(DB_TIME_LAYOUT, m.Date)
    if err != nil {
        addErrorResponse(b.session, i, NewMsg, "Couldn't translate match date from database...")
        return
    }

    if time.Now().After(datetime) {
        addCompInteractionResponse(b.session, i, UpdateMsg, "Du kan inte betta på en match som startat.", []dg.MessageComponent{})
        return
    }

    msg := fmt.Sprintf("Hemmalag: %v\n", m.HomeTeam)
    msg += fmt.Sprintf("Bortalag: %v\n\n", m.AwayTeam)
    msg += fmt.Sprintf("Spelas: %v, klockan %v\n\n", datetime.Format("2006-01-02"), datetime.Format("15:04"))
    msg += fmt.Sprintf("**Mål**")

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: GuessUpdateScoreHome,
                    Placeholder: "Hemmalag",
                    Options: *getScoresAsOptions(m.ID, earlierBetScore[0], m.HomeTeam),
                },
            },
        },
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: GuessUpdateScoreAway,
                    Placeholder: "Bortalag",
                    Options: *getScoresAsOptions(m.ID, earlierBetScore[1], m.AwayTeam),
                },
            },
        },
    }

    addCompInteractionResponse(b.session, i, NewMsg, msg, components)
}

func (b *Bot) guessUpdateGoals(i *dg.InteractionCreate, where BetLocation) {
    values := getValuesOrRespond(b.session, i, UpdateMsg)
    if values == nil { return }

    var (
        uid = getInteractUID(i)
        splitted = strings.Split(values[0], "_")
        mid = splitted[0]
        homescore = "0"
        awayscore = "0"
        score = ""
        awayOrHome = ""
    )

    switch where {
        case Home:
            homescore = splitted[1]
            score = homescore
            awayOrHome = "home"
        case Away:
            awayscore = splitted[1]
            score = awayscore
            awayOrHome = "away"
        default:
            addErrorResponse(b.session, i, UpdateMsg)
            return
    }

    m := getMatch(b.db, "id=$1", mid)
    if m.ID == -1 {
        addErrorResponse(b.session, i, UpdateMsg)
        return
    }

    if matchHasBegun(b.session, i, m) {
        addCompInteractionResponse(b.session, i, UpdateMsg, "Matchen har startat...", []dg.MessageComponent{})
        return
    }

    var err error
    hasBettedBefore := getBet(b.db, "uid=$1 AND matchid=$2", uid, mid).ID != -1
    if hasBettedBefore {
        _, err = b.db.Exec("UPDATE bets SET " + awayOrHome + "score=$1 WHERE uid=$2 AND matchid=$3", score, uid, mid)
        if err != nil { log.Panic(err) }
    } else {
        _, err = b.db.Exec("INSERT INTO bets (uid, matchid, homescore, awayscore, round) VALUES ($1, $2, $3, $4, $5)", uid, mid, homescore, awayscore, m.Round)
        if err != nil { log.Panic(err) }
    }

    addNoInteractionResponse(b.session, i)
}

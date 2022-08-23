package main

import (
    "fmt"
    "strconv"
    "log"
    "time"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) regretCommand(i *dg.InteractionCreate) {
    uid := getInteractUID(i)

    allBets := *getBets(b.db, "uid=$1 AND status=$2", uid, BetStatusUnhandled)

    labels := make(map[int]string)
    dates := make(map[int]string)

    var regrettableBets []Bet

    for _, bet := range allBets {
        m := getMatch(b.db, "id=$1", bet.MatchID)
        matchDate, _ := time.Parse(DB_TIME_LAYOUT, m.Date)

        if time.Now().Before(matchDate) {
            labels[bet.ID] = fmt.Sprintf("%v vs %v [%v-%v]", m.HomeTeam, m.AwayTeam, bet.HomeScore, bet.AwayScore)
            dates[bet.ID] = matchDate.Format(MSG_TIME_LAYOUT)
            regrettableBets = append(regrettableBets, bet)
        }
    }

    if len(regrettableBets) == 0 {
        addInteractionResponse(b.session, i, NewMsg, "Inga framtida vadslagningar...")
        return
    }

    options := []dg.SelectMenuOption{}

    for _, b := range regrettableBets {
        options = append(options, dg.SelectMenuOption{
            Label: labels[b.ID],
            Value: strconv.Itoa(b.ID),
            Description: dates[b.ID],
        })
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en vadslagning",
                    CustomID: RegretSelected,
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(b.session, i, NewMsg, "Dina vadslagningar", components)
}

func (b *Bot) regretSelected(i *dg.InteractionCreate) {
    values := getValuesOrRespond(b.session, i, UpdateMsg)
    if values == nil { return }
    bid := values[0]

    var m Match
    var bet Bet
    err := b.db.QueryRow("SELECT m.date, b.uid FROM bets AS b " +
                       "JOIN matches AS m ON b.matchid=m.id " +
                       "WHERE b.id=$1", bid).Scan(&m.Date, &bet.UserID)
    if err != nil { log.Panic(err) }

    components := []dg.MessageComponent {}
    msg := ""

    if matchHasBegun(b.session, i, m) {
        msg = "Kan inte ta bort en vadslagning om en pågående match..."
    } else {
        if strconv.Itoa(bet.UserID) != getInteractUID(i) {
            addErrorResponse(b.session, i, UpdateMsg, "Du försökte ta bort någon annans vadslagning...")
            return
        }

        _, err = b.db.Exec("DELETE FROM bets WHERE id=$1", bid)
        msg = "Vadslagning borttagen!"
    }

    addCompInteractionResponse(b.session, i, UpdateMsg, msg, components)
}

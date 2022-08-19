package main

import (
    "fmt"
    "strconv"
    "log"
    "time"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func newRegret(b *botHolder) *Regret {
    cmd := new(Regret)
    cmd.bot = b
    cmd.name = RegretCommand
    cmd.description = "testar"
    cmd.addComponents()
    return cmd
}

func (cmd *Regret) addComponents() {
    cmd.bot.addComponent("regretSelected", cmd.regretSelected)
}

func (cmd *Regret) run(i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()
    s := cmd.bot.session

    uid := getInteractUID(i)

    allBets := *getBets(db, "uid=$1 AND status=$2", uid, BetStatusUnhandled)

    labels := make(map[int]string)
    dates := make(map[int]string)

    var regrettableBets []bet

    for _, b := range allBets {
        m := getMatch(db, "id=$1", b.matchid)
        matchDate, _ := time.Parse(DB_TIME_LAYOUT, m.date)

        if time.Now().Before(matchDate) {
            labels[b.id] = fmt.Sprintf("%v vs %v [%v-%v]", m.hometeam, m.awayteam, b.homescore, b.awayscore)
            dates[b.id] = matchDate.Format(MSG_TIME_LAYOUT)
            regrettableBets = append(regrettableBets, b)
        }
    }

    if len(regrettableBets) == 0 {
        addInteractionResponse(s, i, NewMsg, "Inga framtida vadslagningar...")
        return
    }

    options := []dg.SelectMenuOption{}

    for _, b := range regrettableBets {
        options = append(options, dg.SelectMenuOption{
            Label: labels[b.id],
            Value: strconv.Itoa(b.id),
            Description: dates[b.id],
        })
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en vadslagning",
                    CustomID: "regretSelected",
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(s, i, NewMsg, "Dina vadslagningar", components)
}

func (cmd *Regret) regretSelected(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }
    bid := values[0]

    var m match
    var b bet
    err := db.QueryRow("SELECT m.date, b.uid FROM bets AS b " +
                       "JOIN matches AS m ON b.matchid=m.id " +
                       "WHERE b.id=$1", bid).Scan(&m.date, &b.uid)
    if err != nil { log.Panic(err) }

    components := []dg.MessageComponent {}
    msg := ""

    if matchHasBegun(s, i, m) {
        msg = "Kan inte ta bort en vadslagning om en pågående match..."
    } else {
        if strconv.Itoa(b.uid) != getInteractUID(i) {
            addErrorResponse(s, i, UpdateMsg, "Du försökte ta bort någon annans vadslagning...")
            return
        }

        _, err = db.Exec("DELETE FROM bets WHERE id=$1", bid)
        msg = "Vadslagning borttagen!"
    }

    addCompInteractionResponse(s, i, UpdateMsg, msg, components)
}

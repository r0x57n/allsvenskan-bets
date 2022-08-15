package main

import (
	"fmt"
	"strconv"
	"log"
	"time"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

func regretCommand(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
	defer db.Close()

    uid := getInteractUID(i)

    allBets := *getBets(db, "uid=? AND handled=?", uid, 0)

    labels := make(map[int]string)
    dates := make(map[int]string)

    var regrettableBets []bet

    for _, b := range allBets {
        m := getMatch(db, "id=?", b.matchid)
        matchDate, _ := time.Parse(DB_TIME_LAYOUT, m.date)

        if time.Now().Before(matchDate) {
            labels[b.id] = fmt.Sprintf("%v vs %v [%v-%v]", m.homeTeam, m.awayTeam, b.homeScore, b.awayScore)
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

func regretSelected(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
	defer db.Close()

    bid := getValuesOrRespond(s, i, UpdateMsg)
    if bid == nil { return }

    var matchid int
    err := db.QueryRow("SELECT matchid FROM bets WHERE id=?", bid[0]).Scan(&matchid)

    var date string
    err = db.QueryRow("SELECT date FROM matches WHERE id=?", matchid).Scan(&date)
    if err != nil { log.Panic(err) }

    matchDate, err := time.Parse(DB_TIME_LAYOUT, date)
    if err != nil { log.Panic(err) }

    components := []dg.MessageComponent {}

    if time.Now().After(matchDate) {
        msg := "Kan inte ta bort en vadslagning om en pågående match..."
        addCompInteractionResponse(s, i, UpdateMsg, msg, components)
    } else {
        _, err = db.Exec("DELETE FROM bets WHERE id=?", bid[0])
        msg := "Vadslagning borttagen!"
        addCompInteractionResponse(s, i, UpdateMsg, msg, components)
    }
}

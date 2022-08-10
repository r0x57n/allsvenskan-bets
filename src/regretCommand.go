package main

import (
	"fmt"
	"strconv"
	"log"
	"time"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

func regretCommand(s *dg.Session, i *dg.InteractionCreate) {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

    uid := getInteractUID(i)
    var bets []bet
    betsRows, err := db.Query("SELECT id, uid, matchid, homeScore, awayScore FROM bets WHERE uid=? AND handled=0", uid)
    defer betsRows.Close()
    if err != nil { log.Panic(err) }

    labels := make(map[int]string)
    dates := make(map[int]string)

    for betsRows.Next() {
        var b bet
        betsRows.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore)

        var m match

        err := db.QueryRow("SELECT homeTeam, awayTeam, date, round FROM matches WHERE id=?", b.matchid).
                      Scan(&m.homeTeam, &m.awayTeam, &m.date, &m.round)
        if err != nil { log.Panic(err) }
        datetime, _ := time.Parse(TIME_LAYOUT, m.date)

        if time.Now().Before(datetime) {
            labels[b.id] = fmt.Sprintf("%v vs %v [%v-%v]", m.homeTeam, m.awayTeam, b.homeScore, b.awayScore)
            dates[b.id] = datetime.Format("2006-02-01 kl. 15:04")
            bets = append(bets, b)
        }
    }

    options := []dg.SelectMenuOption{}
    disabled := false

    if len(bets) > 0 {
        for _, b := range bets {
            options = append(options, dg.SelectMenuOption{
                Label: labels[b.id],
                Value: strconv.Itoa(b.id),
                Description: dates[b.id],
            })
        }
    } else {
        options = append(options, dg.SelectMenuOption{
            Label: "Inga framtida vadslagningar...",
            Value: "invalid",
            Description: "",
            Default: true,
        })
        disabled = true
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en vadslagning",
                    CustomID: "regretSelected", // component handler
                    Options: options,
                    Disabled: disabled,
                },
            },
        },
    }

    addCompInteractionResponse(s, i, NewMsg, "Dina vadslagningar", components)
}

func regretSelected(s *dg.Session, i *dg.InteractionCreate) {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

    bid := i.MessageComponentData().Values[0]

    var matchid int
    err = db.QueryRow("SELECT matchid FROM bets WHERE id=?", bid).Scan(&matchid)

    var date string
    err = db.QueryRow("SELECT date FROM matches WHERE id=?", matchid).Scan(&date)
    if err != nil { log.Panic(err) }
    datetime, _ := time.Parse(TIME_LAYOUT, date)

    components := []dg.MessageComponent {}

    if time.Now().After(datetime) {
        msg := "Kan inte ta bort en vadslagning för en pågåenge eller spelad match..."
        addCompInteractionResponse(s, i, UpdateMsg, msg, components)
    } else {
        _, err = db.Exec("DELETE FROM bets WHERE id=?", bid)
        msg := "Vadslagning borttaget!"
        addCompInteractionResponse(s, i, UpdateMsg, msg, components)
    }
}
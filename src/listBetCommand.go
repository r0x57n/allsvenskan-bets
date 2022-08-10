package main

import (
	"fmt"
	"strconv"
	"log"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

// Command: vadslagningar
func listBetsCommand(s *dg.Session, i *dg.InteractionCreate) {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

    // Get options and parse
    options := i.Interaction.ApplicationCommandData().Options
	uID := options[0].Value

    betType := 2 // 0 = lost, 1 = won, 2 = all
    if len(options) == 2 {
        betType, _ = strconv.Atoi(fmt.Sprintf("%v", options[1].Value))
    }

    // Get bets
    var bets *sql.Rows
    switch betType {
        case 0:
            bets, err = db.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE uid=? AND handled=1 AND won=0", uID)
        case 1:
            bets, err = db.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE uid=? AND handled=1 AND won=1", uID)
        default:
            bets, err = db.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE uid=? AND handled=1", uID)
    }
	if err != nil { log.Fatal(err) }
    defer bets.Close()

	var viewable = 0

	if err := db.QueryRow("SELECT viewable FROM points WHERE uid=?", uID).Scan(&viewable); err != nil {
		if err != sql.ErrNoRows { log.Panic(err) }
	}

    desc, correct, incorrect := "", "", ""

	if viewable == 1 {
		var b bet

		for bets.Next() {
			bets.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore, &b.won)
			matchRow := db.QueryRow("SELECT homeTeam, awayTeam, date FROM matches WHERE id=?", b.matchid)

			var m match
			matchRow.Scan(&m.homeTeam, &m.awayTeam, &m.date)

            if b.won == 0 {
                incorrect += fmt.Sprintf("%v (**%v**) - %v (**%v**)\n", m.homeTeam, b.homeScore, m.awayTeam, b.awayScore)
            } else if b.won == 1 {
                correct += fmt.Sprintf("%v (**%v**) - %v (**%v**)\n", m.homeTeam, b.homeScore, m.awayTeam, b.awayScore)
            }
		}

		if correct == "" && incorrect == "" {
			desc = fmt.Sprintf("Användaren har inga vadslagningar ännu!", )

            if correct == "" {
                correct = "-"
            }

            if incorrect == "" {
                incorrect = "-"
            }
		}
	} else {
		desc = "Användaren har valt att dölja sina vadslagningar."
        incorrect = "-"
        correct = "-"
	}

    fields := []*dg.MessageEmbedField {}

    if betType == 0 {
        fields = []*dg.MessageEmbedField {
            {
                Name: "Inkorrekta",
                Value: incorrect,
            },
        }
    } else if betType == 1 {
        fields = []*dg.MessageEmbedField {
            {
                Name: "Korrekta",
                Value: correct,
            },
        }
    } else {
        fields = []*dg.MessageEmbedField {
            {
                Name: "Korrekta",
                Value: correct,
            },
            {
                Name: "Inkorrekta",
                Value: incorrect,
            },
        }
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Vadslagningar", desc)
}

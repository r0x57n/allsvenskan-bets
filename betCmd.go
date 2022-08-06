package main

import (
	"fmt"
	"strconv"
    "strings"
	"log"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

// Command: slåvad
func betCommand(s *dg.Session, i *dg.InteractionCreate) {
	options := getRoundMatchesAsOptions()

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en match",
                    CustomID: "betOnSelected", // component handler
                    Options: *options,
                },
            },
        },
    }

    compInteractionResponse(s, i, dg.InteractionResponseChannelMessageWithSource, "Kommande omgångens matcher:", components)
}

func betOnSelected(s *dg.Session, i *dg.InteractionCreate) {
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

	matchID, _ := strconv.Atoi(i.MessageComponentData().Values[0])
	uID := i.Interaction.Member.User.ID

	matchInfo := svffDB.QueryRow("SELECT id, homeTeam, awayTeam, date FROM matches WHERE id=?", matchID)
	earlierBet, _ := betsDB.Query("SELECT homeScore, awayScore FROM bets WHERE (uid, matchid) IS (?, ?)", uID, matchID)
	defer earlierBet.Close()

	var (
		m match
		defHome int = -1
		defAway int = -1
	)

	if earlierBet.Next() { // prior bet
		earlierBet.Scan(&defHome, &defAway)
	}

	if err := matchInfo.Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date); err != nil { log.Panic(err) }
	msg := fmt.Sprintf("%v (h) vs %v (b) @ %v \n\n**Poäng** *(hemmalag överst)*", m.homeTeam, m.awayTeam, m.date)
    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID:    "betScoreHome",
                    Placeholder: "Hemmalag",
                    Options: getScoreMenuOptions(m.id, defHome),
                },
            },
        },
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID:    "betScoreAway",
                    Placeholder: "Bortalag",
                    Options: getScoreMenuOptions(m.id, defAway),
                },
            },
        },
    }

    compInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, msg, components)
}

func betScoreComponent(s *dg.Session, i *dg.InteractionCreate, where location) {
	db, err := sql.Open(DB_TYPE, BETS_DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	data := i.MessageComponentData().Values[0]
	var splitted = strings.Split(data, "_")
	var (
		matchID = splitted[0]
		uID = i.Interaction.Member.User.ID
		home = "0"
		away = "0"
	)

	switch where {
		case Home: home = splitted[1]
		case Away: away = splitted[1]
		default: log.Panic("This shouldn't happen...")
	}

	rows, err := db.Query("SELECT * FROM bets WHERE (uid, matchid) IS (?, ?)", uID, matchID)
	defer rows.Close()
	if err != nil { log.Fatal(err) }

	// Prior bet
	if rows.Next() {
		if where == Home {
			rows.Close()
			if _, err := db.Exec("UPDATE bets SET homeScore=? WHERE (uid, matchid) IS (?, ?)", home, uID, matchID); err != nil { log.Panic(err) }
		} else {
			rows.Close()
			if _, err := db.Exec("UPDATE bets SET awayScore=? WHERE (uid, matchid) IS (?, ?)", away, uID, matchID); err != nil { log.Panic(err) }
		}
	// No prior bet
	} else {
		rows.Close()
		if _, err := db.Exec("INSERT INTO bets (uid, matchid, homeScore, awayScore) VALUES (?, ?, ?, ?)", uID, matchID, home, away); err != nil { log.Panic(err) }
	}

	if err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse {
		Type: dg.InteractionResponseDeferredMessageUpdate,
    }); err != nil { log.Panic(err) }
}

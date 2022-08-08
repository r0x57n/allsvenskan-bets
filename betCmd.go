package main

import (
	"fmt"
	"strconv"
    "strings"
	"log"
	"database/sql"
    "time"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

// Command: slåvad
func betCommand(s *dg.Session, i *dg.InteractionCreate) {
	options := getRoundMatchesAsOptions()

	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

    m := make(map[string]bet)
    uid := i.Interaction.Member.User.ID
    betsRows, err := betsDB.Query("SELECT matchid, homeScore, awayScore FROM bets WHERE handled=0 AND uid=?", uid)
    if err != nil { log.Panic(err) }

    for betsRows.Next() {
        var b bet

        betsRows.Scan(&b.matchid, &b.homeScore, &b.awayScore)
        m[strconv.Itoa(b.matchid)] = b
    }

    if len(m) > 0 {
        for i, elem := range *options {
            b := m[elem.Value]
            (*options)[i].Label += fmt.Sprintf(" [%v-%v]", b.homeScore, b.awayScore)
        }
    }

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

    var msg string
    var components []dg.MessageComponent
	matchID, _ := strconv.Atoi(i.MessageComponentData().Values[0])
	uID := i.Interaction.Member.User.ID

    // Get earlier bet info, if any
    defHome, defAway := -1, -1
	err = betsDB.QueryRow("SELECT homeScore, awayScore FROM bets WHERE (uid, matchid) IS (?, ?)", uID, matchID).
                 Scan(&defHome, &defAway)
    if err != nil && err != sql.ErrNoRows { log.Panic(err) }

    // Get match info
    var m match
	err = svffDB.QueryRow("SELECT id, homeTeam, awayTeam, date FROM matches WHERE id=?", matchID).
                 Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date)
	if err != nil { log.Panic(err) }

    datetime, err := time.Parse(TIME_LAYOUT, m.date)
	if err != nil { log.Panic(err) }

    // Make the response
    if time.Now().After(datetime) {
        msg = "Du kan inte betta på matcher som startat..."
        components = []dg.MessageComponent {}
    } else {
        datetime, _ := time.Parse(TIME_LAYOUT, m.date)

        msg = fmt.Sprintf("Hemmalag: %v\n", m.homeTeam)
        msg += fmt.Sprintf("Bortalag: %v\n\n", m.awayTeam)
        msg += fmt.Sprintf("Spelas: %v, klockan %v\n\n", datetime.Format("2006-01-02"), datetime.Format("15:04"))
        msg += fmt.Sprintf("**Poäng** *(hemmalag överst)*")

        components = []dg.MessageComponent {
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

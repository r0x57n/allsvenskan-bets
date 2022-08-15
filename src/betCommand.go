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
    db := connectDB()
	defer db.Close()

	options := *getCurrentMatchesAsOptions(db)
    if len(options) == 0 {
        addInteractionResponse(s, i, NewMsg, "Inga matcher tillgängliga! :(")
        return
    }

    uid := getInteractUID(i)

    // map matchIDs to bets so we can easier build strings later
    matchidToBet := make(map[string]bet)
    betsRows, err := db.Query("SELECT matchid, homeScore, awayScore FROM bets WHERE handled=0 AND uid=?", uid)
    if err != nil { log.Panic(err) }

    for betsRows.Next() {
        var b bet
        betsRows.Scan(&b.matchid, &b.homeScore, &b.awayScore)
        matchidToBet[strconv.Itoa(b.matchid)] = b
    }

    if len(matchidToBet) > 0 {
        for i, option := range options {
            b := matchidToBet[option.Value]
            options[i].Label += fmt.Sprintf(" [%v-%v]", b.homeScore, b.awayScore)
        }
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en match",
                    CustomID: "betOnSelected", // component handler
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(s, i, NewMsg, "Kommande omgångens matcher.", components)
}

func betOnSelected(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
	defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }

	mid := values[0]
	uid := getInteractUID(i)

    earlierBetScore := [2]int {-1, -1}
    earlierBet := getBet(db, "uid=? AND matchid=?", uid, mid)
    if earlierBet.id != -1 {
        earlierBetScore[0], earlierBetScore[1] = earlierBet.homeScore, earlierBet.awayScore
    }

    m := getMatch(db, "id=?", mid)
    if m.id == -1 {
        addErrorResponse(s, i, UpdateMsg)
        return
    }

    datetime, err := time.Parse(DB_TIME_LAYOUT, m.date)
	if err != nil { log.Panic(err) }

    if time.Now().After(datetime) {
        addCompInteractionResponse(s, i, UpdateMsg, "Du kan inte betta på en match som startat.", []dg.MessageComponent{})
        return
    }

    msg := fmt.Sprintf("Hemmalag: %v\n", m.homeTeam)
    msg += fmt.Sprintf("Bortalag: %v\n\n", m.awayTeam)
    msg += fmt.Sprintf("Spelas: %v, klockan %v\n\n", datetime.Format("2006-01-02"), datetime.Format("15:04"))
    msg += fmt.Sprintf("**Poäng** *(hemmalag överst)*")

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "betScoreHome",
                    Placeholder: "Hemmalag",
                    Options: *getScoresAsOptions(m.id, earlierBetScore[0]),
                },
            },
        },
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "betScoreAway",
                    Placeholder: "Bortalag",
                    Options: *getScoresAsOptions(m.id, earlierBetScore[1]),
                },
            },
        },
    }

    addCompInteractionResponse(s, i, UpdateMsg, msg, components)
}

func betScoreComponent(s *dg.Session, i *dg.InteractionCreate, where location) {
    db := connectDB()
	defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }

	var (
        splitted = strings.Split(values[0], "_")
		mid = splitted[0]
		uid = getInteractUID(i)
		homeScore = "0"
		awayScore = "0"
        score = ""
        awayOrHome = ""
	)

	switch where {
		case Home:
            homeScore = splitted[1]
            score = homeScore
            awayOrHome = "home"
		case Away:
            awayScore = splitted[1]
            score = awayScore
            awayOrHome = "away"
		default:
            addErrorResponse(s, i, UpdateMsg)
            return
	}

    round := -1
    err := db.QueryRow("SELECT round FROM matches WHERE id=?", mid).Scan(&round)
	if err != nil {
        if err == sql.ErrNoRows {
            addErrorResponse(s, i, UpdateMsg, "Kunde inte hitta matchen...")
            return
        } else { log.Fatal(err) }
    }

    hasBettedBefore := getBet(db, "uid=? AND matchid=?", uid, mid).id != -1

	if hasBettedBefore {
        _, err = db.Exec("UPDATE bets SET " + awayOrHome + "Score=? WHERE (uid, matchid) IS (?, ?)", score, uid, mid)
	} else {
		_, err = db.Exec("INSERT INTO bets (uid, matchid, homeScore, awayScore, round) VALUES (?, ?, ?, ?, ?)", uid, mid, homeScore, awayScore, round)
	}

    if err != nil { log.Panic(err) }

    addNoInteractionResponse(s, i)
}

package main

import (
	"math"
	"fmt"
	"strconv"
	"log"
	"time"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

/*
 Helper functions
*/

func getInteractUID(i *dg.InteractionCreate) string {
    uid := ""

    if i.Interaction.Member == nil {
        uid = i.Interaction.User.ID
    } else {
        uid = i.Interaction.Member.User.ID
    }

    if uid == "" {
        log.Panic("Couldn't get user ID.")
    }

    return uid
}

func getUser(db *sql.DB, uid string) user {
    var u user

	err := db.QueryRow("SELECT uid, seasonPoints, bank, viewable, interactable FROM users WHERE uid=?", uid).
              Scan(&u.uid, &u.seasonPoints, &u.bank, &u.viewable, &u.interactable)

	if err != nil {
        if err == sql.ErrNoRows {
            _, err = db.Exec("INSERT INTO users (uid) VALUES (?)", uid)
            if err != nil { log.Panic(err) }
        } else {
            log.Panic(err)
        }
    }

    return u
}

func getUserFromInteraction(db *sql.DB, i *dg.InteractionCreate) user {
    uid := fmt.Sprint(getInteractUID(i))
    return getUser(db, uid)
}

func notOwner(s *dg.Session, i *dg.InteractionCreate) bool {
    isntOwner := getInteractUID(i) != *OWNER

	if isntOwner {
        addInteractionResponse(s, i, NewMsg, "Du har inte rättigheter att köra detta kommando...")
        return true
	}

    return false
}

/*
   Common database stuff (SQL)
*/

func connectDB() *sql.DB {
    db, err := sql.Open(DB_TYPE, DB)
    if err != nil {
        log.Fatalf("Couldn't connect to database: %v", err)
    }

    return db
}

func getCurrentRound(db *sql.DB) int {
    round := -1
	today := time.Now().Format("2006-01-02")

    err := db.QueryRow("SELECT round FROM matches WHERE date(date)>=? AND finished='0' ORDER BY date", today).Scan(&round)
    if err != nil {
        if err == sql.ErrNoRows {
            return round
        } else { log.Panic(err) }
    }

    return round
}

func getUserChallenges(db *sql.DB, uid string) *[]challenge {
    challRows, err := db.Query("SELECT id, challengerUID, challengeeUID, type, matchID, points, condition, status FROM challenges WHERE (challengerUID=? OR challengeeUID=?) AND (status=?)", uid, uid, Accepted)
    if err != nil { log.Panic(err) }
	defer challRows.Close()

    var challenges []challenge
    for challRows.Next() {
        var c challenge
        challRows.Scan(&c.id, &c.challengerUID, &c.challengeeUID, &c.typ, &c.matchID, &c.points, &c.condition, &c.status)
        challenges = append(challenges, c)
    }

    return &challenges
}

func getMatches(db *sql.DB, where string) *[]match {
    var matches []match

	rows, err := db.Query("SELECT id, homeTeam, awayTeam, date, scoreHome, scoreAway, finished FROM matches WHERE " + where)
	defer rows.Close()
    if err != nil {
        if err == sql.ErrNoRows {
            return &matches
        } else { log.Panic(err) }
    }

	for rows.Next() {
		var m match
		if err := rows.Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date, &m.scoreHome, &m.scoreAway, &m.finished); err != nil { log.Panic(err) }
		matches = append(matches, m)
	}

    return &matches
}

/*
   Helpers for select menus
*/

// Parameter is optional string to add to value of the options.
// This is so we can add meta data about things such as challenges, we need
// to remember who the challengee was...
func getRoundMatchesAsOptions(db *sql.DB, addToValue ...string) *[]dg.SelectMenuOption {
    options := []dg.SelectMenuOption{}

    round := getCurrentRound(db)

	today := time.Now().Format(DB_TIME_LAYOUT)
    matches := *getMatches(db, fmt.Sprintf("round=%v AND date>='%v'", round, today))

	if len(matches) == 0 {
        return &options
	}

    for _, m := range matches {
        optionValue := strconv.Itoa(m.id)

        // adding meta data to value
        if len(addToValue) == 1 {
            optionValue = addToValue[0] + "_" + optionValue
        }

        matchDate, err := time.Parse(DB_TIME_LAYOUT, m.date)
        if err != nil { log.Printf("Couldn't parse date: %v", err) }

        daysUntilMatch := math.Round(time.Until(matchDate).Hours() / 24)

        options = append(options, dg.SelectMenuOption{
            Label: fmt.Sprintf("%v vs %v", m.homeTeam, m.awayTeam),
            Value: optionValue,
            Description: fmt.Sprintf("om %v dagar (%v)", daysUntilMatch, matchDate.Format(MSG_TIME_LAYOUT)),
        })
    }

    return &options
}

func getPointsAsOptions(values string, maxPoints int) *[]dg.SelectMenuOption {
    options := []dg.SelectMenuOption{}

    if maxPoints != 0 {
        for i := 1; i <= maxPoints; i++ {
            options = append(options, dg.SelectMenuOption{
                Label: fmt.Sprintf("%v Poäng", i),
                Value: fmt.Sprintf("%v_%v", values, i),
                Description: "",
            })
        }
    } else {
        options = append(options, dg.SelectMenuOption{
            Label: fmt.Sprintf("Du har inga poäng :("),
            Value: "none",
            Description: "",
        })
    }

    return &options
}


func getScoresAsOptions(matchID int, defScore int) *[]dg.SelectMenuOption {
	options := []dg.SelectMenuOption {}

	for i := 0; i < 25; i++ {
        isChosenScore := defScore == i

        options = append(options, dg.SelectMenuOption{
            Label: strconv.Itoa(i),
            Value: fmt.Sprintf("%v_%v", matchID, i),
            Default: isChosenScore,
        })
	}

	return &options
}

func getAcceptDiscardOptions(accept string, discard string, defOption bool) []dg.SelectMenuOption {
    return []dg.SelectMenuOption{
                    {
                        Label: accept,
                        Value: "1",
                        Default: defOption,
                    },
                    {
                        Label: discard,
                        Value: "0",
                        Default: !defOption,
                    },
    }
}

/*
  Interaction response adders
*/

func addInteractionResponse(s *dg.Session,
                            i *dg.InteractionCreate,
                            typ dg.InteractionResponseType,
                            msg string) {
	if err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse {
		Type: typ,
		Data: &dg.InteractionResponseData {
			Content: msg,
			Flags: 1 << 6, // Ephemeral
		},
	}); err != nil { log.Panic(err) }
}

func ignoreInteraction(s *dg.Session,
                       i *dg.InteractionCreate) {
    addInteractionResponse(s, i, Ignore, "")
}

func addEmbeddedInteractionResponse(s *dg.Session,
                                    i *dg.InteractionCreate,
                                    typ dg.InteractionResponseType,
                                    fields []*dg.MessageEmbedField,
                                    title string,
                                    descr string) {
	if err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse {
		Type: typ,
		Data: &dg.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
            Embeds: []*dg.MessageEmbed {
                {
                    Title: title,
                    Description: descr,
                    Fields: fields,
                },
            },

		},
	}); err != nil { log.Panic(err) }

}

func addCompInteractionResponse(s *dg.Session,
                                i *dg.InteractionCreate,
                                typ dg.InteractionResponseType,
                                msg string,
                                components []dg.MessageComponent) {
	if err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse {
		Type: typ,
		Data: &dg.InteractionResponseData {
			Content: msg,
			Components: components,
			Flags: 1 << 6, // Ephemeral
        },
	}); err != nil { log.Panic(err) }
}

func getValuesOrRespond (s *dg.Session,
                         i *dg.InteractionCreate,
                         typ dg.InteractionResponseType) []string {
    vals := i.Interaction.MessageComponentData().Values
    if len(vals) == 0 {
        addErrorResponse(s, i, typ)
        log.Printf("Tried to unpack values but found none...")
        return nil
    }

    return vals
}

func addErrorResponse(s *dg.Session,
                      i *dg.InteractionCreate,
                      typ dg.InteractionResponseType) {
        msg := "Oväntat fel, kontakta ägare och beskriv vad du försökte göra.\n"
        msg += "Timestamp: " + time.Now().Format(DB_TIME_LAYOUT)
        addCompInteractionResponse(s, i, typ, msg, []dg.MessageComponent {})
}

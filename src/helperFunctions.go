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

func getUser(uid string) user {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

    var u user

	err = db.QueryRow("SELECT uid, seasonPoints, bank, viewable, interactable FROM users WHERE uid=?", uid).
                 Scan(&u.uid, &u.season, &u.history, &u.viewable, &u.interactable)
	if err != nil {
        if err == sql.ErrNoRows {
            u.uid, err = strconv.Atoi(uid)
            if err != nil { log.Panic(err) }

            u.season = 0
            u.history = ""
            u.viewable = 1
            u.interactable = 1

            _, err = db.Exec("INSERT INTO users (uid, seasonPoints) VALUES (?, ?)", u.uid, u.season)
            if err != nil { log.Panic(err) }
        } else {
            log.Panic(err)
        }
    }

    return u
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
   Common database getters
*/

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

/*
   Options builders for select menus
*/

// Parameter is optional string to add to value of the options.
// This is so we can add meta data about things such as challenges, we need
// to remember who the challengee was...
func getRoundMatchesAsOptions(value ...string) *[]dg.SelectMenuOption {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	today := time.Now().Format("2006-01-02")
	todayAndTime := time.Now().Format(TIME_LAYOUT)

    round := -1
    err = db.QueryRow("SELECT round FROM matches WHERE date(date)>=? AND finished='0' ORDER BY date", today).Scan(&round)
    if err != nil { log.Panic(err) }

	rows, err := db.Query("SELECT id, homeTeam, awayTeam, date, scoreHome, scoreAway, finished FROM matches WHERE round=? AND date>=?", round, todayAndTime)
	defer rows.Close()
    if err != nil { log.Panic(err) }

	var matches []*match
	for rows.Next() {
		var m match
		if err := rows.Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date, &m.scoreHome, &m.scoreAway, &m.finished); err != nil { log.Panic(err) }
		matches = append(matches, &m)
	}

    options := []dg.SelectMenuOption{}

	if len(matches) == 0 {
		options = append(options, dg.SelectMenuOption{
			Label: "Inga matcher tillgängliga... :(",
			Value: "noMatches",
			Description: "",
            Default: true,
		})
	} else {
		for _, m := range matches {
            val := strconv.Itoa(m.id)

            if len(value) == 1 {
                val = value[0] + "_" + val
            }

            datetime, _ := time.Parse(TIME_LAYOUT, m.date)
            daysUntil := math.Round(time.Until(datetime).Hours() / 24)

            options = append(options, dg.SelectMenuOption{
                Label: fmt.Sprintf("%v vs %v", m.homeTeam, m.awayTeam),
                Value: val,
                Description: fmt.Sprintf("om %v dagar (%v)", daysUntil, datetime.Format("2006-02-01 kl. 15:04")),
            })
		}
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

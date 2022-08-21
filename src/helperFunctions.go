package main

import (
	"math"
	"fmt"
	"strconv"
	"log"
	"time"
	"database/sql"
	_ "github.com/lib/pq"
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

func getUserFromInteraction(db *sql.DB, i *dg.InteractionCreate) user {
    uid := fmt.Sprint(getInteractUID(i))
    return getUser(db, uid)
}

func matchHasBegun(s *dg.Session, i *dg.InteractionCreate, m match) bool {
    matchDate, err := time.Parse(DB_TIME_LAYOUT, m.date)
	if err != nil {
        addErrorResponse(s, i, NewMsg, "Couldn't translate match date from database...")
        return true
    }

    return time.Now().After(matchDate)
}


/*
   Common database stuff.
   We take care to let the SQL package prepare the statements, see: https://go.dev/doc/database/sql-injection
*/

func connectDB(i dbInfo) *sql.DB {
    dbInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
                        i.host, i.port, i.user, i.password, i.name)
    db, err := sql.Open(DB_TYPE, dbInfo)
    if err != nil {
        log.Fatalf("Couldn't connect to database: %v", err)
    }

    return db
}

func getUser(db *sql.DB, uid string) user {
    var u user

	err := db.QueryRow("SELECT uid, points, bank, viewable, interactable FROM users WHERE uid=$1", uid).
              Scan(&u.uid, &u.points, &u.bank, &u.viewable, &u.interactable)

	if err != nil {
        if err == sql.ErrNoRows {
            _, err = db.Exec("INSERT INTO users (uid) VALUES ($1)", uid)
            if err != nil { log.Panic(err) }
        } else {
            log.Panic(err)
        }
    }

    return u
}

func getCurrentRound(db *sql.DB) int {
    round := -1
	today := time.Now().Format("2006-01-02")

    err := db.QueryRow("SELECT round FROM matches WHERE date(date)>=$1 AND finished='0' ORDER BY date", today).Scan(&round)
    if err != nil {
        if err == sql.ErrNoRows {
            return round
        } else { log.Panic(err) }
    }

    return round
}

func getMatches(db *sql.DB, where string, args ...any) *[]match {
    var matches []match

    rows, err := db.Query("SELECT id, hometeam, awayteam, date, homescore, awayscore, finished FROM matches WHERE " + where, args...)
	defer rows.Close()
    if err != nil {
        if err == sql.ErrNoRows {
            return &matches
        } else { log.Panic(err) }
    }

	for rows.Next() {
		var m match
		if err := rows.Scan(&m.id, &m.hometeam, &m.awayteam, &m.date, &m.homescore, &m.awayscore, &m.finished); err != nil { log.Panic(err) }
		matches = append(matches, m)
	}

    return &matches
}

func getMatchesFromRows(rows *sql.Rows) *[]match {
    var matches []match

	for rows.Next() {
		var m match
		if err := rows.Scan(&m.id, &m.hometeam, &m.awayteam, &m.date, &m.homescore, &m.awayscore, &m.finished); err != nil { log.Panic(err) }
		matches = append(matches, m)
	}

    return &matches
}

func getMatch(db *sql.DB, where string, args ...any) match {
    var m match

	err := db.QueryRow("SELECT id, hometeam, awayteam, date, homescore, awayscore, finished, round FROM matches WHERE " + where, args...).
              Scan(&m.id, &m.hometeam, &m.awayteam, &m.date, &m.homescore, &m.awayscore, &m.finished, &m.round)
	if err != nil {
        if err == sql.ErrNoRows {
            return match { id: -1 }
        } else {
            log.Panic(err)
        }
    }

    return m
}

func getBetsFromRows(rows *sql.Rows) *[]bet {
    var bets []bet

	for rows.Next() {
        var b bet
		if err := rows.Scan(&b.id, &b.uid, &b.matchid, &b.homescore, &b.awayscore, &b.status, &b.round); err != nil { log.Panic(err) }
		bets = append(bets, b)
	}

    return &bets
}

func getBets(db *sql.DB, where string, args ...any) *[]bet {
    var bets []bet

	rows, err := db.Query("SELECT id, uid, matchid, homescore, awayscore, status FROM bets WHERE " + where, args...)
	defer rows.Close()
    if err != nil {
        if err == sql.ErrNoRows {
            return &bets
        } else { log.Panic(err) }
    }

	for rows.Next() {
        var b bet
		if err := rows.Scan(&b.id, &b.uid, &b.matchid, &b.homescore, &b.awayscore, &b.status); err != nil { log.Panic(err) }
		bets = append(bets, b)
	}

    return &bets
}

func getBet(db *sql.DB, where string, args ...any) bet {
    var b bet

	err := db.QueryRow("SELECT id, uid, matchid, homescore, awayscore, status FROM bets WHERE " + where, args...).
              Scan(&b.id, &b.uid, &b.matchid, &b.homescore, &b.awayscore, &b.status)
	if err != nil {
        if err == sql.ErrNoRows {
            return bet { id: -1 }
        } else {
            log.Panic(err)
        }
    }

    return b
}

func getChallenges(db *sql.DB, where string, args ...any) *[]challenge {
    var challenges []challenge

	rows, err := db.Query("SELECT id, challengerid, challengeeid, type, matchid, points, condition, status FROM challenges WHERE " + where, args...)
	defer rows.Close()
    if err != nil {
        if err == sql.ErrNoRows {
            return &challenges
        } else { log.Panic(err) }
    }

	for rows.Next() {
        var c challenge
		if err := rows.Scan(&c.id, &c.challengerid, &c.challengeeid, &c.typ, &c.matchid, &c.points, &c.condition, &c.status); err != nil { log.Panic(err) }
		challenges = append(challenges, c)
	}

    return &challenges
}

func getChallenge(db *sql.DB, where string, args ...any) challenge {
    var c challenge

	err := db.QueryRow("SELECT id, challengerid, challengeeid, type, matchid, points, condition, status FROM challenges WHERE " + where, args...).
              Scan(&c.id, &c.challengerid, &c.challengeeid, &c.typ, &c.matchid, &c.points, &c.condition, &c.status)
	if err != nil {
        if err == sql.ErrNoRows {
            return challenge { id: -1 }
        } else {
            log.Panic(err)
        }
    }

    return c
}

/*
   Helpers for select menus
*/

func getOptionsOutOfRows(rows *sql.Rows) *[]dg.SelectMenuOption {
    options := []dg.SelectMenuOption{}

    matches := *getMatchesFromRows(rows)

	if len(matches) == 0 {
        return &options
	}

    for _, m := range matches {
        optionValue := strconv.Itoa(m.id)

        matchDate, err := time.Parse(DB_TIME_LAYOUT, m.date)
        if err != nil { log.Printf("Couldn't parse date: %v", err) }

        daysUntilMatch := math.Round(time.Until(matchDate).Hours() / 24)

        description := ""
        if math.Signbit(daysUntilMatch) {
            description = fmt.Sprintf("spelad (%v)", matchDate.Format(MSG_TIME_LAYOUT))
        } else {
            description = fmt.Sprintf("om %v dagar (%v)", daysUntilMatch, matchDate.Format(MSG_TIME_LAYOUT))
        }

        label := fmt.Sprintf("%v vs %v", m.hometeam, m.awayteam)

        options = append(options, dg.SelectMenuOption{
            Label: label,
            Value: optionValue,
            Description: description,
        })
    }

    return &options
}

// Parameter is optional string to add to value of the options.
// This is so we can add meta data about things such as challenges, we need
// to remember who the challengee was...
func getCurrentMatchesAsOptions(db *sql.DB, addToValue ...string) *[]dg.SelectMenuOption {
    options := []dg.SelectMenuOption{}

    round := getCurrentRound(db)
	today := time.Now().Format(DB_TIME_LAYOUT)
    matches := *getMatches(db, "round=$1 AND date>=$2", round, today)

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

        label := fmt.Sprintf("%v vs %v", m.hometeam, m.awayteam)
        description := fmt.Sprintf("om %v dagar (%v)", daysUntilMatch, matchDate.Format(MSG_TIME_LAYOUT))

        options = append(options, dg.SelectMenuOption{
            Label: label,
            Value: optionValue,
            Description: description,
        })
    }

    return &options
}

func getPointsAsOptions(values string, maxPoints int) *[]dg.SelectMenuOption {
    options := []dg.SelectMenuOption{}

    if maxPoints > 25 {
        maxPoints = 24
    }

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


func getScoresAsOptions(matchid int, defScore int, teamName string) *[]dg.SelectMenuOption {
	options := []dg.SelectMenuOption {}

	for i := 0; i < 25; i++ {
        isChosenScore := defScore == i

        options = append(options, dg.SelectMenuOption{
            Label: fmt.Sprintf("%v - %v", i, teamName),
            Value: fmt.Sprintf("%v_%v", matchid, i),
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

func addNoInteractionResponse(s *dg.Session,
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

func getValuesOrRespond(s *dg.Session,
                        i *dg.InteractionCreate,
                        typ dg.InteractionResponseType) []string {
    if i.Interaction.Type != dg.InteractionMessageComponent {
        log.Printf("Not message component...")
        addErrorResponse(s, i, typ, "Not message component...")
        return nil
    }

    vals := i.Interaction.MessageComponentData().Values
    if len(vals) == 0 {
        addErrorResponse(s, i, typ)
        log.Printf("Tried to unpack values but found none...")
        return nil
    }

    return vals
}

func getOptionsOrRespond(s *dg.Session,
                         i *dg.InteractionCreate,
                         typ dg.InteractionResponseType) []*dg.ApplicationCommandInteractionDataOption {
    if i.Interaction.Type != dg.InteractionApplicationCommand {
        log.Printf("Not application command...")
        addErrorResponse(s, i, typ, "Not application command...")
        return nil
    }

    options := i.Interaction.ApplicationCommandData().Options
    if len(options) == 0 {
        log.Printf("Tried to unpack values but found none...")
        addErrorResponse(s, i, typ, "Försökte packa upp värden utan att ha något.")
        return nil
    }

    return options
}


func addErrorResponse(s *dg.Session,
                      i *dg.InteractionCreate,
                      typ dg.InteractionResponseType,
                      optional ...string) {
        msg := "Oväntat fel, kontakta ägare och beskriv vad du försökte göra.\n"
        msg += "Timestamp: \n" + time.Now().Format(DB_TIME_LAYOUT)
        addCompInteractionResponse(s, i, typ, msg + optional[0], []dg.MessageComponent {})
}

func addErrorsResponse(s *dg.Session,
                       i *dg.InteractionCreate,
                       typ dg.InteractionResponseType,
                       errors []CommandError,
                       add string) {
    msg := ""
    msg += add + "\n"

    for _, e := range errors {
        switch (e) {
            case ErrorNoRights:
                msg += "- Inga rättigheter att köra kommandot.\n"
            case ErrorMatchStarted:
                msg += "- Matchen har redan startat.\n"
            case ErrorOtherNotInteractable:
                msg += "- Användaren tillåter inte utmaningar.\n"
            case ErrorSelfNotInteractable:
                msg += "- Du måste själv tillåta utmaningar (se /installningar).\n"
            case ErrorInteractingWithSelf:
                msg += "- Du kan inte utmana dig själv.\n"
            case ErrorMaxChallenges:
                msg += "- Du kan inte ha mer än 25 utmaningar.\n"
            case ErrorNoMatches:
                msg += "- Det finns inga matcher vad slå om.\n"
            case ErrorIdenticalChallenge:
                msg += "- Du kan inte utmana samma spelare om samma sak igen.\n"
            case ErrorNotEnoughPoints:
                msg += "- Du eller den utmanade har inte nog med poäng.\n"
            case ErrorChallengeHandled:
                msg += "- Utmaningen har redan blivit hanterad.\n"
            case ErrorUserNotViewable:
                msg += "- Användaren har valt att dölja sina vadslagningar."
            default:
                log.Printf("Unknown error code in makeErrorsResponse: %v", e)
        }
    }

    addCompInteractionResponse(s, i, typ, msg, []dg.MessageComponent {})
}

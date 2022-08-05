package main

import (
	"flag"
	"github.com/robfig/cron/v3"
	"fmt"
	"strings"
	"strconv"
	"log"
	"time"
	"os"
	"os/signal"
	"github.com/bwmarrin/discordgo"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

/*
 Paths
*/

const (
	SVFF_DB = "../svffscraper/foo.db"
	BETS_DB = "./bets.db"
	DB_TYPE = "sqlite3"
)


/*
 Bot parameters
*/

var (
	GUILD_ID  = flag.String("guild", "", "Test guild ID")
	BOT_TOKEN = flag.String("token", "MTAwMDgzNDQ3MzIyODc3OTU4Mg.GMTTc2.8vE1wGIbRP41q6G_md3FhXfHAISDZww2Ja0aTs", "Bot access token")
	APP_ID    = flag.String("app", "1000834473228779582", "Application ID")
)

const (
	OWNER = "436614981283217418"
)


/*
 Types
*/

type match struct {
	id int
	homeTeam string
	awayTeam string
	date string
	scoreHome int
	scoreAway int
	finished int
}


/*
 Commands and handlers
*/

var (
	COMMANDS = []discordgo.ApplicationCommand {
		{
			Name: "hjälp",
			Description: "Få hjälp med hur denna bot fungerar.",
		},
		{
			Name: "vadslå",
			Description: "Slå vad om en kommande match.",
		},
		{
			Name: "minavad",
			Description: "Lista dina vadslagningar.",
		},
	}

	COMMAND_HANDLERS = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		"hjälp": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData {
					Flags: 1 << 6, // Ephemeral
					Content: getHelpContent(),
				},
			})
		},
		"vadslå": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData {
					Flags: 1 << 6, // Ephemeral
					Content: "Kommande omgångens matcher:",
					Components: []discordgo.MessageComponent {
						discordgo.ActionsRow {
							Components: []discordgo.MessageComponent {
								discordgo.SelectMenu {
									Placeholder: "Välj en match",
									CustomID: "selectMatch",
									Options: getRoundMatchesAsOptions(),
								},
							},
						},
					},
				},
			})
		},
		"minavad": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData {
					Flags: 1 << 6, // Ephemeral
					Content: getUserBets(i),
				},
			})
		},
	}

	COMPONENT_HANDLERS = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		"selectMatch": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			response := buildMatchBetResponse(i)
			err := s.InteractionRespond(i.Interaction, response)
			if err != nil {
				panic(err)
			}
		},
		"scoreHome": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			response := setScore(i, Home)
			err := s.InteractionRespond(i.Interaction, response)
			if err != nil {
				panic(err)
			}
		},
		"scoreAway": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			response := setScore(i, Away)
			err := s.InteractionRespond(i.Interaction, response)
			if err != nil {
				panic(err)
			}
		},
	}
)

/*
 Content functions
*/

func getHelpContent() string {
	help := "Denna bot är till för att kunna vadslå om hur olika Allsvenska matcher kommer sluta.\n" +
		    "\n" +
	        "**Kommandon**\n"

	for _, elem := range COMMANDS {
		help = help + fmt.Sprintf("/%v - %v\n", elem.Name, elem.Description)
	}

	return help
}

func getRoundMatchesAsOptions() []discordgo.SelectMenuOption {
	db, err := sql.Open(DB_TYPE, SVFF_DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	today := time.Now().Format("2006-01-02")
	tenFromToday := time.Now().Add(time.Hour * 24 * 10).Format("2006-01-02")

	rows, _ := db.Query("SELECT id, homeTeam, awayTeam, date, scoreHome, scoreAway, finished FROM matches WHERE date BETWEEN ? and ?", today, tenFromToday)
	defer rows.Close()

	var matches []match
	for rows.Next() {
		var m match
		if err := rows.Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date, &m.scoreHome, &m.scoreAway, &m.finished); err != nil { log.Panic(err) }
		matches = append(matches, m)
	}

	response := []discordgo.SelectMenuOption{}

	if len(matches) == 0 {
		response = append(response, discordgo.SelectMenuOption{
			Label: "Inga matcher tillgängliga kommande tio dagar.",
			Value: "",
			Description: "",
		})
	} else {
		for _, m := range matches {
			response = append(response, discordgo.SelectMenuOption{
				Label: fmt.Sprintf("%v vs %v", m.homeTeam, m.awayTeam),
				Value: strconv.Itoa(m.id),
				Description: m.date,
			})
		}
	}

	return response
}

func getUserBets(i *discordgo.InteractionCreate) string {
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

	uID := i.Interaction.Member.User.ID

	bets, _ := betsDB.Query("SELECT * FROM bets WHERE uid=?", uID)
	defer bets.Close()

	var (
		id int
		uid int
		matchID int
		homeScore int
		awayScore int
	)

	userBets := ""

	for bets.Next() {
		bets.Scan(&id, &uid, &matchID, &homeScore, &awayScore)
		match := svffDB.QueryRow("SELECT homeTeam, awayTeam, date FROM matches WHERE id=?", matchID)

		var (
			homeTeam string
			awayTeam string
			date string
		)

		match.Scan(&homeTeam, &awayTeam, &date)

		userBets = userBets + fmt.Sprintf("%v vs %v @ %v, ditt vad är: %v - %v\n", homeTeam, awayTeam, date, homeScore, awayScore)
	}

	if userBets == "" {
		userBets = "Inga vadslagningar ännu!"
	}

	return userBets
}

func buildMatchBetResponse(i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
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

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%v (h) vs %v (b) @ %v \n\n**Poäng** *(hemmalag överst)*", m.homeTeam, m.awayTeam, m.date),
			Flags: 1 << 6, // Ephemeral
			Components: []discordgo.MessageComponent {
				discordgo.ActionsRow {
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							// Select menu, as other components, must have a customID, so we set it to this value.
							CustomID:    "scoreHome",
							Placeholder: "Hemmalag",
							Options: getScoreMenuOptions(m.id, defHome),
						},
					},
				},
				discordgo.ActionsRow {
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							// Select menu, as other components, must have a customID, so we set it to this value.
							CustomID:    "scoreAway",
							Placeholder: "Bortalag",
							Options: getScoreMenuOptions(m.id, defAway),
						},
					},
				},
			},
		},
	}

	return response
}

type Test int
const (
	Home Test = iota
	Away
)

func setScore(i *discordgo.InteractionCreate, where Test) *discordgo.InteractionResponse {
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

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}
}


/*
 Helper functions
*/

func getScoreMenuOptions(matchID int, defScore int) []discordgo.SelectMenuOption {
	scores := []discordgo.SelectMenuOption {}

	for i := 0; i < 25; i++ {
		if defScore == i {
			scores = append(scores, discordgo.SelectMenuOption{
				Label: strconv.Itoa(i),
				Value: fmt.Sprintf("%v_%v", matchID, i),
				Default: true,
			})
		} else {
			scores = append(scores, discordgo.SelectMenuOption{
				Label: strconv.Itoa(i),
				Value: fmt.Sprintf("%v_%v", matchID, i),
				Default: false,
			})
		}
	}

	return scores
}


/*
 Interval functions
*/

func handleTodaysBets() {
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	today := time.Now().Format("2006-01-12")

	log.Printf("Handling bets for %v...", today)

	matches, err := svffDB.Query("SELECT id FROM matches WHERE date=?", today)
	if err != nil { log.Panic(err) }

	var (

	)

	for matches.Next() {

	}
}

/*
  Initialization
*/

func initializeBot() *discordgo.Session {
	// Login bot
	s, err := discordgo.New("Bot " + *BOT_TOKEN)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	// Initialize commands
	cmdIDs := make(map[string]string, len(COMMANDS))

	for _, cmd := range COMMANDS {
		rcmd, err := s.ApplicationCommandCreate(*APP_ID, *GUILD_ID, &cmd)
		if err != nil {
			log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
		}

		cmdIDs[rcmd.ID] = rcmd.Name
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
			case discordgo.InteractionApplicationCommand:
				if h, ok := COMMAND_HANDLERS[i.ApplicationCommandData().Name]; ok { h(s, i) }
			case discordgo.InteractionMessageComponent:
				if h, ok := COMPONENT_HANDLERS[i.MessageComponentData().CustomID]; ok { h(s, i) }
		}
	})

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	// Start bot
	err = s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	return s
}

/*
 Main
*/

func main() {
	log.Print("Starting...")

	// Initialize and start the bot
	s := initializeBot()
	defer s.Close()

	// Check the bets on a timed interval
	c := cron.New()
	c.AddFunc("@every 5s", func() { log.Print("Every 30s") })
	//c.Start()

	// Wait for stop signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Print("Quitting...")
}

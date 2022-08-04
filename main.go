package main

import (
	"flag"
	"cron"
	"strings"
	"strconv"
	"os"
	"log"
	"os/signal"
	"github.com/bwmarrin/discordgo"
	"time"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

/* =======
    Paths
   =======*/

const (
	SVFF_DB = "../svffscraper/foo.db"
	BETS_DB = "./bets.db"
)


/* ================
    Bot parameters
   ================ */

var (
	GUILD_ID  = flag.String("guild", "", "Test guild ID")
	BOT_TOKEN = flag.String("token", "MTAwMDgzNDQ3MzIyODc3OTU4Mg.GMTTc2.8vE1wGIbRP41q6G_md3FhXfHAISDZww2Ja0aTs", "Bot access token")
	APP_ID    = flag.String("app", "1000834473228779582", "Application ID")
)

const (
	OWNER = "436614981283217418"
)


/* =======================
    Commands and handlers
   ======================= */

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

/* ===================
    Content functions
   =================== */

func getHelpContent() string {
	help := "Denna bot är till för att kunna vadslå om hur olika Allsvenska matcher kommer sluta.\n" +
		    "\n" +
	        "**Kommandon**\n"

	for _, elem := range COMMANDS {
		help = help + "/" + elem.Name + " - " + elem.Description + "\n"
	}

	return help
}

func getRoundMatchesAsOptions() []discordgo.SelectMenuOption {
	db, err := sql.Open("sqlite3", SVFF_DB)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	today := time.Now().Format("2006-01-02")
	tenFromToday := time.Now().Add(time.Hour * 24 * 10).Format("2006-01-02")

	rows, _ := db.Query("SELECT * FROM matches WHERE date BETWEEN '" + today + "' and '" + tenFromToday + "'")
	defer rows.Close()

	response := []discordgo.SelectMenuOption{}

	var (
		id int
		home string
		away string
		date string
	)

	for rows.Next() {
		rows.Scan(&id, &home, &away, &date)
		response = append(response, discordgo.SelectMenuOption{
			Label: home + " vs " + away,
			Value: strconv.Itoa(id),
			Description: date,
		})
	}

	return response
}

func getUserBets(i *discordgo.InteractionCreate) string {
	svffDB, err := sql.Open("sqlite3", SVFF_DB)
	defer svffDB.Close()
	if err != nil {
		log.Fatal(err)
	}

	betsDB, err := sql.Open("sqlite3", BETS_DB)
	defer betsDB.Close()
	if err != nil {
		log.Fatal(err)
	}

	uID := i.Interaction.Member.User.ID

	bets, _ := betsDB.Query("SELECT * FROM bets WHERE uid=" + uID)
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
		match := svffDB.QueryRow("SELECT home_team, away_team, date FROM matches WHERE id=" + strconv.Itoa(matchID))

		var (
			homeTeam string
			awayTeam string
			date string
		)

		match.Scan(&homeTeam, &awayTeam, &date)

		userBets = userBets + homeTeam + " vs " + awayTeam + " @ " + date + ", ditt vad är: " + strconv.Itoa(homeScore) + " - " + strconv.Itoa(awayScore) + "\n"
	}

	if userBets == "" {
		userBets = "Inga vadslagningar ännu!"
	}

	return userBets
}

func buildMatchBetResponse(i *discordgo.InteractionCreate) *discordgo.InteractionResponse {
	svffDB, err := sql.Open("sqlite3", SVFF_DB)
	defer svffDB.Close()
	if err != nil {
		log.Fatal(err)
	}

	betsDB, err := sql.Open("sqlite3", BETS_DB)
	defer betsDB.Close()
	if err != nil {
		log.Fatal(err)
	}

	matchID, _ := strconv.Atoi(i.MessageComponentData().Values[0])
	uID := i.Interaction.Member.User.ID

	matchInfo := svffDB.QueryRow("SELECT * FROM matches WHERE id=" + strconv.Itoa(matchID))
	earlierBet, _ := betsDB.Query("SELECT homeScore, awayScore FROM bets WHERE (uid, matchid) IS (" + uID + ", " + strconv.Itoa(matchID) + ")")
	defer earlierBet.Close()

	var (
		id int
		home string
		away string
		date string
		defHome int = -1
		defAway int = -1
	)

	if earlierBet.Next() { // prior bet
		earlierBet.Scan(&defHome, &defAway)
	}

	matchInfo.Scan(&id, &home, &away, &date)

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: home + " (h) vs " + away + " (b) @ " + date + "\n\n**Poäng** *(hemmalag överst)*",
			Flags: 1 << 6, // Ephemeral
			Components: []discordgo.MessageComponent {
				discordgo.ActionsRow {
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							// Select menu, as other components, must have a customID, so we set it to this value.
							CustomID:    "scoreHome",
							Placeholder: "Hemmalag",
							Options: getScoreMenuOptions(id, defHome),
						},
					},
				},
				discordgo.ActionsRow {
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							// Select menu, as other components, must have a customID, so we set it to this value.
							CustomID:    "scoreAway",
							Placeholder: "Bortalag",
							Options: getScoreMenuOptions(id, defAway),
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
	db, err := sql.Open("sqlite3", BETS_DB)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	data := i.MessageComponentData().Values[0]
	var splitted = strings.Split(data, "_")
	var (
		matchID = splitted[0]
		uID = i.Interaction.Member.User.ID
		home = "0"
		away = "0"
	)

	if where == Home {
		home = splitted[1]
	} else {
		away = splitted[1]
	}

	rows, err := db.Query("SELECT * FROM bets WHERE (uid, matchid) IS (" + uID + ", " + matchID + ")")
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
	}

	if rows.Next() { // prior bet
		if where == Home {
			rows.Close()
			_, err = db.Exec("UPDATE bets SET homeScore = " + home + " WHERE (uid, matchid) IS (" + uID + ", " + matchID + ")")
			if err != nil {
				log.Fatal(err)
			}
		} else {
			rows.Close()
			_, err = db.Exec("UPDATE bets SET awayScore = " + away + " WHERE (uid, matchid) IS (" + uID + ", " + matchID + ")")
			if err != nil {
				log.Fatal(err)
			}
		}
	} else { // no prior bet
		rows.Close()
		_, err = db.Exec("INSERT INTO bets (uid, matchid, homeScore, awayScore) VALUES " +
											"('" + uID  + "', '" + matchID  + "', " +
											" '" + home + "', '" + away     + "');")
		if err != nil {
			log.Fatal(err)
		}
	}

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}
}


/* ===================
    Helper functions
   =================== */

func getScoreMenuOptions(matchID int, defScore int) []discordgo.SelectMenuOption {
	scores := []discordgo.SelectMenuOption {}

	for i := 0; i < 25; i++ {
		if defScore == i {
			scores = append(scores, discordgo.SelectMenuOption{
				Label: strconv.Itoa(i),
				Value: strconv.Itoa(matchID) + "_" + strconv.Itoa(i),
				Default: true,
			})
		} else {
			scores = append(scores, discordgo.SelectMenuOption{
				Label: strconv.Itoa(i),
				Value: strconv.Itoa(matchID) + "_" + strconv.Itoa(i),
				Default: false,
			})
		}
	}

	return scores
}


/* ======
    Main
   ======*/

func main() {
	var s *discordgo.Session
	var err error

	s, err = discordgo.New("Bot " + *BOT_TOKEN)
	if err != nil {
		log.Printf("Invalid bot parameters: %v", err)
	}

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	cmdIDs := make(map[string]string, len(COMMANDS))

	for _, cmd := range COMMANDS {
		rcmd, err := s.ApplicationCommandCreate(*APP_ID, *GUILD_ID, &cmd)
		if err != nil {
			log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
		}

		cmdIDs[rcmd.ID] = rcmd.Name
	}

	// Components are part of interactions, so we register InteractionCreate handler
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
			case discordgo.InteractionApplicationCommand:
				if h, ok := COMMAND_HANDLERS[i.ApplicationCommandData().Name]; ok {
					h(s, i)
				}
			case discordgo.InteractionMessageComponent:

				if h, ok := COMPONENT_HANDLERS[i.MessageComponentData().CustomID]; ok {
					h(s, i)
				}
		}
	})

	log.Print("Starting...")
	err = s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Print("Quitting...")
}

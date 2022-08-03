package main

import (
	"flag"
	"fmt"
	"strconv"
	"os"
	"log"
	"os/signal"
	"github.com/bwmarrin/discordgo"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

// Bot parameters
var (
	GuildID  = flag.String("guild", "", "Test guild ID")
	BotToken = flag.String("token", "MTAwMDgzNDQ3MzIyODc3OTU4Mg.GMTTc2.8vE1wGIbRP41q6G_md3FhXfHAISDZww2Ja0aTs", "Bot access token")
	AppID    = flag.String("app", "1000834473228779582", "Application ID")
)

func getNextMatches() string {
	db, err := sql.Open("sqlite3", "../svffscraper/foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	today := time.Now().Format("2006-01-02")
	tenFromToday := time.Now().Add(time.Hour * 24 * 10).Format("2006-01-02")

	rows, _ := db.Query("SELECT * FROM matches WHERE date BETWEEN '" + today + "' and '" + tenFromToday + "'")

	var response = ""

	var (
		id int
		home string
		away string
		date string
	)

	for rows.Next() {
		rows.Scan(&id, &home, &away, &date)
		response = response + home + " vs " + away + ", " + date + "\n"
	}

	if response == "" {
		response = "Inga resultat..."
	}

	return response
}

func fetchBettableMatches() []discordgo.SelectMenuOption {
	db, err := sql.Open("sqlite3", "../svffscraper/foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	today := time.Now().Format("2006-01-02")
	tenFromToday := time.Now().Add(time.Hour * 24 * 10).Format("2006-01-02")

	rows, _ := db.Query("SELECT * FROM matches WHERE date BETWEEN '" + today + "' and '" + tenFromToday + "'")

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

func fetchResponseForMatchBet(matchID int) *discordgo.InteractionResponse{
	db, err := sql.Open("sqlite3", "../svffscraper/foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	row := db.QueryRow("SELECT * FROM matches WHERE id=" + strconv.Itoa(matchID))

	var (
		id int
		home string
		away string
		date string
	)

	row.Scan(&id, &home, &away, &date)

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: home + " (h) vs " + away + " (b) @ " + date + "\n\n**Poäng**",
			Flags: 1 << 6, // Ephemeral
			Components: []discordgo.MessageComponent {
				discordgo.ActionsRow {
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							// Select menu, as other components, must have a customID, so we set it to this value.
							CustomID:    "scoreHome",
							Placeholder: "Hemmalag",
							Options: getScoreMenuOptions(id),
						},
					},
				},
				discordgo.ActionsRow {
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							// Select menu, as other components, must have a customID, so we set it to this value.
							CustomID:    "scoreAway",
							Placeholder: "Bortalag",
							Options: getScoreMenuOptions(id),
						},
					},
				},
			},
		},
	}

	return response
}

func setScoreHome(value int) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: strconv.Itoa(value),
			Flags: 1 << 6, // Ephemeral
		},
	}
}

func setScoreAway(value int) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: strconv.Itoa(value),
			Flags: 1 << 6, // Ephemeral
		},
	}
}

func getScoreMenuOptions(matchID int) []discordgo.SelectMenuOption {
	scores := []discordgo.SelectMenuOption {}

	for i := 0; i < 25; i++ {
		scores = append(scores, discordgo.SelectMenuOption{
			Label: strconv.Itoa(i),
			Value: strconv.Itoa(matchID) + "_" + strconv.Itoa(i),
		})
	}

	return scores
}

func main() {
	var s *discordgo.Session
	var err error

	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		fmt.Println("Invalid bot parameters: %v", err)
	}

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	var (
		commands = []discordgo.ApplicationCommand {
		{
			Name: "hjälp",
			Description: "Få hjälp med hur denna bot fungerar.",
		},
		{
			Name: "kommande",
			Description: "Kommande matcher.",
		},
		{
			Name: "vadslå",
			Description: "Vadslå på kommande match.",
		},
	})

	var commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		"hjälp": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData {
					Flags: 1 << 6, // Ephemeral
					Content: "/vadslå - Starta en ny vadslagning över en specifik match.",
					},
			})
		},
		"kommande": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData {
					Flags: 1 << 6, // Ephemeral
					Content: getNextMatches(),
				},
			})
		},
		"vadslå": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData {
					Flags: 1 << 6, // Ephemeral
					Content: "Matcher",
					Components: []discordgo.MessageComponent {
						// ActionRow is a container of all buttons within the same row.
						discordgo.ActionsRow {
							Components: []discordgo.MessageComponent {
								discordgo.SelectMenu {
									Placeholder: "Välj en match",
									CustomID: "selectMatch",
									Options: fetchBettableMatches(),
								},
							},
						},
					},
				},
			})
		},
	}

	var componentHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		"selectMatch": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			data, _ := strconv.Atoi(i.MessageComponentData().Values[0])
			response := fetchResponseForMatchBet(data)
			err := s.InteractionRespond(i.Interaction, response)
			if err != nil {
				panic(err)
			}
		},
		"scoreHome": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			data, _ := strconv.Atoi(i.MessageComponentData().Values[0])
			response := setScoreHome(data)
			err := s.InteractionRespond(i.Interaction, response)
			if err != nil {
				panic(err)
			}
		},
		"scoreAway": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			data, _ := strconv.Atoi(i.MessageComponentData().Values[0])
			response := setScoreAway(data)
			err := s.InteractionRespond(i.Interaction, response)
			if err != nil {
				panic(err)
			}
		},
	}

	cmdIDs := make(map[string]string, len(commands))

	for _, cmd := range commands {
		rcmd, err := s.ApplicationCommandCreate(*AppID, *GuildID, &cmd)
		if err != nil {
			log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
		}

		cmdIDs[rcmd.ID] = rcmd.Name
	}

	// Components are part of interactions, so we register InteractionCreate handler
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
			case discordgo.InteractionApplicationCommand:
				if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
					h(s, i)
				}
			case discordgo.InteractionMessageComponent:

				if h, ok := componentHandlers[i.MessageComponentData().CustomID]; ok {
					h(s, i)
				}
		}
	})

	fmt.Println("Starting...")
	err = s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	fmt.Println("Press Ctrl+C to exit")
	<-stop

	fmt.Println("Quitting...")
}

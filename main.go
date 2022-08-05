package main

import (
	"math"
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
	TIME_LAYOUT = time.RFC3339
)


/*
 Bot parameters
*/

var (
	GUILD_ID  = flag.String("guild", "", "Test guild ID")
	BOT_TOKEN = flag.String("token", "MTAwMDgzNDQ3MzIyODc3OTU4Mg.GMTTc2.8vE1wGIbRP41q6G_md3FhXfHAISDZww2Ja0aTs", "Bot access token")
	APP_ID    = flag.String("app", "1000834473228779582", "Application ID")
    REFRESH   = flag.Bool("refresh", false, "Refresh commands on start")
)

const (
	OWNER = 436614981283217418
)


/*
 Types and structs
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

type bet struct {
	id int
	uid int
	matchid int
	homeScore int
	awayScore int
	handled int
}

type location int
const (
	Home location = iota
	Away
)

/*
  Command functions
*/

// Command: hjälp
func helpCommand(s *discordgo.Session, i *discordgo.InteractionCreate, COMMANDS *[]discordgo.ApplicationCommand) {
	help := "Denna bot är till för att kunna slå vad om hur olika Allsvenska matcher kommer sluta.\n" +
		    "\n" +
	        "**Kommandon**\n"

	for _, elem := range *COMMANDS {
		if elem.Name != "sammanfatta" && elem.Name != "refresh" {
			help = help + fmt.Sprintf("/%v - %v\n", elem.Name, elem.Description)
		}
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
			Content: help,
		},
	}); err != nil { log.Panic(err) }
}

// Command: slåvad
func betCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	db, err := sql.Open(DB_TYPE, SVFF_DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	today := time.Now().Format("2006-01-02")
	tenFromToday := time.Now().AddDate(0, 0, 10).Format("2006-01-02")

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

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
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
							Options: response,
						},
					},
				},
			},
		},
	}); err != nil { log.Panic(err) }
}

// Command: utmana
func challengeCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
			Content: "Utmana",
		},
	}); err != nil { log.Panic(err) }
}

// Command: kommande
func upcomingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

	uID := i.Interaction.Member.User.ID

	bets, _ := betsDB.Query("SELECT id, uid, matchid, homeScore, awayScore FROM bets WHERE uid=? and handled=0", uID)
	defer bets.Close()

	var b bet

	userBets := ""

	for bets.Next() {
		bets.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore)
		matchRow := svffDB.QueryRow("SELECT homeTeam, awayTeam, date FROM matches WHERE id=?", b.matchid)

		var m match
		matchRow.Scan(&m.homeTeam, &m.awayTeam, &m.date)

		date, _ := time.Parse(TIME_LAYOUT, m.date)
		daysUntil := math.Round(time.Until(date).Hours() / 24)

		userBets = userBets + fmt.Sprintf("%v (**%v**) - %v (**%v**), spelas om %v dagar.\n", m.homeTeam, b.homeScore, m.awayTeam, b.awayScore, daysUntil)
	}

	if userBets == "" {
		userBets = "Inga vadslagningar ännu!"
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
			Content: userBets,
		},
	}); err != nil { log.Panic(err) }
}

// Command: tidigare
func earlierCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

	uID := i.Interaction.ApplicationCommandData().Options[0].Value

	bets, _ := betsDB.Query("SELECT id, uid, matchid, homeScore, awayScore FROM bets WHERE uid=? and handled=1", uID)
	defer bets.Close()

	var hidden = 0

	if err := betsDB.QueryRow("SELECT hidden FROM points WHERE uid=?", uID).Scan(&hidden); err != nil {
		if err != sql.ErrNoRows { log.Panic(err) }
	}

	userBets := ""

	if hidden != 1 {
		var b bet

		for bets.Next() {
			bets.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore)
			matchRow := svffDB.QueryRow("SELECT homeTeam, awayTeam, date FROM matches WHERE id=?", b.matchid)

			var m match
			matchRow.Scan(&m.homeTeam, &m.awayTeam, &m.date)

			userBets = userBets + fmt.Sprintf("%v (**%v**) - %v (**%v**)", m.homeTeam, b.homeScore, m.awayTeam, b.awayScore)
		}

		if userBets == "" {
			userBets = fmt.Sprintf("Användaren har inga vadslagningar ännu!", )
		}
	} else {
		userBets = "Användaren har valt att dölja sina vadslagningar."
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
			Content: userBets,
		},
	}); err != nil { log.Panic(err) }
}

// Command: poäng
func pointsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

	rows, err := betsDB.Query("SELECT uid, season FROM points ORDER BY season DESC LIMIT 10")
	defer rows.Close()
	if err != nil { log.Panic(err) }

	str := ""
	pos := 1

	for rows.Next() {
		var (
			uid int
			season int
		)

		rows.Scan(&uid, &season)
		user, _ := s.User(strconv.Itoa(uid))

		str += fmt.Sprintf("#%v @%v med %v poäng\n", pos, user.Username, season)
	}

	str += "--------------\n"

	uPoints := 0
	if err := betsDB.QueryRow("SELECT season FROM points WHERE uid=?", i.Member.User.ID).Scan(&uPoints); err != nil {
		if err == sql.ErrNoRows {
			// skip
		} else {
			log.Panic(err)
		}
	}

	str += fmt.Sprintf("Dina poäng i år: %v", uPoints)

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
			Content: str,
		},
	}); err != nil { log.Panic(err) }
}

// Command: inställningar
func settingsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
			Content: "...",
		},
	}); err != nil { log.Panic(err) }
}

// Command: sammanfatta
func summaryCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if isOwner(i) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData {
				Flags: 1 << 6, // Ephemeral
				Content: "Sammmanfatta",
			},
		}); err != nil { log.Panic(err) }
	} else {
		if err := s.InteractionRespond(i.Interaction, notOwnerResponse()); err != nil { log.Panic(err) }
	}
}

// Command: refresh
func refreshCommand(s *discordgo.Session, i *discordgo.InteractionCreate, COMMANDS *[]discordgo.ApplicationCommand) {
	if isOwner(i) {
        log.Println("Refreshing commands...")

		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData {
				Flags: 1 << 6, // Ephemeral
				Content: "Refreshing commands...",
            },
        }); err != nil { log.Panic(err) }

        cID := ""

        if len(i.Interaction.ApplicationCommandData().Options) == 1 {
            cID = fmt.Sprintf("%v", i.Interaction.ApplicationCommandData().Options[0].Value)
        }

        cmds, _ := s.ApplicationCommands(*APP_ID, *GUILD_ID)

        for _, cmd := range cmds {
            if cmd.Name == cID {
                log.Printf("Deleting: %v", cmd.Name)
                s.ApplicationCommandDelete(*APP_ID, *GUILD_ID, cmd.ID)
            } else if cID == "" {
                log.Printf("Deleting: %v", cmd.Name)
                s.ApplicationCommandDelete(*APP_ID, *GUILD_ID, cmd.ID)
            }
        }

        // Initialize commands
        cmdIDs := make(map[string]string, len(*COMMANDS))

        for _, cmd := range *COMMANDS {
            if cmd.Name == cID {
                log.Printf("Adding: %v", cmd.Name)

                rcmd, err := s.ApplicationCommandCreate(*APP_ID, *GUILD_ID, &cmd)
                if err != nil {
                    log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
                }

                cmdIDs[rcmd.ID] = rcmd.Name
            } else if cID == "" {
                log.Printf("Adding: %v", cmd.Name)

                rcmd, err := s.ApplicationCommandCreate(*APP_ID, *GUILD_ID, &cmd)
                if err != nil {
                    log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
                }

                cmdIDs[rcmd.ID] = rcmd.Name
            }
        }

        log.Println("Finished refreshing!")
	} else {
		if err := s.InteractionRespond(i.Interaction, notOwnerResponse()); err != nil { log.Panic(err) }
	}
}

/*
  Component handlers
*/

func matchBetComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData {
			Content: fmt.Sprintf("%v (h) vs %v (b) @ %v \n\n**Poäng** *(hemmalag överst)*", m.homeTeam, m.awayTeam, m.date),
			Flags: 1 << 6, // Ephemeral
			Components: []discordgo.MessageComponent {
				discordgo.ActionsRow {
					Components: []discordgo.MessageComponent {
						discordgo.SelectMenu {
							// Select menu, as other components, must have a customID, so we set it to this value.
							CustomID:    "scoreHome",
							Placeholder: "Hemmalag",
							Options: getScoreMenuOptions(m.id, defHome),
						},
					},
				},
				discordgo.ActionsRow {
					Components: []discordgo.MessageComponent {
						discordgo.SelectMenu {
							// Select menu, as other components, must have a customID, so we set it to this value.
							CustomID:    "scoreAway",
							Placeholder: "Bortalag",
							Options: getScoreMenuOptions(m.id, defAway),
						},
					},
				},
			},
		},
	}); err != nil { log.Panic(err) }
}

func scoreComponent(s *discordgo.Session, i *discordgo.InteractionCreate, where location) {
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

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
    }); err != nil { log.Panic(err) }
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

func getCommandsAsChoices(s *discordgo.Session) []*discordgo.ApplicationCommandOptionChoice {
    cmds, err := s.ApplicationCommands(*APP_ID, *GUILD_ID)
    if err != nil { log.Panic(err) }

    var choices []*discordgo.ApplicationCommandOptionChoice

    for _, cmd := range cmds {
        choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
            Name: cmd.Name,
            Value: cmd.Name,
        })
    }

    return choices
}

func isOwner(i *discordgo.InteractionCreate) bool {
	if i.Member.User.ID == strconv.Itoa(OWNER) {
		return true
	}

	return false
}

func notOwnerResponse() *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse {
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData {
            Flags: 1 << 6, // Ephemeral
            Content: "Du har inte rättigheter att köra kommandot.",
        },
    }
}


/*
 Interval functions
*/

func handleTodaysBets() {
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	log.Printf("Handling bets for %v...", yesterday)

	// Fetch all match IDs for today
	// date(date) tells SQLite to turn date='2006-01-02T00:00:00.00' to '2006-01-02'
	rows, err := svffDB.Query("SELECT id FROM matches WHERE date(date)=? AND finished=1", yesterday)
	if err != nil { log.Panic(err) }

	var matchIDs []int

	for rows.Next() {
		var matchID int
		rows.Scan(&matchID)
		matchIDs = append(matchIDs, matchID)
	}

	rows.Close()

	if len(matchIDs) == 0 {
		log.Print("No matches to handle!")
	} else {
		log.Printf("%v matches to handle...", len(matchIDs))

		// Handle bets for each match individually
		for _, mID := range matchIDs {
			var (
				rHomeScore int
				rAwayScore int
			)

			row := svffDB.QueryRow("SELECT scoreHome, scoreAway FROM matches WHERE id=?", mID)
			if err := row.Scan(&rHomeScore, &rAwayScore); err != nil { log.Panic(err) }

			betRows, err := betsDB.Query("SELECT id, uid, homeScore, awayScore FROM bets WHERE matchid=? and handled=0", mID)
			defer betRows.Close()
			if err != nil { log.Panic(err) }

			var bets []bet

			for betRows.Next() {
				var bet bet

				betRows.Scan(&bet.id, &bet.uid, &bet.homeScore, &bet.awayScore)

				bets = append(bets, bet)
			}

			betRows.Close()


			for _, bet := range bets {
				if rHomeScore == bet.homeScore && rAwayScore == bet.awayScore {
					addPoints(bet, 1)
				} else {
					addPoints(bet, 0)
				}
			}

			log.Printf("%v bets handled!", len(bets))
		}
	}
}

func addPoints(b bet, points int) {
	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

	row := betsDB.QueryRow("SELECT season FROM points WHERE uid=?", b.uid)
	if err != nil { log.Panic(err) }

	var currPoints int
	if err := row.Scan(&currPoints); err != nil {
		if err == sql.ErrNoRows {
			if _, err := betsDB.Exec("INSERT INTO points (uid, season) VALUES (?, ?)", b.uid, points); err != nil { log.Panic(err) }
		} else {
			log.Panic(err)
		}
	} else {
		if _, err := betsDB.Exec("UPDATE points SET season=season + ? WHERE uid=?", points, b.uid); err != nil { log.Panic(err) }
	}

	if _, err := betsDB.Exec("UPDATE bets SET handled=1 WHERE id=?", b.id); err != nil { log.Panic(err) }
}


/*
  Initialization
*/

func initializeBot() *discordgo.Session {

	log.Print("Initializing...")

	// Login bot
	s, err := discordgo.New("Bot " + *BOT_TOKEN)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
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
				Name: "slåvad",
				Description: "Slå vad om en kommande match.",
			},
			{
				Name: "utmana",
				Description: "Utmana en annan användare om en kommande match.",
				Options: []*discordgo.ApplicationCommandOption {
					{
						Type: discordgo.ApplicationCommandOptionUser,
						Name: "namn",
						Description: "Användare att utmana.",
                        Required: true,
					},
					{
						Type: discordgo.ApplicationCommandOptionString,
						Name: "typ",
                        Description: "Vilken sorts utmaning?",
                        Required: true,
                        Choices: []*discordgo.ApplicationCommandOptionChoice {
                            {
                                Name: "Matchvinnare",
                                Value: "matchvinnare",
                            },
                        },
					},
				},
			},
			{
				Name: "kommande",
				Description: "Lista dina kommande vadslagningar.",
			},
			{
				Name: "tidigare",
				Description: "Lista en annan användares tidigare vadslagningar.",
				Options: []*discordgo.ApplicationCommandOption {
					{
						Type: discordgo.ApplicationCommandOptionUser,
						Name: "namn",
						Description: "Användare att visa vadslagningar för.",
					},
				},
			},
			{
				Name: "poäng",
				Description: "Visa dina poäng och topp 10 på servern.",
			},
			{
				Name: "inställningar",
				Description: "Ställ in inställningar för din användare.",
			},
			{
				Name: "sammanfatta",
				Description: "Sammanfatta denna omgång till #bets.",
			},
			{
				Name: "refresh",
				Description: "Refresha enskilt eller alla kommandon.",
				Options: []*discordgo.ApplicationCommandOption {
					{
						Type: discordgo.ApplicationCommandOptionString,
						Name: "kommando",
						Description: "Kommando att refresha",
                        Choices: getCommandsAsChoices(s),
					},
                },
			},
		}

		COMMAND_HANDLERS = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			"hjälp": func(s *discordgo.Session, i *discordgo.InteractionCreate)         {   helpCommand(s, i, &COMMANDS)    },
			"slåvad": func(s *discordgo.Session, i *discordgo.InteractionCreate)        {   betCommand(s, i)                },
			"utmana": func(s *discordgo.Session, i *discordgo.InteractionCreate)        {   challengeCommand(s, i)          },
			"kommande": func(s *discordgo.Session, i *discordgo.InteractionCreate)      {   upcomingCommand(s, i)           },
			"tidigare": func(s *discordgo.Session, i *discordgo.InteractionCreate)      {   earlierCommand(s, i)            },
			"poäng": func(s *discordgo.Session, i *discordgo.InteractionCreate)         {   pointsCommand(s, i)             },
			"inställningar": func(s *discordgo.Session, i *discordgo.InteractionCreate) {   settingsCommand(s, i)           },
			"sammanfatta": func(s *discordgo.Session, i *discordgo.InteractionCreate)   {   summaryCommand(s,i )            },
			"refresh": func(s *discordgo.Session, i *discordgo.InteractionCreate)       {   refreshCommand(s, i, &COMMANDS) },
		}

		COMPONENT_HANDLERS = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			"selectMatch": func(s *discordgo.Session, i *discordgo.InteractionCreate)   {   matchBetComponent(s, i)         },
			"scoreHome": func(s *discordgo.Session, i *discordgo.InteractionCreate)     {   scoreComponent(s, i, Home)      },
			"scoreAway": func(s *discordgo.Session, i *discordgo.InteractionCreate)     {   scoreComponent(s, i, Away)      },
		}
	)

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

    if *REFRESH {
        cmds, _ := s.ApplicationCommands(*APP_ID, *GUILD_ID)

        for _, cmd := range cmds {
            log.Printf("Deleting: %v", cmd.Name)
            s.ApplicationCommandDelete(*APP_ID, *GUILD_ID, cmd.ID)
        }

        // Initialize commands
        cmdIDs := make(map[string]string, len(COMMANDS))

        for _, cmd := range COMMANDS {
            log.Printf("Adding: %v", cmd.Name)

            rcmd, err := s.ApplicationCommandCreate(*APP_ID, *GUILD_ID, &cmd)
            if err != nil {
                log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
            }

            cmdIDs[rcmd.ID] = rcmd.Name
        }
    }

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

    flag.Parse()

	// Initialize and start the bot
	s := initializeBot()
	defer s.Close()

	// Check the bets on a timed interval
	c := cron.New()
	c.AddFunc("@every 30m", handleTodaysBets)
	c.Start()

	// Wait for stop signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Print("Quitting...")
}

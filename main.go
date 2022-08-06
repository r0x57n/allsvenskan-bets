package main

import (
	"math"
	"flag"
	"fmt"
	"strconv"
	"log"
	"time"
	"os"
	"os/signal"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
	dg "github.com/bwmarrin/discordgo"
)


/*
 Paths
*/

const (
	SVFF_DB = "../svffscraper/foo.db"
	BETS_DB = "./bets.db"
	DB_TYPE = "sqlite3"
	TIME_LAYOUT = time.RFC3339
    VERSION = "0.1.0" // major.minor.patch
)


/*
 Bot parameters
*/

var (
	GUILD_ID  = flag.String("guild", "", "Test guild ID")
	BOT_TOKEN = flag.String("token", "MTAwMDgzNDQ3MzIyODc3OTU4Mg.GMTTc2.8vE1wGIbRP41q6G_md3FhXfHAISDZww2Ja0aTs", "Bot access token")
	APP_ID    = flag.String("app", "1000834473228779582", "Application ID")
    RR        = flag.Bool("RR", false, "Remove commands and refresh") // Be careful with this, max of 200 commands can be added in a day, removing and adding counts...
    UPDATE    = flag.Bool("update", false, "Update commands")
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

type user struct {
    uid int
    season int
    history string
    viewable int
    interactable int
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
func helpCommand(s *dg.Session, i *dg.InteractionCreate, COMMANDS *[]dg.ApplicationCommand) {
	help := "Denna bot är till för att kunna slå vad om hur olika Allsvenska matcher kommer sluta.\n" +
		    "\n" +
            "Du kan */vadslå* över en match. Då slår du vad om hur du tror en match kommer sluta poängmässigt. Har du rätt vinner du poäng som kan användas till antingen **skryträtt**, eller för att */utmana* andra användare.\n" +
            "När du utmanar en annan användare väljer du en utmaning och hur många poäng du satsar på ditt utfall. Vinnaren tar alla poängen.\n" +
            "\n" +
            "Alla vadslagningar kollas runt midnatt och det är först då poängen delas ut.\n" +
            "\n" +
	        "**Kommandon**\n"

    adminOnly := map[string]int{"sammanfatta": 1, "update": 1, "delete": 1}

	for _, elem := range *COMMANDS {
		if _, ok := adminOnly[elem.Name]; !ok {
			help = help + fmt.Sprintf("/%v - %v\n", elem.Name, elem.Description)
		}
	}

    msgStdInteractionResponse(s, i, help)
}

// Command: kommande
func upcomingCommand(s *dg.Session, i *dg.InteractionCreate) {
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

    msgStdInteractionResponse(s, i, userBets)
}

func regretCommand(s *dg.Session, i *dg.InteractionCreate) {
    msgStdInteractionResponse(s, i, "Ej implementerat.")
}

// Command: tidigare
func earlierCommand(s *dg.Session, i *dg.InteractionCreate) {
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

	uID := i.Interaction.ApplicationCommandData().Options[0].Value

	bets, _ := betsDB.Query("SELECT id, uid, matchid, homeScore, awayScore FROM bets WHERE uid=? and handled=1", uID)
	defer bets.Close()

	var viewable = 0

	if err := betsDB.QueryRow("SELECT viewable FROM points WHERE uid=?", uID).Scan(&viewable); err != nil {
		if err != sql.ErrNoRows { log.Panic(err) }
	}

	userBets := ""

	if viewable == 1 {
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

    msgStdInteractionResponse(s, i, userBets)
}

// Command: poäng
func pointsCommand(s *dg.Session, i *dg.InteractionCreate) {
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

    msgStdInteractionResponse(s, i, str)
}

// Command: sammanfatta
func summaryCommand(s *dg.Session, i *dg.InteractionCreate) {
	if isOwner(i) {
        msgStdInteractionResponse(s, i, "Sammanfatta")
	} else {
        msgStdInteractionResponse(s, i, "Du har inte rättigheter att köra detta kommando.")
	}
}

// Command: info
func infoCommand(s *dg.Session, i *dg.InteractionCreate) {
    str := "Jag är en bot gjord i Go med hjälp av [discordgo](https://github.com/bwmarrin/discordgo) paketet. Min källkod finns på [Github](https://github.com/r0x57n/allsvenskanBets)." +
           "\n\n" +
           "Den version jag kör är: " + VERSION

	if err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse {
		Type: dg.InteractionResponseChannelMessageWithSource,
		Data: &dg.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
			Content: "",
            Embeds: []*dg.MessageEmbed {
                {
                    Title:     "Hej!",
                    Description: str,
                },
            },

		},
	}); err != nil { log.Panic(err) }
}


/*
 Helper functions
*/

func getScoreMenuOptions(matchID int, defScore int) []dg.SelectMenuOption {
	scores := []dg.SelectMenuOption {}

	for i := 0; i < 25; i++ {
		if defScore == i {
			scores = append(scores, dg.SelectMenuOption{
				Label: strconv.Itoa(i),
				Value: fmt.Sprintf("%v_%v", matchID, i),
				Default: true,
			})
		} else {
			scores = append(scores, dg.SelectMenuOption{
				Label: strconv.Itoa(i),
				Value: fmt.Sprintf("%v_%v", matchID, i),
				Default: false,
			})
		}
	}

	return scores
}

func getCommandsAsChoices(s *dg.Session) []*dg.ApplicationCommandOptionChoice {
    cmds, err := s.ApplicationCommands(*APP_ID, *GUILD_ID)
    if err != nil { log.Panic(err) }

    var choices []*dg.ApplicationCommandOptionChoice

    for _, cmd := range cmds {
        choices = append(choices, &dg.ApplicationCommandOptionChoice{
            Name: cmd.Name,
            Value: cmd.ID,
        })
    }

    return choices
}

func isOwner(i *dg.InteractionCreate) bool {
	if i.Member.User.ID == strconv.Itoa(OWNER) {
		return true
	}

	return false
}

func addInteractionResponse(s *dg.Session,
                         i *dg.InteractionCreate,
                         interactionType dg.InteractionResponseType,
                         msg string) {
	if err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse {
		Type: interactionType,
		Data: &dg.InteractionResponseData {
			Content: msg,
			Flags: 1 << 6, // Ephemeral
		},
	}); err != nil { log.Panic(err) }

}

func msgStdInteractionResponse(s *dg.Session, i *dg.InteractionCreate, msg string) {
	if err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse {
		Type: dg.InteractionResponseChannelMessageWithSource,
		Data: &dg.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
			Content: msg,
		},
	}); err != nil { log.Panic(err) }
}

func compInteractionResponse(s *dg.Session,
                                i *dg.InteractionCreate,
                                interactionType dg.InteractionResponseType,
                                msg string,
                                components []dg.MessageComponent) {
	if err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse {
		Type: interactionType,
		Data: &dg.InteractionResponseData {
			Content: msg,
			Components: components,
			Flags: 1 << 6, // Ephemeral
        },
	}); err != nil { log.Panic(err) }
}

func getUser(uid string) user {
	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

    var u user

	err = betsDB.QueryRow("SELECT uid, season, history, viewable, interactable FROM points WHERE uid=?", uid).
                 Scan(&u.uid, &u.season, &u.history, &u.viewable, &u.interactable)
	if err != nil {
        if err == sql.ErrNoRows {
            u.uid, err = strconv.Atoi(uid)
            if err != nil { log.Panic(err) }

            u.season = 0
            u.history = ""
            u.viewable = 1
            u.interactable = 1

            _, err = betsDB.Exec("INSERT INTO points (uid, season) VALUES (?, ?)", u.uid, u.season)
            if err != nil { log.Panic(err) }
        } else {
            log.Panic(err)
        }
    }

    return u
}

// Parameter is optional string to add to value of the options.
// This is so we can add meta data about things such as challenges, we need
// to remember who the challengee was...
func getRoundMatchesAsOptions(value ...string) *[]dg.SelectMenuOption {
	db, err := sql.Open(DB_TYPE, SVFF_DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	today := time.Now().Format("2006-01-02")
	tenFromToday := time.Now().AddDate(0, 0, 10).Format("2006-01-02")

	rows, _ := db.Query("SELECT id, homeTeam, awayTeam, date, scoreHome, scoreAway, finished FROM matches WHERE date BETWEEN ? and ?", today, tenFromToday)
	defer rows.Close()

	var matches []*match
	for rows.Next() {
		var m match
		if err := rows.Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date, &m.scoreHome, &m.scoreAway, &m.finished); err != nil { log.Panic(err) }
		matches = append(matches, &m)
	}

    options := []dg.SelectMenuOption{}

	if len(matches) == 0 {
		options = append(options, dg.SelectMenuOption{
			Label: "Inga matcher tillgängliga kommande tio dagar.",
			Value: "",
			Description: "",
		})
	} else {
		for _, m := range matches {
            val := strconv.Itoa(m.id)

            if len(value) == 1 {
                val = value[0] + "_" + val
            }

            options = append(options, dg.SelectMenuOption{
                Label: fmt.Sprintf("%v vs %v", m.homeTeam, m.awayTeam),
                Value: val,
                Description: m.date,
            })
		}
	}

    return &options
}


/*
  Initialization
*/

func initializeBot() *dg.Session {

	log.Print("Initializing...")

	// Login bot
	s, err := dg.New("Bot " + *BOT_TOKEN)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	/*
	Commands and handlers declarations
	*/

	var (
		COMMANDS = []dg.ApplicationCommand {
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
				Options: []*dg.ApplicationCommandOption {
					{
						Type: dg.ApplicationCommandOptionUser,
						Name: "användarnamn",
						Description: "Användare att utmana.",
                        Required: true,
					},
					{
						Type: dg.ApplicationCommandOptionString,
						Name: "typ",
                        Description: "Vilken sorts utmaning?",
                        Required: true,
                        Choices: []*dg.ApplicationCommandOptionChoice {
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
				Name: "ångra",
				Description: "Ta bort ett vad som du gjort, om utmaning måste andra användaren också acceptera.",
			},
			{
				Name: "tidigare",
				Description: "Lista en annan användares tidigare vadslagningar.",
				Options: []*dg.ApplicationCommandOption {
					{
						Type: dg.ApplicationCommandOptionUser,
						Name: "användarnamn",
						Description: "Användare att visa vadslagningar för.",
                        Required: true,
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
				Name: "info",
				Description: "Teknisk info om mig.",
			},
			{
				Name: "sammanfatta",
				Description: "Sammanfatta denna omgång till #bets.",
			},
			{
				Name: "update",
				Description: "Uppdatera alla kommandon eller ett enskilt.",
				Options: []*dg.ApplicationCommandOption {
					{
						Type: dg.ApplicationCommandOptionString,
						Name: "kommando",
						Description: "Kommando att uppdatera.",
                        Choices: getCommandsAsChoices(s),
					},
                },
			},
			{
				Name: "delete",
				Description: "Ta bort ett enskilt kommando.",
				Options: []*dg.ApplicationCommandOption {
					{
						Type: dg.ApplicationCommandOptionString,
						Name: "kommando",
						Description: "Kommando att ta bort.",
                        Required: true,
                        Choices: getCommandsAsChoices(s),
					},
                },
			},
		}

		COMMAND_HANDLERS = map[string]func(s *dg.Session, i *dg.InteractionCreate) {
            // User commands
			"hjälp": func(s *dg.Session, i *dg.InteractionCreate)         {   helpCommand(s, i, &COMMANDS)    },
			"slåvad": func(s *dg.Session, i *dg.InteractionCreate)        {   betCommand(s, i)                },
			"utmana": func(s *dg.Session, i *dg.InteractionCreate)        {   challengeCommand(s, i)          },
			"kommande": func(s *dg.Session, i *dg.InteractionCreate)      {   upcomingCommand(s, i)           },
			"ångra": func(s *dg.Session, i *dg.InteractionCreate)         {   regretCommand(s, i)             },
			"tidigare": func(s *dg.Session, i *dg.InteractionCreate)      {   earlierCommand(s, i)            },
			"poäng": func(s *dg.Session, i *dg.InteractionCreate)         {   pointsCommand(s, i)             },
			"inställningar": func(s *dg.Session, i *dg.InteractionCreate) {   settingsCommand(s, i)           },
			"info": func(s *dg.Session, i *dg.InteractionCreate)          {   infoCommand(s, i)              },

            // Admin commands
			"sammanfatta": func(s *dg.Session, i *dg.InteractionCreate)   {   summaryCommand(s,i )            },
			"update": func(s *dg.Session, i *dg.InteractionCreate)        {   updateCommand(s, i, &COMMANDS)  },
			"delete": func(s *dg.Session, i *dg.InteractionCreate)        {   deleteCommand(s, i)             },
		}

        // Component handlers
		COMPONENT_HANDLERS = map[string]func(s *dg.Session, i *dg.InteractionCreate) {
			"betOnSelected": func(s *dg.Session, i *dg.InteractionCreate)      {   betOnSelected(s, i)             },
			"betScoreHome": func(s *dg.Session, i *dg.InteractionCreate)       {   betScoreComponent(s, i, Home)      },
			"betScoreAway": func(s *dg.Session, i *dg.InteractionCreate)       {   betScoreComponent(s, i, Away)      },
			"challSelectWinner": func(s *dg.Session, i *dg.InteractionCreate)  {   challSelectWinner(s, i)           },
			"challSelectPoints": func(s *dg.Session, i *dg.InteractionCreate)  {   challSelectPoints(s, i)               },
			"challAcceptDiscard": func(s *dg.Session, i *dg.InteractionCreate) {   challAcceptDiscard(s, i)               },
            "settingsVisibility": func(s *dg.Session, i *dg.InteractionCreate) {   settingsVisibility(s, i)               },
            "settingsChall": func(s *dg.Session, i *dg.InteractionCreate)      {   settingsChall(s, i)               },
		}
	)

	s.AddHandler(func(s *dg.Session, i *dg.InteractionCreate) {
		switch i.Type {
			case dg.InteractionApplicationCommand:
				if h, ok := COMMAND_HANDLERS[i.ApplicationCommandData().Name]; ok { h(s, i) }
			case dg.InteractionMessageComponent:
				if h, ok := COMPONENT_HANDLERS[i.MessageComponentData().CustomID]; ok { h(s, i) }
		}
	})

	s.AddHandler(func(s *dg.Session, r *dg.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

    if *RR || *UPDATE {

        // Delete earlier commands
        if *RR {
            cmds, _ := s.ApplicationCommands(*APP_ID, *GUILD_ID)

            for _, cmd := range cmds {
                log.Printf("Deleting: %v", cmd.Name)
                s.ApplicationCommandDelete(*APP_ID, *GUILD_ID, cmd.ID)
            }
        }

        // Update/add commands
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

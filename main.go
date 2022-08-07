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
    VERSION = "0.2.0" // major.minor.patch
)


/*
 Bot parameters
*/

var (
	GUILD_ID  = flag.String("guild", "", "Test guild ID")
	BOT_TOKEN = flag.String("token", "MTAwMDgzNDQ3MzIyODc3OTU4Mg.GMTTc2.8vE1wGIbRP41q6G_md3FhXfHAISDZww2Ja0aTs", "Bot access token")
	APP_ID    = flag.String("app", "1000834473228779582", "Application ID")
    DELETE    = flag.Bool("delete", false, "Remove all commands")
    UPDATE    = flag.Bool("update", false, "Update/add all commands")
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
    won int
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
            "Resultaten för matcherna uppdateras lite då och då under dagen, därför kan det ta ett tag till att poängen delas ut efter en match är spelad.\n" +
            "\n" +
	        "**Kommandon**\n"

    adminOnly := map[string]int{"sammanfatta": 1, "update": 1, "delete": 1, "checkbets": 1}

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

    // Get options and parse
    options := i.Interaction.ApplicationCommandData().Options
	uID := options[0].Value

    betType := 2 // 0 = lost, 1 = won, 2 = all
    if len(options) == 2 {
        betType, _ = strconv.Atoi(fmt.Sprintf("%v", options[1].Value))
    }

    // Get bets
    var bets *sql.Rows
    switch betType {
        case 0:
            bets, err = betsDB.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE uid=? AND handled=1 AND won=0", uID)
        case 1:
            bets, err = betsDB.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE uid=? AND handled=1 AND won=1", uID)
        default:
            bets, err = betsDB.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE uid=? AND handled=1", uID)
    }
	if err != nil { log.Fatal(err) }
    defer bets.Close()

	var viewable = 0

	if err := betsDB.QueryRow("SELECT viewable FROM points WHERE uid=?", uID).Scan(&viewable); err != nil {
		if err != sql.ErrNoRows { log.Panic(err) }
	}

	userBets := ""

	if viewable == 1 {
		var b bet

		for bets.Next() {
			bets.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore, &b.won)
			matchRow := svffDB.QueryRow("SELECT homeTeam, awayTeam, date FROM matches WHERE id=?", b.matchid)

			var m match
			matchRow.Scan(&m.homeTeam, &m.awayTeam, &m.date)

            wonStr := ""
            if betType == 2 {
                wonStr = " - Korrekt"
                if b.won == 0 {
                    wonStr = " - Inkorrekt"
                }
            }

			userBets = userBets + fmt.Sprintf("%v (**%v**) - %v (**%v**)%v\n", m.homeTeam, b.homeScore, m.awayTeam, b.awayScore, wonStr)
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

		str += fmt.Sprintf("#%v **%v** med %v poäng\n", pos, user.Username, season)
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

// Command: checkbets
func checkBetsCommand(s *dg.Session, i *dg.InteractionCreate) {
	if isOwner(i) {
        msgStdInteractionResponse(s, i, "Checking bets...")
        checkUnhandledBets()
	} else {
        msgStdInteractionResponse(s, i, "Du har inte rättigheter att köra detta kommando.")
	}
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
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	today := time.Now().Format("2006-01-02")
	todayAndTime := time.Now().Format(TIME_LAYOUT)

    round := -1
    err = svffDB.QueryRow("SELECT round FROM matches WHERE date(date)>=? AND finished='0' ORDER BY date", today).Scan(&round)
    if err != nil { log.Panic(err) }

	rows, err := svffDB.Query("SELECT id, homeTeam, awayTeam, date, scoreHome, scoreAway, finished FROM matches WHERE round=? AND date>=?", round, todayAndTime)
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
			Value: "",
			Description: "",
		})
	} else {
		for _, m := range matches {
            val := strconv.Itoa(m.id)

            if len(value) == 1 {
                val = value[0] + "_" + val
            }

            datetime, _ := time.Parse(TIME_LAYOUT, m.date)

            options = append(options, dg.SelectMenuOption{
                Label: fmt.Sprintf("%v vs %v", m.homeTeam, m.awayTeam),
                Value: val,
                Description: datetime.Format("2006-02-01 kl. 15:04"),
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

    // Delete commands
    if *DELETE {
        cmds, _ := s.ApplicationCommands(*APP_ID, *GUILD_ID)

        for _, cmd := range cmds {
            log.Printf("Deleting: %v", cmd.Name)
            s.ApplicationCommandDelete(*APP_ID, *GUILD_ID, cmd.ID)
        }
    }

    // Update/add commands
    if *UPDATE {
        for _, cmd := range COMMANDS {
            log.Printf("Adding: %v", cmd.Name)

            _, err := s.ApplicationCommandCreate(*APP_ID, *GUILD_ID, &cmd)
            if err != nil {
                log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
            }
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
	c.AddFunc("@every 30m", checkUnhandledBets)
	c.Start()

	// Wait for stop signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Print("Quitting...")
}

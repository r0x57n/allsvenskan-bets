package main

import (
	"math"
    "sort"
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
    "github.com/gookit/config/v2"
    "github.com/gookit/config/v2/yaml"
)


/*
 Paths
*/

const (
	DB = "./bets.db"
	DB_TYPE = "sqlite3"
	TIME_LAYOUT = time.RFC3339
    VERSION = "0.5.0" // major.minor.patch
    CHECK_BETS_INTERVAL = "30m"
    CHECK_CHALL_INTERVAL = "5s"
)


/*
 Bot parameters
*/

var (
	GUILD_ID  = flag.String("guild", "", "Test guild ID")
	BOT_TOKEN = flag.String("token", "", "Bot access token")
	APP_ID    = flag.String("app", "", "Application ID")
    OWNER     = flag.String("owner", "", "Owner of the bot")
    DELETE    = flag.Bool("delete", false, "Remove all commands")
    UPDATE    = flag.Bool("update", false, "Update/add all commands")
)


/*
  Structs
*/

type cmd struct {
    name string
    category string
    admin int
}

type match struct {
	id int
	homeTeam string
	awayTeam string
	date string
	scoreHome int
	scoreAway int
	finished int
    round int
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

type challenge struct {
    id int
    challengerUID int
    challengeeUID int
    typ int
    matchID int
    points int
    condition string
    status status
}

type user struct {
    uid int
    season int
    history string
    viewable int
    interactable int
}


/*
  Enums
*/

type status int
const (
    Unhandled = iota
    Sent
    Accepted
    Declined
    RequestForfeit
    Forfeited
)

type location int
const (
	Home = iota
	Away
)

type InteractionType dg.InteractionResponseType
const (
	NewMsg = dg.InteractionResponseChannelMessageWithSource
	UpdateMsg = dg.InteractionResponseUpdateMessage
)


/*
  Command functions
*/

// Command: hjälp
func helpCommand(s *dg.Session, i *dg.InteractionCreate, COMMANDS *[]dg.ApplicationCommand) {
	help := "Denna bot är till för att kunna slå vad om hur olika Allsvenska matcher kommer sluta.\n" +
		    "\n" +
            "Du kan */slåvad* över en match. Då slår du vad om hur du tror en match kommer sluta poängmässigt. Har du rätt vinner du poäng som kan användas till antingen **skryträtt**, eller för att */utmana* andra användare. " +
            "När du utmanar en annan användare väljer du en utmaning och hur många poäng du satsar på ditt utfall. Vinnaren tar alla poängen.\n" +
            "\n" +
            "Resultaten för matcherna uppdateras lite då och då under dagen, därför kan det ta ett tag tills dess att poängen delas ut efter att en match är spelad."

    adminOnly := map[string]int{"sammanfatta": 1, "update": 1, "delete": 1, "checkbets": 1}
    isOwner := getInteractUID(i) == *OWNER

    cmds := ""

	for _, elem := range *COMMANDS {
		if _, adminCmd := adminOnly[elem.Name]; !adminCmd || isOwner {
            if adminCmd {
                cmds += fmt.Sprintf("*/%v - %v*\n", elem.Name, elem.Description)
            } else {
                cmds += fmt.Sprintf("/%v - %v\n", elem.Name, elem.Description)
            }
		}
	}

    fields := []*dg.MessageEmbedField {
        {
            Name: "Kommandon",
            Value: cmds,
        },
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Hjälp", help)
}

// Command: vadslagningar
func listBetsCommand(s *dg.Session, i *dg.InteractionCreate) {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
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
            bets, err = db.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE uid=? AND handled=1 AND won=0", uID)
        case 1:
            bets, err = db.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE uid=? AND handled=1 AND won=1", uID)
        default:
            bets, err = db.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE uid=? AND handled=1", uID)
    }
	if err != nil { log.Fatal(err) }
    defer bets.Close()

	var viewable = 0

	if err := db.QueryRow("SELECT viewable FROM points WHERE uid=?", uID).Scan(&viewable); err != nil {
		if err != sql.ErrNoRows { log.Panic(err) }
	}

    desc, correct, incorrect := "", "", ""

	if viewable == 1 {
		var b bet

		for bets.Next() {
			bets.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore, &b.won)
			matchRow := db.QueryRow("SELECT homeTeam, awayTeam, date FROM matches WHERE id=?", b.matchid)

			var m match
			matchRow.Scan(&m.homeTeam, &m.awayTeam, &m.date)

            if b.won == 0 {
                incorrect += fmt.Sprintf("%v (**%v**) - %v (**%v**)\n", m.homeTeam, b.homeScore, m.awayTeam, b.awayScore)
            } else if b.won == 1 {
                correct += fmt.Sprintf("%v (**%v**) - %v (**%v**)\n", m.homeTeam, b.homeScore, m.awayTeam, b.awayScore)
            }
		}

		if correct == "" && incorrect == "" {
			desc = fmt.Sprintf("Användaren har inga vadslagningar ännu!", )

            if correct == "" {
                correct = "-"
            }

            if incorrect == "" {
                incorrect = "-"
            }
		}
	} else {
		desc = "Användaren har valt att dölja sina vadslagningar."
        incorrect = "-"
        correct = "-"
	}

    fields := []*dg.MessageEmbedField {}

    if betType == 0 {
        fields = []*dg.MessageEmbedField {
            {
                Name: "Inkorrekta",
                Value: incorrect,
            },
        }
    } else if betType == 1 {
        fields = []*dg.MessageEmbedField {
            {
                Name: "Korrekta",
                Value: correct,
            },
        }
    } else {
        fields = []*dg.MessageEmbedField {
            {
                Name: "Korrekta",
                Value: correct,
            },
            {
                Name: "Inkorrekta",
                Value: incorrect,
            },
        }
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Vadslagningar", desc)
}

// Command: poäng
func pointsCommand(s *dg.Session, i *dg.InteractionCreate) {
	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	rows, err := db.Query("SELECT uid, season FROM points ORDER BY season DESC LIMIT 10")
	defer rows.Close()
	if err != nil { log.Panic(err) }

	top10 := ""
	pos := 0

	for rows.Next() {
		var (
			uid int
			season int
		)

		rows.Scan(&uid, &season)
		user, _ := s.User(strconv.Itoa(uid))
        pos++

		top10 += fmt.Sprintf("#%v **%v** med %v poäng\n", pos, user.Username, season)
	}

	uPoints := 0
	if err := db.QueryRow("SELECT season FROM points WHERE uid=?", getInteractUID(i)).Scan(&uPoints); err != nil {
		if err == sql.ErrNoRows {
			// skip
		} else {
			log.Panic(err)
		}
	}

	userPoints := fmt.Sprintf("Du har samlat ihop **%v** poäng i år!", uPoints)

    fields := []*dg.MessageEmbedField {
        {
            Name: "Top 10",
            Value: top10,
        },
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Poäng", userPoints)
}

// Command: sammanfatta
func summaryCommand(s *dg.Session, i *dg.InteractionCreate) {
	if notOwner(s, i) { return }

	db, err := sql.Open(DB_TYPE, DB)
	defer db.Close()
	if err != nil { log.Fatal(err) }

	today := time.Now().Format("2006-01-02")

    round := -1
    err = db.QueryRow("SELECT round FROM matches WHERE date(date)>=? AND finished='0' ORDER BY date", today).Scan(&round)
    if err != nil { log.Panic(err) }

    var matches []match
    matchesRows, err := db.Query("SELECT id, homeTeam, awayTeam, date, scoreHome, scoreAway, finished FROM matches WHERE round=?", round)
    if err != nil { log.Panic(err) }

    won, lost := 0, 0
    err = db.QueryRow("SELECT COUNT(id) FROM bets WHERE round=? AND won=1 AND handled=1", round).Scan(&lost)
    if err != nil { log.Panic(err) }
    err = db.QueryRow("SELECT COUNT(id) FROM bets WHERE round=? AND won=0 AND handled=1", round).Scan(&won)
    if err != nil { log.Panic(err) }

    var bets []bet
    wins := make(map[int]int)

    for matchesRows.Next() {
        var m match
        matchesRows.Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date, &m.scoreHome, &m.scoreAway, &m.finished)
        matches = append(matches, m)

        betsRows, err := db.Query("SELECT id, uid, matchid, homeScore, awayScore, won FROM bets WHERE matchid=?", m.id)
        if err != nil { log.Panic(err) }

        for betsRows.Next() {
            var b bet
            betsRows.Scan(&b.id, &b.uid, &b.matchid, &b.homeScore, &b.awayScore, &b.won)
            bets = append(bets, b)

            if b.won == 1 {
                wins[b.uid]++
            }
        }
    }

    // Top three wins
    topThree := "Dom med flest vinster är:\n"
    keys := make([]int, 0, len(wins))
    for k := range wins {
        keys = append(keys, k)
    }

    sort.Ints(keys)

    for i, k := range keys {
        if i <= 3 {
            username, _ := s.User(strconv.Itoa(k))
            topThree += fmt.Sprintf("#%v - %v med %v vinster\n", i + 1, username.Username, wins[k])
        }
    }

    msg := fmt.Sprintf("Denna omgång spelades **%v** matcher och **%v** vadslagningar las.\n\n", len(matches), len(bets))
    msg += fmt.Sprintf("**%v**:st vann sina vad medans **%v**:st förlorade.\n\n", won, lost)
    msg += topThree

    addInteractionResponse(s, i, NewMsg, msg)
}

// Command: info
func infoCommand(s *dg.Session, i *dg.InteractionCreate) {
    str := "Jag är en bot gjord i Go med hjälp av [discordgo](https://github.com/bwmarrin/discordgo) paketet. Min källkod finns på [Github](https://github.com/r0x57n/allsvenskanBets)."
    str += "\n\n"
    str += "*v" + VERSION + "*"

    fields := []*dg.MessageEmbedField {}
    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Hej!", str)
}

// Command: checkbets
func checkBetsCommand(s *dg.Session, i *dg.InteractionCreate) {
	if notOwner(s, i) { return }

    addInteractionResponse(s, i, NewMsg, "Checking bets...")
    checkUnhandledBets()
}

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

	err = db.QueryRow("SELECT uid, season, history, viewable, interactable FROM points WHERE uid=?", uid).
                 Scan(&u.uid, &u.season, &u.history, &u.viewable, &u.interactable)
	if err != nil {
        if err == sql.ErrNoRows {
            u.uid, err = strconv.Atoi(uid)
            if err != nil { log.Panic(err) }

            u.season = 0
            u.history = ""
            u.viewable = 1
            u.interactable = 1

            _, err = db.Exec("INSERT INTO points (uid, season) VALUES (?, ?)", u.uid, u.season)
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

    // Add command/component handlers
	s.AddHandler(func(s *dg.Session, i *dg.InteractionCreate) {
		switch i.Type {
			case dg.InteractionApplicationCommand:
				if h, ok := COMMAND_HANDLERS[i.ApplicationCommandData().Name]; ok { h(s, i) }
			case dg.InteractionMessageComponent:
				if h, ok := COMPONENT_HANDLERS[i.MessageComponentData().CustomID]; ok { h(s, i) }
		}
	})

    // Tell us when we manage to login
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

    config.WithOptions(config.ParseEnv)
    config.AddDriver(yaml.Driver)
    err := config.LoadFiles("config.yml")
	if err != nil { panic(err) }

    flag.Parse()
    if *BOT_TOKEN == "" {
        *BOT_TOKEN = config.String("botToken")
    }

    if *APP_ID == "" {
        *APP_ID = config.String("appID")
    }

    if *OWNER == "" {
        *OWNER = config.String("owner")
    }

	// Initialize and start the bot
	s := initializeBot()
	defer s.Close()

    c := cron.New()
    if CHECK_BETS_INTERVAL != "" {
        // Check the bets on a timed interval
        c.AddFunc("@every " + CHECK_BETS_INTERVAL, checkUnhandledBets)
        log.Printf("Checking bets every %v", CHECK_BETS_INTERVAL)
    }
    if CHECK_CHALL_INTERVAL != "" {
        /*c.AddFunc("@every " + CHECK_CHALL_INTERVAL, checkUnhandledChallenges())*/
        log.Printf("Checking challenges every %v", CHECK_CHALL_INTERVAL)
    }
    c.Start()

	// Wait for stop signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Print("Quitting...")
}

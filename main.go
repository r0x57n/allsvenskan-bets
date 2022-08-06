package main

import (
	"math"
	"flag"
	"fmt"
	"strings"
	"strconv"
	"log"
	"time"
	"os"
	"os/signal"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
	"github.com/bwmarrin/discordgo"
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
            "Du kan */vadslå* över en match. Då slår du vad om hur du tror en match kommer sluta poängmässigt. Har du rätt vinner du poäng som kan användas till antingen **skryträtt**, eller för att */utmana* andra användare.\n" +
            "När du utmanar en annan användare väljer du en utmaning och hur många poäng du satsar på ditt utfall. Vinnaren tar alla poängen.\n" +
            "\n" +
            "Alla vadslagningar kollas runt midnatt och det är först då poängen delas ut.\n" +
            "\n" +
	        "**Kommandon**\n"

	for _, elem := range *COMMANDS {
		if elem.Name != "sammanfatta" && elem.Name != "refresh" {
			help = help + fmt.Sprintf("/%v - %v\n", elem.Name, elem.Description)
		}
	}

    msgStdInteractionResponse(s, i, help)
}

// Command: slåvad
func betCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := getRoundMatchesAsOptions()

    components := []discordgo.MessageComponent {
        discordgo.ActionsRow {
            Components: []discordgo.MessageComponent {
                discordgo.SelectMenu {
                    Placeholder: "Välj en match",
                    CustomID: "betOnSelected", // component handler
                    Options: *options,
                },
            },
        },
    }

    compInteractionResponse(s, i, discordgo.InteractionResponseChannelMessageWithSource, "Kommande omgångens matcher:", components)
}

// Command: utmana
func challengeCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    if len(i.Interaction.ApplicationCommandData().Options) != 2 {
        log.Panic("Not enough argumenst given...")
    }

    option := i.Interaction.ApplicationCommandData().Options[0]
    challengee, err := s.User(fmt.Sprintf("%v", option.Value))
    if err != nil { log.Panic(err) }

    msg := fmt.Sprintf("Utmana användare %v om hur följande match kommer sluta.", challengee.Username)

    options := getRoundMatchesAsOptions(challengee.ID)

    components := []discordgo.MessageComponent {
        discordgo.ActionsRow {
            Components: []discordgo.MessageComponent {
                discordgo.SelectMenu {
                    Placeholder: "Vilken match vill du utmana om?",
                    CustomID: "challOnSelected",
                    Options: *options,
                },
            },
        },
    }

    compInteractionResponse(s, i, discordgo.InteractionResponseChannelMessageWithSource, msg, components)
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

    msgStdInteractionResponse(s, i, userBets)
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

    msgStdInteractionResponse(s, i, userBets)
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

    msgStdInteractionResponse(s, i, str)
}

// Command: inställningar
func settingsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    msgStdInteractionResponse(s, i, "...")
}

// Command: sammanfatta
func summaryCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if isOwner(i) {
        msgStdInteractionResponse(s, i, "Sammanfatta")
	} else {
        msgStdInteractionResponse(s, i, "Du har inte rättigheter att köra detta kommando.")
	}
}

// Command: refresh
func refreshCommand(s *discordgo.Session, i *discordgo.InteractionCreate, COMMANDS *[]discordgo.ApplicationCommand) {
	if isOwner(i) {
        log.Println("Refreshing commands...")

        msgStdInteractionResponse(s, i, "Refreshing commands...")

        cID := ""

        if len(i.Interaction.ApplicationCommandData().Options) == 1 {
            cID = fmt.Sprintf("%v", i.Interaction.ApplicationCommandData().Options[0].Value)
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
        msgStdInteractionResponse(s, i, "Du har inte rättigheter att köra detta kommando...")
	}
}

func aboutCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    str := "Jag är en bot gjort i Go med hjälp av discordgo paketet. Min källkod finns på https://github.com/r0x57n/allsvenskanBets." +
           "\n" +
           "Den version jag kör är: " + VERSION

    msgStdInteractionResponse(s, i, str)
}

/*
  Component handlers
*/

func betOnSelected(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	msg := fmt.Sprintf("%v (h) vs %v (b) @ %v \n\n**Poäng** *(hemmalag överst)*", m.homeTeam, m.awayTeam, m.date)
    components := []discordgo.MessageComponent {
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
    }

    compInteractionResponse(s, i, discordgo.InteractionResponseUpdateMessage, msg, components)
}

func challOnSelected(s *discordgo.Session, i *discordgo.InteractionCreate) {
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

    // Parsing values
	vals := i.MessageComponentData().Values
    if len(vals) == 0 { log.Panic(err) }

    splitted := strings.Split(vals[0], "_")

    _, err = s.User(splitted[0])
	if err != nil { log.Panic(err) }

    matchID := splitted[1]

	_, err = s.User(i.Interaction.Member.User.ID)
	if err != nil { log.Panic(err) }

    // Do things
    var m match

	err = svffDB.QueryRow("SELECT id, homeTeam, awayTeam, date FROM matches WHERE id=?", matchID).Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date)
	if err != nil { log.Panic(err) }

    components := []discordgo.MessageComponent {
        discordgo.ActionsRow {
            Components: []discordgo.MessageComponent {
                discordgo.SelectMenu {
                    CustomID: "challWinner",
                    Options: []discordgo.SelectMenuOption{
                        {
                            Label: m.homeTeam,
                            Value: "home",
                            Default: true,
                        },
                        {
                            Label: m.awayTeam,
                            Value: "away",
                        },
                    },
                },
            },
        },
        discordgo.ActionsRow {
            Components: []discordgo.MessageComponent {
                discordgo.SelectMenu {
                    CustomID: "pointsChall",
                    Placeholder: "Poäng att satsa.",
                    Options: []discordgo.SelectMenuOption{
                        {
                            Label: "1 Poäng",
                            Value: "1",
                        },
                        {
                            Label: "5 Poäng",
                            Value: "5",
                        },
                    },
                },
            },
        },
    }

    msg := "Vem tror du vinner och hur mycket poäng vill du satsa?"

    compInteractionResponse(s, i, discordgo.InteractionResponseUpdateMessage, msg, components)
}

func challWinner(s *discordgo.Session, i *discordgo.InteractionCreate) {
    addInteractionResponse(s, i, discordgo.InteractionResponseDeferredMessageUpdate, "")
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

func addInteractionResponse(s *discordgo.Session,
                         i *discordgo.InteractionCreate,
                         interactionType discordgo.InteractionResponseType,
                         msg string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: interactionType,
		Data: &discordgo.InteractionResponseData {
			Content: msg,
			Flags: 1 << 6, // Ephemeral
		},
	}); err != nil { log.Panic(err) }

}

func msgStdInteractionResponse(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
			Content: msg,
		},
	}); err != nil { log.Panic(err) }
}

func compInteractionResponse(s *discordgo.Session,
                                i *discordgo.InteractionCreate,
                                interactionType discordgo.InteractionResponseType,
                                msg string,
                                components []discordgo.MessageComponent) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse {
		Type: interactionType,
		Data: &discordgo.InteractionResponseData {
			Content: msg,
			Components: components,
			Flags: 1 << 6, // Ephemeral
        },
	}); err != nil { log.Panic(err) }
}

// Parameter is optional string to add to value of the options.
// This is so we can add meta data about things such as challenges, we need
// to remember who the challengee was...
func getRoundMatchesAsOptions(value ...string) *[]discordgo.SelectMenuOption {
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

    options := []discordgo.SelectMenuOption{}

	if len(matches) == 0 {
		options = append(options, discordgo.SelectMenuOption{
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

            options = append(options, discordgo.SelectMenuOption{
                Label: fmt.Sprintf("%v vs %v", m.homeTeam, m.awayTeam),
                Value: val,
                Description: m.date,
            })
		}
	}

    return &options
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
	Commands and handlers declarations
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
				Description: "Utmana en annan användare om en kommande match. Välj användare och vilket typ av utmaning, skicka för att fylla i mer detaljer.",
				Options: []*discordgo.ApplicationCommandOption {
					{
						Type: discordgo.ApplicationCommandOptionUser,
						Name: "användarnamn",
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
						Name: "användare",
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
			{
				Name: "ommig",
				Description: "Teknisk info om mig.",
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
			"ommig": func(s *discordgo.Session, i *discordgo.InteractionCreate)         {   aboutCommand(s, i)              },
		}

        // Component handlers
		COMPONENT_HANDLERS = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			"betOnSelected": func(s *discordgo.Session, i *discordgo.InteractionCreate)   {   betOnSelected(s, i)             },
			"challOnSelected": func(s *discordgo.Session, i *discordgo.InteractionCreate) {   challOnSelected(s, i)           },
			"challWinner": func(s *discordgo.Session, i *discordgo.InteractionCreate)     {   challWinner(s, i)               },
			"scoreHome": func(s *discordgo.Session, i *discordgo.InteractionCreate)       {   scoreComponent(s, i, Home)      },
			"scoreAway": func(s *discordgo.Session, i *discordgo.InteractionCreate)       {   scoreComponent(s, i, Away)      },
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

    if *RR {
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

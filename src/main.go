package main

import (
	"flag"
	"log"
	"time"
	"os"
	"os/signal"
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
	DB_TIME_LAYOUT = time.RFC3339
    MSG_TIME_LAYOUT = "2006-02-01 kl. 15:04"
    VERSION = "0.6.0" // major.minor.patch
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
    description string
    category CommandCategory
    admin bool
    options []*dg.ApplicationCommandOption
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
    round int
}

type challenge struct {
    id int
    challengerUID int
    challengeeUID int
    typ int
    matchID int
    points int
    condition string
    status ChallengeStatus
}

type user struct {
    uid int
    seasonPoints int
    bank string
    viewable int
    interactable int
}


/*
  Enums
*/

type CommandCategory string
const (
    General = "Allmänt"
    Betting = "Vadslagning"
    Admin = "Admin"
)

type ChallengeStatus int
const (
    Unhandled = iota
    Sent
    Accepted
    Declined
    RequestForfeit
    Forfeited
)

type BetType int
const (
    Lost = iota
    Won
    All
)

type BetLocation int
const (
	Home = iota
	Away
)

type InteractionType dg.InteractionResponseType
const (
	NewMsg = dg.InteractionResponseChannelMessageWithSource
	UpdateMsg = dg.InteractionResponseUpdateMessage
    Ignore = dg.InteractionResponseDeferredMessageUpdate
)


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
        for _, c := range COMMANDS {
            cmd := dg.ApplicationCommand {
                Name: c.name,
                Description: c.description,
                Options: c.options,
            }

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

package main

import (
	"log"
	"time"
    "flag"
	"os"
	"os/signal"
	"github.com/robfig/cron/v3"
    "github.com/gookit/config/v2"
    "github.com/gookit/config/v2/yaml"
)

/*
  Globals
*/

const (
    VERSION                 = "1.5.2" // major.minor.patch
	DB_TYPE                 = "postgres"
	DB_TIME_LAYOUT          = time.RFC3339
    MSG_TIME_LAYOUT         = "2006-01-02 kl. 15:04"
    CHECK_BETS_INTERVAL     = "30m"
    UPDATE_MATCHES_INTERVAL = "2h"
)

var (
    ADD_COMMANDS = flag.Bool("add", false, "Add all commands on startup.")
    CONFIG_PATH = flag.String("c", "../config.yml", "Path to config file.")
    LOG_TO_FILE = flag.Bool("l", false, "Log to file.")
)

/*
 Main
*/

func main() {
    log.Print("Parsing flags...")
    flag.Parse()

    // Setup logging to file
    if *LOG_TO_FILE {
        f, err := os.OpenFile("log.txt", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
        if err != nil { log.Panic(err) }
        defer f.Close()

        log.SetOutput(f)
    }

    // Start running
	log.Print("Starting...")

    // Load config
    config.WithOptions(config.ParseEnv)
    config.AddDriver(yaml.Driver)
    if err := config.LoadFiles(*CONFIG_PATH); err != nil { panic(err) }

    dbinfo := InfoDB{
        Host: config.String("dbHost"),
        User: config.String("dbUser"),
        Password: config.String("dbPass"),
        Name: config.String("dbName"),
        Port: config.Int("dbPort"),
    }

    bot := &Bot{
        token: config.String("botToken"),
        appID: config.String("appID"),
        owner: config.String("owner"),
        db: connectDB(dbinfo),
        updaterPath: config.String("updaterPath"),
    }

	// Initialize and start the bot
	bot.Init()
    bot.Start()
	defer bot.Close()

    // Interval checking stuff
    bot.cron = cron.New()
    if CHECK_BETS_INTERVAL != "" {
        bot.cron.AddFunc("@every " + CHECK_BETS_INTERVAL, func() { bot.checkStuff(false) })
        log.Printf("Checking bets/challenges/sumaries every %v", CHECK_BETS_INTERVAL)
    }
    if UPDATE_MATCHES_INTERVAL != "" {
        bot.cron.AddFunc("@every " + UPDATE_MATCHES_INTERVAL, func(){ bot.updateMatches(false) })
        log.Printf("Updating matches every %v", UPDATE_MATCHES_INTERVAL)
    }
    bot.cron.Start()

	// Stop signal to quit
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Print("Quitting...")
}

package main

import (
	"log"
	"time"
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
    VERSION                 = "0.8.0" // major.minor.patch
	DB_TYPE                 = "postgres"
	DB_TIME_LAYOUT          = time.RFC3339
    MSG_TIME_LAYOUT         = "2006-01-02 kl. 15:04"
    CONFIG_PATH             = "../config.yml"
    CHECK_BETS_INTERVAL     = "30m"
    CHECK_CHALL_INTERVAL    = "30m"
)

var DB database

/*
 Main
*/

func main() {
	log.Print("Starting...")

    bot := &botHolder{}

    // Load config
    config.WithOptions(config.ParseEnv)
    config.AddDriver(yaml.Driver)
    if err := config.LoadFiles(CONFIG_PATH); err != nil { panic(err) }

    bot.token   = config.String("botToken")
    bot.appID   = config.String("botToken")
    bot.owner   = config.String("owner")
    DB.host     = config.String("dbHost")
    DB.user     = config.String("dbName")
    DB.password = config.String("dbPass")
    DB.name     = config.String("dbName")
    DB.port     = config.Int("dbPort")

	// Initialize and start the bot
	bot.Init()
    bot.Start()
	defer bot.Close()

    // Interval checking stuff
    c := cron.New()
    if CHECK_BETS_INTERVAL != "" {
        c.AddFunc("@every " + CHECK_BETS_INTERVAL, checkUnhandledBets)
        log.Printf("Checking bets every %v", CHECK_BETS_INTERVAL)
    }
    if CHECK_CHALL_INTERVAL != "" {
        c.AddFunc("@every " + CHECK_CHALL_INTERVAL, checkUnhandledChallenges)
        log.Printf("Checking challenges every %v", CHECK_CHALL_INTERVAL)
    }
    c.Start()

	// Stop signal to quit
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Print("Quitting...")
}

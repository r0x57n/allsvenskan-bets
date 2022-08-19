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
    VERSION                 = "0.9.0" // major.minor.patch
	DB_TYPE                 = "postgres"
	DB_TIME_LAYOUT          = time.RFC3339
    MSG_TIME_LAYOUT         = "2006-01-02 kl. 15:04"
    CONFIG_PATH             = "../config.yml"
    CHECK_BETS_INTERVAL     = "30m"
    CHECK_CHALL_INTERVAL    = "30m"
    UPDATE_MATCHES_INTERVAL = "2h"
)

/*
 Main
*/

func main() {
	log.Print("Starting...")

    // Load config
    config.WithOptions(config.ParseEnv)
    config.AddDriver(yaml.Driver)
    if err := config.LoadFiles(CONFIG_PATH); err != nil { panic(err) }

    dbinfo := dbInfo{
        host: config.String("dbHost"),
        user: config.String("dbName"),
        password: config.String("dbPass"),
        name: config.String("dbName"),
        port: config.Int("dbPort"),
    }

    bot := &botHolder{
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
    c := cron.New()
    if CHECK_BETS_INTERVAL != "" {
        c.AddFunc("@every " + CHECK_BETS_INTERVAL, bot.checkUnhandledBets)
        log.Printf("Checking bets every %v", CHECK_BETS_INTERVAL)
    }
    if CHECK_CHALL_INTERVAL != "" {
        c.AddFunc("@every " + CHECK_CHALL_INTERVAL, bot.checkUnhandledChallenges)
        log.Printf("Checking challenges every %v", CHECK_CHALL_INTERVAL)
    }
    if UPDATE_MATCHES_INTERVAL != "" {
        c.AddFunc("@every " + UPDATE_MATCHES_INTERVAL, func(){ bot.updateMatches(false) })
        log.Printf("Updating matches every %v", UPDATE_MATCHES_INTERVAL)
    }
    c.Start()

	// Stop signal to quit
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Print("Quitting...")
}

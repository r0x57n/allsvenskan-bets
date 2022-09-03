package main

import (
	"database/sql"
	"github.com/robfig/cron/v3"
	dg "github.com/bwmarrin/discordgo"
)

type Bot struct {
    token string
    appID string
    guildID string
    allsvenskanGuildID string
    owner string
    updaterPath string
    session *dg.Session
    db *sql.DB
    cron *cron.Cron
    commands []Command
    commandHandlers map[string]func(s *dg.Session, i *dg.InteractionCreate)
    componentHandlers map[string]func(s *dg.Session, i *dg.InteractionCreate)
}

type InfoDB struct {
    Host string
    Port int
    User string
    Password string
    Name  string
}

type BotInfo struct {
    CurrentRound int
    LastUpdate string
}

type Match struct {
	ID int
	HomeTeam string
	AwayTeam string
	Date string
	HomeScore int
	AwayScore int
    Round int
	Finished bool
}

type MatchSummary struct {
    Info string
    Fields []*dg.MessageEmbedField
}

type Bet struct {
	ID int
	UserID int
	MatchID int
	HomeScore int
	AwayScore int
    Status BetStatus
    Round int
}

type Challenge struct {
    ID int
    ChallengerID int
    ChallengeeID int
    Type ChallengeType
    MatchID int
    Points int
    Condition ChallengeCondition
    Status ChallengeStatus
    Winner ChallengeWinner
}

type Round struct {
    Num string `json:"num"`
    NumMatches int `json:"numMatches"`
    NumBets int `json:"numBets"`
    NumWins int `json:"numWins"`
    NumLoss int `json:"numLoss"`
    TopFive string `json:"topFive"`
    BotFive string `json:"botFive"`
}

type User struct {
    UserID int
    Points int
    Bank string
    Viewable bool
    Interactable bool
}

package main

import (
	"database/sql"
	dg "github.com/bwmarrin/discordgo"
)

type botHolder struct {
    token string
    appID string
    guildID string
    owner string
    session *dg.Session
    db *sql.DB
}

type database struct {
    host string
    port int
    user string
    password string
    name  string
}

type cmd struct {
    name string
    description string
    category CommandCategory
    admin bool
    options []*dg.ApplicationCommandOption
}

type match struct {
	id int
	hometeam string
	awayteam string
	date string
	homescore int
	awayscore int
    round int
	finished bool
}

type bet struct {
	id int
	uid int
	matchid int
	homescore int
	awayscore int
    status BetStatus
}

type challenge struct {
    id int
    challengerid int
    challengeeid int
    typ ChallengeType
    matchid int
    points int
    condition ChallengeCondition
    status ChallengeStatus
}

type user struct {
    uid int
    points int
    bank string
    viewable bool
    interactable bool
}

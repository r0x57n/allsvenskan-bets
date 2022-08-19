package main

import (
	"database/sql"
	dg "github.com/bwmarrin/discordgo"
)

/*
  Structs
*/

type botHolder struct {
    token string
    appID string
    guildID string
    owner string
    session *dg.Session
    db *sql.DB
    commands map[CommandName]Command
    componentHandlers map[string]func(s *dg.Session, i *dg.InteractionCreate)
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


/*
  Enums
*/

type CommandName string
const (
    BetCommand = "slåvad"
)

type ComponentName string
const (
    BetOnSelected = "BetOnSelected"
)

type CommandCategory string
const (
    CommandCategoryGeneral = "Allmänt"
    CommandCategoryBetting = "Slå vad"
    CommandCategoryListing = "Vadslagningar"
    CommandCategoryAdmin   = "Admin"
)

type BetStatus int
const (
    BetStatusUnhandled = iota
    BetStatusWon
    BetStatusLost
)

type ChallengeStatus int
const (
    ChallengeStatusUnhandled = iota
    ChallengeStatusSent
    ChallengeStatusAccepted
    ChallengeStatusDeclined
    ChallengeStatusRequestForfeit
    ChallengeStatusForfeited
    ChallengeStatusHandled
)

type ChallengeType int
const (
    ChallengeTypeWinner = iota
)

type ChallengeCondition int
const (
    ChallengeConditionWinnerHome = iota
    ChallengeConditionWinnerAway
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

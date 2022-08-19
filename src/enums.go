package main

import (
    dg "github.com/bwmarrin/discordgo"
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

type BetStatus int
const (
    BetStatusUnhandled = iota
    BetStatusWon
    BetStatusLost
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
  Command enums
*/

type CommandName string
const (
    HelpCommand         = "hjälp"
    BetCommand          = "slåvad"
    RegretCommand       = "ångra"
    ChallengeCommand    = "utmana"
    ChickenCommand      = "fegaur"
    UpcomingCommand     = "kommande"
    BetsCommand         = "vadslagningar"
    PointsCommand       = "poäng"
    SettingsCommand     = "inställningar"
    InfoCommand         = "info"
    SummaryCommand      = "sammanfatta"
    RefreshCommand      = "refresh"
    RemoveCommand       = "remove"
    CheckCommand        = "checkbets"
    UpdateCommand       = "update"
)

type CommandCategory string
const (
    CommandCategoryGeneral = "Allmänt"
    CommandCategoryBetting = "Slå vad"
    CommandCategoryListing = "Vadslagningar"
    CommandCategoryAdmin   = "Admin"
)

type ComponentName string
const (
    BetSelectScore = "betOnSelected"
    BetUpdateScoreHome = "betScoreHome"
    BetUpdateScoreAway = "betScoreAway"
    ChallSelectWinner = "challSelectWinner"
    ChallSelectPoints = "challSelectPoints"
    ChallAcceptDiscard = "challAcceptDiscard"
    ChallAcceptDiscardDo = "challAcceptDiscardDo"
    SettingsVisibility = "settingsVisibility"
    SettingsChall = "settingsChall"
    RefreshCommandDo = "updateCommandDo"
    RemoveCommandDo = "deleteCommandDo"
    RegretSelected = "regretSelected"
    ChallAnswer = "challAnswer"
    ChickenSelected = "chickenSelected"
    ChickenAnswer = "chickenAnswer"
)

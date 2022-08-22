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

type CommandError int
const (
    ErrorNoRights = iota
    ErrorMatchStarted
    ErrorOtherNotInteractable
    ErrorUserNotViewable
    ErrorSelfNotInteractable
    ErrorInteractingWithSelf
    ErrorMaxChallenges
    ErrorNoMatches
    ErrorIdenticalChallenge
    ErrorNotEnoughPoints
    ErrorChallengeHandled
)

type BetStatus int
const (
    BetStatusUnhandled = iota
    BetStatusWon
    BetStatusLost
    BetStatusAlmostWon
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
    BetCommand          = "gissa"
    RegretCommand       = "ångra"
    ChallengeCommand    = "utmana"
    ChickenCommand      = "fegaur"
    BetsCommand         = "vad"
    PointsCommand       = "poäng"
    SettingsCommand     = "inställningar"
    InfoCommand         = "info"
    SummaryCommand      = "omgång"
    RefreshCommand      = "add"
    RemoveCommand       = "remove"
    CheckCommand        = "checkbets"
    UpdateCommand       = "update"
    SummaryAllCommand   = "sammanfatta"
    MatchCommand        = "match"
)

type CommandCategory string
const (
    CommandCategoryGeneral = "Allmänt"
    CommandCategoryBetting = "Vadslagningar"
    CommandCategoryListing = "Statistik"
    CommandCategoryAdmin   = "Admin"
)

type ComponentName string
const (
    BetSelectScore          = "betOnSelected"
    BetUpdateScoreHome      = "betScoreHome"
    BetUpdateScoreAway      = "betScoreAway"
    ChallSelectWinner       = "challSelectWinner"
    ChallSelectPoints       = "challSelectPoints"
    ChallAcceptDiscard      = "challAcceptDiscard"
    ChallAcceptDiscardDo    = "challAcceptDiscardDo"
    SettingsVisibility      = "settingsVisibility"
    SettingsChall           = "settingsChall"
    RefreshCommandDo        = "updateCommandDo"
    RemoveCommandDo         = "deleteCommandDo"
    RegretSelected          = "regretSelected"
    ChallAnswer             = "challAnswer"
    ChickenSelected         = "chickenSelected"
    ChickenAnswer           = "chickenAnswer"
    MatchSendInfo           = "matchSendInfo"
    SummaryMatchDo          = "summaryMatchDo"
)

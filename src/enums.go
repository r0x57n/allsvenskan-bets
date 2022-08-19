package main

import (
    dg "github.com/bwmarrin/discordgo"
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

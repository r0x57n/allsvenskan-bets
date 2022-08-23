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

type ChallengeWinner int
const (
    ChallengeWinnerUndecided = iota
    ChallengeWinnerChallenger
    ChallengeWinnerChallengee
    ChallengeWinnerNone
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

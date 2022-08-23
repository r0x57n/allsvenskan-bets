package main

import (
    "strconv"
    dg "github.com/bwmarrin/discordgo"
)
type Command struct {
    Name string
    Description string
    Category CommandCategory
    Admin bool
    Options []*dg.ApplicationCommandOption
}

type CommandName string
const (
    HelpCommand         = "hjälp"
    GuessCommand        = "gissa"
    RegretCommand       = "ångra"
    ChallengeCommand    = "utmana"
    ChickenCommand      = "fegaur"
    BetsCommand         = "vad"
    PointsCommand       = "poäng"
    SettingsCommand     = "inställningar"
    InfoCommand         = "info"
    RoundCommand        = "omgång"
    AddCommand          = "add"
    RemoveCommand       = "remove"
    CheckCommand        = "checkbets"
    UpdateCommand       = "update"
    SummariseCommand    = "sammanfatta"
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
    GuessSelectScore        = "betOnSelected"
    GuessUpdateScoreHome    = "betScoreHome"
    GuessUpdateScoreAway    = "betScoreAway"
    ChallSelectWinner       = "challSelectWinner"
    ChallSelectPoints       = "challSelectPoints"
    ChallAcceptDiscard      = "challAcceptDiscard"
    ChallAcceptDiscardDo    = "challAcceptDiscardDo"
    SettingsVisibility      = "settingsVisibility"
    SettingsChall           = "settingsChall"
    AddCommandDo            = "updateCommandDo"
    RemoveCommandDo         = "deleteCommandDo"
    RegretSelected          = "regretSelected"
    ChallAnswer             = "challAnswer"
    ChickenSelected         = "chickenSelected"
    ChickenAnswer           = "chickenAnswer"
    MatchSummarySend        = "matchSendInfo"
    SummariseMatchSend      = "summaryMatchDo"
)

func (b *Bot) addCommands() {
    b.commands = []Command {
        {
            Name: HelpCommand,
            Description: "få hjälp med hur denna bot fungerar",
            Category: CommandCategoryGeneral,
        },
        {
            Name: GuessCommand,
            Description: "gör en gissning över slutresultatet för en kommande match",
            Category: CommandCategoryBetting,
        },
        {
            Name: RegretCommand,
            Description: "ångra en gissning du har gjort",
            Category: CommandCategoryBetting,
        },
        {
            Name: ChallengeCommand,
            Description: "utmana en annan användare om en kommande match",
            Category: CommandCategoryBetting,
            Options: []*dg.ApplicationCommandOption {
                {
                    Type: dg.ApplicationCommandOptionUser,
                    Name: "användarnamn",
                    Description: "användare att utmana",
                    Required: true,
                },
                {
                    Type: dg.ApplicationCommandOptionString,
                    Name: "sort",
                    Description: "vilken sorts utmaning?",
                    Required: true,
                    Choices: []*dg.ApplicationCommandOptionChoice {
                        {
                            Name: "vinnare",
                            Value: "matchvinnare",
                        },
                    },
                },
            },
        },
        {
            Name: ChickenCommand,
            Description: "skapa en förfrågan om att stoppa en utmaning",
            Category: CommandCategoryBetting,
        },
        {
            Name: BetsCommand,
            Description: "lista en användares gissningar/utmaningar",
            Category: CommandCategoryListing,
            Options: []*dg.ApplicationCommandOption {
                {
                    Type: dg.ApplicationCommandOptionUser,
                    Name: "användarnamn",
                    Description: "Användare att visa vadslagningar för.",
                    Required: true,
                },
                {
                    Type: dg.ApplicationCommandOptionString,
                    Name: "sort",
                    Description: "lista bara korrekta/inkorrekta/kommande",
                    Choices: []*dg.ApplicationCommandOptionChoice {
                        {
                            Name: "vunna",
                            Value: strconv.Itoa(BetStatusWon),
                        },
                        {
                            Name: "förlorade",
                            Value: strconv.Itoa(BetStatusLost),
                        },
                        {
                            Name: "kommande",
                            Value: strconv.Itoa(BetStatusUnhandled),
                        },
                    },
                },
            },
        },
        {
            Name: PointsCommand,
            Description: "visa dina poäng och topp 10 på servern",
            Category: CommandCategoryListing,
        },
        {
            Name: RoundCommand,
            Description: "visa en sammanfattning av en tidigare omgång",
            Category: CommandCategoryListing,
            Options: []*dg.ApplicationCommandOption {
                {
                    Type: dg.ApplicationCommandOptionInteger,
                    Name: "omgång",
                    Description: "vilken omgång att sammanfatta (0 == nuvarande)",
                    Required: true,
                },
            },
        },
        {
            Name: MatchCommand,
            Description: "visa bets/utmaningar för en viss match",
            Category: CommandCategoryListing,
            Options: []*dg.ApplicationCommandOption {
                {
                    Type: dg.ApplicationCommandOptionInteger,
                    Name: "omgång",
                    Description: "välj en omgång du vill visa matcher för, lämna tom eller '0' för aktuella",
                },
            },
        },
        {
            Name: SettingsCommand,
            Description: "inställningar för din användare",
            Category: CommandCategoryGeneral,
        },
        {
            Name: InfoCommand,
            Description: "teknisk info om botten",
            Category: CommandCategoryGeneral,
        },
        {
            Name: AddCommand,
            Description: "uppdatera alla kommandon eller ett enskilt",
            Category: CommandCategoryAdmin,
            Admin: true,
        },
        {
            Name: RemoveCommand,
            Description: "ta bort ett enskilt kommando",
            Category: CommandCategoryAdmin,
            Admin: true,
        },
        {
            Name: CheckCommand,
            Description: "kör checks för challenges/bets",
            Category: CommandCategoryAdmin,
            Admin: true,
        },
        {
            Name: UpdateCommand,
            Description: "uppdaterar matcher manuellt",
            Category: CommandCategoryAdmin,
            Admin: true,
        },
        {
            Name: SummariseCommand,
            Description: "sammanfattar en omgång eller match till #bets",
            Category: CommandCategoryAdmin,
            Admin: true,
            Options: []*dg.ApplicationCommandOption {
                {
                    Type: dg.ApplicationCommandOptionString,
                    Name: "sort",
                    Description: "vad ska sammanfattas?",
                    Required: true,
                    Choices: []*dg.ApplicationCommandOptionChoice {
                        {
                            Name: "omgång",
                            Value: "0",
                        },
                        {
                            Name: "match",
                            Value: "1",
                        },
                    },
                },
                {
                    Type: dg.ApplicationCommandOptionInteger,
                    Name: "omgång",
                    Description: "vilken omgång?",
                },
            },
        },
    }

    // Link functions to commands
    b.commandHandlers = map[string]func(s *dg.Session, i *dg.InteractionCreate) {
        // User commands
        HelpCommand:        func(s *dg.Session, i *dg.InteractionCreate) {    b.helpCommand(i)        },
        GuessCommand:       func(s *dg.Session, i *dg.InteractionCreate) {    b.guessCommand(i)       },
        RegretCommand:      func(s *dg.Session, i *dg.InteractionCreate) {    b.regretCommand(i)      },
        ChallengeCommand:   func(s *dg.Session, i *dg.InteractionCreate) {    b.challengeCommand(i)   },
        ChickenCommand:     func(s *dg.Session, i *dg.InteractionCreate) {    b.chickenCommand(i)     },
        BetsCommand:        func(s *dg.Session, i *dg.InteractionCreate) {    b.listBetsCommand(i)    },
        PointsCommand:      func(s *dg.Session, i *dg.InteractionCreate) {    b.pointsCommand(i)      },
        SettingsCommand:    func(s *dg.Session, i *dg.InteractionCreate) {    b.settingsCommand(i)    },
        InfoCommand:        func(s *dg.Session, i *dg.InteractionCreate) {    b.infoCommand(i)        },
        RoundCommand:       func(s *dg.Session, i *dg.InteractionCreate) {    b.roundCommand(i)       },
        MatchCommand:       func(s *dg.Session, i *dg.InteractionCreate) {    b.matchCommand(i)       },

        // Admin commands
        AddCommand:         func(s *dg.Session, i *dg.InteractionCreate) {    b.addCommand(i)         },
        RemoveCommand:      func(s *dg.Session, i *dg.InteractionCreate) {    b.removeCommand(i)      },
        CheckCommand:       func(s *dg.Session, i *dg.InteractionCreate) {    b.checkBetsCommand(i)   },
        UpdateCommand:      func(s *dg.Session, i *dg.InteractionCreate) {    b.updateCommand(i)      },
        SummariseCommand:   func(s *dg.Session, i *dg.InteractionCreate) {    b.summariseCommand(i)   },
    }

    // Link functions to command components
    b.componentHandlers = map[string]func(s *dg.Session, i *dg.InteractionCreate) {
        GuessSelectScore:     func(s *dg.Session, i *dg.InteractionCreate) {   b.guessSelectGoals(i)           },
        GuessUpdateScoreHome: func(s *dg.Session, i *dg.InteractionCreate) {   b.guessUpdateGoals(i, Home)     },
        GuessUpdateScoreAway: func(s *dg.Session, i *dg.InteractionCreate) {   b.guessUpdateGoals(i, Away)     },
        ChallSelectWinner:    func(s *dg.Session, i *dg.InteractionCreate) {   b.challSelectWinner(i)          },
        ChallSelectPoints:    func(s *dg.Session, i *dg.InteractionCreate) {   b.challSelectPoints(i)          },
        ChallAcceptDiscard:   func(s *dg.Session, i *dg.InteractionCreate) {   b.challAcceptDiscard(i)         },
        ChallAcceptDiscardDo: func(s *dg.Session, i *dg.InteractionCreate) {   b.challAcceptDiscardDo(i)       },
        ChallAnswer:          func(s *dg.Session, i *dg.InteractionCreate) {   b.challAnswer(i)                },
        SettingsVisibility:   func(s *dg.Session, i *dg.InteractionCreate) {   b.settingsVisibility(i)         },
        SettingsChall:        func(s *dg.Session, i *dg.InteractionCreate) {   b.settingsChall(i)              },
        AddCommandDo:         func(s *dg.Session, i *dg.InteractionCreate) {   b.addCommandDo(i)               },
        RemoveCommandDo:      func(s *dg.Session, i *dg.InteractionCreate) {   b.removeCommandDo(i)            },
        RegretSelected:       func(s *dg.Session, i *dg.InteractionCreate) {   b.regretSelected(i)             },
        ChickenSelected:      func(s *dg.Session, i *dg.InteractionCreate) {   b.chickenChallengeSelected(i)   },
        ChickenAnswer:        func(s *dg.Session, i *dg.InteractionCreate) {   b.chickenAnswer(i)              },
        MatchSummarySend:     func(s *dg.Session, i *dg.InteractionCreate) {   b.matchSummarySend(i)           },
        SummariseMatchSend:   func(s *dg.Session, i *dg.InteractionCreate) {   b.summariseMatchSend(i)         },
    }
}

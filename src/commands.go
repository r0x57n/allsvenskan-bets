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
    SummaryCommand      = "sammanfatta"
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
    MatchSendInfo           = "matchSendInfo"
    SummaryMatchDo          = "summaryMatchDo"
)

func (b *Bot) addCommands() {
    b.commands = []Command {
        {
            Name: HelpCommand,
            Description: "Få hjälp med hur denna bot fungerar.",
            Category: CommandCategoryGeneral,
        },
        {
            Name: GuessCommand,
            Description: "Gör en gissning över slutresultatet för en kommande match.",
            Category: CommandCategoryBetting,
        },
        {
            Name: RegretCommand,
            Description: "Ångra en gissning du har gjort.",
            Category: CommandCategoryBetting,
        },
        {
            Name: ChallengeCommand,
            Description: "Utmana en annan användare om en kommande match.",
            Category: CommandCategoryBetting,
            Options: []*dg.ApplicationCommandOption {
                {
                    Type: dg.ApplicationCommandOptionUser,
                    Name: "användarnamn",
                    Description: "Användare att utmana.",
                    Required: true,
                },
                {
                    Type: dg.ApplicationCommandOptionString,
                    Name: "sort",
                    Description: "Vilken sorts utmaning?",
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
            Description: "Skapa en förfrågan om att stoppa en utmaning.",
            Category: CommandCategoryBetting,
        },
        {
            Name: BetsCommand,
            Description: "Lista en användares gissningar/utmaningar.",
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
                    Description: "Lista bara korrekta/inkorrekta/kommande.",
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
            Description: "Visa dina poäng och topp 10 på servern.",
            Category: CommandCategoryListing,
        },
        {
            Name: RoundCommand,
            Description: "Visa en sammanfattning av en tidigare omgång.",
            Category: CommandCategoryListing,
            Options: []*dg.ApplicationCommandOption {
                {
                    Type: dg.ApplicationCommandOptionInteger,
                    Name: "omgång",
                    Description: "Vilken omgång att sammanfatta (0 == nuvarande).",
                    Required: true,
                },
            },
        },
        {
            Name: MatchCommand,
            Description: "Visa bets/utmaningar för en viss match.",
            Category: CommandCategoryListing,
        },
        {
            Name: SettingsCommand,
            Description: "Inställningar för din användare.",
            Category: CommandCategoryGeneral,
        },
        {
            Name: InfoCommand,
            Description: "Teknisk info om botten.",
            Category: CommandCategoryGeneral,
        },
        {
            Name: AddCommand,
            Description: "Uppdatera alla kommandon eller ett enskilt.",
            Category: CommandCategoryAdmin,
            Admin: true,
        },
        {
            Name: RemoveCommand,
            Description: "Ta bort ett enskilt kommando.",
            Category: CommandCategoryAdmin,
            Admin: true,
        },
        {
            Name: CheckCommand,
            Description: "Kör checks för challenges/bets.",
            Category: CommandCategoryAdmin,
            Admin: true,
        },
        {
            Name: UpdateCommand,
            Description: "Uppdaterar matcher manuellt.",
            Category: CommandCategoryAdmin,
            Admin: true,
        },
        {
            Name: SummaryCommand,
            Description: "Sammanfattar en omgång eller match till #bets",
            Category: CommandCategoryAdmin,
            Admin: true,
            Options: []*dg.ApplicationCommandOption {
                {
                    Type: dg.ApplicationCommandOptionString,
                    Name: "sort",
                    Description: "Vad ska sammanfattas?",
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
                    Description: "Vilken omgång?",
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
        SummaryCommand:     func(s *dg.Session, i *dg.InteractionCreate) {    b.summaryCommand(i)     },
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
        MatchSendInfo:        func(s *dg.Session, i *dg.InteractionCreate) {   b.matchSendInfo(i)              },
        SummaryMatchDo:       func(s *dg.Session, i *dg.InteractionCreate) {   b.summaryMatchDo(i)             },
    }
}

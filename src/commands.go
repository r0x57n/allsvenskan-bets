package main

import (
    dg "github.com/bwmarrin/discordgo"
)

/*
   Structs
*/

type CommandName string
const (
    BetCommand = "slåvad"
    HelpCommand = "hjälp"
    ChallengeCommand = "utmana"
    SettingsCommand = "inställningar"
    ChickenCommand = "fegaur"
    RegretCommand = "ångra"
    UpcomingCommand = "kommande"
    BetsCommand = "vadslagningar"
    PointsCommand = "poäng"
    InfoCommand = "info"
    SummaryCommand = "sammanfatta"
    UpdateCommand = "update"
    DeleteCommand = "delete"
    CheckCommand = "checkbets"
)

type ComponentName string
const (
    BetOnSelected = "BetOnSelected"
    BetSetHome = "betScoreHome"
    BetSetAway = "betScoreAway"

)

type CommandCategory string
const (
    CommandCategoryGeneral = "Allmänt"
    CommandCategoryBetting = "Slå vad"
    CommandCategoryListing = "Vadslagningar"
    CommandCategoryAdmin   = "Admin"
)

/*
  Enums
 */

type Command interface {
    init(b *botHolder)
    settings()
    run(i *dg.InteractionCreate)
}

type CommandInfo struct {
    name string
    description string
    category CommandCategory
    admin bool
    options []*dg.ApplicationCommandOption
}

type aCommand struct {
    Command
    CommandInfo
    bot *botHolder
}

type Bet struct { aCommand }
type Help struct { aCommand }
type Challenge struct { aCommand }
type Settings struct { aCommand }
type Chicken struct { aCommand }
type Regret struct { aCommand }
type Upcoming struct { aCommand }
type Bets struct { aCommand }
type Points struct { aCommand }
type Info struct { aCommand }
type Summary struct { aCommand }
type Update struct { aCommand }
type Delete struct { aCommand }
type Check struct { aCommand }


/*
Commands and handlers declarations
*/

var (
    COMMANDS = []cmd {
        {
            name: "slåvad",
            description: "Slå vad om en kommande match.",
            category: CommandCategoryBetting,
            admin: false,
        },
        {
            name: "ångra",
            description: "Ångra ett vad du har lagt.",
            category: CommandCategoryBetting,
            admin: false,
        },
        {
            name: "utmana",
            description: "Utmana en annan användare.",
            category: CommandCategoryBetting,
            admin: false,
            options: []*dg.ApplicationCommandOption {
                {
                    Type: dg.ApplicationCommandOptionUser,
                    Name: "användarnamn",
                    Description: "Användare att utmana.",
                    Required: true,
                },
                {
                    Type: dg.ApplicationCommandOptionString,
                    Name: "typ",
                    Description: "Vilken sorts utmaning?",
                    Required: true,
                    Choices: []*dg.ApplicationCommandOptionChoice {
                        {
                            Name: "Matchvinnare",
                            Value: "matchvinnare",
                        },
                    },
                },
            },
        },
        {
            name: "fegaur",
            description: "Be om att få fega ur en utmaning.",
            category: CommandCategoryBetting,
            admin: false,
        },
        {
            name: "kommande",
            description: "Lista dina kommande vadslagningar.",
            category: CommandCategoryListing,
            admin: false,
        },
        {
            name: "vadslagningar",
            description: "Lista en annan användares tidigare vadslagningar.",
            category: CommandCategoryListing,
            admin: false,
            options: []*dg.ApplicationCommandOption {
                {
                    Type: dg.ApplicationCommandOptionUser,
                    Name: "användarnamn",
                    Description: "Användare att visa vadslagningar för.",
                    Required: true,
                },
                {
                    Type: dg.ApplicationCommandOptionString,
                    Name: "typ",
                    Description: "Vill du enbart visa en viss typ av vad?",
                    Choices: []*dg.ApplicationCommandOptionChoice {
                        {
                            Name: "vunna",
                            Value: "1",
                        },
                        {
                            Name: "förlorade",
                            Value: "0",
                        },
                    },
                },
            },
        },
        {
            name: "poäng",
            description: "Visa dina poäng och topp 10 på servern.",
            category: CommandCategoryListing,
            admin: false,
        },
        {
            name: "inställningar",
            description: "Inställningar för din användare.",
            category: CommandCategoryGeneral,
            admin: false,
        },
        {
            name: "info",
            description: "Teknisk info om botten.",
            category: CommandCategoryGeneral,
            admin: false,
        },
        {
            name: "sammanfatta",
            description: "Sammanfatta denna omgång till #bets.",
            category: CommandCategoryAdmin,
            admin: true,
        },
        {
            name: "update",
            description: "Uppdatera alla kommandon eller ett enskilt.",
            category: CommandCategoryAdmin,
            admin: true,
        },
        {
            name: "delete",
            description: "Ta bort ett enskilt kommando.",
            category: CommandCategoryAdmin,
            admin: true,
        },
        {
            name: "checkbets",
            description: "Kör checks för challenges/bets.",
            category: CommandCategoryAdmin,
            admin: true,
        },
    }
)

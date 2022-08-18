package main

import (
    dg "github.com/bwmarrin/discordgo"
)

/*
Commands and handlers declarations
*/

var (
    COMMANDS = []cmd {
        {
            name: "hjälp",
            description: "Få hjälp med hur denna bot fungerar.",
            category: CommandCategoryGeneral,
            admin: false,
        },
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

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
            category: General,
            admin: false,
        },
        {
            name: "slåvad",
            description: "Slå vad om en kommande match.",
            category: Betting,
            admin: false,
        },
        {
            name: "ångra",
            description: "Ångra ett vad du har lagt.",
            category: Betting,
            admin: false,
        },
        {
            name: "utmana",
            description: "Utmana en annan användare.",
            category: Betting,
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
            description: "Be den du utmanat om att få dra tillbaka utmaningen.",
            category: Betting,
            admin: false,
        },
        {
            name: "kommande",
            description: "Lista dina kommande vadslagningar.",
            category: Betting,
            admin: false,
        },
        {
            name: "vadslagningar",
            description: "Lista en annan användares tidigare vadslagningar.",
            category: Betting,
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
            category: Betting,
            admin: false,
        },
        {
            name: "inställningar",
            description: "Inställningar för din användare.",
            category: General,
            admin: false,
        },
        {
            name: "info",
            description: "Teknisk info om botten.",
            category: General,
            admin: false,
        },
        {
            name: "sammanfatta",
            description: "Sammanfatta denna omgång till #bets.",
            category: Admin,
            admin: true,
        },
        {
            name: "update",
            description: "Uppdatera alla kommandon eller ett enskilt.",
            category: Admin,
            admin: true,
        },
        {
            name: "delete",
            description: "Ta bort ett enskilt kommando.",
            category: Admin,
            admin: true,
        },
        {
            name: "checkbets",
            description: "Kollar om alla lagda bets överenstämmer med resultat.",
            category: Admin,
            admin: true,
        },
    }

    COMMAND_HANDLERS = map[string]func(s *dg.Session, i *dg.InteractionCreate) {
        // User commands
        "hjälp": func(s *dg.Session, i *dg.InteractionCreate)         {   helpCommand(s, i)               },
        "slåvad": func(s *dg.Session, i *dg.InteractionCreate)        {   betCommand(s, i)                },
        "ångra": func(s *dg.Session, i *dg.InteractionCreate)         {   regretCommand(s, i)             },
        "utmana": func(s *dg.Session, i *dg.InteractionCreate)        {   challengeCommand(s, i)          },
        "fegaur": func(s *dg.Session, i *dg.InteractionCreate)        {   chickenCommand(s, i)            },
        "kommande": func(s *dg.Session, i *dg.InteractionCreate)      {   upcomingCommand(s, i)           },
        "vadslagningar": func(s *dg.Session, i *dg.InteractionCreate) {   listBetsCommand(s, i)           },
        "poäng": func(s *dg.Session, i *dg.InteractionCreate)         {   pointsCommand(s, i)             },
        "inställningar": func(s *dg.Session, i *dg.InteractionCreate) {   settingsCommand(s, i)           },
        "info": func(s *dg.Session, i *dg.InteractionCreate)          {   infoCommand(s, i)               },

        // Admin commands
        "sammanfatta": func(s *dg.Session, i *dg.InteractionCreate)   {   summaryCommand(s,i)             },
        "update": func(s *dg.Session, i *dg.InteractionCreate)        {   updateCommand(s, i)             },
        "delete": func(s *dg.Session, i *dg.InteractionCreate)        {   deleteCommand(s, i)             },
        "checkbets": func(s *dg.Session, i *dg.InteractionCreate)     {   checkBetsCommand(s,i)           },
    }

    // Component handlers
    COMPONENT_HANDLERS = map[string]func(s *dg.Session, i *dg.InteractionCreate) {
        "betOnSelected": func(s *dg.Session, i *dg.InteractionCreate)           {   betOnSelected(s, i)              },
        "betScoreHome": func(s *dg.Session, i *dg.InteractionCreate)            {   betScoreComponent(s, i, Home)    },
        "betScoreAway": func(s *dg.Session, i *dg.InteractionCreate)            {   betScoreComponent(s, i, Away)    },
        "challSelectWinner": func(s *dg.Session, i *dg.InteractionCreate)       {   challSelectWinner(s, i)          },
        "challSelectPoints": func(s *dg.Session, i *dg.InteractionCreate)       {   challSelectPoints(s, i)          },
        "challAcceptDiscard": func(s *dg.Session, i *dg.InteractionCreate)      {   challAcceptDiscard(s, i)         },
        "challAcceptDiscardDo": func(s *dg.Session, i *dg.InteractionCreate)    {   challAcceptDiscardDo(s, i)       },
        "settingsVisibility": func(s *dg.Session, i *dg.InteractionCreate)      {   settingsVisibility(s, i)         },
        "settingsChall": func(s *dg.Session, i *dg.InteractionCreate)           {   settingsChall(s, i)              },
        "updateCommandDo": func(s *dg.Session, i *dg.InteractionCreate)         {   updateCommandDo(s, i)            },
        "deleteCommandDo": func(s *dg.Session, i *dg.InteractionCreate)         {   deleteCommandDo(s, i)            },
        "regretSelected": func(s *dg.Session, i *dg.InteractionCreate)          {   regretSelected(s, i)             },
        "challAnswer": func(s *dg.Session, i *dg.InteractionCreate)             {   challAnswer(s, i)                },
        "chickenSelected": func(s *dg.Session, i *dg.InteractionCreate)         {   chickenSelected(s, i)            },
        "chickenAnswer": func(s *dg.Session, i *dg.InteractionCreate)           {   chickenAnswer(s, i)              },
    }
)

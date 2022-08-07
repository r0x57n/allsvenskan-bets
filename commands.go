package main

import (
	dg "github.com/bwmarrin/discordgo"
)

/*
Commands and handlers declarations
*/

var (
    COMMANDS = []dg.ApplicationCommand {
        {
            Name: "hjälp",
            Description: "Få hjälp med hur denna bot fungerar.",
        },
        {
            Name: "slåvad",
            Description: "Slå vad om en kommande match.",
        },
        {
            Name: "utmana",
            Description: "Utmana en annan användare om en kommande match.",
            Options: []*dg.ApplicationCommandOption {
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
            Name: "kommande",
            Description: "Lista dina kommande vadslagningar.",
        },
        {
            Name: "ångra",
            Description: "Ta bort ett vad som du gjort, om utmaning måste andra användaren också acceptera.",
        },
        {
            Name: "tidigare",
            Description: "Lista en annan användares tidigare vadslagningar.",
            Options: []*dg.ApplicationCommandOption {
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
            Name: "poäng",
            Description: "Visa dina poäng och topp 10 på servern.",
        },
        {
            Name: "inställningar",
            Description: "Ställ in inställningar för din användare.",
        },
        {
            Name: "info",
            Description: "Teknisk info om mig.",
        },
        {
            Name: "sammanfatta",
            Description: "Sammanfatta denna omgång till #bets.",
        },
        {
            Name: "update",
            Description: "Uppdatera alla kommandon eller ett enskilt.",
        },
        {
            Name: "delete",
            Description: "Ta bort ett enskilt kommando.",
        },
        {
            Name: "checkbets",
            Description: "Kollar om alla lagda bets överenstämmer med resultat.",
        },
    }

    COMMAND_HANDLERS = map[string]func(s *dg.Session, i *dg.InteractionCreate) {
        // User commands
        "hjälp": func(s *dg.Session, i *dg.InteractionCreate)         {   helpCommand(s, i, &COMMANDS)    },
        "slåvad": func(s *dg.Session, i *dg.InteractionCreate)        {   betCommand(s, i)                },
        "utmana": func(s *dg.Session, i *dg.InteractionCreate)        {   challengeCommand(s, i)          },
        "kommande": func(s *dg.Session, i *dg.InteractionCreate)      {   upcomingCommand(s, i)           },
        "ångra": func(s *dg.Session, i *dg.InteractionCreate)         {   regretCommand(s, i)             },
        "tidigare": func(s *dg.Session, i *dg.InteractionCreate)      {   earlierCommand(s, i)            },
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
        "betOnSelected": func(s *dg.Session, i *dg.InteractionCreate)      {   betOnSelected(s, i)              },
        "betScoreHome": func(s *dg.Session, i *dg.InteractionCreate)       {   betScoreComponent(s, i, Home)    },
        "betScoreAway": func(s *dg.Session, i *dg.InteractionCreate)       {   betScoreComponent(s, i, Away)    },
        "challSelectWinner": func(s *dg.Session, i *dg.InteractionCreate)  {   challSelectWinner(s, i)          },
        "challSelectPoints": func(s *dg.Session, i *dg.InteractionCreate)  {   challSelectPoints(s, i)          },
        "challAcceptDiscard": func(s *dg.Session, i *dg.InteractionCreate) {   challAcceptDiscard(s, i)         },
        "settingsVisibility": func(s *dg.Session, i *dg.InteractionCreate) {   settingsVisibility(s, i)         },
        "settingsChall": func(s *dg.Session, i *dg.InteractionCreate)      {   settingsChall(s, i)              },
        "updateCommandDo": func(s *dg.Session, i *dg.InteractionCreate)    {   updateCommandDo(s, i, &COMMANDS) },
        "deleteCommandDo": func(s *dg.Session, i *dg.InteractionCreate)    {   deleteCommandDo(s, i)            },
    }
)

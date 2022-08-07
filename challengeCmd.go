package main

import (
	"log"
    "fmt"
    "strings"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	dg "github.com/bwmarrin/discordgo"
)

func challengeCommand(s *dg.Session, i *dg.InteractionCreate) {
    if len(i.Interaction.ApplicationCommandData().Options) != 2 {
        log.Panic("Not enough argumenst given...")
    }

    option := i.Interaction.ApplicationCommandData().Options[0]
    challengee, err := s.User(fmt.Sprintf("%v", option.Value))
    if err != nil { log.Panic(err) }

    // Check if user allows challenges
    u := getUser(challengee.ID)
    if u.interactable == 0 {
        msgStdInteractionResponse(s, i, "Användaren tillåter inte utmaningar.")
        return
    }

    msg := fmt.Sprintf("Utmana användare %v om hur följande match kommer sluta.", challengee.Username)

    options := getRoundMatchesAsOptions(challengee.ID)

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Vilken match vill du utmana om?",
                    CustomID: "challSelectWinner",
                    Options: *options,
                },
            },
        },
    }

    compInteractionResponse(s, i, dg.InteractionResponseChannelMessageWithSource, msg, components)
}

func challSelectWinner(s *dg.Session, i *dg.InteractionCreate) {
	svffDB, err := sql.Open(DB_TYPE, SVFF_DB)
	defer svffDB.Close()
	if err != nil { log.Fatal(err) }

	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

    // Parsing values
	vals := i.MessageComponentData().Values
    if len(vals) == 0 { log.Panic(err) }

    splitted := strings.Split(vals[0], "_")

    challengee, err := s.User(splitted[0])
	if err != nil { log.Panic(err) }

    matchID := splitted[1]

	_, err = s.User(i.Interaction.Member.User.ID)
	if err != nil { log.Panic(err) }

    // Do things
    var m match

	err = svffDB.QueryRow("SELECT id, homeTeam, awayTeam, date FROM matches WHERE id=?", matchID).Scan(&m.id, &m.homeTeam, &m.awayTeam, &m.date)
	if err != nil { log.Panic(err) }

    val := challengee.ID + "_" + matchID + "_"

    msg := "Vem tror du vinner?"

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "challSelectPoints",
                    Options: []dg.SelectMenuOption{
                        {
                            Label: m.homeTeam,
                            Value: val + "home",
                            Default: true,
                        },
                        {
                            Label: m.awayTeam,
                            Value: val + "away",
                        },
                    },
                },
            },
        },
    }

    compInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, msg, components)
}

func challSelectPoints(s *dg.Session, i *dg.InteractionCreate) {
	betsDB, err := sql.Open(DB_TYPE, BETS_DB)
	defer betsDB.Close()
	if err != nil { log.Fatal(err) }

    msg := "Hur mycket poäng vill du satsa?"

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "challAcceptDiscard",
                    Placeholder: "Poäng att satsa.",
                    Options: []dg.SelectMenuOption{
                        {
                            Label: "1 Poäng",
                            Value: "1",
                        },
                        {
                            Label: "5 Poäng",
                            Value: "5",
                        },
                    },
                },
            },
        },
    }

    compInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, msg, components)
}

func challAcceptDiscard(s *dg.Session, i *dg.InteractionCreate) {
    msg := fmt.Sprintf("\nDu tror att %v vinner för %v poäng.\n\n", 1, 2)
    msg += "Är du säker? En utmaning kan bara tas bort om den du utmanar accepterar borttagningen.\n"

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.Button{
                    Label: "Skicka utmaning",
                    Style: dg.PrimaryButton,
                    CustomID: "challAccept",
                },
                dg.Button{
                    Label: "Släng",
                    Style: dg.DangerButton,
                    CustomID: "challDiscard",
                },
            },
        },
    }

    compInteractionResponse(s, i, dg.InteractionResponseUpdateMessage, msg, components)
}

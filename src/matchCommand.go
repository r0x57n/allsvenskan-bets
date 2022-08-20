package main

import (
	"fmt"
	"log"
	"strconv"
	dg "github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

func (b *botHolder) matchCommand(i *dg.InteractionCreate) {

    rows, err := b.db.Query("SELECT m.id, m.hometeam, m.awayteam, m.date, m.homescore, m.awayscore, m.finished " +
                            "FROM matches AS m " +
                            "WHERE round=$1", getCurrentRound(b.db))
    if err != nil { log.Panic(err) }

    options := *getOptionsOutOfRows(rows)

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en match",
                    CustomID: MatchSendInfo, // component handler
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(b.session, i, NewMsg, "Omgångens matcher.", components)
}

func (b *botHolder) matchSendInfo(i *dg.InteractionCreate) {
    vals := getValuesOrRespond(b.session, i, NewMsg)
    if vals == nil { return }

    mid := vals[0]
    m := getMatch(b.db, "id=$1", mid)

    //datetime, _ := time.Parse(DB_TIME_LAYOUT, m.date)

    matchInfo := fmt.Sprintf("%v - %v, slutade %v - %v\n", m.hometeam, m.awayteam, m.homescore, m.awayscore)

    // all bets
    bets := *getBets(b.db, "matchid=$1", mid)

    msgBets := ""
    if len(bets) == 0 {
        msgBets = "-"
    }

    for _, bet := range bets {
        user, _ := b.session.User(strconv.Itoa(bet.uid))
        username := ""
        if user == nil {
            username = strconv.Itoa(bet.uid)
        } else {
            username = user.Username
        }

        msgBets += fmt.Sprintf("%v gissade på %v - %v", username, bet.homescore, bet.awayscore)
    }

    // all challenges
    challenges := *getChallenges(b.db, "matchid=$1 AND (status=$2 OR status=$3)",
                                 mid, ChallengeStatusHandled, ChallengeStatusAccepted)

    msgChalls := ""
    if len(challenges) == 0 {
        msgChalls = "-"
    }

    for _, c := range challenges {
        userChallenger, _ := b.session.User(strconv.Itoa(c.challengerid))
        userChallengee, _ := b.session.User(strconv.Itoa(c.challengeeid))
        usernameChallenger := ""
        usernameChallengee := ""
        if userChallenger == nil {
            usernameChallenger = strconv.Itoa(c.challengerid)
        } else {
            usernameChallenger = userChallenger.Username
        }

        if userChallengee == nil {
            usernameChallengee = strconv.Itoa(c.challengeeid)
        } else {
            usernameChallengee = userChallengee.Username
        }

        winner := ""
        if c.condition == ChallengeConditionWinnerHome {
            winner = m.hometeam
        } else {
            winner = m.awayteam
        }

        msgChalls += fmt.Sprintf("%v utmanade %v om att %v skulle vinna\n",
                                 usernameChallenger, usernameChallengee, winner)
    }

    fields := []*dg.MessageEmbedField {
        {
            Name: "Gissningar",
            Value: msgBets,
        },
        {
            Name: "Utmaningar",
            Value: msgChalls,
        },
    }

	if err := b.session.InteractionRespond(i.Interaction, &dg.InteractionResponse {
		Type: UpdateMsg,
		Data: &dg.InteractionResponseData {
			Flags: 1 << 6, // Ephemeral
            Components: []dg.MessageComponent{},
            Embeds: []*dg.MessageEmbed {
                {
                    Title: "Resultat",
                    Description: matchInfo,
                    Fields: fields,
                },
            },

		},
	}); err != nil { log.Panic(err) }
}

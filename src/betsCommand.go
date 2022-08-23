package main

import (
    "fmt"
    "math"
    "time"
    "strconv"
    "log"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) listBetsCommand(i *dg.InteractionCreate) {
    // Get options and parse
    options := getOptionsOrRespond(b.session, i, NewMsg)
    if options == nil { return }
    uid := options[0].Value

    listTypes := -1
    if len(options) == 2 {
        listTypes, _ = strconv.Atoi(fmt.Sprintf("%v", options[1].Value))
    }

    desc := ""
    userToView := getUser(b.db, fmt.Sprint(uid))
    userNotViewingThemselves := userToView.UserID != getUserFromInteraction(b.db, i).UserID

    // Error checking
    errors := []CommandError{}
    if !userToView.Viewable && userNotViewingThemselves {
        errors = append(errors, ErrorUserNotViewable)
    } else if !userToView.Viewable && !userNotViewingThemselves {
        desc = "Andra användare kan inte se dina vadslagningar.\n"
    }

    desc += "Hemmalag - Bortalag [resultat] (gissning)"

    if len(errors) != 0 {
        addErrorsResponse(b.session , i, NewMsg, errors, "Kan inte visa spelarens gissningar/utmaningar.")
        return
    }

    where := ""
    switch listTypes {
        case BetStatusLost:
            where = fmt.Sprintf("uid=%v AND status=%v", uid, BetStatusLost)
        case BetStatusWon:
            where = fmt.Sprintf("uid=%v AND (status=%v OR status=%v)", uid, BetStatusWon, BetStatusAlmostWon)
        case BetStatusUnhandled:
            where = fmt.Sprintf("uid=%v AND status=%v", uid, BetStatusUnhandled)
        default: // all
            where = fmt.Sprintf("uid=%v", uid)
    }

    rows, err := b.db.Query("SELECT b.homescore, b.awayscore, b.matchid, b.status, m.hometeam, m.awayteam, m.date, m.homescore, m.awayscore " +
                           "FROM bets AS b " +
                           "JOIN matches AS m ON b.matchid=m.id " +
                           "WHERE " + where)
    if err != nil { log.Panic(err) }
    defer rows.Close()

    comingBets, wonBets, lostBets := "", "", ""

    for rows.Next() {
        var bet Bet
        var m Match

        rows.Scan(&bet.HomeScore, &bet.AwayScore, &bet.MatchID, &bet.Status, &m.HomeTeam, &m.AwayTeam, &m.Date, &m.HomeScore, &m.AwayScore)

        switch bet.Status {
            case BetStatusLost:
                lostBets += fmt.Sprintf("%v - %v [%v-%v] (%v-%v)\n", m.HomeTeam, m.AwayTeam, m.HomeScore, m.AwayScore, bet.HomeScore, bet.AwayScore)
            case BetStatusWon, BetStatusAlmostWon:
                wonBets += fmt.Sprintf("%v - %v [%v-%v] (%v-%v)\n", m.HomeTeam, m.AwayTeam, m.HomeScore, m.AwayScore, bet.HomeScore, bet.AwayScore)
            case BetStatusUnhandled:
                matchDate, err := time.Parse(DB_TIME_LAYOUT, m.Date)
                if err != nil { log.Printf("Couldn't parse date: %v", err) }
                daysUntilMatch := math.Round(time.Until(matchDate).Hours() / 24)
                played := ""

                if math.Signbit(daysUntilMatch) {
                    played = "spelas nu"
                } else if daysUntilMatch == 0 {
                    played = "spelas idag"
                } else {
                    played = fmt.Sprintf("om %v dagar", daysUntilMatch)
                }
                comingBets += fmt.Sprintf("%v (**%v**) - %v (**%v**), %v.\n", m.HomeTeam, bet.HomeScore, m.AwayTeam, bet.AwayScore, played)
            default:
                addErrorResponse(b.session, i, NewMsg, "Got unidentifiable b.status in listBetsCommand.")
                return
        }
    }

    if comingBets == "" { comingBets = "-" }
    if lostBets == "" { lostBets = "-" }
    if wonBets == "" { wonBets = "-" }

    rows, err = b.db.Query("SELECT c.challengerid, c.challengeeid, c.points, c.condition, c.winner, " +
                           "m.hometeam, m.awayteam " +
                           "FROM challenges AS c " +
                           "JOIN matches AS m ON c.matchid=m.id " +
                           "WHERE (c.challengerid=$1 OR c.challengeeid=$2) AND " +
                           "(c.status=$3 OR c.status=$4)", uid, uid,
                            ChallengeStatusAccepted, ChallengeStatusHandled)
    if err != nil { log.Panic(err) }
    defer rows.Close()

    type challAndMatch struct {
        c Challenge
        m Match
    }

    challenges := "-"

    for rows.Next() {
        var c Challenge
        var m Match

        rows.Scan(&c.ChallengerID, &c.ChallengeeID, &c.Points, &c.Condition, &c.Winner,
                  &m.HomeTeam, &m.AwayTeam)

        challenger, err := b.session.User(strconv.Itoa(c.ChallengerID))
        if err != nil { log.Panic(err) }
        challengee, err := b.session.User(strconv.Itoa(c.ChallengeeID))
        if err != nil { log.Panic(err) }

        winOrLose := "vinna"
        if c.Condition == ChallengeConditionWinnerAway {
            winOrLose = "förlora"
        }


        winner := ""
        if c.Winner != 0 {
            userIsChallenger := fmt.Sprint(getUserFromInteraction(b.db, i).UserID) == challenger.ID

            winner = "[+]"
            if userIsChallenger && c.Winner == ChallengeWinnerChallengee {
                winner = "[-]"
            } else if c.Winner == ChallengeWinnerNone {
                winner = "[/]"
            }

            if challenges == "-" { challenges = "" }
            challenges += fmt.Sprintf("**%v** utmanade **%v** om att **%v** skulle %v mot **%v** för **%v** poäng. %v\n",
                                        challenger.Username, challengee.Username, m.HomeTeam, winOrLose, m.AwayTeam, c.Points, winner)
        } else {
            if comingBets == "-" { comingBets = "" }
            comingBets += fmt.Sprintf("**%v** utmanar **%v** om att **%v** ska %v mot **%v** för **%v** poäng. %v\n",
                                       challenger.Username, challengee.Username, m.HomeTeam, winOrLose, m.AwayTeam, c.Points, winner)
        }
    }

    fields := []*dg.MessageEmbedField {}

    switch (listTypes) {
        case BetStatusLost:
            fields = append(fields, &dg.MessageEmbedField{
                Name: "Förlorade",
                Value: lostBets,
            })
        case BetStatusWon:
            fields = append(fields, &dg.MessageEmbedField{
                Name: "Vunna",
                Value: wonBets,
            })
        case BetStatusUnhandled:
            fields = append(fields, &dg.MessageEmbedField{
                Name: "Kommande",
                Value: comingBets,
            })
        default:
            fields = []*dg.MessageEmbedField {
                {
                    Name: "Kommande",
                    Value: comingBets,
                },
                {
                    Name: "Vunna",
                    Value: wonBets,
                },
                {
                    Name: "Förlorade",
                    Value: lostBets,
                },
                {
                    Name: "Utmaningar",
                    Value: challenges,
                },
            }
    }

    addEmbeddedInteractionResponse(b.session, i, NewMsg, fields, "Vadslagningar", desc)
}

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

func (b *botHolder) listBetsCommand(i *dg.InteractionCreate) {
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
    userNotViewingThemselves := userToView.uid != getUserFromInteraction(b.db, i).uid

    // Error checking
    errors := []CommandError{}
    if !userToView.viewable && userNotViewingThemselves {
        errors = append(errors, ErrorUserNotViewable)
    } else if !userToView.viewable && !userNotViewingThemselves {
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
        var bet bet
        var m match

        rows.Scan(&bet.homescore, &bet.awayscore, &bet.matchid, &bet.status, &m.hometeam, &m.awayteam, &m.date, &m.homescore, &m.awayscore)

        switch bet.status {
            case BetStatusLost:
                lostBets += fmt.Sprintf("%v - %v [%v-%v] (%v-%v)\n", m.hometeam, m.awayteam, m.homescore, m.awayscore, bet.homescore, bet.awayscore)
            case BetStatusWon, BetStatusAlmostWon:
                wonBets += fmt.Sprintf("%v - %v [%v-%v] (%v-%v)\n", m.hometeam, m.awayteam, m.homescore, m.awayscore, bet.homescore, bet.awayscore)
            case BetStatusUnhandled:
                matchDate, err := time.Parse(DB_TIME_LAYOUT, m.date)
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
                comingBets += fmt.Sprintf("%v (**%v**) - %v (**%v**), %v.\n", m.hometeam, bet.homescore, m.awayteam, bet.awayscore, played)
            default:
                addErrorResponse(b.session, i, NewMsg, "Got unidentifiable b.status in listBetsCommand.")
                return
        }
    }

    if comingBets == "" { comingBets = "-" }
    if lostBets == "" { lostBets = "-" }
    if wonBets == "" { wonBets = "-" }

    rows, err = b.db.Query("SELECT c.challengerid, c.challengeeid, c.points, c.condition, " +
                           "m.hometeam, m.awayteam " +
                           "FROM challenges AS c " +
                           "JOIN matches AS m ON c.matchid=m.id " +
                           "WHERE (c.challengerid=$1 OR c.challengeeid=$2) AND c.status=$3", uid, uid, ChallengeStatusHandled)
    if err != nil { log.Panic(err) }
    defer rows.Close()

    type challAndMatch struct {
        c challenge
        m match
    }

    challenges := "-"

    for rows.Next() {
        var c challenge
        var m match

        rows.Scan(&c.challengerid, &c.challengeeid, &c.points, &c.condition,
                  &m.hometeam, &m.awayteam)

        challenger, err := b.session.User(strconv.Itoa(c.challengerid))
        if err != nil { log.Panic(err) }
        challengee, err := b.session.User(strconv.Itoa(c.challengeeid))
        if err != nil { log.Panic(err) }

        winOrLose := "vinna"
        if c.condition == ChallengeConditionWinnerAway {
            winOrLose = "förlora"
        }

        if challenges == "-" { challenges = "" }
        challenges += fmt.Sprintf("**%v** utmanade **%v** om att **%v** skulle %v mot **%v** för **%v** poäng.\n",
                                    challenger.Username, challengee.Username, m.hometeam, winOrLose, m.awayteam, c.points)
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
            }
    }

    fields = append(fields, &dg.MessageEmbedField{
        Name: "Utmaningar",
        Value: challenges,
    })

    addEmbeddedInteractionResponse(b.session, i, NewMsg, fields, "Vadslagningar", desc)
}

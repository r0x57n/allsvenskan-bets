package main

import (
    "fmt"
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

    listTypes := All
    if len(options) == 2 {
        listTypes, _ = strconv.Atoi(fmt.Sprintf("%v", options[1].Value))
    }

    desc := ""

    userToView := getUser(b.db, fmt.Sprint(uid))
    userNotViewingThemselves := userToView.uid != getUserFromInteraction(b.db, i).uid
    if !userToView.viewable && userNotViewingThemselves {
        desc := "Användaren har valt att dölja sina vadslagningar."
        addInteractionResponse(b.session, i, NewMsg, desc)
        return
    } else if !userToView.viewable && !userNotViewingThemselves {
        desc = "Andra användare kan inte se dina vadslagningar."
    }

    where := ""
    switch listTypes {
        case Lost:
            where = fmt.Sprintf("uid=%v AND status=%v", uid, BetStatusWon)
        case Won:
            where = fmt.Sprintf("uid=%v AND status=%v", uid, BetStatusLost)
        case All:
            where = fmt.Sprintf("uid=%v AND (status=%v OR status=%v)", uid, BetStatusWon, BetStatusLost)
        default:
            addErrorResponse(b.session, i, NewMsg, "Got unidentifiable listTypes in listBetsCommand.")
            return
    }

    rows, err := b.db.Query("SELECT b.homescore, b.awayscore, b.matchid, b.status, m.hometeam, m.awayteam " +
                            "FROM bets AS b " +
                            "JOIN matches AS m ON b.matchid=m.id " +
                            "WHERE " + where)
    defer rows.Close()
    if err != nil { log.Panic(err) }

    wonBets, lostBets := "-", "-"

    for rows.Next() {
        var bet bet
        var m match

        rows.Scan(&bet.homescore, &bet.awayscore, &bet.matchid, &bet.status, &m.hometeam, &m.awayteam)

        switch bet.status {
            case BetStatusLost:
                if lostBets == "-" { lostBets = "" }
                lostBets += fmt.Sprintf("%v (**%v**) - %v (**%v**)\n", m.hometeam, bet.homescore, m.awayteam, bet.awayscore)
            case BetStatusWon:
                if wonBets == "-" { wonBets = "" }
                wonBets += fmt.Sprintf("%v (**%v**) - %v (**%v**)\n", m.hometeam, bet.homescore, m.awayteam, bet.awayscore)
            case BetStatusUnhandled:
                continue
            default:
                addErrorResponse(b.session, i, NewMsg, "Got unidentifiable b.status in listBetsCommand.")
                return
        }
    }

    if wonBets == "-" && lostBets == "-" {
        desc = fmt.Sprintf("Användaren har inga vadslagningar ännu!", )
    }

    rows, err = b.db.Query("SELECT c.challengerid, c.challengeeid, c.points, c.condition, " +
                           "m.hometeam, m.awayteam " +
                           "FROM challenges AS c " +
                           "JOIN matches AS m ON c.matchid=m.id " +
                           "WHERE (c.challengerid=$1 OR c.challengeeid=$2) AND c.status=$3", uid, uid, ChallengeStatusHandled)
    defer rows.Close()
    if err != nil { log.Panic(err) }

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
        challenges += fmt.Sprintf("**%v** utmanade **%v** om att **%v** skulle %v mot **%v** för **%v** poäng\n",
                                    challenger.Username, challengee.Username, m.hometeam, winOrLose, m.awayteam, c.points)
    }

    fields := []*dg.MessageEmbedField {}

    if listTypes == 0 {
        fields = []*dg.MessageEmbedField {
            {
                Name: "Förlorade",
                Value: lostBets,
            },
        }
    } else if listTypes == 1 {
        fields = []*dg.MessageEmbedField {
            {
                Name: "Vunna",
                Value: wonBets,
            },
        }
    } else {
        fields = []*dg.MessageEmbedField {
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
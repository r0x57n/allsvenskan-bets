package main

import (
    "fmt"
    "strconv"
    "strings"
    "log"
    "time"
    _ "github.com/lib/pq"
    dg "github.com/bwmarrin/discordgo"
)

// Command: slåvad
func betCommand(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
    defer db.Close()

    options := *getCurrentMatchesAsOptions(db)
    if len(options) == 0 {
        addInteractionResponse(s, i, NewMsg, "Inga matcher tillgängliga! :(")
        return
    }

    uid := getInteractUID(i)

    // map match id:s to bets so we can easier build strings later
    matchidToBet := make(map[string]bet)
    betsRows, err := db.Query("SELECT matchid, homescore, awayscore FROM bets WHERE status=$1 AND uid=$2", BetStatusUnhandled, uid)
    if err != nil { log.Panic(err) }

    for betsRows.Next() {
        var b bet
        betsRows.Scan(&b.matchid, &b.homescore, &b.awayscore)
        matchidToBet[strconv.Itoa(b.matchid)] = b
    }

    if len(matchidToBet) > 0 {
        for i, option := range options {
            b := matchidToBet[option.Value]
            options[i].Label += fmt.Sprintf(" [%v-%v]", b.homescore, b.awayscore)
        }
    }

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    Placeholder: "Välj en match",
                    CustomID: "betOnSelected", // component handler
                    Options: options,
                },
            },
        },
    }

    addCompInteractionResponse(s, i, NewMsg, "Kommande omgångens matcher.", components)
}

func betOnSelected(s *dg.Session, i *dg.InteractionCreate) {
    db := connectDB()
	defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }

	mid := values[0]
	uid := getInteractUID(i)

    earlierBetScore := [2]int {-1, -1}
    earlierBet := getBet(db, "uid=$1 AND matchid=$2", uid, mid)
    if earlierBet.id != -1 {
        earlierBetScore[0], earlierBetScore[1] = earlierBet.homescore, earlierBet.awayscore
    }

    m := getMatch(db, "id=$1", mid)
    if m.id == -1 {
        addErrorResponse(s, i, UpdateMsg)
        return
    }

    datetime, err := time.Parse(DB_TIME_LAYOUT, m.date)
	if err != nil {
        addErrorResponse(s, i, NewMsg, "Couldn't translate match date from database...")
        return
    }

    if time.Now().After(datetime) {
        addCompInteractionResponse(s, i, UpdateMsg, "Du kan inte betta på en match som startat.", []dg.MessageComponent{})
        return
    }

    msg := fmt.Sprintf("Hemmalag: %v\n", m.hometeam)
    msg += fmt.Sprintf("Bortalag: %v\n\n", m.awayteam)
    msg += fmt.Sprintf("Spelas: %v, klockan %v\n\n", datetime.Format("2006-01-02"), datetime.Format("15:04"))
    msg += fmt.Sprintf("**Poäng** *(hemmalag överst)*")

    components := []dg.MessageComponent {
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "betScoreHome",
                    Placeholder: "Hemmalag",
                    Options: *getScoresAsOptions(m.id, earlierBetScore[0]),
                },
            },
        },
        dg.ActionsRow {
            Components: []dg.MessageComponent {
                dg.SelectMenu {
                    CustomID: "betScoreAway",
                    Placeholder: "Bortalag",
                    Options: *getScoresAsOptions(m.id, earlierBetScore[1]),
                },
            },
        },
    }

    addCompInteractionResponse(s, i, UpdateMsg, msg, components)
}

func betScoreComponent(s *dg.Session, i *dg.InteractionCreate, where BetLocation) {
    db := connectDB()
    defer db.Close()

    values := getValuesOrRespond(s, i, UpdateMsg)
    if values == nil { return }

    var (
        uid = getInteractUID(i)
        splitted = strings.Split(values[0], "_")
        mid = splitted[0]
        homescore = "0"
        awayscore = "0"
        score = ""
        awayOrHome = ""
    )

    switch where {
        case Home:
            homescore = splitted[1]
            score = homescore
            awayOrHome = "home"
        case Away:
            awayscore = splitted[1]
            score = awayscore
            awayOrHome = "away"
        default:
            addErrorResponse(s, i, UpdateMsg)
            return
    }

    m := getMatch(db, "id=$1", mid)
    if m.id == -1 {
        addErrorResponse(s, i, UpdateMsg)
        return
    }

    if matchHasBegun(s, i, m) {
        addCompInteractionResponse(s, i, UpdateMsg, "Matchen har startat...", []dg.MessageComponent{})
        return
    }

    var err error
    hasBettedBefore := getBet(db, "uid=$1 AND matchid=$2", uid, mid).id != -1
    if hasBettedBefore {
        _, err = db.Exec("UPDATE bets SET " + awayOrHome + "score=$1 WHERE uid=$2 AND matchid=$3", score, uid, mid)
        if err != nil { log.Panic(err) }
    } else {
        _, err = db.Exec("INSERT INTO bets (uid, matchid, homescore, awayscore) VALUES ($1, $2, $3, $4)", uid, mid, homescore, awayscore)
        if err != nil { log.Panic(err) }
    }

    addNoInteractionResponse(s, i)
}

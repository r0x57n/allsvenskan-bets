package main

import (
    "database/sql"
    "fmt"
    "log"
    "strconv"
    "encoding/json"
    "time"
    dg "github.com/bwmarrin/discordgo"
    _ "github.com/lib/pq"
)

func (b *Bot) roundCommand(i *dg.InteractionCreate) {
    options := getOptionsOrRespond(b.session, i, NewMsg)
    if options == nil { return }

    round, _ := strconv.Atoi(fmt.Sprint(options[0].Value))

    if round < 0 || round > 30 {
        addInteractionResponse(b.session, i, NewMsg, "Välj en omgång mellan 0-30.")
        return
    }

    if round == 0 {
        round = getCurrentRound(b.db) - 1
    }

    var jsonD []byte
    err := b.db.QueryRow("SELECT data FROM summaries WHERE round=$1 AND year=$2", round, time.Now().Year()).Scan(&jsonD)
    if err != nil {
        if err != sql.ErrNoRows {
            log.Panic(err)
        } else {
            addInteractionResponse(b.session, i, NewMsg, fmt.Sprintf("Omgången %v finns inte sammanfattad ännu.", round))
            return
        }
    }

    var roundData Round
    json.Unmarshal(jsonD, &roundData)

    title := fmt.Sprintf("Sammanfattning av omgång %v", roundData.Num)

    msg := fmt.Sprintf("**%v** matcher spelades och **%v** vadslagningar las.\n",
                       roundData.NumMatches, roundData.NumBets)
    msg += fmt.Sprintf("Av dessa var det **%v** korrekta, **%v** inkorrekta och **%v** nära gissningar.",
                       roundData.NumWins, roundData.NumLoss, roundData.NumClose)

    fields := b.getRoundEmbedFields(roundData)

    addEmbeddedInteractionResponse(b.session, i, NewMsg, fields, title, msg)
}

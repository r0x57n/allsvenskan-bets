package main

import (
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) infoCommand(i *dg.InteractionCreate) {
    str := "Jag är en bot gjord i Go med hjälp av [discordgo](https://github.com/bwmarrin/discordgo) paketet. Min källkod finns på [Github](https://github.com/r0x57n/allsvenskanBets). "
    str += "Just nu kör jag version " + VERSION + ".\n\n"
    str += "Förbättringsförslag mottages i #bets kanalen."

    fields := []*dg.MessageEmbedField {}
    addEmbeddedInteractionResponse(b.session, i, NewMsg, fields, "Hej!", str)
}

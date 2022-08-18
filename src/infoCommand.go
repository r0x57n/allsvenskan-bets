package main

import (
    dg "github.com/bwmarrin/discordgo"
)

// Command: info
func (b *botHolder) infoCommand(i *dg.InteractionCreate) {
    str := "Jag är en bot gjord i Go med hjälp av [discordgo](https://github.com/bwmarrin/discordgo) paketet. Min källkod finns på [Github](https://github.com/r0x57n/allsvenskanBets). "
    str += "Just nu kör jag version " + VERSION + "."

    fields := []*dg.MessageEmbedField {}
    addEmbeddedInteractionResponse(b.session, i, NewMsg, fields, "Hej!", str)
}

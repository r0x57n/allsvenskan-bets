package main

import (
    dg "github.com/bwmarrin/discordgo"
)

func newInfo(b *botHolder) *Info {
    cmd := new(Info)
    cmd.bot = b
    cmd.name = InfoCommand
    cmd.description = "testar"
    return cmd
}

// Command: info
func (cmd *Info) run(i *dg.InteractionCreate) {
    str := "Jag är en bot gjord i Go med hjälp av [discordgo](https://github.com/bwmarrin/discordgo) paketet. Min källkod finns på [Github](https://github.com/r0x57n/allsvenskanBets). "
    str += "Just nu kör jag version " + VERSION + "."

    fields := []*dg.MessageEmbedField {}
    addEmbeddedInteractionResponse(cmd.bot.session, i, NewMsg, fields, "Hej!", str)
}

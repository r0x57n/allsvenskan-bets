package main

import (
    dg "github.com/bwmarrin/discordgo"
)

func (b *botHolder) checkBetsCommand(i *dg.InteractionCreate) {
    if b.notOwnerRespond(i) { return }
    addInteractionResponse(b.session, i, NewMsg, "Checking stuff...")

    b.checkStuff(true)
}

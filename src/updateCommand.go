package main

import (
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) updateCommand(i *dg.InteractionCreate) {
    if b.notOwnerRespond(i) { return }

    addInteractionResponse(b.session, i, NewMsg, "Börjar uppdatera...")

    b.updateMatches(true)
}

package main

import (
    dg "github.com/bwmarrin/discordgo"
)

func (b *botHolder) checkBetsCommand(i *dg.InteractionCreate) {
    if b.notOwner(getInteractUID(i)) { return }

    addInteractionResponse(b.session, i, NewMsg, "Checking bets...")
    checkUnhandledBets()
    checkUnhandledChallenges()
}

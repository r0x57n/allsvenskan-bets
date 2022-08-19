package main

import (
    dg "github.com/bwmarrin/discordgo"
)

func newCheck(b *botHolder) *Check {
    cmd := new(Check)
    cmd.bot = b
    cmd.name = CheckCommand
    cmd.description = "testar"
    return cmd
}

// Command: checkbets
func (cmd *Check) run(i *dg.InteractionCreate) {
    b := cmd.bot
    if b.notOwner(getInteractUID(i)) { return }

    addInteractionResponse(b.session, i, NewMsg, "Checking bets...")
    checkUnhandledBets()
    checkUnhandledChallenges()
}

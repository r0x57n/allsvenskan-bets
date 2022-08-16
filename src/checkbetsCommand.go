package main

import (
    dg "github.com/bwmarrin/discordgo"
)

// Command: checkbets
func checkBetsCommand(s *dg.Session, i *dg.InteractionCreate) {
	if notOwner(s, i) { return }

    addInteractionResponse(s, i, NewMsg, "Checking bets...")
    checkUnhandledBets()
    checkUnhandledChallenges()
}

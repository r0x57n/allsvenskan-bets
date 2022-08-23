package main

func (b *Bot) checkStuff(interactive bool) {
    b.checkUnhandledBets(interactive)
    b.checkUnhandledChallenges(interactive)
    b.sendSummaries()
}

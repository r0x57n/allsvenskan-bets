package main

func (b *botHolder) checkStuff(interactive bool) {
    b.checkUnhandledBets(interactive)
    b.checkUnhandledChallenges(interactive)
    b.sendSummaries()
}

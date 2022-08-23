package main

import (
    "log"
    "os/exec"
    "fmt"
)

func (b *Bot) updateMatches(interactive bool) {
    log.Printf("Starting updater...")

    cmd := exec.Command("./updater")
    cmd.Dir = b.updaterPath

    if err := cmd.Run(); err != nil {
        log.Printf("Couldn't run updater: %v", err)
        b.messageOwner(fmt.Sprintf("Gick inte att uppdatera matcher: %v", err))
        return
    }

    if (interactive) {
        b.messageOwner("Matcher uppdaterade.")
    }

    log.Printf("Finished updater...")
}

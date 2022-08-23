package main

import (
    "fmt"
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) helpCommand(i *dg.InteractionCreate) {
    isOwner := getInteractUID(i) == b.owner

    help := fmt.Sprintf("Använd */%v* för att gissa vad en match kommer sluta målmässigt. Gissar du helt rätt får du 3 poäng, om du är ett mål ifrån ", GuessCommand)
    help += fmt.Sprintf("får du endast 1 poäng. Poängen kan användas i utmaningar mot andra spelare. För att utmana en spelare, kör kommandot ")
    help += fmt.Sprintf("*/%v* med en spelare vald och vilken typ av utmaning du vill skapa. Du kan bara utmana folk som har interagerat med botten. \n\n", ChallengeCommand)
    help += fmt.Sprintf("Resultaten för matcherna uppdateras lite då och då under dagen, därför kan det ta ett tag tills dess att poängen delas ut efter att en match är spelad.\n\n")
    help += fmt.Sprintf("Samtliga kommandon listas här nedan.")

    adminCmds := ""
    generalCmds := ""
    bettingCmds := ""
    listingCmds := ""

    for _, cmd := range b.commands {
        switch cmd.Category {
            case CommandCategoryAdmin:
                adminCmds += fmt.Sprintf("/%v - %v\n", cmd.Name, cmd.Description)
            case CommandCategoryGeneral:
                generalCmds += fmt.Sprintf("/%v - %v\n", cmd.Name, cmd.Description)
            case CommandCategoryBetting:
                bettingCmds += fmt.Sprintf("/%v - %v\n", cmd.Name, cmd.Description)
            case CommandCategoryListing:
                listingCmds += fmt.Sprintf("/%v - %v\n", cmd.Name, cmd.Description)
        }
    }

    fields := []*dg.MessageEmbedField {
        {
            Name: CommandCategoryGeneral,
            Value: generalCmds,
        },
        {
            Name: CommandCategoryBetting,
            Value: bettingCmds,
        },
        {
            Name: CommandCategoryListing,
            Value: listingCmds,
        },
    }

    if isOwner {
        fields = append(fields, &dg.MessageEmbedField{
            Name: CommandCategoryAdmin,
            Value: adminCmds,
        })
    }

    addEmbeddedInteractionResponse(b.session, i, NewMsg, fields, "Hjälpsida", help)
}

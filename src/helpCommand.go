package main

import (
    "fmt"
    dg "github.com/bwmarrin/discordgo"
)

func newHelp(b *botHolder) *Help {
    h := new(Help)
    h.bot = b
    h.name = "hjälp"
    h.description = "Få hjälp med hur denna bot fungerar."
    h.category = CommandCategoryGeneral
    return h
}

func (cmd *Help) run(i *dg.InteractionCreate) {
    isOwner := getInteractUID(i) == cmd.bot.owner

    help := "Du kan */slåvad* över en match. Då slår du vad om hur du tror en match kommer sluta poängmässigt. Har du rätt vinner du poäng som kan användas till antingen **skryträtt**, eller för att */utmana* andra användare. "
    help += "När du utmanar en annan användare väljer du en utmaning och hur många poäng du satsar på ditt utfall. Vinnaren tar alla poängen.\n"
    help += "\n"
    help += "Resultaten för matcherna uppdateras lite då och då under dagen, därför kan det ta ett tag tills dess att poängen delas ut efter att en match är spelad."

    adminCmds := ""
    generalCmds := ""
    bettingCmds := ""
    listingCmds := ""

    for _, cmd := range cmd.bot.commands {
        switch cmd.category {
            case CommandCategoryAdmin:
                adminCmds += fmt.Sprintf("/%v - %v\n", cmd.name, cmd.description)
            case CommandCategoryGeneral:
                generalCmds += fmt.Sprintf("/%v - %v\n", cmd.name, cmd.description)
            case CommandCategoryBetting:
                bettingCmds += fmt.Sprintf("/%v - %v\n", cmd.name, cmd.description)
            case CommandCategoryListing:
                listingCmds += fmt.Sprintf("/%v - %v\n", cmd.name, cmd.description)
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

    addEmbeddedInteractionResponse(cmd.bot.session, i, NewMsg, fields, "Hjälpsida", help)
}

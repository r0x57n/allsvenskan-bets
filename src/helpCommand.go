package main

import (
    "fmt"
	dg "github.com/bwmarrin/discordgo"
)

// Command: hjälp
func helpCommand(s *dg.Session, i *dg.InteractionCreate, COMMANDS *[]cmd) {
	help := "Du kan */slåvad* över en match. Då slår du vad om hur du tror en match kommer sluta poängmässigt. Har du rätt vinner du poäng som kan användas till antingen **skryträtt**, eller för att */utmana* andra användare. " +
            "När du utmanar en annan användare väljer du en utmaning och hur många poäng du satsar på ditt utfall. Vinnaren tar alla poängen.\n" +
            "\n" +
            "Resultaten för matcherna uppdateras lite då och då under dagen, därför kan det ta ett tag tills dess att poängen delas ut efter att en match är spelad."

    isOwner := getInteractUID(i) == *OWNER

    adminCmds := ""
    generalCmds := ""
    bettingCmds := ""


	for _, cmd := range *COMMANDS {
        switch cmd.category {
            case "Admin":
                adminCmds += fmt.Sprintf("/%v - %v\n", cmd.name, cmd.description)
            case "Allmänt":
                generalCmds += fmt.Sprintf("/%v - %v\n", cmd.name, cmd.description)
            case "Vadslagning":
                bettingCmds += fmt.Sprintf("/%v - %v\n", cmd.name, cmd.description)
        }
	}

    fields := []*dg.MessageEmbedField {
        {
            Name: "Allmänt",
            Value: generalCmds,
        },
        {
            Name: "Vadslagning",
            Value: bettingCmds,
        },
    }

    if isOwner {
        fields = append(fields, &dg.MessageEmbedField{
            Name: "Admin",
            Value: adminCmds,
        })
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Hjälpsida", help)
}

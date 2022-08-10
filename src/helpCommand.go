package main

import (
    "fmt"
	dg "github.com/bwmarrin/discordgo"
)

// Command: hjälp
func helpCommand(s *dg.Session, i *dg.InteractionCreate, COMMANDS *[]dg.ApplicationCommand) {
	help := "Denna bot är till för att kunna slå vad om hur olika Allsvenska matcher kommer sluta.\n" +
		    "\n" +
            "Du kan */slåvad* över en match. Då slår du vad om hur du tror en match kommer sluta poängmässigt. Har du rätt vinner du poäng som kan användas till antingen **skryträtt**, eller för att */utmana* andra användare. " +
            "När du utmanar en annan användare väljer du en utmaning och hur många poäng du satsar på ditt utfall. Vinnaren tar alla poängen.\n" +
            "\n" +
            "Resultaten för matcherna uppdateras lite då och då under dagen, därför kan det ta ett tag tills dess att poängen delas ut efter att en match är spelad."

    adminOnly := map[string]int{"sammanfatta": 1, "update": 1, "delete": 1, "checkbets": 1}
    isOwner := getInteractUID(i) == *OWNER

    cmds := ""

	for _, elem := range *COMMANDS {
		if _, adminCmd := adminOnly[elem.Name]; !adminCmd || isOwner {
            if adminCmd {
                cmds += fmt.Sprintf("*/%v - %v*\n", elem.Name, elem.Description)
            } else {
                cmds += fmt.Sprintf("/%v - %v\n", elem.Name, elem.Description)
            }
		}
	}

    fields := []*dg.MessageEmbedField {
        {
            Name: "Kommandon",
            Value: cmds,
        },
    }

    addEmbeddedInteractionResponse(s, i, NewMsg, fields, "Hjälp", help)
}

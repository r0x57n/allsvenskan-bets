package main

import (
    "log"
    dg "github.com/bwmarrin/discordgo"
)

func (b *Bot) Init() {
    log.Print("Initializing...")

    // Login bot to get the active session
    s, err := dg.New("Bot " + b.token)
    if err != nil {
        log.Fatalf("Invalid bot parameters: %v", err)
    }

    b.session = s
    b.addCommands()

    s.AddHandler(func(s *dg.Session, i *dg.InteractionCreate) {
        switch i.Type {
            case dg.InteractionApplicationCommand:
                if h, ok := b.commandHandlers[i.ApplicationCommandData().Name]; ok { h(s, i) }
            case dg.InteractionMessageComponent:
                if h, ok := b.componentHandlers[i.MessageComponentData().CustomID]; ok { h(s, i) }
        }
    })

    // Handler to tell us when we logged in
    s.AddHandler(func(s *dg.Session, r *dg.Ready) {
        log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
        g, _ := s.Guild(s.State.Guilds[0].ID)
        log.Printf("In the following guild: %v", g.Name)
        b.allsvenskanGuildID = s.State.Guilds[0].ID

        if *ADD_COMMANDS {
            log.Printf("Adding commands...")
            for _, c := range b.commands {
                cmd := dg.ApplicationCommand {
                    Name: c.Name,
                    Description: c.Description,
                    Options: c.Options,
                }

                log.Printf("Adding: %v", c.Name)
                _, err := b.session.ApplicationCommandCreate(b.appID, b.guildID, &cmd)
                if err != nil {
                    log.Fatalf("Cannot create slash command %q, %v", cmd.Name, err)
                }
            }
            log.Printf("Added all commands!")
        }
    })
}

func (b *Bot) Start() {
    err := b.session.Open()
    if err != nil {
        log.Panicf("Cannot open the session: %v", err)
    }
}

func (b *Bot) Close() {
    b.session.Close()
    b.db.Close()
}

func (b *Bot) notOwnerRespond(i *dg.InteractionCreate) bool {
    if b.owner != getInteractUID(i) {
        addInteractionResponse(b.session, i, NewMsg, "Saknar behörighet.")
        return true
    }
    return false
}

func (b *Bot) messageOwner(msg string) {
    channelID, _ := b.session.UserChannelCreate(b.owner)
    b.session.ChannelMessageSend(channelID.ID, msg)
}


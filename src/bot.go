package main

import (
	"log"
	dg "github.com/bwmarrin/discordgo"
)

func (b *botHolder) Init() {
	log.Print("Initializing...")

	// Login bot to get the active session
	s, err := dg.New("Bot " + b.token)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

    b.session = s

    // Add handlers for command/component
    b.commandHandlers = map[string]func(s *dg.Session, i *dg.InteractionCreate) {
        // User commands
        "hjälp": func(s *dg.Session, i *dg.InteractionCreate)         {   b.helpCommand(i)               },
        "slåvad": func(s *dg.Session, i *dg.InteractionCreate)        {   b.betCommand(i)                },
        "ångra": func(s *dg.Session, i *dg.InteractionCreate)         {   b.regretCommand(i)             },
        "utmana": func(s *dg.Session, i *dg.InteractionCreate)        {   b.challengeCommand(i)          },
        "fegaur": func(s *dg.Session, i *dg.InteractionCreate)        {   b.chickenCommand(i)            },
        "kommande": func(s *dg.Session, i *dg.InteractionCreate)      {   b.upcomingCommand(i)           },
        "vadslagningar": func(s *dg.Session, i *dg.InteractionCreate) {   b.listBetsCommand(i)           },
        "poäng": func(s *dg.Session, i *dg.InteractionCreate)         {   b.pointsCommand(i)             },
        "inställningar": func(s *dg.Session, i *dg.InteractionCreate) {   b.settingsCommand(i)           },
        "info": func(s *dg.Session, i *dg.InteractionCreate)          {   b.infoCommand(i)               },

        // Admin commands
        "sammanfatta": func(s *dg.Session, i *dg.InteractionCreate)   {   b.summaryCommand(i)            },
        "update": func(s *dg.Session, i *dg.InteractionCreate)        {   b.updateCommand(i)             },
        "delete": func(s *dg.Session, i *dg.InteractionCreate)        {   b.deleteCommand(i)             },
        "checkbets": func(s *dg.Session, i *dg.InteractionCreate)     {   b.checkBetsCommand(i)          },
    }

    // Component handlers
    b.componentHandlers = map[string]func(s *dg.Session, i *dg.InteractionCreate) {
        "betOnSelected": func(s *dg.Session, i *dg.InteractionCreate)           {   betOnSelected(s, i)   },
        "betScoreHome": func(s *dg.Session, i *dg.InteractionCreate)            {   betScoreComponent(s, i, Home)    },
        "betScoreAway": func(s *dg.Session, i *dg.InteractionCreate)            {   betScoreComponent(s, i, Away)    },
        "challSelectWinner": func(s *dg.Session, i *dg.InteractionCreate)       {   challSelectWinner(s, i)          },
        "challSelectPoints": func(s *dg.Session, i *dg.InteractionCreate)       {   challSelectPoints(s, i)          },
        "challAcceptDiscard": func(s *dg.Session, i *dg.InteractionCreate)      {   challAcceptDiscard(s, i)         },
        "challAcceptDiscardDo": func(s *dg.Session, i *dg.InteractionCreate)    {   challAcceptDiscardDo(s, i)       },
        "settingsVisibility": func(s *dg.Session, i *dg.InteractionCreate)      {   settingsVisibility(s, i)         },
        "settingsChall": func(s *dg.Session, i *dg.InteractionCreate)           {   settingsChall(s, i)              },
        "updateCommandDo": func(s *dg.Session, i *dg.InteractionCreate)         {   b.updateCommandDo(s, i)            },
        "deleteCommandDo": func(s *dg.Session, i *dg.InteractionCreate)         {   b.deleteCommandDo(s, i)            },
        "regretSelected": func(s *dg.Session, i *dg.InteractionCreate)          {   regretSelected(s, i)             },
        "challAnswer": func(s *dg.Session, i *dg.InteractionCreate)             {   challAnswer(s, i)                },
        "chickenSelected": func(s *dg.Session, i *dg.InteractionCreate)         {   chickenSelected(s, i)            },
        "chickenAnswer": func(s *dg.Session, i *dg.InteractionCreate)           {   chickenAnswer(s, i)              },
    }

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
	})
}

func (b *botHolder) Start() {
	err := b.session.Open()
	if err != nil {
		log.Panicf("Cannot open the session: %v", err)
	}
}

func (b *botHolder) Close() {
    b.session.Close()
}

func (b *botHolder) notOwner(uid string) bool {
    if b.owner != uid { return true }
    return false
}

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
    commandHandlers := map[string]func(s *dg.Session, i *dg.InteractionCreate) {
        // User commands
        HelpCommand: func(s *dg.Session, i *dg.InteractionCreate)     {   b.commands[HelpCommand].run(i) },
        BetCommand: func(s *dg.Session, i *dg.InteractionCreate)      {   b.commands[BetCommand].run(i)  },
        RegretCommand: func(s *dg.Session, i *dg.InteractionCreate)         {   b.commands[RegretCommand].run(i) },
        ChallengeCommand: func(s *dg.Session, i *dg.InteractionCreate)        {   b.commands[ChallengeCommand].run(i)},
        ChickenCommand: func(s *dg.Session, i *dg.InteractionCreate)        {   b.commands[ChickenCommand].run(i) },
        UpcomingCommand: func(s *dg.Session, i *dg.InteractionCreate)      {   b.commands[UpcomingCommand].run(i) },
        BetsCommand: func(s *dg.Session, i *dg.InteractionCreate) {   b.commands[BetsCommand].run(i) },
        PointsCommand: func(s *dg.Session, i *dg.InteractionCreate)         {   b.commands[PointsCommand].run(i) },
        SettingsCommand: func(s *dg.Session, i *dg.InteractionCreate) {   b.commands[SettingsCommand].run(i)},
        InfoCommand: func(s *dg.Session, i *dg.InteractionCreate)          {   b.commands[InfoCommand].run(i)},

        // Admin commands
        SummaryCommand: func(s *dg.Session, i *dg.InteractionCreate)   {   b.commands[SummaryCommand].run(i)},
        UpdateCommand: func(s *dg.Session, i *dg.InteractionCreate)        {  b.commands[UpdateCommand].run(i) },
        DeleteCommand: func(s *dg.Session, i *dg.InteractionCreate)        {   b.commands[DeleteCommand].run(i)},
        CheckCommand: func(s *dg.Session, i *dg.InteractionCreate)     {   b.commands[CheckCommand].run(i)},
    }

    // Component handlers are initialized within each command
    b.componentHandlers = map[string]func(s *dg.Session, i *dg.InteractionCreate) { }

    b.commands = map[CommandName]aCommand{}
    b.commands[BetCommand] = newBet(b)
    b.commands[HelpCommand] = newHelp(b)
    b.commands[ChallengeCommand] = newChallenge(b)
    b.commands[SettingsCommand] = newSettings(b)
    b.commands[ChickenCommand] = newChicken(b)
    b.commands[RegretCommand] = newRegret(b)
    b.commands[UpcomingCommand] = newUpcoming(b)
    b.commands[BetsCommand] = newBets(b)
    b.commands[PointsCommand] = newPoints(b)
    b.commands[InfoCommand] = newInfo(b)
    b.commands[SummaryCommand] = newSummary(b)
    b.commands[UpdateCommand] = newUpdate(b)
    b.commands[DeleteCommand] = newDelete(b)
    b.commands[CheckCommand] = newCheck(b)

	s.AddHandler(func(s *dg.Session, i *dg.InteractionCreate) {
		switch i.Type {
			case dg.InteractionApplicationCommand:
				if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok { h(s, i) }
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

func (b *botHolder) addComponent(comp string, fn func(s *dg.Session, i *dg.InteractionCreate)) {
    b.componentHandlers[comp] = func(s *dg.Session, i *dg.InteractionCreate) { fn(s, i) }
}

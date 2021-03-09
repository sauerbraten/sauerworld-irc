package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/sauerbraten/pubsub"
	"github.com/sauerbraten/sauerworld-irc/config"
)

func setupDiscord(pub *pubsub.Publisher) (*discordgo.Session, func()) {
	d, err := discordgo.New("Bot " + config.Discord.Token)
	if err != nil {
		log.Fatalf("discord: error creating session: %v\n", err)
	}

	d.AddHandler(func(_ *discordgo.Session, _ *discordgo.Connect) {
		log.Println("discord: connected")
	})
	d.AddHandler(func(_ *discordgo.Session, _ *discordgo.Disconnect) {
		log.Println("discord: disconnected")
	})

	err = d.Open()
	if err != nil {
		log.Fatalf("discord: error opening session: %v\n", err)
	}

	d.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.ChannelID != config.Discord.ChannelID || m.WebhookID != "" || m.Author.ID == d.State.User.ID {
			return
		}
		pub.Publish([]byte(d2i(s, m)))
	})

	return d, func() {
		err = d.Close()
		if err != nil {
			log.Printf("discord: error closing connection: %v\n", err)
		}
		pub.Close()
	}
}

func d2i(s *discordgo.Session, m *discordgo.MessageCreate) string {
	authorName := m.Author.Username
	if m.Member.Nick != "" {
		authorName = m.Member.Nick
	}
	content, err := m.ContentWithMoreMentionsReplaced(s)
	if err != nil {
		log.Printf("error replacing mentions in Discord message: %v\n", err)
		// content is still a usable string
	}
	return fmt.Sprintf("<%s> %s", authorName, content)
}

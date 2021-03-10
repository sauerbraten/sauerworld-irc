package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	irc "github.com/fluffle/goirc/client"
	"github.com/sauerbraten/sauerworld-irc/config"
)

func setupIRC(d *discordgo.Session) (*irc.Conn, <-chan string, func()) {
	i := irc.SimpleClient(config.IRC.Nick, config.IRC.Username, config.IRC.Realname)

	var ownJoinHandler irc.Remover
	ownJoinHandler = i.HandleFunc(irc.JOIN, func(i *irc.Conn, line *irc.Line) {
		if line.Nick == i.Me().Nick {
			log.Println("irc: joined", config.IRC.Channel)
			ownJoinHandler.Remove()
		}
	})

	i.HandleFunc(irc.CONNECTED, func(i *irc.Conn, _ *irc.Line) {
		log.Println("irc: connected")
		i.Join(config.IRC.Channel)
	})

	disconnectHandler := i.HandleFunc(irc.DISCONNECTED, func(i *irc.Conn, _ *irc.Line) {
		log.Println("irc: disconnected")
		err := i.ConnectTo(config.IRC.ServerAddress)
		if err != nil {
			log.Fatalf("irc: could not reconnect to server: %s\n", err)
		}
	})

	fromIRC := make(chan string, 10)
	i.HandleFunc(irc.PRIVMSG, func(i *irc.Conn, line *irc.Line) {
		if !line.Public() || line.Target() != config.IRC.Channel || line.Nick == i.Me().Nick {
			return
		}
		fromIRC <- i2d(d, line)
	})

	err := i.ConnectTo(config.IRC.ServerAddress)
	if err != nil {
		log.Fatalf("irc: could not connect to server: %s\n", err)
	}

	return i, fromIRC, func() {
		disconnectHandler.Remove()
		ircDisconnected := make(chan struct{}, 1)
		i.HandleFunc(irc.DISCONNECTED, func(i *irc.Conn, _ *irc.Line) {
			close(ircDisconnected)
		})
		i.Quit("see you!")
		<-ircDisconnected
	}
}

var mentionPattern = regexp.MustCompile(`@[^\s]+`)

func i2d(d *discordgo.Session, l *irc.Line) string {
	content := l.Text()
	content = mentionPattern.ReplaceAllStringFunc(content, func(mention string) string {
		name := strings.TrimSpace(mention)[1:]
		if mention := name2mention(d, name); mention != "" {
			return mention
		}
		return mention
	})

	return fmt.Sprintf("**<%s>** %s", l.Nick, content)
}

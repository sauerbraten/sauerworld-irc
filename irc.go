package main

import (
	"fmt"
	"log"

	irc "github.com/fluffle/goirc/client"
	"github.com/sauerbraten/pubsub"
	"github.com/sauerbraten/sauerworld-irc/config"
)

func setupIRC(pub *pubsub.Publisher) (*irc.Conn, func()) {
	i := irc.SimpleClient(config.IRC.BotName, config.IRC.BotName, config.IRC.BotMaintainer)

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
	reconnectHandler := i.HandleFunc(irc.DISCONNECTED, func(i *irc.Conn, _ *irc.Line) {
		log.Println("irc: disconnected")
		err := i.ConnectTo(config.IRC.ServerAddress)
		if err != nil {
			log.Fatalf("irc: could not reconnect to server: %s\n", err)
		}
	})

	err := i.ConnectTo(config.IRC.ServerAddress)
	if err != nil {
		log.Fatalf("irc: could not connect to server: %s\n", err)
	}

	i.HandleFunc(irc.PRIVMSG, func(i *irc.Conn, line *irc.Line) {
		if !line.Public() || line.Target() != config.IRC.Channel || line.Nick == i.Me().Nick {
			return
		}
		pub.Publish([]byte(i2d(line)))
	})

	return i, func() {
		reconnectHandler.Remove()
		ircDisconnected := make(chan struct{}, 1)
		i.HandleFunc(irc.DISCONNECTED, func(i *irc.Conn, _ *irc.Line) {
			close(ircDisconnected)
		})
		i.Quit("see you!")
		<-ircDisconnected
		pub.Close()
	}
}

func i2d(l *irc.Line) string {
	return fmt.Sprintf("**<%s>** %s", l.Nick, l.Text())
}

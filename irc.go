package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"regexp"
	"strings"

	irc "github.com/fluffle/goirc/client"

	"github.com/sauerbraten/sauerworld-irc/config"
)

func setupIRC() (<-chan string, func()) {
	c := irc.NewConfig(config.IRC.Nick, config.IRC.Username, config.IRC.Realname)
	if config.IRC.TLS {
		c.SSL = true
		c.SSLConfig = &tls.Config{ServerName: config.IRC.ServerName}
	}
	c.Server = config.IRC.ServerName + ":" + config.IRC.ServerPort
	c.Pass = config.IRC.SASLPassword

	i = irc.Client(c)

	var ownJoinHandler irc.Remover
	ownJoinHandler = i.HandleFunc(irc.JOIN, func(_ *irc.Conn, line *irc.Line) {
		if line.Nick == i.Me().Nick {
			log.Println("irc: joined", config.IRC.Channel)
			ownJoinHandler.Remove()
		}
	})

	i.HandleFunc(irc.CONNECTED, func(_ *irc.Conn, _ *irc.Line) {
		log.Println("irc: connected")
		i.Join(config.IRC.Channel)
	})

	disconnectHandler := i.HandleFunc(irc.DISCONNECTED, func(i *irc.Conn, l *irc.Line) {
		log.Printf("irc: disconnected: %v\n", l.Raw)
		err := i.Connect()
		if err != nil {
			log.Fatalf("irc: could not reconnect to server: %s\n", err)
		}
	})

	fromIRC := make(chan string, 10)
	i.HandleFunc(irc.PRIVMSG, func(_ *irc.Conn, line *irc.Line) {
		if !line.Public() || line.Target() != config.IRC.Channel || line.Nick == i.Me().Nick {
			return
		}
		if _, ok := config.IRC.IgnoreNicks[line.Nick]; ok {
			return
		}
		fromIRC <- i2d(line)
	})

	err := i.Connect()
	if err != nil {
		log.Fatalf("irc: could not connect to server: %s\n", err)
	}

	return fromIRC, func() {
		disconnectHandler.Remove()
		ircDisconnected := make(chan struct{}, 1)
		i.HandleFunc(irc.DISCONNECTED, func(_ *irc.Conn, _ *irc.Line) {
			close(ircDisconnected)
		})
		i.Quit("see you soon!")
		<-ircDisconnected
	}
}

var mentionPattern = regexp.MustCompile(`@[^\s]+`)

func i2d(l *irc.Line) string {
	content := l.Text()
	content = mentionPattern.ReplaceAllStringFunc(content, func(mention string) string {
		name := strings.TrimSpace(mention)[1:]
		if mention := name2mention(name); mention != "" {
			return mention
		}
		return mention
	})

	return fmt.Sprintf("**<%s>** %s", l.Nick, content)
}

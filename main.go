package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sauerbraten/pubsub"
	"github.com/sauerbraten/sauerworld-irc/config"
)

func main() {
	proxy := pubsub.NewBroker()

	fromIRC, pub := proxy.Subscribe("fromirc")
	i, stopIRC := setupIRC(pub)

	fromDiscord, pub := proxy.Subscribe("fromdiscord")
	d, stopDiscord := setupDiscord(pub)

	go func() {
		for m := range fromIRC {
			_, err := d.ChannelMessageSend(config.Discord.ChannelID, string(m))
			if err != nil {
				log.Printf("discord: sending message '%s': %v\n", m, err)
			}
		}
	}()

	go func() {
		for m := range fromDiscord {
			i.Privmsg(config.IRC.Channel, string(m))
		}
	}()

	log.Println("proxying messages")

	// wait for kill signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-stop
	log.Println("received interrupt, shutting down")

	stopIRC()
	stopDiscord()
}

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sauerbraten/sauerworld-irc/config"
)

func main() {
	d, fromDiscord, stopDiscord := setupDiscord()
	i, fromIRC, stopIRC := setupIRC(d)

	go func() {
		for m := range fromIRC {
			_, err := d.ChannelMessageSend(config.Discord.ChannelID, m)
			if err != nil {
				log.Printf("discord: sending message '%s': %v\n", m, err)
			}
		}
	}()

	go func() {
		for m := range fromDiscord {
			i.Privmsg(config.IRC.Channel, m)
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

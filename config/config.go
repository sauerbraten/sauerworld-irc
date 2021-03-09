package config

import (
	"log"
	"os"
)

var (
	IRC = struct {
		ServerAddress string
		Channel       string
		BotName       string
		BotMaintainer string
	}{
		mustEnv("IRC_SERVER_ADDRESS"),
		mustEnv("IRC_CHANNEL_NAME"),
		mustEnv("IRC_BOT_NAME"),
		mustEnv("IRC_BOT_MAINTAINER"),
	}

	Discord = struct {
		Token     string
		ChannelID string
	}{
		mustEnv("DISCORD_TOKEN"),
		mustEnv("DISCORD_CHANNEL_ID"),
	}
)

func mustEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s not set\n", name)
	}
	return value
}

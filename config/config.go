package config

import (
	"log"
	"os"
)

var (
	IRC = struct {
		ServerAddress string
		Channel       string
		Nick          string
		Username      string
		Realname      string
	}{
		mustEnv("IRC_SERVER_ADDRESS"),
		mustEnv("IRC_CHANNEL_NAME"),
		mustEnv("IRC_NICK"),
		mustEnv("IRC_USERNAME"),
		mustEnv("IRC_REALNAME"),
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

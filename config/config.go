package config

import (
	"log"
	"os"
	"strconv"
)

var (
	IRC = struct {
		ServerName string
		ServerPort string
		TLS        bool
		Channel    string
		Nick       string
		Username   string
		Realname   string
	}{
		mustEnv("IRC_SERVER_NAME"),
		mustEnv("IRC_SERVER_PORT"),
		mustBool(mustEnv("IRC_TLS")),
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

func mustBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		log.Fatalf("parsing '%s' as boolean: %v\n", s, err)
	}
	return b
}

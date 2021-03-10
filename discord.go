package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sauerbraten/sauerworld-irc/config"
)

func setupDiscord() (*discordgo.Session, <-chan string, func()) {
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

	fromDiscord := make(chan string, 10)
	d.AddHandler(func(d *discordgo.Session, m *discordgo.MessageCreate) {
		if m.ChannelID != config.Discord.ChannelID || m.WebhookID != "" || m.Author.ID == d.State.User.ID {
			return
		}
		for i, line := range strings.Split(strings.TrimSpace(d2i(d, m)), "\n") {
			if i > 0 {
				line = "    " + line
			}
			fromDiscord <- line
		}
	})

	d.Identify.Intents = discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessages

	err = d.Open()
	if err != nil {
		log.Fatalf("discord: error opening session: %v\n", err)
	}

	return d, fromDiscord, func() {
		err := d.Close()
		if err != nil {
			log.Printf("discord: error closing connection: %v\n", err)
		}
	}
}

var (
	channelPattern     = regexp.MustCompile("<#[^>]+>")
	customEmojiPattern = regexp.MustCompile(`<(:[^:]+:)\d+>`)
)

func d2i(d *discordgo.Session, m *discordgo.MessageCreate) string {
	// user and role mentions

	replacements := []string{}
	for _, user := range m.Mentions {
		nick := user.Username
		member, err := getMember(d, m.GuildID, user.ID)
		if err != nil {
			log.Printf("discord: error getting member: %v\n", err)
		} else if member.Nick != "" {
			nick = member.Nick
		}
		replacements = append(replacements, "<@"+user.ID+">", "@"+nick, "<@!"+user.ID+">", "@"+nick)
	}
	for _, roleID := range m.MentionRoles {
		role, err := getRole(d, m.GuildID, roleID)
		if err != nil {
			log.Printf("discord: error getting role: %v\n", err)
			continue
		}
		if !role.Mentionable {
			continue
		}
		replacements = append(replacements, "<@&"+role.ID+">", "@"+role.Name)
	}
	content := strings.NewReplacer(replacements...).Replace(m.Content)

	// channel mentions

	content = channelPattern.ReplaceAllStringFunc(content, func(mention string) string {
		channel, err := getChannel(d, mention[2:len(mention)-1])
		if err != nil {
			log.Printf("discord: error getting channel: %v\n", err)
			return mention
		}
		return "#" + channel.Name
	})

	// custom emojis

	content = customEmojiPattern.ReplaceAllString(content, "$1")

	authorName := m.Author.Username
	if m.Member.Nick != "" {
		authorName = m.Member.Nick
	}

	attachmentURLs := []string{}
	for _, a := range m.Attachments {
		attachmentURLs = append(attachmentURLs, a.ProxyURL)
	}

	if len(attachmentURLs) > 0 {
		if len(content) > 0 {
			return fmt.Sprintf("<%s> %s %s", authorName, content, strings.Join(attachmentURLs, ", "))
		}
		return fmt.Sprintf("<%s> %s", authorName, strings.Join(attachmentURLs, ", "))
	}
	return fmt.Sprintf("<%s> %s", authorName, content)
}

func getMember(d *discordgo.Session, guildID, userID string) (*discordgo.Member, error) {
	member, err := d.State.Member(guildID, userID)
	if err == discordgo.ErrStateNotFound {
		member, err = d.GuildMember(guildID, userID)
		if err == nil {
			d.State.MemberAdd(member)
		}
	}
	return member, err
}

func getChannel(d *discordgo.Session, channelID string) (*discordgo.Channel, error) {
	channel, err := d.State.Channel(channelID)
	if err == discordgo.ErrStateNotFound {
		channel, err = d.Channel(channelID)
		if err == nil {
			d.State.ChannelAdd(channel)
		}
	}
	return channel, err
}

func getRole(d *discordgo.Session, guildID, roleID string) (*discordgo.Role, error) {
	role, err := d.State.Role(guildID, roleID)
	if err == discordgo.ErrStateNotFound {
		roles, err := d.GuildRoles(guildID)
		if err == nil {
			for _, r := range roles {
				d.State.RoleAdd(guildID, role)
				if r.ID == roleID {
					role = r
				}
			}
		}
	}
	return role, err
}

func name2mention(d *discordgo.Session, name string) string {
	for _, guild := range d.State.Guilds {
		for _, member := range guild.Members {
			if member.Nick == name {
				return "<@!" + member.User.ID + ">"
			}
			if member.User.Username == name {
				return "<@" + member.User.ID + ">"
			}
		}
		members, err := d.GuildMembers(guild.ID, "", 1000)
		if err != nil {
			continue
		}
		for _, member := range members {
			d.State.MemberAdd(member)
			if member.Nick == name {
				return "<@!" + member.User.ID + ">"
			}
			if member.User.Username == name {
				return "<@" + member.User.ID + ">"
			}
		}
	}
	return ""
}

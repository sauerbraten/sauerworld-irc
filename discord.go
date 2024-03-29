package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/sauerbraten/sauerworld-irc/config"
)

func setupDiscord() (<-chan string, func()) {
	var err error
	d, err = discordgo.New("Bot " + config.Discord.Token)
	if err != nil {
		log.Fatalf("discord: error creating session: %v\n", err)
	}

	d.AddHandler(func(s *discordgo.Session, _ *discordgo.Connect) {
		log.Println("discord: connected")
		// can't use custom status as a bot: https://github.com/discord/discord-api-docs/issues/1160#issuecomment-546549516
		err := s.UpdateGameStatus(0, fmt.Sprintf("%s on %s", config.IRC.Channel, config.IRC.ServerName))
		if err != nil {
			log.Printf("setting 'listening' activity: %v\n", err)
		}
	})

	d.AddHandler(func(_ *discordgo.Session, _ *discordgo.Disconnect) {
		log.Println("discord: disconnected")
	})

	fromDiscord := make(chan string, 10)
	d.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if m.ChannelID != config.Discord.ChannelID ||
			(m.Type != discordgo.MessageTypeDefault && m.Type != discordgo.MessageTypeReply) ||
			m.WebhookID != "" ||
			m.Author.ID == d.State.User.ID {
			return
		}

		proxyMessage(m.Message, fromDiscord)
	})

	d.Identify.Intents = discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessages

	err = d.Open()
	if err != nil {
		log.Fatalf("discord: error opening session: %v\n", err)
	}

	return fromDiscord, func() {
		err := d.Close()
		if err != nil {
			log.Printf("discord: error closing connection: %v\n", err)
		}
	}
}

func proxyMessage(m *discordgo.Message, fromDiscord chan<- string) {
	if m.Type == discordgo.MessageTypeReply {
		m.ReferencedMessage.GuildID = m.GuildID
		inReplyTo := d2i(m.ReferencedMessage)
		fromDiscord <- fmt.Sprintf("<%s> %s", author(m), inReplyTo)
	}

	for i, line := range strings.Split(strings.TrimSpace(d2i(m)), "\n") {
		if i > 0 {
			line = "    " + line
		}
		fromDiscord <- line
	}
}

func author(m *discordgo.Message) string {
	if m.Member != nil && m.Member.Nick != "" {
		return m.Member.Nick
	}
	author, err := getMember(m.GuildID, m.Author.ID)
	if err != nil {
		log.Printf("resolving message author name: %v\n", err)
		log.Printf("message: %+v\n", m)
		log.Printf("author:  %+v\n", m.Author)
	} else if author.Nick != "" {
		return author.Nick
	}
	return m.Author.Username
}

var (
	channelPattern     = regexp.MustCompile("<#[^>]+>")
	customEmojiPattern = regexp.MustCompile(`<(:[^:]+:)\d+>`)
)

func d2i(m *discordgo.Message) string {
	// user and role mentions
	replacements := []string{}
	for _, user := range m.Mentions {
		nick := user.Username
		member, err := getMember(m.GuildID, user.ID)
		if err != nil {
			log.Printf("discord: error getting member: %v\n", err)
		} else if member.Nick != "" {
			nick = member.Nick
		}
		replacements = append(replacements, "<@"+user.ID+">", "@"+nick, "<@!"+user.ID+">", "@"+nick)
	}
	for _, roleID := range m.MentionRoles {
		role, err := getRole(m.GuildID, roleID)
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
		channel, err := getChannel(mention[2 : len(mention)-1])
		if err != nil {
			log.Printf("discord: error getting channel: %v\n", err)
			return mention
		}
		return "#" + channel.Name
	})

	// custom emojis
	content = customEmojiPattern.ReplaceAllString(content, "$1")

	// attachments (files, e.g. images)
	attachmentURLs := []string{}
	for _, a := range m.Attachments {
		attachmentURLs = append(attachmentURLs, a.URL)
	}

	authorName := ""
	if m.Author.ID == d.State.User.ID {
		// we're formatting one of our own message (for example, as context for
		// a reply to something someone on IRC said), so we don't prepend our
		// own nick and instead rely on the IRC nick being part of the message
	} else {
		authorName = fmt.Sprintf("<%s> ", author(m))
	}

	if len(attachmentURLs) > 0 {
		if len(content) > 0 {
			return fmt.Sprintf("%s%s %s", authorName, content, strings.Join(attachmentURLs, " "))
		}
		return fmt.Sprintf("%s%s", authorName, strings.Join(attachmentURLs, " "))
	}
	return fmt.Sprintf("%s%s", authorName, content)
}

func getMember(guildID, userID string) (*discordgo.Member, error) {
	member, err := d.State.Member(guildID, userID)
	if err == discordgo.ErrStateNotFound {
		member, err = d.GuildMember(guildID, userID)
		if err == nil {
			d.State.MemberAdd(member)
		}
	}
	return member, err
}

func getChannel(channelID string) (*discordgo.Channel, error) {
	channel, err := d.State.Channel(channelID)
	if err == discordgo.ErrStateNotFound {
		channel, err = d.Channel(channelID)
		if err == nil {
			d.State.ChannelAdd(channel)
		}
	}
	return channel, err
}

func getRole(guildID, roleID string) (*discordgo.Role, error) {
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

func name2mention(name string) string {
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

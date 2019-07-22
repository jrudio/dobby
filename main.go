package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const (
	// keyword is the trigger word for our program to listen to
	keyword = "dobby"
)

var (
	keywordLen  = 0
	commandList commands
	isVerbose   bool
	version     string
	versionFlag *bool
)

type commands interface {
	execute(channelID, cmd string, args ...string)
	isValid(cmd string) bool
	showHelp(channelID string)
	showError(channelID string, msg string)
	addCommand(cmd string, fn func(channelID string, args ...string))
}

type serviceCredentials struct {
	DiscordToken string `toml:"discordToken"`
	PlexToken    string
}

type clients struct {
	// TODO: maybe add discord here as well?
	// radarr radarr.Client
	// sonarr *sonarr.Sonarr
}

func checkErrAndExit(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {

	credentials, err := getCredentials()

	if err != nil {
		// most likely errTokenRequired error because user did not pass info via flags
		// try secrets.toml
		credentials, err = getCredentialsTOML("./secrets.toml")

		if err != nil {
			fmt.Printf("need credentials: %v\n", err)
			os.Exit(1)
		}
	}

	if keyword == "" {
		fmt.Println("a keyword (or trigger) is required for dobby to work")
		os.Exit(1)
	}

	services, err := initializeClients(credentials)

	checkErrAndExit(err)

	discord, err := discordgo.New("Bot " + credentials.DiscordToken)

	checkErrAndExit(err)

	// get keyword length
	keywordLen = len(keyword)

	commandList := newDiscord(discord)

	commandList = addCommands(commandList, services)

	discord.AddHandler(onMsgCreate(commandList))

	err = discord.Open()

	checkErrAndExit(err)

	defer discord.Close()

	fmt.Println("bot is listening...")

	ctrlC := make(chan os.Signal, 1)

	signal.Notify(ctrlC, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	<-ctrlC
}

func onMsgCreate(commandList commands) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		if isVerbose {
			fmt.Println(m.Content)
		}

		messageLen := len(m.Content)

		if messageLen < keywordLen {
			return
		}

		// our keyword was not triggered -- ignore
		if keyword != m.Content[:keywordLen] {
			return
		}

		// user triggered keyword so lets see what subcommand was requested
		if messageLen > keywordLen {
			// user has a subcommand

			args := strings.Split(m.Content, " ")
			argCount := len(args)

			// remove the keyword
			args = args[1:argCount]

			argCount--

			subcommand := args[0]

			if !commandList.isValid(subcommand) {
				// let user know that command wasn't valid
				commandList.showError(m.ChannelID, "invalid command")
				return
			}

			// remove the subcommand
			args = args[1:argCount]

			commandList.execute(m.ChannelID, subcommand, args...)
		} else {
			// it's only the keyword so return a list of subcommands
			commandList.showHelp(m.ChannelID)
		}

		// TODO: maybe keep track of user and their subsequent commands
		// so multiple users don't mess each other up

		// fmt.Println(m.Content)
	}
}

func addCommands(commandList d, services clients) d {

	// clear deletes messages in a channel -- user can delete x messages
	commandList.addCommand("clear", clearMessages(commandList, services))

	return commandList
}

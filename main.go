package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jrudio/go-plex-client"
)

const (
	// keyword is the trigger word for our program to listen to
	keyword         = "dobby"
	secretsFilepath = "./secrets.toml"
)

var (
	keywordLen            = 0
	commandList           commands
	isVerbose             bool
	isPlexTokenAuthorized bool
	isRequestingPlexPIN   bool
	version               string
	versionFlag           *bool
	plexPIN               chan plex.PinResponse
)

type commands interface {
	execute(channelID, cmd string, args ...string)
	isValid(cmd string) bool
	showHelp(channelID string)
	showError(channelID string, msg string)
	addCommand(cmd string, fn ...func(channelID string, args ...string) bool)
}

type serviceCredentials struct {
	DiscordToken string          `toml:"discordToken"`
	Plex         plexCredentials `toml:"plex"`
}

type plexCredentials struct {
	Token     string
	Host      string
	machineID string
}

type clients struct {
	plex *plex.Plex
	lock sync.Mutex
}

func (c *clients) setPlexRequestTimeout(timeout int) {
	// in seconds
	c.lock.Lock()
	c.plex.HTTPClient.Timeout = time.Duration(timeout) * time.Second
	c.lock.Unlock()
}

func (c *clients) setPlexClientID(clientID string) {
	c.lock.Lock()
	c.plex.ClientIdentifier = clientID
	c.plex.Headers.ClientIdentifier = clientID
	c.lock.Unlock()
}

func (c *clients) setPlexToken(authToken string) {
	c.lock.Lock()
	c.plex.Token = authToken
	c.lock.Unlock()
}

func (c *clients) setPlexHost(host string) {
	c.lock.Lock()
	c.plex.URL = host
	c.lock.Unlock()
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
		credentials, err = getCredentialsTOML(secretsFilepath)

		if err != nil {
			fmt.Printf("need credentials: %v\n", err)
			os.Exit(1)
		}
	}

	if keyword == "" {
		fmt.Println("a keyword (or trigger) is required for dobby to work")
		os.Exit(1)
	}

	services := clients{
		plex: &plex.Plex{},
		lock: sync.Mutex{},
	}

	plexClientID := "Dobby (discord bot)" + version

	if credentials.Plex.Token != "" {

		// initialize plex client
		services.plex, err = plex.New("", credentials.Plex.Token)

		if err != nil {
			fmt.Printf("failed to initialize plex client: %v\n", err)
			return
		}

		// change plex client information to match Dobby
		services.setPlexClientID(plexClientID)

		// pick a server and test the auth token
		serverInfo, err := services.plex.GetServersInfo()

		if err != nil {
			fmt.Printf("plex.GetServersInfo() - failed testing auth token: %v\n", err)
			return
		}

		// TODO: refactor -- too many if-statements!

		if serverInfo.Size > 0 {
			plexServer := serverInfo.Server[0]
			plexHost := plexServer.Scheme + "://" + plexServer.Address + ":" + plexServer.Port

			services.setPlexHost(plexHost)

			// check if plex auth token is valid
			isOK, err := services.plex.Test()

			if err != nil {
				fmt.Printf("plex.Test() - auth test failed: %v\n", err)
			}

			if !isOK {
				fmt.Println("we are not authorized. prompt to authorize plex PIN")
			} else {
				isPlexTokenAuthorized = true
			}
		} else {
			fmt.Println("we are not authorized. prompt to authorize plex PIN")
		}

	}

	if services.plex.Headers.ClientIdentifier != "Dobby (discord bot)"+version || services.plex.ClientIdentifier != "Dobby (discord bot)"+version {
		services.setPlexClientID(plexClientID)
	}

	services.lock = sync.Mutex{}

	services.setPlexRequestTimeout(10)

	plexPIN = make(chan plex.PinResponse)

	// connect to Discord server
	discord, err := discordgo.New("Bot " + credentials.DiscordToken)

	checkErrAndExit(err)

	// get keyword length
	keywordLen = len(keyword)

	commandList := newDiscord(discord)

	commandList = addCommands(commandList, &services)

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

func addCommands(commandList d, services *clients) d {

	// clear deletes messages in a channel -- user can delete x messages
	commandList.addCommand("clear", clearMessages(commandList, services))

	// plex-specific commands
	commandList.addCommand("invite", displayPlexPIN(commandList, services), invite(commandList, services))

	return commandList
}

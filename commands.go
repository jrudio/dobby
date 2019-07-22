package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

type d struct {
	cmds    map[string]func(channelID string, args ...string)
	discord *discordgo.Session
}

func newDiscord(session *discordgo.Session) d {
	return d{
		cmds:    map[string]func(channelID string, args ...string){},
		discord: session,
	}
}

func (discord d) addCommand(cmd string, fn func(channelID string, args ...string)) {
	discord.cmds[cmd] = fn
}

func (discord d) execute(channelID, cmd string, args ...string) {
	if fn, ok := discord.cmds[cmd]; ok {
		fn(channelID, args...)
	} else {
		if isVerbose {
			fmt.Printf("invalid command: %s\n", cmd)
		}
	}
}

func (discord d) isValid(cmd string) bool {
	_, ok := discord.cmds[cmd]

	return ok
}

func (discord d) showHelp(channelID string) {
	msg := "Here is a list of available commands: \n"

	for key := range discord.cmds {
		msg += "`" + key + "`\n"
	}

	_, err := discord.discord.ChannelMessageSend(channelID, msg)

	if err != nil {
		fmt.Printf("failed to send command list to channel %s: %v\n",
			channelID,
			err)
	}
}

func (discord d) getCommands() []string {
	cmdsLen := len(discord.cmds)

	cmds := make([]string, cmdsLen)

	i := 0

	for commandName := range discord.cmds {
		cmds[i] = commandName

		if i++; i > cmdsLen {
			break
		}
	}

	return cmds
}

func (discord d) showError(channelID, msg string) {
	_, err := discord.discord.ChannelMessageSend(channelID, msg)

	if err != nil && isVerbose {
		fmt.Printf("send message failed: %v", err)
	}
}

func clearMessages(commandList d, services clients) func(channelID string, args ...string) {
	return func(channelID string, args ...string) {
		argCount := len(args)
		messageLimit := 0

		if argCount > 0 {
			// make sure arg is an int
			limit, err := strconv.Atoi(args[0])

			if err != nil {
				fmt.Printf("%v - clear command - channel id %s - failed because arg: %v\n",
					time.Now().String(),
					channelID,
					err)

				return
			}

			messageLimit = limit
		}

		messages, err := commandList.discord.ChannelMessages(channelID, messageLimit, "", "", "")

		if err != nil {
			fmt.Printf("failed to retrieve message ids: %v\n", err)
			return
		}

		messageIDs := make([]string, len(messages))

		for i, message := range messages {
			messageIDs[i] = message.ID
		}

		if err := commandList.discord.ChannelMessagesBulkDelete(channelID, messageIDs); err != nil {
			fmt.Printf("failed to delete messages: %v\n", err)
			commandList.showError(channelID, err.Error())
		}
	}
}

// func showLibrary(commandList d, services clients) func(channelID string, args ...string) {
// 	return func(channelID string, args ...string) {
// 		// command: library <movie|show> [missing, downloaded, or missing] <page-number>
// 		// page number is optional -- w/o page number we'll show the first page of results
// 		//
// 		// examples:
// 		// library movie
// 		// library movie 3
// 		// library movie missing
// 		// library movie missing 3
// 		// library movie downloaded 6

// 		argCount := len(args)

// 		if argCount < 1 {
// 			commandList.showError(channelID, "need arg `movie|show`")
// 			return
// 		}

// 		mediaType := args[0]

// 		args = args[1:argCount]
// 		argCount--

// 		// args should be: "", "1" (page), "monitored" (a filter type), "monitored 2" (a filter type + page number)

// 		page := "1"
// 		pageSize := "40"

// 		switch mediaType {
// 		case "movie":
// 			options := radarr.GetMovieOptions{
// 				Page:     page,
// 				PageSize: pageSize,
// 				SortKey:  "sortTitle",
// 				SortDir:  "asc",
// 			}

// 			if argCount > 0 {
// 				// check for a page number
// 				// if successful there's no filter
// 				if _, err := strconv.Atoi(args[0]); err != nil {
// 					// we could not convert so we most likely have a filter
// 					filter := args[0]
// 					filterValue := "true"

// 					switch filter {
// 					case "monitored":
// 						filter = "monitored"
// 					case "downloaded":
// 						filter = "downloaded"
// 					case "missing":
// 						filter = "downloaded"
// 						filterValue = "false"
// 					case "released":
// 						filter = "status"
// 						filterValue = "released"
// 					case "announced":
// 						filter = "status"
// 						filterValue = "announced"
// 					case "cinemas":
// 						filter = "status"
// 						filterValue = "inCinemas"
// 					default:
// 						commandList.showError(channelID, fmt.Sprintf("unknown filter `%s` for command `library movie`", filter))
// 						return
// 					}

// 					options.FilterKey = filter
// 					options.FilterValue = filterValue
// 					options.FilterType = "equal"

// 					// check for page number
// 					if argCount > 1 {
// 						if _, err := strconv.Atoi(args[1]); err == nil {
// 							page = args[1]
// 							options.Page = page
// 						}
// 					}

// 				} else {
// 					// we converted the argument to a number
// 					// so we have a page number
// 					page = args[0]
// 					options.Page = page
// 				}
// 			}

// 			movies, err := services.radarr.GetMovies(options)

// 			if err != nil {
// 				output := fmt.Sprintf("fetch movies from radarr failed: %v", err)

// 				commandList.showError(channelID, output)
// 				logPrint(channelID, output)
// 				return
// 			}

// 			movieCount := len(movies)
// 			// the output preface has 35 chars
// 			output := fmt.Sprintf("showing %d movies on page %s:\n\n",
// 				movieCount, page)
// 			titleLen := 0
// 			yearLen := 0

// 			// no movies but there is a page argument
// 			if movieCount < 1 && page != "" {
// 				output += "uh oh! try going back a page!"
// 			} else if movieCount < 1 && page == "" {
// 				output += "add some movies to your library! :smile:"
// 			}

// 			for _, movie := range movies {
// 				yearStr := strconv.Itoa(movie.Year)

// 				// '<title> (2000) - downloaded\n'
// 				//  <-x-->|<------26 chars ---->|
// 				//
// 				// x == title; average char length is 14
// 				// if not downloaded subtract 13 chars
// 				// if downloaded total would be 82
// 				// not downloaded total is 69 for each movie
// 				//
// 				// we can average ~50 movies before we need to go to the next page
// 				output += movie.Title + " (" + yearStr + ") "

// 				if movie.Downloaded {
// 					output += " - `downloaded`"
// 				}

// 				output += "\n"

// 				titleLen += len(movie.Title)
// 				yearLen += len(yearStr)
// 			}

// 			if isVerbose {
// 				// get the average movie + year length to determine how many movies we can show in discord
// 				// without going over the 2000 char limit
// 				fmt.Printf("movie count: %d\n", movieCount)
// 				fmt.Printf("\ttotal title length: %d\n\ttotal year length: %d\n", titleLen, yearLen)

// 				averageTitleLen := 0
// 				averageYearLen := 0

// 				if movieCount != 0 {
// 					averageTitleLen = titleLen / movieCount
// 					averageYearLen = yearLen / movieCount
// 				}

// 				fmt.Printf("\taverage title length: %d\n\taverage year length: %d\n", averageTitleLen, averageYearLen)
// 				fmt.Printf("\ttotal message length: %d\n", len(output))
// 			}

// 			if _, err := commandList.discord.ChannelMessageSend(channelID, output); err != nil {
// 				fmt.Printf("message sent to discord failed: %v\n", err)
// 				commandList.discord.ChannelMessageSend(channelID, fmt.Sprintf("could not reply back: %v", err))
// 			}
// 		case "show":
// 			output := "`library show` not implemented"
// 			if _, err := commandList.discord.ChannelMessageSend(channelID, output); err != nil {
// 				fmt.Printf("message sent to discord failed: %v\n", err)
// 				commandList.discord.ChannelMessageSend(channelID, fmt.Sprintf("could not reply back: %v", err))
// 			}
// 		default:
// 			output := "unknown command"

// 			logPrint(channelID, output)
// 		}
// 	}
// }

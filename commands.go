package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jrudio/go-plex-client"
)

type plexCommands struct {
	hooks []func(channelID string, args ...string) bool
}

type d struct {
	cmds    map[string][]func(channelID string, args ...string) bool
	discord *discordgo.Session
}

func newDiscord(session *discordgo.Session) d {
	return d{
		cmds:    map[string][]func(channelID string, args ...string) bool{},
		discord: session,
	}
}

func (discord d) addCommand(cmd string, fn ...func(channelID string, args ...string) bool) {
	discord.cmds[cmd] = fn
}

func (discord d) execute(channelID, cmd string, args ...string) {
	if functions, ok := discord.cmds[cmd]; ok {
		for _, fn := range functions {
			if _ok := fn(channelID, args...); !_ok {
				// stop subsequent commands if current function returns false
				break
			}
		}
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

func clearMessages(commandList d, services *clients) func(channelID string, args ...string) bool {
	return func(channelID string, args ...string) bool {
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

				return true
			}

			messageLimit = limit
		}

		messages, err := commandList.discord.ChannelMessages(channelID, messageLimit, "", "", "")

		if err != nil {
			fmt.Printf("failed to retrieve message ids: %v\n", err)
			return true
		}

		messageIDs := make([]string, len(messages))

		for i, message := range messages {
			messageIDs[i] = message.ID
		}

		if err := commandList.discord.ChannelMessagesBulkDelete(channelID, messageIDs); err != nil {
			fmt.Printf("failed to delete messages: %v\n", err)
			commandList.showError(channelID, err.Error())
		}

		return true
	}
}

func displayPlexPIN(commandList d, services *clients) func(channelID string, args ...string) bool {
	return func(channelID string, args ...string) bool {
		if isPlexTokenAuthorized {
			if isVerbose {
				fmt.Println("displayPlexPIN() - dobby is already authorized")
			}
			return true
		}

		if isRequestingPlexPIN {
			return true
		}

		message := "Dobby is not authorized to access your Plex Media Server\n"

		isRequestingPlexPIN = true

		requestHeaders := services.plex.Headers

		resp, err := plex.RequestPIN(requestHeaders)

		if err != nil {
			isRequestingPlexPIN = false
			return false
		}

		message += fmt.Sprintf("Plex PIN: `%s`\nPlease go to https://plex.tv/link and link your account using the code above", resp.Code)

		commandList.discord.ChannelMessageSend(channelID, message)

		checkPlexPIN(resp, func(plexAuthToken string) {
			// when we are authorized

			services.setPlexToken(plexAuthToken)

			message = "Successfully linked Dobby! :D"

			commandList.discord.ChannelMessageSend(channelID, message)

			// persist plex auth token
			creds, err := getCredentialsTOML(secretsFilepath)

			if err != nil {
				fmt.Printf("checkPlexPIN() - error getting credentials: %v\n", err)
				message = "`internal error - could not save plex authorization token`"
				commandList.discord.ChannelMessageSend(channelID, message)
				isRequestingPlexPIN = false
				return
			}

			creds.PlexToken = plexAuthToken

			if err := saveCredentials(creds, secretsFilepath); err != nil {
				fmt.Printf("checkPlexPIN() - saveCredentials failed: %v\n", err)
				message = "`internal error - could not save plex authorization token`"
				commandList.discord.ChannelMessageSend(channelID, message)
				isRequestingPlexPIN = false
				return
			}

			if isVerbose {
				fmt.Println("saved plex auth token to file")
			}

			isRequestingPlexPIN = false
		}, func(errMessage string) {
			// when we encounter an error
			message = fmt.Sprintf("we have encountered an error :frowning2: :\n%v", errMessage)

			commandList.discord.ChannelMessageSend(channelID, message)

			isRequestingPlexPIN = false
		})

		return false
	}
}

// checkPlexPIN is a loop to check if we are authorized to a Plex server
func checkPlexPIN(_plexPIN plex.PinResponse, onSuccess func(plexAuthToken string), onError func(errMessage string)) {
	// TODO: check expiration
	fmt.Println("checkPlexPIN()")
	i := 0
	for {
		if _plexPIN.Code == "" {
			fmt.Println("checkPlexPIN() invalid or no plex pin was passed")
			onError("internal error: checkPlexPIN() invalid or no plex pin was passed")
			return
		}

		fmt.Println("checkPlexPIN() - loop", i)

		// end loop when we are authorized
		resp, err := plex.CheckPIN(_plexPIN.ID, _plexPIN.ClientIdentifier)
		i++

		if err != nil && err.Error() == "pin is not authorized yet" {
			// not authorized keep checking
			fmt.Println("not authorized yet -- sleeping for 1 second")
			time.Sleep(1 * time.Second)
			continue
		} else if err != nil {
			onError(err.Error())
			return
		}

		// we are authorized!
		onSuccess(resp.AuthToken)

		return
	}
}

// invite invite a plex user to your Plex Media Server
func invite(commandList d, services *clients) func(channelID string, args ...string) bool {
	return func(channelID string, args ...string) bool {
		if !isPlexTokenAuthorized {
			fmt.Println("invite() - dobby is not authorized")
			commandList.discord.ChannelMessageSend(channelID, "dobby is not authorized to send invites!")
			return false
		}

		commandList.discord.ChannelMessageSend(channelID, "inviting user to our Plex Media Server")

		return true
	}
}

// search search media for on your Plex Media Server
// func search(commandList d, services *clients) func(channelID string, args ...string) bool {
// 	return func(channelID string, args ...string) bool {
// 		commandList.discord.ChannelMessageSend(channelID, "inviting user to our Plex Media Server")

// 		return true
// 	}
// }

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

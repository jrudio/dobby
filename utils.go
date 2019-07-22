package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

const errDiscordTokenRequired = "a discord token is required"

// utils.go holds network utils and function helpers

func get(query string) (*http.Response, error) {
	client := http.Client{
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequest("GET", query, nil)

	if err != nil {
		return &http.Response{}, err
	}

	return client.Do(req)
}

func post(query string, body []byte) (*http.Response, error) {
	client := http.Client{
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequest("POST", query, bytes.NewBuffer(body))

	if err != nil {
		return &http.Response{}, err
	}

	req.Header.Set("Content-type", "application/json")

	return client.Do(req)
}

func encodeURL(str string) (string, error) {
	u, err := url.Parse(str)

	if err != nil {
		return "", err
	}

	return u.String(), nil
}

// getCredentials grabs apikeys and auth tokens via flags or environment vars
// prioritizes flags
func getCredentials() (serviceCredentials, error) {
	// TODO: implement environment vars

	credentials := serviceCredentials{}

	flag.StringVar(&credentials.DiscordToken, "discord-token", "", "token used for bot authentication")
	flag.BoolVar(&isVerbose, "verbose", false, "output more inforation")
	versionFlag = flag.Bool("version", false, "get program version")

	flag.Parse()

	// check for version flag
	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	if credentials.DiscordToken == "" {
		return credentials, errors.New(errDiscordTokenRequired)
	}

	return credentials, nil
}

// getCredentialsTOML grabs apikeys and auth tokens via .toml file
func getCredentialsTOML(path string) (serviceCredentials, error) {
	credentials := serviceCredentials{}

	fileBytes, err := ioutil.ReadFile(path)

	if err != nil {
		return credentials, err
	}

	if err := toml.Unmarshal(fileBytes, &credentials); err != nil {
		return credentials, err
	}

	if credentials.DiscordToken == "" {
		return credentials, errors.New(errDiscordTokenRequired)
	}

	return credentials, nil
}

func initializeClients(credentials serviceCredentials) (clients, error) {
	services := clients{}

	// init plex client

	return services, nil
}

func logPrint(chanID, message string) {
	fmt.Printf("%s - channel id: %s - %s\n", time.Now().String(), chanID, message)
}

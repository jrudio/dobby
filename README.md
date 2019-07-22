Dobby
===

Use Discord to manage your Plex Media Server

Commands:

- `add-friend` invite a plex user to your plex media server

Install
===

Some features won't work if Dobby is not installed on your Plex Media Server

- Install the latest Go compiler
- clone this project
- run `go mod init` in project folder
- run `go build -o dobby`
- run `./dobby`


To get a discord token go to `https://discordapp.com/developers/applications/me` 
- click `New App`
- fill out required information
- click save
- click on the side tab that says `Bot`
- copy `https://discordapp.com/api/oauth2/authorize?client_id=<client-id>&scope=bot&permissions=10240` and change the `client-id` to your client id in the Discord developer portal
- go to url
- authorize bot to access your discord server
- go back to `https://discordapp.com/developers/applications/me` 
- copy `token` and put in `secrets.toml`

Docker
===

Work In Progress

<!-- Simply pull the docker image from docker hub

`docker pull jrudio/shart`

then

- run `docker run -d jrudio/shart -token abc123 -radarr-url http://192.168.1.15:7878 -sonarr-url http://192.168.1.15:8989 -radarr-key abc123 -sonarr-key abc123`

Build Image Yourself (BIY)

- clone this repo onto target machine
- make sure you're in the repo directory
- run `docker build -t jrudio/shart .`
- run `docker run -d jrudio/shart -token abc123 -radarr-url http://192.168.1.15:7878 -sonarr-url http://192.168.1.15:8989 -radarr-key abc123 -sonarr-key abc123`

OPTIONAL: 

- use [docker-compose](https://docs.docker.com/compose/install) for an easy startup
- edit `docker-compose.yml` to match your discord/sonarr/radarr url and api key
- run `docker-compose up -d shart` -->

Usage
===

This bot will respond to the trigger word `dobby`

We need to link Dobby to our Plex Media Server

On first run Dobby will give you a Plex PIN to authorize. So you will need go to `plex.tv/activate` and link Dobby to your server

Work In Progress...


Develop
===

Build a binary with versioning

`go build -i -v -ldflags="-X main.version=$(git describe --always --long --dirty)" -o shart`

# substitute-bot-go

[![CircleCI](https://circleci.com/gh/anirbanmu/substitute-bot-go.svg?style=shield)](https://circleci.com/gh/anirbanmu/substitute-bot-go)

A [Reddit](https://www.reddit.com/) bot that provides the ability to search and replace parent comments. Supports Ruby regular expression syntax. This is a port of the original Ruby [substitute-bot](https://github.com/anirbanmu/substitute-bot) into Go.

## Compatibility
This project uses Go modules so please use a version of Go (>= 1.11) that supports them. This has been developed and tested with Go 1.13.1.

## Setup
- Clone this repo
- The following environment variables are required to run:
  - `SUBSTITUTE_BOT_CLIENT_ID=<YOUR_REDDIT_CLIENT_ID>`
  - `SUBSTITUTE_BOT_CLIENT_SECRET=<YOUR_REDDIT_CLIENT_SECRET>`
  - `SUBSTITUTE_BOT_USERNAME=<YOUR_REDDIT_USERNAME>`
  - `SUBSTITUTE_BOT_PASSWORD=<YOUR_REDDIT_PASSWORD>`
  - `SUBSTITUTE_BOT_USER_AGENT=<USER_AGENT_TO_USE_WITH_REDDIT_API_CALLS>`
- The following environment variables are optional:
  - `SUBSTITUTE_BOT_PORT=<PORT_NUMBER_FOR_WEB_FRONTEND>` (only used by web frontend; defaults to 3000)
- To run the bot: `go run cmd/bot/main.go`
- To run the web frontend that shows recent replies: `go run cmd/bot/main.go cmd/bot/index.html.go cmd/bot/style.scss.go`

## Testing
- Some of the tests utilize [Gingko/Gomega](https://onsi.github.io/ginkgo/)
- `go test --cover --short ./...`

## Live

You can check out the live running web frontend @ https://substitute-bot.electrostat.xyz

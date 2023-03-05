package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/anirbanmu/substitute-bot-go/pkg/persistence"
	"github.com/ugorji/go/codec"
)

func getStyleHandler() (func(http.ResponseWriter, *http.Request), error) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		fmt.Fprintf(w, styleCss)
	}, nil
}

type replyFetcher interface {
	FetchReply(count int64) ([]persistence.Reply, error)
}

func getIndexHandler(botUsername string, fetcher replyFetcher) func(http.ResponseWriter, *http.Request) {
	t := template.Must(template.New("index").Parse(indexTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		replies, err := fetcher.FetchReply(50)
		if err != nil {
			log.Println("error fetching replies:", err)
			http.Error(w, "Something went wrong", 500)
			return
		}

		args := struct {
			BotUsername string
			Replies     []persistence.Reply
		}{botUsername, replies}

		t.Execute(w, args)
	}
}

func main() {
	botUsername, ok := os.LookupEnv("SUBSTITUTE_BOT_USERNAME")
	if !ok {
		log.Panic("environment variable SUBSTITUTE_BOT_USERNAME is required to be set")
	}

	port, ok := os.LookupEnv("SUBSTITUTE_BOT_PORT")
	if !ok {
		port = ":3000"
	} else {
		port = ":" + port
	}

	styleHandler, err := getStyleHandler()
	if err != nil {
		log.Panicf("unable to get style handler: %s", err)
	}

	store, err := persistence.NewStore(nil, &codec.CborHandle{}, nil, nil)
	if err != nil {
		log.Panicf("unable to get persistence.DefaultStore: %s", err)
	}

	http.HandleFunc("/stylesheets/style.css", styleHandler)
	http.HandleFunc("/", getIndexHandler(botUsername, store))
	http.ListenAndServe(port, nil)
}

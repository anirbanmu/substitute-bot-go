package main

import (
	"bytes"
	"fmt"
	"github.com/anirbanmu/substitute-bot-go/pkg/replystorage"
	"github.com/ugorji/go/codec"
	libsass "github.com/wellington/go-libsass"
	"html/template"
	"log"
	"net/http"
	"os"
)

func compileStyle() (*string, error) {
	var buf bytes.Buffer

	comp, err := libsass.New(&buf, bytes.NewBufferString(styleScss))
	if err != nil {
		return nil, err
	}

	if err := comp.Run(); err != nil {
		return nil, err
	}

	str := buf.String()
	return &str, nil
}

func GetStyleHandler() (func(http.ResponseWriter, *http.Request), error) {
	css, err := compileStyle()
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		fmt.Fprintf(w, *css)
	}, nil
}

type ReplyFetcher interface {
	Fetch(count int64) ([]replystorage.Reply, error)
}

func GetIndexHandler(botUsername string, fetcher ReplyFetcher) func(http.ResponseWriter, *http.Request) {
	t := template.Must(template.New("index").Parse(indexTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		replies, err := fetcher.Fetch(50)
		if err != nil {
			log.Println("error fetching replies:", err)
			http.Error(w, "Something went wrong", 500)
			return
		}

		args := struct {
			BotUsername string
			Replies     []replystorage.Reply
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

	styleHandler, err := GetStyleHandler()
	if err != nil {
		log.Panicf("unable to get style handler: %s", err)
	}

	store, err := replystorage.NewStore(nil, &codec.CborHandle{})
	if err != nil {
		log.Panicf("unable to get replystorage.DefaultStore: %s", err)
	}

	http.HandleFunc("/stylesheets/style.css", styleHandler)
	http.HandleFunc("/", GetIndexHandler(botUsername, store))
	http.ListenAndServe(port, nil)
}

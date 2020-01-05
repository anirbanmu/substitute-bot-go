package main

import (
	"encoding/json"
	"fmt"
	"github.com/anirbanmu/substitute-bot-go/pkg/persistence"
	"github.com/anirbanmu/substitute-bot-go/pkg/reddit"
	"github.com/anirbanmu/substitute-bot-go/pkg/sse"
	"github.com/anirbanmu/substitute-bot-go/pkg/substitution"
	"github.com/ugorji/go/codec"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type atomicCounter struct{ c uint64 }

func (a *atomicCounter) incr()         { atomic.AddUint64(&a.c, 1) }
func (a *atomicCounter) count() uint64 { return atomic.LoadUint64(&a.c) }

func constructStoredReplyFromPosted(requester string, posted reddit.Comment) persistence.Reply {
	return persistence.Reply{
		Author:         posted.Author,
		AuthorFullname: posted.Author,
		Body:           posted.Body,
		BodyHTML:       posted.BodyHTML,
		CreatedUtc:     int64(posted.CreatedUtc),
		ID:             posted.ID,
		Name:           posted.Name,
		ParentID:       posted.ParentID,
		Permalink:      posted.Permalink,
		Requester:      requester,
	}
}

func processComment(botUser string, comment reddit.Comment, api *reddit.API, store *persistence.Store) {
	if comment.IsDeleted() || comment.Author == botUser || !reddit.IsFullnameComment(comment.ParentID) {
		return
	}

	cmd, err := substitution.ParseSubstitutionCommand(comment.Body)
	if err != nil {
		return
	}

	if len(cmd.ReplaceWith) > 0 {
		cmd.ReplaceWith = "**" + cmd.ReplaceWith + "**"
	}

	parent, err := api.GetComment(comment.ParentID)
	if err != nil || parent.IsDeleted() || parent.Author == botUser {
		return
	}

	body, err := cmd.Run(parent.Body)
	if err != nil {
		log.Printf("processing comment %s - error trying to run substitution.Command{%s, %s}.Run(%s): %s", comment.Name, cmd.ToReplace, cmd.ReplaceWith, parent.Body, err)
		return
	}

	if len(body) == 0 {
		log.Printf("processing comment %s - 0 length body for substitution.Command{%s, %s}.Run(%s)", comment.Name, cmd.ToReplace, cmd.ReplaceWith, parent.Body)
		return
	}

	posted, err := api.PostComment(comment.Name, body+"\n\n^^This ^^was ^^posted ^^by ^^a ^^bot. ^^[Source](https://github.com/anirbanmu/substitute-bot-go)")
	if err != nil {
		log.Printf("processing comment %s - failed to post comment reply: %s", comment.Name, err)
		return
	}

	log.Printf("processing comment %s - posted reply (%s)", comment.Name, posted.Name)

	if _, err := store.AddReplyWithTrim(constructStoredReplyFromPosted(comment.Author, *posted), 50); err != nil {
		log.Printf("processing comment %s - failed to store comment reply: %s", comment.Name, err)
	}
}

func createAPIAndStore(creds reddit.Credentials) (*reddit.API, *persistence.Store) {
	api, err := reddit.InitAPI(creds, nil)
	if err != nil {
		log.Panicf("failed to initialize Reddit API: %s", err)
	}

	store, err := persistence.NewStore(nil, &codec.CborHandle{}, nil)
	if err != nil {
		log.Panicf("failed to get persistence.DefaultStore (is redis running? does REDIS_URL need to be set?): %s", err)
	}

	return api, store
}

func processCommentEvents(idx int, counter *atomicCounter, wg *sync.WaitGroup, botUsername string, api *reddit.API, store *persistence.Store, events <-chan sse.Event) {
	wg.Add(1)
	defer wg.Done()

	log.Printf("worker %d started", idx)

	comment := reddit.Comment{}
	// reader := strings.NewReader("")
	replacer := strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&")
	// decoder := codec.NewDecoder(reader, &codec.JsonHandle{})

	for e := range events {
		if e.Event != "rc" {
			continue
		}

		counter.incr()

		_, err := store.AddNewCommentID(e.ID)
		if err != nil {
			log.Printf("worker %d failed to store max comment event id %s", idx, e.ID)
		}

		// reader.Reset(replacer.Replace(string(e.Data)))
		// if err := decoder.Decode(&comment); err != nil {
		if err := json.Unmarshal([]byte(replacer.Replace(string(e.Data))), &comment); err != nil {
			log.Printf("worker %d failed to decode json into reddit.Comment: %s", idx, err)
			continue
		}

		processComment(botUsername, comment, api, store)
	}
}

const baseStreamURL = "http://stream.pushshift.io/?type=comments&filter=author,author_fullname,body,body_html,created_utc,id,name,parent_id,permalink"

func main() {
	// For detecting shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Holds unprocessed comments
	events := make(chan sse.Event, 4096)

	creds := reddit.Credentials{
		Username:     os.Getenv("SUBSTITUTE_BOT_USERNAME"),
		Password:     os.Getenv("SUBSTITUTE_BOT_PASSWORD"),
		ClientID:     os.Getenv("SUBSTITUTE_BOT_CLIENT_ID"),
		ClientSecret: os.Getenv("SUBSTITUTE_BOT_CLIENT_SECRET"),
		UserAgent:    os.Getenv("SUBSTITUTE_BOT_USER_AGENT"),
	}

	var clients [16]struct {
		api   *reddit.API
		store *persistence.Store
	}

	for i := 0; i < len(clients); i++ {
		api, store := createAPIAndStore(creds)
		clients[i].api = api
		clients[i].store = store
	}

	streamURL := baseStreamURL

	maxID, err := clients[0].store.MaxCommentID()
	if err == nil {
		streamURL = fmt.Sprintf("%s&comment_start_id=%d", streamURL, maxID+1)
	}

	wg := sync.WaitGroup{}
	defer wg.Wait()

	// Will hold total seen comments
	counter := atomicCounter{}

	// Start comment processors
	for i := 0; i < len(clients); i++ {
		go processCommentEvents(i, &counter, &wg, creds.Username, clients[i].api, clients[i].store, events)
	}

	// Heartbeat logger (don't care about waiting for this goroutine to exit)
	go func() {
		for {
			time.Sleep(1 * 60 * time.Second)
			log.Printf("processed %d comments in total. queue length: %d", counter.count(), len(events))
		}
	}()

	// Start comment streamer
	// We give up control of events here
	log.Printf("using %s as URL for stream", streamURL)
	cancel, errChan, err := sse.Stream(events, &wg, streamURL, nil)
	if err != nil {
		log.Panicf("stream initialization errored: %s", err)
	}

	log.Printf("started sse stream")

	// Run till we get a signal / error from stream
	select {
	case <-signals:
	case e := <-errChan:
		log.Printf("received error from stream: %s", e)
	}

	// Clean up
	cancel()
	for e := range errChan {
		log.Printf("received error from stream: %s", e)
	}
}

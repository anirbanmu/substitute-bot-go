package main

import (
	"github.com/anirbanmu/substitute-bot-go/pkg/reddit"
	"github.com/anirbanmu/substitute-bot-go/pkg/replystorage"
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

func constructStoredReplyFromPosted(requester string, posted reddit.Comment) replystorage.Reply {
	return replystorage.Reply{
		Author:         posted.Author,
		AuthorFullname: posted.Author,
		Body:           posted.Body,
		BodyHtml:       posted.BodyHtml,
		CreatedUtc:     posted.CreatedUtc,
		Id:             posted.Id,
		Name:           posted.Name,
		ParentId:       posted.ParentId,
		Permalink:      posted.Permalink,
		Requester:      requester,
	}
}

func processComment(botUser string, comment reddit.Comment, api *reddit.Api, store *replystorage.Store) {
	if comment.IsDeleted() || comment.Author == botUser || !reddit.IsFullnameComment(comment.ParentId) {
		return
	}

	cmd, err := substitution.ParseSubstitutionCommand(comment.Body)
	if err != nil {
		return
	}

	if len(cmd.ReplaceWith) > 0 {
		cmd.ReplaceWith = "**" + cmd.ReplaceWith + "**"
	}

	parent, err := api.GetComment(comment.ParentId)
	if err != nil || parent.IsDeleted() || parent.Author == botUser {
		return
	}

	body, err := cmd.Run(parent.Body + "\n\n^^This ^^was ^^posted ^^by ^^a ^^bot. ^^[Source](https://github.com/anirbanmu/substitute-bot-go)")
	if err != nil {
		log.Printf("processing comment %s - error trying to run SubstitutionCommand{%s, %s}.Run(%s): %s", comment.Name, cmd.ToReplace, cmd.ReplaceWith, parent.Body, err)
		return
	}

	if len(body) == 0 {
		log.Printf("processing comment %s - 0 length body for SubstitutionCommand{%s, %s}.Run(%s)", comment.Name, cmd.ToReplace, cmd.ReplaceWith, parent.Body)
		return
	}

	posted, err := api.PostComment(comment.Name, body)
	if err != nil {
		log.Printf("processing comment %s - failed to post comment reply: %s", comment.Name, err)
		return
	}

	log.Printf("processing comment %s - posted reply (%s)", comment.Name, posted.Name)

	if _, err := store.Add(constructStoredReplyFromPosted(comment.Author, *posted)); err != nil {
		log.Printf("processing comment %s - failed to store comment reply: %s", comment.Name, err)
	}
}

func createApiAndStore(creds reddit.Credentials) (*reddit.Api, *replystorage.Store) {
	api, err := reddit.InitApi(creds, nil)
	if err != nil {
		log.Panicf("failed to initialize Reddit API: %s", err)
	}

	store, err := replystorage.NewStore(nil, &codec.CborHandle{})
	if err != nil {
		log.Panicf("failed to get replystorage.DefaultStore (is redis running? does REDIS_URL need to be set?): %s", err)
	}

	return api, store
}

func processCommentEvents(idx int, counter *atomicCounter, wg *sync.WaitGroup, botUsername string, api *reddit.Api, store *replystorage.Store, events <-chan sse.Event) {
	wg.Add(1)
	defer wg.Done()

	log.Printf("worker %d started", idx)

	comment := reddit.Comment{}
	reader := strings.NewReader("")
	replacer := strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&")
	decoder := codec.NewDecoder(reader, &codec.JsonHandle{})

	for e := range events {
		if e.Event != "rc" {
			continue
		}

		counter.incr()

		reader.Reset(replacer.Replace(string(e.Data)))
		if err := decoder.Decode(&comment); err != nil {
			log.Printf("failed to decode json into reddit.Comment: %s", err)
			continue
		}

		processComment(botUsername, comment, api, store)
	}
}

func main() {
	// For detecting shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Holds unprocessed comments
	events := make(chan sse.Event, 4096)

	creds := reddit.Credentials{
		os.Getenv("SUBSTITUTE_BOT_USERNAME"),
		os.Getenv("SUBSTITUTE_BOT_PASSWORD"),
		os.Getenv("SUBSTITUTE_BOT_CLIENT_ID"),
		os.Getenv("SUBSTITUTE_BOT_CLIENT_SECRET"),
		os.Getenv("SUBSTITUTE_BOT_USER_AGENT"),
	}

	var processorData [16]struct {
		api   *reddit.Api
		store *replystorage.Store
	}

	for _, p := range processorData {
		api, store := createApiAndStore(creds)
		p.api = api
		p.store = store
	}

	wg := sync.WaitGroup{}
	defer wg.Wait()

	// Will hold total seen comments
	counter := atomicCounter{}

	// Start comment processors
	for w := 0; w < 16; w++ {
		go processCommentEvents(w, &counter, &wg, creds.Username, processorData[w].api, processorData[w].store, events)
	}

	// Heartbeat logger (don't care about waiting for this goroutine to exit)
	go func() {
		for {
			time.Sleep(1 * 60 * time.Second)
			log.Printf("processed %d comments in total. queue length: %d", counter.count(), len(events))
		}
	}()

	// Start comment streamer
	const streamUrl = "http://stream.pushshift.io/?type=comments&filter=author,author_fullname,body,body_html,created_utc,id,name,parent_id,permalink"

	// We give up control of events here
	cancel, errChan, err := sse.Stream(events, &wg, streamUrl, nil)
	if err != nil {
		log.Panicf("stream initialization errored: %s", err)
	}

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

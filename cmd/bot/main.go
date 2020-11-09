package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anirbanmu/substitute-bot-go/pkg/persistence"
	"github.com/anirbanmu/substitute-bot-go/pkg/reddit"
	"github.com/anirbanmu/substitute-bot-go/pkg/substitution"
	"github.com/turnage/graw"
	grawReddit "github.com/turnage/graw/reddit"
	"github.com/ugorji/go/codec"
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

func createAPIAndStore(creds reddit.Credentials) (*reddit.API, *persistence.Store) {
	api, err := reddit.InitAPI(creds, nil)
	if err != nil {
		log.Panicf("failed to initialize Reddit API: %s", err)
	}

	store, err := persistence.NewStore(nil, &codec.CborHandle{}, nil, nil)
	if err != nil {
		log.Panicf("failed to get persistence.DefaultStore (is redis running? does REDIS_URL need to be set?): %s", err)
	}

	return api, store
}

type substituteBot struct {
	commentCounter atomicCounter
	store          *persistence.Store
	api            *reddit.API
	botUsername    string
	bot            grawReddit.Bot
}

func (r *substituteBot) Comment(comment *grawReddit.Comment) error {
	r.commentCounter.incr()

	if comment.Author == "[deleted]" || comment.Body == "[removed]" || comment.Author == r.botUsername || !reddit.IsFullnameComment(comment.ParentID) {
		return nil
	}

	processed, err := r.store.AlreadyProcessedCommentID(comment.ID)
	if err != nil || processed {
		return nil
	}
	r.store.AddProcessedCommentID(comment.ID)

	cmd, err := substitution.ParseSubstitutionCommand(comment.Body)
	if err != nil {
		return nil
	}

	if len(cmd.ReplaceWith) > 0 {
		cmd.ReplaceWith = "**" + cmd.ReplaceWith + "**"
	}

	parent, err := r.api.GetComment(comment.ParentID)
	if err != nil || parent.IsDeleted() || parent.Author == r.botUsername {
		return nil
	}

	body, err := cmd.Run(parent.Body)
	if err != nil {
		log.Printf("processing comment %s - error trying to run substitution.Command{%s, %s}.Run(%s): %s", comment.Name, cmd.ToReplace, cmd.ReplaceWith, parent.Body, err)
		return nil
	}

	if len(body) == 0 {
		log.Printf("processing comment %s - 0 length body for substitution.Command{%s, %s}.Run(%s)", comment.Name, cmd.ToReplace, cmd.ReplaceWith, parent.Body)
		return nil
	}

	posted, err := r.api.PostComment(comment.Name, body+"\n\n^^This ^^was ^^posted ^^by ^^a ^^bot. ^^[Source](https://github.com/anirbanmu/substitute-bot-go)")
	if err != nil {
		log.Printf("processing comment %s - failed to post comment reply: %s", comment.Name, err)
		return nil
	}

	log.Printf("processing comment %s - posted reply (%s)", comment.Name, posted.Name)

	if _, err := r.store.AddReplyWithTrim(constructStoredReplyFromPosted(comment.Author, *posted), 50); err != nil {
		log.Printf("processing comment %s - failed to store comment reply: %s", comment.Name, err)
	}

	return nil
}

func main() {
	creds := reddit.Credentials{
		Username:     os.Getenv("SUBSTITUTE_BOT_USERNAME"),
		Password:     os.Getenv("SUBSTITUTE_BOT_PASSWORD"),
		ClientID:     os.Getenv("SUBSTITUTE_BOT_CLIENT_ID"),
		ClientSecret: os.Getenv("SUBSTITUTE_BOT_CLIENT_SECRET"),
		UserAgent:    os.Getenv("SUBSTITUTE_BOT_USER_AGENT"),
	}

	botConfig := grawReddit.BotConfig{
		Agent: creds.UserAgent,
		App: grawReddit.App{
			ID:       creds.ClientID,
			Secret:   creds.ClientSecret,
			Username: creds.Username,
			Password: creds.Password,
		},
		Client: &http.Client{Timeout: time.Second * 30},
	}
	bot, err := grawReddit.NewBot(botConfig)
	if err != nil {
		log.Panicf("reddit bot initialization errored: %s", err)
	}

	cfg := graw.Config{SubredditComments: []string{"all"}}

	api, store := createAPIAndStore(creds)
	handler := &substituteBot{store: store, api: api, botUsername: creds.Username, bot: bot}
	_, wait, err := graw.Run(handler, bot, cfg)
	if err != nil {
		log.Panicf("Failed to start graw run: %s", err)
	}

	// Heartbeat logger (don't care about waiting for this goroutine to exit)
	wg := sync.WaitGroup{}
	defer wg.Wait()

	done := make(chan bool, 1)
	go func() {
		wg.Add(1)
		defer wg.Done()

		for {
			select {
			case <-done:
				return
			case <-time.After(60 * time.Second):
				log.Printf("processed %d comments in total.", handler.commentCounter.count())
			}
		}
	}()

	wait()
	done <- true
}

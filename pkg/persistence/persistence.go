package persistence

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/ugorji/go/codec"
)

const (
	prefix                                    = "substitute-bot-go"
	repliesKey                                = "substitute-bot-go:comments"
	maxCommentIDKey                           = "substitute-bot-go:max-comment-id"
	processedCommentIDPrefix                  = "substitute-bot-go:processed-comments"
	defaultMaxCommentIDExpirationSeconds      = 15 * 60
	defaultProcessedCommentIDExpirationSecond = 5 * 60
)

// Store represents a reply store that can be used to store/retrieve replies
type Store struct {
	Client                  *redis.Client
	handle                  codec.Handle
	storeMaxScript          *redis.Script
	seenCommentIDExpiration time.Duration
	ctx                     context.Context
}

func defaultRedisClient() *redis.Client {
	address := os.Getenv("REDIS_URL")
	if len(address) == 0 {
		address = "localhost:6379"
	}

	return redis.NewClient(
		&redis.Options{
			Addr:     address,
			Password: "", // no password set
			DB:       0,  // use default DB
		},
	)
}

func defaultCodecHandle() codec.Handle {
	return &codec.JsonHandle{}
}

// NewStore creates a new Store with provided redis client & codec
func NewStore(client *redis.Client, handle codec.Handle, maxCommentIDExpirationSeconds *int, seenCommentIDExpirationSeconds *int) (*Store, error) {
	if client == nil {
		client = defaultRedisClient()
	}

	ctx := context.Background()

	// Test out client to make sure we're good to go
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	if handle == nil {
		handle = defaultCodecHandle()
	}

	if maxCommentIDExpirationSeconds == nil {
		defaultSeconds := defaultMaxCommentIDExpirationSeconds
		maxCommentIDExpirationSeconds = &defaultSeconds
	}

	if seenCommentIDExpirationSeconds == nil {
		defaultSeenSeconds := defaultProcessedCommentIDExpirationSecond
		seenCommentIDExpirationSeconds = &defaultSeenSeconds
	}

	scriptText := fmt.Sprintf(`
		local existing = redis.call("GET", KEYS[1])
		local num = tonumber(ARGV[1])
		if (existing ~= false)
		then
			local existingnum = tonumber(existing)
			local newmax = math.max(existingnum, num)

			redis.call("SETEX", KEYS[1], %d, newmax)
			return newmax
		end

		redis.call("SETEX", KEYS[1], %d, num)
		return num
	`, *maxCommentIDExpirationSeconds, *maxCommentIDExpirationSeconds)

	script := redis.NewScript(scriptText)

	return &Store{
		Client:                  client,
		handle:                  handle,
		storeMaxScript:          script,
		seenCommentIDExpiration: time.Second * time.Duration(*seenCommentIDExpirationSeconds),
		ctx:                     ctx,
	}, nil
}

// DefaultStore creates a store with defaults (default redis & json)
func DefaultStore() (*Store, error) {
	return NewStore(defaultRedisClient(), defaultCodecHandle(), nil, nil)
}

// AddReply pesists a Reply to the store
func (s *Store) AddReply(reply Reply) (int64, error) {
	encodeBuffer := bytes.Buffer{}
	encoder := codec.NewEncoder(&encodeBuffer, s.handle)
	if err := encoder.Encode(reply); err != nil {
		return -1, err
	}

	return s.Client.LPush(s.ctx, repliesKey, encodeBuffer.Bytes()).Result()
}

// FetchReply retrieves count Reply's from the store
func (s *Store) FetchReply(count int64) ([]Reply, error) {
	encodedReplies, err := s.Client.LRange(s.ctx, repliesKey, 0, count-1).Result()
	if err != nil {
		return []Reply{}, err
	}

	decoder := codec.NewDecoderBytes(nil, s.handle)

	replies := make([]Reply, len(encodedReplies))
	for i := 0; i < len(encodedReplies); i++ {
		decoder.ResetBytes([]byte(encodedReplies[i]))
		if err := decoder.Decode(&replies[i]); err != nil {
			return []Reply{}, err
		}
	}

	return replies, nil
}

// TrimReplies trims the list of Reply's stored to count
func (s *Store) TrimReplies(count int) error {
	_, err := s.Client.LTrim(s.ctx, repliesKey, 0, int64(count-1)).Result()
	return err
}

// AddReplyWithTrim persists a Reply to the store & trims the list to count atomically
func (s *Store) AddReplyWithTrim(reply Reply, trimCount int64) (int64, error) {
	encodeBuffer := bytes.Buffer{}
	encoder := codec.NewEncoder(&encodeBuffer, s.handle)

	if err := encoder.Encode(reply); err != nil {
		return -1, err
	}

	pipe := s.Client.Pipeline()

	pipe.LPush(s.ctx, repliesKey, encodeBuffer.Bytes())
	pipe.LTrim(s.ctx, repliesKey, 0, int64(trimCount-1))
	length := pipe.LLen(s.ctx, repliesKey)

	if _, err := pipe.Exec(s.ctx); err != nil {
		return -1, err
	}

	return length.Val(), nil
}

// AddNewCommentID stores a new max comment ID seen if it's greater than what's already stored (if any)
func (s *Store) AddNewCommentID(stringID string) (int64, error) {
	ID, err := strconv.ParseInt(stringID, 10, 64)
	if err != nil {
		return -1, err
	}

	max, err := s.storeMaxScript.Run(s.ctx, s.Client, []string{maxCommentIDKey}, ID).Result()
	if err != nil {
		return -1, err
	}

	return max.(int64), nil
}

// MaxCommentID retrieves the last stored max comment id if it exists
func (s *Store) MaxCommentID() (int64, error) {
	max, err := s.Client.Get(s.ctx, maxCommentIDKey).Result()
	if err != nil {
		return -1, err
	}

	ID, err := strconv.ParseInt(max, 10, 64)
	if err != nil {
		return -1, err
	}

	return ID, err
}

func processedCommentKey(stringID string) string {
	return fmt.Sprintf("%s:%s", processedCommentIDPrefix, stringID)
}

// AddProcessedCommentID marks stringID as processed
func (s *Store) AddProcessedCommentID(stringID string) error {
	err := s.Client.Set(s.ctx, processedCommentKey(stringID), true, s.seenCommentIDExpiration).Err()
	if err != nil {
		return err
	}

	return nil
}

// AlreadyProcessedCommentID checks if stringID has already been processed
func (s *Store) AlreadyProcessedCommentID(stringID string) (bool, error) {
	exists, err := s.Client.Exists(s.ctx, processedCommentKey(stringID)).Result()
	if err != nil {
		return false, err
	}

	return exists == 1, nil
}

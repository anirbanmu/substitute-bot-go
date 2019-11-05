package persistence

import (
	"bytes"
	"github.com/go-redis/redis"
	"github.com/ugorji/go/codec"
	"os"
	"strconv"
)

const (
	prefix          = "substitute-bot-go"
	repliesKey      = "substitute-bot-go:comments"
	maxCommentIDKey = "substitute-bot-go:max-comment-id"
)

// Store represents a reply store that can be used to store/retrieve replies
type Store struct {
	Client         *redis.Client
	EncodeBuffer   *bytes.Buffer
	Encoder        *codec.Encoder
	Decoder        *codec.Decoder
	storeMaxScript *redis.Script
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
func NewStore(client *redis.Client, handle codec.Handle) (*Store, error) {
	if client == nil {
		client = defaultRedisClient()
	}

	// Test out client to make sure we're good to go
	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}

	if handle == nil {
		handle = defaultCodecHandle()
	}

	encodeBuffer := bytes.Buffer{}
	encoder := codec.NewEncoder(&encodeBuffer, handle)

	script := redis.NewScript(`
		local existing = redis.call("GET", KEYS[1])
		local num = tonumber(ARGV[1])
		if (existing ~= false)
		then
			local existingnum = tonumber(existing)
			local newmax = math.max(existingnum, num)

			redis.call("SET", KEYS[1], newmax)
			return newmax
		end

		redis.call("SET", KEYS[1], num)
		return num
	`)

	return &Store{
		Client:         client,
		EncodeBuffer:   &encodeBuffer,
		Encoder:        encoder,
		Decoder:        codec.NewDecoderBytes(nil, handle),
		storeMaxScript: script,
	}, nil
}

// DefaultStore creates a store with defaults (default redis & json)
func DefaultStore() (*Store, error) {
	return NewStore(defaultRedisClient(), defaultCodecHandle())
}

// AddReply pesists a Reply to the store
func (s *Store) AddReply(reply Reply) (int64, error) {
	s.EncodeBuffer.Reset()

	if err := s.Encoder.Encode(reply); err != nil {
		return -1, err
	}

	return s.Client.LPush(repliesKey, s.EncodeBuffer.Bytes()).Result()
}

// FetchReply retrieves count Reply's from the store
func (s *Store) FetchReply(count int64) ([]Reply, error) {
	encodedReplies, err := s.Client.LRange(repliesKey, 0, count-1).Result()
	if err != nil {
		return []Reply{}, err
	}

	replies := make([]Reply, len(encodedReplies))
	for i := 0; i < len(encodedReplies); i++ {
		s.Decoder.ResetBytes([]byte(encodedReplies[i]))
		if err := s.Decoder.Decode(&replies[i]); err != nil {
			return []Reply{}, err
		}
	}

	return replies, nil
}

// TrimReplies trims the list of Reply's stored to count
func (s *Store) TrimReplies(count int) error {
	_, err := s.Client.LTrim(repliesKey, 0, int64(count-1)).Result()
	return err
}

// AddReplyWithTrim persists a Reply to the store & trims the list to count atomically
func (s *Store) AddReplyWithTrim(reply Reply, trimCount int64) (int64, error) {
	s.EncodeBuffer.Reset()

	if err := s.Encoder.Encode(reply); err != nil {
		return -1, err
	}

	pipe := s.Client.Pipeline()

	pipe.LPush(repliesKey, s.EncodeBuffer.Bytes())
	pipe.LTrim(repliesKey, 0, int64(trimCount-1))
	length := pipe.LLen(repliesKey)

	if _, err := pipe.Exec(); err != nil {
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

	max, err := s.storeMaxScript.Run(s.Client, []string{maxCommentIDKey}, ID).Result()
	if err != nil {
		return -1, err
	}

	return max.(int64), nil
}

// MaxCommentID retrieves the last stored max comment id if it exists
func (s *Store) MaxCommentID() (int64, error) {
	max, err := s.Client.Get(maxCommentIDKey).Result()
	if err != nil {
		return -1, err
	}

	ID, err := strconv.ParseInt(max, 10, 64)
	if err != nil {
		return -1, err
	}

	return ID, err
}

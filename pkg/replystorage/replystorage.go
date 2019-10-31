package replystorage

import (
	"bytes"
	"github.com/go-redis/redis"
	"github.com/ugorji/go/codec"
	"os"
)

const (
	prefix     = "substitute-bot-go"
	repliesKey = "substitute-bot-go:comments"
)

type Store struct {
	Client       *redis.Client
	EncodeBuffer *bytes.Buffer
	Encoder      *codec.Encoder
	Decoder      *codec.Decoder
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

	return &Store{
		Client:       client,
		EncodeBuffer: &encodeBuffer,
		Encoder:      encoder,
		Decoder:      codec.NewDecoderBytes(nil, handle),
	}, nil
}

func DefaultStore() (*Store, error) {
	return NewStore(defaultRedisClient(), defaultCodecHandle())
}

func (s *Store) Add(reply Reply) (int64, error) {
	s.EncodeBuffer.Reset()

	if err := s.Encoder.Encode(reply); err != nil {
		return -1, err
	}

	return s.Client.LPush(repliesKey, s.EncodeBuffer.Bytes()).Result()
}

func (s *Store) Fetch(count int64) ([]Reply, error) {
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

func (s *Store) Trim(count int) error {
	_, err := s.Client.LTrim(repliesKey, 0, int64(count-1)).Result()
	return err
}

func (s *Store) AddWithTrim(reply Reply, trimCount int64) (int64, error) {
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

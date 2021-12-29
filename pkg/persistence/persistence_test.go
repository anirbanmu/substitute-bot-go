package persistence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/ugorji/go/codec"
)

var _ = Describe("persistence", func() {
	address := os.Getenv("REDIS_URL")
	if len(address) == 0 {
		address = "localhost:6379"
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	defaultStore, _ := DefaultStore()

	replies := [10]Reply{}
	repliesJSON := [10](*[]byte){}
	for i := 0; i < 10; i++ {
		replies[i] = Reply{
			Author:         "username",
			AuthorFullname: "t3_b49jk",
			Body:           "body",
			BodyHTML:       "html",
			CreatedUtc:     1571371710,
			ID:             "f5uyrhf",
			Name:           "t1_f5uyrhf",
			ParentID:       "t1_f5uyrdf",
			Permalink:      "r/subreddit/comments/de31f1/title/f5uyrhf",
			Requester:      fmt.Sprintf("requester-username-%d", i),
		}

		b, _ := json.Marshal(&replies[i])
		repliesJSON[i] = &b
	}

	ctx := context.Background()

	BeforeEach(func() {
		Expect(redisClient.FlushDB(ctx).Err()).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(redisClient.FlushDB(ctx).Err()).NotTo(HaveOccurred())
	})

	Describe("Store", func() {
		Context("creation", func() {
			Describe("DefaultStore", func() {
				It("returns a Store with a non-nil client & encoder/decoder", func() {
					Expect(defaultStore.Client).NotTo(BeNil())
					Expect(defaultStore.handle).NotTo(BeNil())
				})
			})

			Describe("NewStore", func() {
				Context("when no parameters are given", func() {
					store, err := NewStore(nil, nil, nil, nil)
					It("returns a Store with a non-nil client & encoder/decoder", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(store.Client).NotTo(BeNil())
						Expect(store.handle).NotTo(BeNil())
					})
				})

				Context("when bad client is given", func() {
					It("returns no Store & error", func() {
						client := redis.NewClient(&redis.Options{
							Addr:     "ewriotjkldfsgoierut:4444",
							Password: "", // no password set
							DB:       0,  // use default DB
						})

						store, err := NewStore(client, nil, nil, nil)
						Expect(err).To(HaveOccurred())
						Expect(store).To(BeNil())
					})
				})

				Context("when only client is given", func() {
					store, err := NewStore(redisClient, nil, nil, nil)
					It("returns a Store with a same client as given & encoder/decoder", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(store.Client).To(Equal(redisClient))
						Expect(store.handle).NotTo(BeNil())
					})
				})

				Context("when only encoding is given", func() {
					store, err := NewStore(nil, &codec.CborHandle{}, nil, nil)
					It("returns a Store with a non-nil client & encoder/decoder", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(store.Client).NotTo(BeNil())
						Expect(store.handle).NotTo(BeNil())
					})
				})
			})
		})

		Context("methods", func() {
			Describe("AddReply", func() {
				It("correctly adds reply to store", func() {
					replyCount, err := defaultStore.AddReply(replies[0])
					Expect(err).NotTo(HaveOccurred())
					Expect(replyCount).To(Equal(int64(1)))

					b, err := redisClient.LRange(ctx, repliesKey, 0, 0).Result()
					Expect(err).NotTo(HaveOccurred())
					Expect(bytes.Equal([]byte(b[0]), *repliesJSON[0])).To(BeTrue())
				})
			})

			Describe("FetchReply", func() {
				BeforeEach(func() {
					for i := 0; i < len(repliesJSON); i++ {
						_, err := redisClient.LPush(ctx, repliesKey, *repliesJSON[i]).Result()
						Expect(err).NotTo(HaveOccurred())
					}
				})

				It("correctly fetches given number of replies from store", func() {
					returnedReplies, err := defaultStore.FetchReply(7)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(returnedReplies)).To(Equal(7))

					for i := 0; i < len(returnedReplies); i++ {
						Expect(returnedReplies[i]).To(Equal(replies[len(replies)-i-1]))
					}
				})
			})

			Describe("TrimReplies", func() {
				BeforeEach(func() {
					for i := 0; i < len(repliesJSON); i++ {
						_, err := redisClient.LPush(ctx, repliesKey, *repliesJSON[i]).Result()
						Expect(err).NotTo(HaveOccurred())
					}
				})

				It("does nothing if number of replies is less than count", func() {
					err := defaultStore.TrimReplies(len(repliesJSON) * 2)
					Expect(err).NotTo(HaveOccurred())

					count, err := redisClient.LLen(ctx, repliesKey).Result()
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(int64(len(repliesJSON))))
				})

				It("correctly trims number of replies to given count", func() {
					err := defaultStore.TrimReplies(4)
					Expect(err).NotTo(HaveOccurred())

					count, err := redisClient.LLen(ctx, repliesKey).Result()
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(int64(4)))
				})
			})

			Describe("AddReplyWithTrim", func() {
				BeforeEach(func() {
					for i := 0; i < len(repliesJSON); i++ {
						_, err := redisClient.LPush(ctx, repliesKey, *repliesJSON[i]).Result()
						Expect(err).NotTo(HaveOccurred())
					}
				})

				Context("when number of replies is less than trimCount", func() {
					It("correctly adds reply to store & doesn't trim the list", func() {
						replyCount, err := defaultStore.AddReplyWithTrim(replies[0], int64(len(repliesJSON)*2+1))
						Expect(err).NotTo(HaveOccurred())
						Expect(replyCount).To(Equal(int64(len(repliesJSON) + 1)))

						b, err := redisClient.LRange(ctx, repliesKey, 0, 0).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(bytes.Equal([]byte(b[0]), *repliesJSON[0])).To(BeTrue())

						count, err := redisClient.LLen(ctx, repliesKey).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(count).To(Equal(int64(len(repliesJSON) + 1)))
					})
				})

				Context("when number of replies is greater than trimCount", func() {
					It("correctly adds reply to store & trims the list", func() {
						replyCount, err := defaultStore.AddReplyWithTrim(replies[0], 2)
						Expect(err).NotTo(HaveOccurred())
						Expect(replyCount).To(Equal(int64(2)))

						b, err := redisClient.LRange(ctx, repliesKey, 0, 0).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(bytes.Equal([]byte(b[0]), *repliesJSON[0])).To(BeTrue())

						count, err := redisClient.LLen(ctx, repliesKey).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(count).To(Equal(int64(2)))
					})
				})
			})

			Describe("AddNewCommentID", func() {
				zeroDuration, _ := time.ParseDuration("0s")
				setExpirationDuration, _ := time.ParseDuration(fmt.Sprintf("%ds", defaultMaxCommentIDExpirationSeconds))

				Context("when there is no existing max id", func() {
					It("sets given id to max and returns it", func() {
						max, err := defaultStore.AddNewCommentID("34849")
						Expect(err).NotTo(HaveOccurred())
						Expect(max).To(Equal(int64(34849)))

						m, err := redisClient.Get(ctx, maxCommentIDKey).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(m).To(Equal("34849"))

						// Make sure expiration is being set
						exp, err := redisClient.TTL(ctx, maxCommentIDKey).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(exp).To(SatisfyAll(BeNumerically(">", zeroDuration), BeNumerically("<=", setExpirationDuration)))
					})
				})

				Context("when there is an existing max id", func() {
					Context("and the existing max id is lower", func() {
						It("sets given id to max and returns it", func() {
							_, err := redisClient.Set(ctx, maxCommentIDKey, 90, 0).Result()
							Expect(err).NotTo(HaveOccurred())

							max, err := defaultStore.AddNewCommentID("100")
							Expect(err).NotTo(HaveOccurred())
							Expect(max).To(Equal(int64(100)))

							m, err := redisClient.Get(ctx, maxCommentIDKey).Result()
							Expect(err).NotTo(HaveOccurred())
							Expect(m).To(Equal("100"))

							// Make sure expiration is being set
							exp, err := redisClient.TTL(ctx, maxCommentIDKey).Result()
							Expect(err).NotTo(HaveOccurred())
							Expect(exp).To(SatisfyAll(BeNumerically(">", zeroDuration), BeNumerically("<=", setExpirationDuration)))
						})
					})

					Context("and the existing max id is higher", func() {
						It("returns existing id", func() {
							_, err := redisClient.Set(ctx, maxCommentIDKey, 100, 0).Result()
							Expect(err).NotTo(HaveOccurred())

							max, err := defaultStore.AddNewCommentID("90")
							Expect(err).NotTo(HaveOccurred())
							Expect(max).To(Equal(int64(100)))

							m, err := redisClient.Get(ctx, maxCommentIDKey).Result()
							Expect(err).NotTo(HaveOccurred())
							Expect(m).To(Equal("100"))
						})
					})
				})
			})

			Describe("MaxCommentID", func() {
				Context("when there is no existing max id", func() {
					It("returns err", func() {
						_, err := defaultStore.MaxCommentID()
						Expect(err).To(HaveOccurred())
					})
				})

				Context("when there is an existing max id", func() {
					It("returns id", func() {
						_, err := redisClient.Set(ctx, maxCommentIDKey, 100, 0).Result()
						Expect(err).NotTo(HaveOccurred())

						m, err := defaultStore.MaxCommentID()
						Expect(err).NotTo(HaveOccurred())
						Expect(m).To(Equal(int64(100)))
					})
				})
			})

			Describe("AddProcessedCommentID", func() {
				zeroDuration, _ := time.ParseDuration("0s")
				setExpirationDuration, _ := time.ParseDuration(fmt.Sprintf("%ds", defaultProcessedCommentIDExpirationSecond))

				Context("when using default expiration", func() {
					It("set given id with prefix key with proper TTL", func() {
						err := defaultStore.AddProcessedCommentID("34849")
						Expect(err).NotTo(HaveOccurred())

						// Make sure expiration is being set
						exp, err := redisClient.TTL(ctx, fmt.Sprintf("%s:%s", processedCommentIDPrefix, "34849")).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(exp).To(SatisfyAll(BeNumerically(">", zeroDuration), BeNumerically("<=", setExpirationDuration)))
					})
				})
			})

			Describe("AlreadyProcessedCommentID", func() {
				Context("when using default expiration", func() {
					It("returns true when key exists", func() {
						err := redisClient.Set(ctx, fmt.Sprintf("%s:%s", processedCommentIDPrefix, "34849"), true, 0).Err()
						Expect(err).NotTo(HaveOccurred())

						exists, err := defaultStore.AlreadyProcessedCommentID("34849")
						Expect(err).NotTo(HaveOccurred())
						Expect(exists).To(Equal(true))
					})

					It("returns false when key doesn't exist", func() {
						exists, err := defaultStore.AlreadyProcessedCommentID("34849")
						Expect(err).NotTo(HaveOccurred())
						Expect(exists).To(Equal(false))
					})
				})
			})

			Describe("processedCommentKey", func() {
				It("prefixes with proper string", func() {
					key := processedCommentKey("34849")
					Expect(key).To(Equal(fmt.Sprintf("%s:%s", processedCommentIDPrefix, "34849")))
				})
			})
		})
	})
})

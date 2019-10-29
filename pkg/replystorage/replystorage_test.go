package replystorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/ugorji/go/codec"
)

var _ = Describe("replystorage", func() {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	defaultStore, _ := DefaultStore()

	replies := [10]Reply{}
	repliesJson := [10](*[]byte){}
	for i := 0; i < 10; i++ {
		replies[i] = Reply{
			Author:         "username",
			AuthorFullname: "t3_b49jk",
			Body:           "body",
			BodyHtml:       "html",
			CreatedUtc:     1571371710,
			Id:             "f5uyrhf",
			Name:           "t1_f5uyrhf",
			ParentId:       "t1_f5uyrdf",
			Permalink:      "r/subreddit/comments/de31f1/title/f5uyrhf",
			Requester:      fmt.Sprintf("requester-username-%d", i),
		}

		b, _ := json.Marshal(&replies[i])
		repliesJson[i] = &b
	}

	BeforeEach(func() {
		Expect(redisClient.FlushDB().Err()).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(redisClient.FlushDB().Err()).NotTo(HaveOccurred())
	})

	Describe("Store", func() {
		Context("creation", func() {
			Describe("DefaultStore", func() {
				It("returns a Store with a non-nil client & encoder/decoder", func() {
					Expect(defaultStore.Client).NotTo(BeNil())
					Expect(defaultStore.Encoder).NotTo(BeNil())
					Expect(defaultStore.Decoder).NotTo(BeNil())
				})
			})

			Describe("NewStore", func() {
				Context("when no parameters are given", func() {
					store, err := NewStore(nil, nil)
					It("returns a Store with a non-nil client & encoder/decoder", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(store.Client).NotTo(BeNil())
						Expect(store.Encoder).NotTo(BeNil())
						Expect(store.Decoder).NotTo(BeNil())
					})
				})

				Context("when bad client is given", func() {
					It("returns no Store & error", func() {
						client := redis.NewClient(&redis.Options{
							Addr:     "ewriotjkldfsgoierut:4444",
							Password: "", // no password set
							DB:       0,  // use default DB
						})

						store, err := NewStore(client, nil)
						Expect(err).To(HaveOccurred())
						Expect(store).To(BeNil())
					})
				})

				Context("when only client is given", func() {
					store, err := NewStore(redisClient, nil)
					It("returns a Store with a same client as given & encoder/decoder", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(store.Client).To(Equal(redisClient))
						Expect(store.Encoder).NotTo(BeNil())
						Expect(store.Decoder).NotTo(BeNil())
					})
				})

				Context("when only encoding is given", func() {
					store, err := NewStore(nil, &codec.CborHandle{})
					It("returns a Store with a non-nil client & encoder/decoder", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(store.Client).NotTo(BeNil())
						Expect(store.Encoder).NotTo(BeNil())
						Expect(store.Decoder).NotTo(BeNil())
					})
				})
			})
		})

		Context("methods", func() {
			Describe("Add", func() {
				It("correctly adds reply to store", func() {
					replyCount, err := defaultStore.Add(replies[0])
					Expect(err).NotTo(HaveOccurred())
					Expect(replyCount).To(Equal(int64(1)))

					b, err := redisClient.LRange(repliesKey, 0, 0).Result()
					Expect(err).NotTo(HaveOccurred())
					Expect(bytes.Equal([]byte(b[0]), *repliesJson[0])).To(BeTrue())
				})
			})

			Describe("Fetch", func() {
				BeforeEach(func() {
					for i := 0; i < len(repliesJson); i++ {
						_, err := redisClient.LPush(repliesKey, *repliesJson[i]).Result()
						Expect(err).NotTo(HaveOccurred())
					}
				})

				It("correctly fetches given number of replies from store", func() {
					returnedReplies, err := defaultStore.Fetch(7)
					Expect(err).NotTo(HaveOccurred())
					Expect(len(returnedReplies)).To(Equal(7))

					for i := 0; i < len(returnedReplies); i++ {
						Expect(returnedReplies[i]).To(Equal(replies[len(replies)-i-1]))
					}
				})
			})

			Describe("Trim", func() {
				BeforeEach(func() {
					for i := 0; i < len(repliesJson); i++ {
						_, err := redisClient.LPush(repliesKey, *repliesJson[i]).Result()
						Expect(err).NotTo(HaveOccurred())
					}
				})

				It("does nothing if number of replies is less than count", func() {
					err := defaultStore.Trim(len(repliesJson) * 2)
					Expect(err).NotTo(HaveOccurred())

					count, err := redisClient.LLen(repliesKey).Result()
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(int64(len(repliesJson))))
				})

				It("correctly trims number of replies to given count", func() {
					err := defaultStore.Trim(4)
					Expect(err).NotTo(HaveOccurred())

					count, err := redisClient.LLen(repliesKey).Result()
					Expect(err).NotTo(HaveOccurred())
					Expect(count).To(Equal(int64(4)))
				})
			})

			Describe("AddWithTrim", func() {
				BeforeEach(func() {
					for i := 0; i < len(repliesJson); i++ {
						_, err := redisClient.LPush(repliesKey, *repliesJson[i]).Result()
						Expect(err).NotTo(HaveOccurred())
					}
				})

				Context("when number of replies is less than trimCount", func() {
					It("correctly adds reply to store & doesn't trim the list", func() {
						replyCount, err := defaultStore.AddWithTrim(replies[0], int64(len(repliesJson)*2+1))
						Expect(err).NotTo(HaveOccurred())
						Expect(replyCount).To(Equal(int64(len(repliesJson) + 1)))

						b, err := redisClient.LRange(repliesKey, 0, 0).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(bytes.Equal([]byte(b[0]), *repliesJson[0])).To(BeTrue())

						count, err := redisClient.LLen(repliesKey).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(count).To(Equal(int64(len(repliesJson) + 1)))
					})
				})

				Context("when number of replies is greater than trimCount", func() {
					It("correctly adds reply to store & trims the list", func() {
						replyCount, err := defaultStore.AddWithTrim(replies[0], 2)
						Expect(err).NotTo(HaveOccurred())
						Expect(replyCount).To(Equal(int64(2)))

						b, err := redisClient.LRange(repliesKey, 0, 0).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(bytes.Equal([]byte(b[0]), *repliesJson[0])).To(BeTrue())

						count, err := redisClient.LLen(repliesKey).Result()
						Expect(err).NotTo(HaveOccurred())
						Expect(count).To(Equal(int64(2)))
					})
				})
			})
		})
	})
})

package main

import (
	"errors"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/anirbanmu/substitute-bot-go/pkg/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWeb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "web Suite")
}

type FailingReplyFetcher struct{}

func (r *FailingReplyFetcher) FetchReply(count int64) ([]persistence.Reply, error) {
	return nil, errors.New("some error")
}

type SuccessfulReplyFetcher struct{}

func (r *SuccessfulReplyFetcher) FetchReply(count int64) ([]persistence.Reply, error) {
	replies := []persistence.Reply{
		{
			Author:         "username",
			AuthorFullname: "t3_b49jk",
			Body:           "body",
			BodyHTML:       "html",
			CreatedUtc:     1571371710, // October 18, 2019
			ID:             "f5uyrhf",
			Name:           "t1_f5uyrhf",
			ParentID:       "t1_f5uyrdf",
			Permalink:      "r/subreddit/comments/de31f1/title/f5uyrhf",
			Requester:      "requester-username-user",
		},
	}
	return replies, nil
}

var _ = Describe("web", func() {
	Describe("getStyleHandler", func() {
		It("returns CSS", func() {
			handler, err := getStyleHandler()
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest("GET", "http://example.com/some/path", nil)
			w := httptest.NewRecorder()
			handler(w, req)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			Expect(resp.Header.Get("Content-Type")).To(Equal("text/css; charset=utf-8"))
			Expect(len(body)).To(BeNumerically(">", 0))
		})
	})

	Describe("getIndexHandler", func() {
		Context("when path matches / exactly", func() {
			Context("and reply fetching fails", func() {
				It("returns 500", func() {
					handler := getIndexHandler("bot-username", &FailingReplyFetcher{})

					req := httptest.NewRequest("GET", "http://example.com/", nil)
					w := httptest.NewRecorder()
					handler(w, req)

					resp := w.Result()
					Expect(resp.StatusCode).To(Equal(500))
				})
			})

			Context("and reply fetching succeeds", func() {
				It("returns 200 & renders HTML", func() {
					handler := getIndexHandler("bot-username", &SuccessfulReplyFetcher{})

					req := httptest.NewRequest("GET", "http://example.com/", nil)
					w := httptest.NewRecorder()
					handler(w, req)

					resp := w.Result()
					Expect(resp.StatusCode).To(Equal(200))

					body, err := ioutil.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(body)).To(ContainSubstring("October 18, 2019"))
				})
			})
		})

		Context("when path does not match / exactly", func() {
			It("returns 404", func() {
				handler := getIndexHandler("bot-username", &SuccessfulReplyFetcher{})

				req := httptest.NewRequest("GET", "http://example.com/some/path", nil)
				w := httptest.NewRecorder()
				handler(w, req)

				resp := w.Result()
				Expect(resp.StatusCode).To(Equal(404))
			})
		})
	})
})

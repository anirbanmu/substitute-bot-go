package reddit

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Comment", func() {
	var comment Comment

	BeforeEach(func() {
		comment = Comment{
			Author:         "dummy-author",
			AuthorFullname: "t2_4jtui7g8",
			Body:           "this is fake text",
			BodyHtml:       "<div class=\"md\"><p>this is fake text</p></div>",
			CreatedUtc:     1571002615,
			Id:             "g7krui4",
			Name:           "t1_g7krui4",
			ParentId:       "t1_h7kxui2",
			Permalink:      "/r/dummy-subreddit/comments/krtjrk/dummy-topic/g7krui4/",
		}
	})

	Describe("IsDeleted", func() {
		Context("when comment has author & body", func() {
			It("returns false", func() {
				Expect(comment.IsDeleted()).To(BeFalse())
			})
		})

		Context("when comment author is [deleted]", func() {
			It("returns true", func() {
				comment.Author = "[deleted]"
				Expect(comment.IsDeleted()).To(BeTrue())
			})
		})

		Context("when comment body is [removed]", func() {
			It("returns true", func() {
				comment.Body = "[removed]"
				Expect(comment.IsDeleted()).To(BeTrue())
			})
		})
	})
})

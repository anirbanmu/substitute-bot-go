package persistence

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("persistence", func() {
	var reply Reply

	BeforeEach(func() {
		reply = Reply{
			Author:         "username",
			AuthorFullname: "t3_b49jk",
			Body:           "body",
			BodyHTML:       "html",
			CreatedUtc:     1571371710,
			ID:             "f5uyrhf",
			Name:           "t1_f5uyrhf",
			ParentID:       "t1_f5uyrdf",
			Permalink:      "r/subreddit/comments/de31f1/title/f5uyrhf",
			Requester:      "requester-username-user",
		}
	})

	Describe("RenderMarkdown", func() {
		Context("when body is empty", func() {
			It("returns empty byte slice", func() {
				reply.Body = ""

				rendered := reply.RenderMarkdown()
				Expect(len(rendered)).To(Equal(0))
			})
		})

		Context("when body exists", func() {
			Context("with no need of escaping", func() {
				It("returns correct HTML rendering", func() {
					reply.Body = "**bolded** text"

					rendered := reply.RenderMarkdown()
					Expect(len(rendered)).NotTo(Equal(0))
					Expect(string(rendered)).To(Equal("<p><strong>bolded</strong> text</p>\n"))
				})
			})

			Context("with need of escaping", func() {
				It("returns correct HTML rendering", func() {
					reply.Body = "**bolded<script>alert(\"dsfsdf\")</script>** text"

					rendered := reply.RenderMarkdown()
					Expect(len(rendered)).NotTo(Equal(0))
					Expect(string(rendered)).To(Equal("<p><strong>bolded</strong> text</p>\n"))
				})
			})
		})
	})

	Describe("RenderMarkdownForTemplate", func() {
		Context("when body is empty", func() {
			It("returns empty string", func() {
				reply.Body = ""

				rendered := reply.RenderMarkdownForTemplate()
				Expect(len(rendered)).To(Equal(0))
			})
		})

		Context("when body exists", func() {
			Context("with no need of escaping", func() {
				It("returns correct HTML rendering", func() {
					reply.Body = "**bolded** text"

					rendered := reply.RenderMarkdownForTemplate()
					Expect(len(rendered)).NotTo(Equal(0))
					Expect(string(rendered)).To(Equal("<p><strong>bolded</strong> text</p>\n"))
				})
			})

			Context("with need of escaping", func() {
				It("returns correct HTML rendering", func() {
					reply.Body = "**bolded<script>alert(\"dsfsdf\")</script>** text"

					rendered := reply.RenderMarkdownForTemplate()
					Expect(len(rendered)).NotTo(Equal(0))
					Expect(string(rendered)).To(Equal("<p><strong>bolded</strong> text</p>\n"))
				})
			})
		})
	})

	Describe("RenderCreatedDateForTemplate", func() {
		It("returns correct formatted date string", func() {
			rendered := reply.RenderCreatedDateForTemplate()
			Expect(string(rendered)).To(Equal("October 18, 2019"))
		})
	})

	Describe("RenderSanitizedHTMLForTemplate", func() {
		It("returns sanitized html", func() {
			reply.BodyHTML = "<script>alert(\"dsfsdf\")</script><p>hello</p>"

			rendered := reply.RenderSanitizedHTMLForTemplate()
			Expect(string(rendered)).To(Equal("<p>hello</p>"))
		})
	})
})

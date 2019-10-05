package replystorage

import (
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"html/template"
	"time"
)

type Reply struct {
	Author         string `json:"author"`
	AuthorFullname string `json:"author_fullname"`
	Body           string `json:"body"`
	BodyHtml       string `json:"body_html"`
	CreatedUtc     int64  `json:"created_utc"`
	Id             string `json:"id"`
	Name           string `json:"name"`
	ParentId       string `json:"parent_id"`
	Permalink      string `json:"permalink"`
	Requester      string `json:"requester"`
}

func (r *Reply) RenderMarkdown() []byte {
	unsafe := blackfriday.Run([]byte(r.Body))
	return bluemonday.UGCPolicy().SanitizeBytes(unsafe)
}

func (r *Reply) RenderSanitizedHtmlForTemplate() template.HTML {
	return template.HTML(bluemonday.UGCPolicy().Sanitize(r.BodyHtml))
}

func (r *Reply) RenderMarkdownForTemplate() template.HTML {
	return template.HTML(string(r.RenderMarkdown()))
}

func (r *Reply) RenderCreatedDateForTemplate() template.HTML {
	return template.HTML(time.Unix(r.CreatedUtc, 0).UTC().Format("January 02, 2006"))
}

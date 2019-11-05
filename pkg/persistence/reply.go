package persistence

import (
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"html/template"
	"time"
)

// Reply is the representation of a reddit comment reply that can be stored
type Reply struct {
	Author         string `json:"author"`
	AuthorFullname string `json:"author_fullname"`
	Body           string `json:"body"`
	BodyHTML       string `json:"body_html"`
	CreatedUtc     int64  `json:"created_utc"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	ParentID       string `json:"parent_id"`
	Permalink      string `json:"permalink"`
	Requester      string `json:"requester"`
}

// RenderMarkdown renders & sanitizes the stored markdown into a HTML string
func (r *Reply) RenderMarkdown() []byte {
	unsafe := blackfriday.Run([]byte(r.Body))
	return bluemonday.UGCPolicy().SanitizeBytes(unsafe)
}

// RenderSanitizedHTMLForTemplate sanitized the stored HTML body into a template.HTML
func (r *Reply) RenderSanitizedHTMLForTemplate() template.HTML {
	return template.HTML(bluemonday.UGCPolicy().Sanitize(r.BodyHTML))
}

// RenderMarkdownForTemplate renders & sanitizes the stored markdown into a template.HTML
func (r *Reply) RenderMarkdownForTemplate() template.HTML {
	return template.HTML(string(r.RenderMarkdown()))
}

// RenderCreatedDateForTemplate renders a stored created_utc timestamp into a day representation template.HTML
func (r *Reply) RenderCreatedDateForTemplate() template.HTML {
	return template.HTML(time.Unix(r.CreatedUtc, 0).UTC().Format("January 02, 2006"))
}

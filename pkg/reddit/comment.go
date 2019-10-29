package reddit

type Comment struct {
	Author         string `json:"author"`
	AuthorFullname string `json:"author_fullname"`
	Body           string `json:"body"`
	BodyHtml       string `json:"body_html"`
	CreatedUtc     int64  `json:"created_utc"`
	Id             string `json:"id"`
	Name           string `json:"name"`
	ParentId       string `json:"parent_id"`
	Permalink      string `json:"permalink"`
}

func (c *Comment) IsDeleted() bool {
	return c.Author == "[deleted]" || c.Body == "[deleted]"
}

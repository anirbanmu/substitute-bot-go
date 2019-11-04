package reddit

// Comment represents a reddit comment's core properties (subset of all properties)
type Comment struct {
	Author         string  `json:"author"`
	AuthorFullname string  `json:"author_fullname"`
	Body           string  `json:"body"`
	BodyHTML       string  `json:"body_html"`
	CreatedUtc     float64 `json:"created_utc"`
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	ParentID       string  `json:"parent_id"`
	Permalink      string  `json:"permalink"`
}

// IsDeleted returns true if comment seems to be deleted
func (c *Comment) IsDeleted() bool {
	return c.Author == "[deleted]" || c.Body == "[removed]"
}

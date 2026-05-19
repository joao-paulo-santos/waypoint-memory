package models

type Label struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	HexColor    string `json:"hex_color"`
	CreatedAt   string `json:"created_at"`
}

type CreateLabelRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	HexColor    string `json:"hex_color,omitempty"`
}

type Comment struct {
	ID        int64  `json:"id"`
	TaskID    int64  `json:"task_id"`
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreateCommentRequest struct {
	Author string `json:"author,omitempty"`
	Body   string `json:"body"`
}

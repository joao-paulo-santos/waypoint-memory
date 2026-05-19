package models

type Label struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	HexColor    string `json:"hex_color"`
	CreatedAt   string `json:"created_at"`
}

package models

type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"is_dir"`
	Children []*FileNode `json:"children,omitempty"`
}

type WikiPage struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Path    string `json:"path"`
}

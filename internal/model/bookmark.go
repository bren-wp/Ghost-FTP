package model

const (
	BookmarkKindLocal  = "local"
	BookmarkKindRemote = "remote"
)

// Bookmark is deliberately non-secret navigation metadata. Remote bookmarks
// carry only the account identity needed to prevent a path from being reused by
// a different login on the same host; credentials and trust material never
// belong in this model.
type Bookmark struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Protocol string `json:"protocol,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
}

type BookmarkInput struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Protocol string `json:"protocol,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
}

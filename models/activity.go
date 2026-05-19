package models

type ActivityEntry struct {
	ID         int64  `json:"id"`
	Action     string `json:"action"`
	EntityType string `json:"entity_type"`
	EntityID   *int64 `json:"entity_id"`
	Actor      string `json:"actor"`
	Details    string `json:"details"`
	CreatedAt  string `json:"created_at"`
}

type GlobalActivityEntry struct {
	ActivityEntry
	ProjectID   int64  `json:"project_id"`
	ProjectName string `json:"project_name"`
}

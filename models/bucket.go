package models

type Bucket struct {
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	Position     float64 `json:"position"`
	IsDoneBucket bool    `json:"is_done_bucket"`
	WIPLimit     int     `json:"wip_limit"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type CreateBucketRequest struct {
	Title        string  `json:"title"`
	Position     float64 `json:"position,omitempty"`
	IsDoneBucket bool    `json:"is_done_bucket,omitempty"`
	WIPLimit     int     `json:"wip_limit,omitempty"`
}

type UpdateBucketRequest struct {
	Title        *string  `json:"title,omitempty"`
	Position     *float64 `json:"position,omitempty"`
	IsDoneBucket *bool    `json:"is_done_bucket,omitempty"`
	WIPLimit     *int     `json:"wip_limit,omitempty"`
}

type ReorderBucketsRequest struct {
	BucketIDs []int64 `json:"bucket_ids"`
}

type Board struct {
	Buckets []BucketWithTasks `json:"buckets"`
}

type BucketWithTasks struct {
	Bucket Bucket `json:"bucket"`
	Tasks  []Task `json:"tasks"`
}

package models

type Contact struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Company   string `json:"company"`
	Role      string `json:"role"`
	Notes     string `json:"notes"`
	Tags      string `json:"tags"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreateContactRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Company   string `json:"company,omitempty"`
	Role      string `json:"role,omitempty"`
	Notes     string `json:"notes,omitempty"`
	Tags      string `json:"tags,omitempty"`
}

type UpdateContactRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Company   *string `json:"company,omitempty"`
	Role      *string `json:"role,omitempty"`
	Notes     *string `json:"notes,omitempty"`
	Tags      *string `json:"tags,omitempty"`
}

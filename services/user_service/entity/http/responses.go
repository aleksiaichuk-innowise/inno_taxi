package http

type UserResp struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Phone string   `json:"phone"`
	Roles []string `json:"roles"`
}

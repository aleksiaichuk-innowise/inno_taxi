package http

type RegisterReq struct {
	Name     string `json:"name" binding:"required" validate:"required,min=2,max=32"`
	Email    string `json:"email" binding:"required" validate:"required,email"`
	Password string `json:"password" binding:"required" validate:"required,min=8,max=32"`
	Phone    string `json:"phone" binding:"required"  validate:"required,phone"`
	Role     string `json:"role" binding:"required" validate:"required,role"`
}

type VerifyCredentialsReq struct {
	Login    string `json:"login" binding:"required" validate:"required"`
	Password string `json:"password" binding:"required" validate:"required,min=8,max=32"`
}

type UpdateProfileReq struct {
	Name  string `json:"name" binding:"required" validate:"required,min=2,max=32"`
	Email string `json:"email" binding:"required" validate:"required,email"`
	Phone string `json:"phone" binding:"required" validate:"required,phone"`
}

type UpdatePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required" validate:"required,min=8,max=32"`
	NewPassword     string `json:"new_password" binding:"required" validate:"required,min=8,max=32"`
	ConfirmPassword string `json:"confirm_password" binding:"required" validate:"required,min=8,max=32,eqfield=NewPassword"`
}

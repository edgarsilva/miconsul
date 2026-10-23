package auth

// signinInput is the API sign-in request body.
type signinInput struct {
	Email      string `json:"email" form:"email"`
	Password   string `json:"password" form:"password"`
	RememberMe bool   `json:"remember_me" form:"remember_me"`
}

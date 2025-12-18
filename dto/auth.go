package dto

import "time"

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin finance customer"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,min=6,max=6"`
}

type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	OTP         string `json:"otp" binding:"required,min=6,max=6"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type APIResponse struct {
	Status string      `json:"status"`
	Code   int         `json:"code"`
	Data   interface{} `json:"data"`
}

type LoginResponseData struct {
	Token        string `json:"token,omitempty"`
	RequiresOTP  bool   `json:"requires_otp"`
	Message      string `json:"message,omitempty"`
	OTPExpiresIn int    `json:"otp_expires_in,omitempty"`
	User         struct {
		ID         uint      `json:"id"`
		Name       string    `json:"name"`
		Email      string    `json:"email"`
		Role       string    `json:"role"`
		Status     string    `json:"status"`
		IsVerified bool      `json:"is_verified"`
		LastLogin  *string   `json:"last_login,omitempty"`
		CreatedAt  time.Time `json:"created_at"`
	} `json:"user,omitempty"`
}

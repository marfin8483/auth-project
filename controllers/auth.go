package controllers

import (
	"auth-api/config"
	"auth-api/database"
	"auth-api/dto"
	"auth-api/middleware"
	"auth-api/models"
	"auth-api/utils"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthController struct {
	cfg *config.Config
	db  *gorm.DB
}

func NewAuthController(cfg *config.Config, db *gorm.DB) *AuthController {
	return &AuthController{cfg: cfg, db: db}
}

// Register - Mendaftarkan user baru
func (ac *AuthController) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Check if email already exists
	var existingUser models.User
	if err := ac.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		utils.ErrorResponse(c, 400, gin.H{"message": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to hash password"})
		return
	}

	// Create user
	user := models.User{
		Name:       req.Name,
		Email:      req.Email,
		Password:   hashedPassword,
		Role:       req.Role,
		Status:     "active",
		IsVerified: false,
	}

	if err := ac.db.Create(&user).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to create user"})
		return
	}

	// Generate OTP
	otp, err := utils.GenerateOTP(ac.cfg.Security.OTPLength)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to generate OTP"})
		return
	}

	// Store OTP in Redis
	err = database.StoreOTP(user.Email, otp, ac.cfg.Security.OTPExpiry)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to store OTP"})
		return
	}

	// Send OTP via email
	err = utils.SendOTPEmail(ac.cfg, user.Email, user.Name, otp, int(ac.cfg.Security.OTPExpiry.Minutes()))
	if err != nil {
		// Log error but don't fail registration
		fmt.Printf("⚠️ Failed to send OTP email: %v\n", err)
	}

	// Prepare response
	response := gin.H{
		"id":           user.ID,
		"name":         user.Name,
		"email":        user.Email,
		"role":         user.Role,
		"status":       user.Status,
		"is_verified":  user.IsVerified,
		"created_at":   user.CreatedAt,
		"message":      "Registration successful. Please check your email for OTP verification.",
		"requires_otp": true,
	}

	utils.SuccessResponse(c, 201, response)
}

// Login - Login user dengan password
func (ac *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Check if user is blocked
	blocked, err := database.IsBlocked(req.Email)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Internal server error"})
		return
	}
	if blocked {
		utils.ErrorResponse(c, 429, gin.H{
			"message": "Account temporarily blocked due to too many failed attempts. Please try again after 10 minutes.",
		})
		return
	}

	// Find user
	var user models.User
	if err := ac.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Untuk keamanan, tetap increment attempts meski user tidak ditemukan
			database.IncrementLoginAttempts(req.Email, ac.cfg)
			utils.ErrorResponse(c, 401, gin.H{
				"message": "Invalid email or password",
			})
			return
		}
		utils.ErrorResponse(c, 500, gin.H{"message": "Internal server error"})
		return
	}

	// Check if user is active
	if user.Status != "active" {
		utils.ErrorResponse(c, 401, gin.H{"message": "Account is not active"})
		return
	}

	// Verify password
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		// Increment failed login attempts
		database.IncrementLoginAttempts(req.Email, ac.cfg)

		// Get current attempts
		attempts, _ := database.CheckLoginAttempts(req.Email, ac.cfg)
		remaining := ac.cfg.Security.MaxLoginAttempts - attempts

		errorData := gin.H{
			"message": "Invalid email or password",
		}

		// Add attempts info if there are remaining attempts
		if remaining > 0 {
			errorData["attempts"] = attempts
			errorData["remaining"] = remaining
			errorData["message"] = fmt.Sprintf("Invalid email or password. %d attempts remaining.", remaining)
		} else {
			errorData["message"] = "Account blocked due to too many failed attempts. Please try again after 10 minutes."
		}

		utils.ErrorResponse(c, 401, errorData)
		return
	}

	// Reset login attempts on successful password verification
	database.ResetLoginAttempts(req.Email)

	// Check if user needs OTP verification
	if !user.IsVerified {
		// Generate and send OTP for unverified users
		otp, err := utils.GenerateOTP(ac.cfg.Security.OTPLength)
		if err != nil {
			utils.ErrorResponse(c, 500, gin.H{"message": "Failed to generate OTP"})
			return
		}

		// Store OTP in Redis
		err = database.StoreOTP(user.Email, otp, ac.cfg.Security.OTPExpiry)
		if err != nil {
			utils.ErrorResponse(c, 500, gin.H{"message": "Failed to store OTP"})
			return
		}

		// Send OTP via email
		err = utils.SendOTPEmail(ac.cfg, user.Email, user.Name, otp, int(ac.cfg.Security.OTPExpiry.Minutes()))
		if err != nil {
			fmt.Printf("⚠️ Failed to send OTP email: %v\n", err)
			utils.ErrorResponse(c, 500, gin.H{"message": "Failed to send OTP email"})
			return
		}

		// Update last login (temporary, before OTP verification)
		now := time.Now()
		user.LastLogin = &now
		ac.db.Save(&user)

		// Get OTP TTL
		ttl, _ := database.GetOTPTTL(user.Email)

		var lastLoginStr *string
		if user.LastLogin != nil {
			str := user.LastLogin.Format(time.RFC3339)
			lastLoginStr = &str
		}

		response := gin.H{
			"requires_otp":   true,
			"message":        "OTP has been sent to your email for verification",
			"otp_expires_in": int(ttl.Seconds()),
			"user": gin.H{
				"id":          user.ID,
				"name":        user.Name,
				"email":       user.Email,
				"role":        user.Role,
				"status":      user.Status,
				"is_verified": user.IsVerified,
				"last_login":  lastLoginStr,
				"created_at":  user.CreatedAt,
			},
		}

		utils.SuccessResponse(c, 200, response)
		return
	}

	// User is already verified, generate token immediately
	now := time.Now()
	user.LastLogin = &now
	ac.db.Save(&user)

	// Generate JWT token
	token, err := middleware.GenerateToken(user.ID, user.Email, user.Role, ac.cfg)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to generate token"})
		return
	}

	var lastLoginStr *string
	if user.LastLogin != nil {
		str := user.LastLogin.Format(time.RFC3339)
		lastLoginStr = &str
	}

	response := gin.H{
		"token":        token,
		"requires_otp": false,
		"user": gin.H{
			"id":          user.ID,
			"name":        user.Name,
			"email":       user.Email,
			"role":        user.Role,
			"status":      user.Status,
			"is_verified": user.IsVerified,
			"last_login":  lastLoginStr,
			"created_at":  user.CreatedAt,
		},
	}

	utils.SuccessResponse(c, 200, response)
}

// VerifyOTP - Verifikasi OTP untuk login
func (ac *AuthController) VerifyOTP(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := ac.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, 404, gin.H{"message": "User not found"})
			return
		}
		utils.ErrorResponse(c, 500, gin.H{"message": "Internal server error"})
		return
	}

	// Get OTP from Redis
	storedOTP, err := database.GetOTP(req.Email)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Internal server error"})
		return
	}

	// Check if OTP exists
	if storedOTP == "" {
		utils.ErrorResponse(c, 400, gin.H{"message": "OTP has expired or not found"})
		return
	}

	// Verify OTP
	if storedOTP != req.OTP {
		utils.ErrorResponse(c, 400, gin.H{"message": "Invalid OTP"})
		return
	}

	// OTP valid, delete from Redis
	database.DeleteOTP(req.Email)

	// Update user verification status
	user.IsVerified = true
	ac.db.Save(&user)

	// Generate JWT token
	token, err := middleware.GenerateToken(user.ID, user.Email, user.Role, ac.cfg)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to generate token"})
		return
	}

	// Update last login time
	now := time.Now()
	user.LastLogin = &now
	ac.db.Save(&user)

	var lastLoginStr *string
	if user.LastLogin != nil {
		str := user.LastLogin.Format(time.RFC3339)
		lastLoginStr = &str
	}

	response := gin.H{
		"token": token,
		"user": gin.H{
			"id":          user.ID,
			"name":        user.Name,
			"email":       user.Email,
			"role":        user.Role,
			"status":      user.Status,
			"is_verified": user.IsVerified,
			"last_login":  lastLoginStr,
			"created_at":  user.CreatedAt,
		},
	}

	utils.SuccessResponse(c, 200, response)
}

// ResendOTP - Kirim ulang OTP
func (ac *AuthController) ResendOTP(c *gin.Context) {
	var req dto.ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := ac.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, 404, gin.H{"message": "User not found"})
			return
		}
		utils.ErrorResponse(c, 500, gin.H{"message": "Internal server error"})
		return
	}

	// Check if user is active
	if user.Status != "active" {
		utils.ErrorResponse(c, 401, gin.H{"message": "Account is not active"})
		return
	}

	// Generate new OTP
	otp, err := utils.GenerateOTP(ac.cfg.Security.OTPLength)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to generate OTP"})
		return
	}

	// Store OTP in Redis
	err = database.StoreOTP(user.Email, otp, ac.cfg.Security.OTPExpiry)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to store OTP"})
		return
	}

	// Send OTP via email
	err = utils.SendOTPEmail(ac.cfg, user.Email, user.Name, otp, int(ac.cfg.Security.OTPExpiry.Minutes()))
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to send OTP email"})
		return
	}

	// Get TTL
	ttl, _ := database.GetOTPTTL(user.Email)

	response := gin.H{
		"message":        "New OTP has been sent to your email",
		"otp_expires_in": int(ttl.Seconds()),
	}

	utils.SuccessResponse(c, 200, response)
}

// ForgotPassword - Request reset password
func (ac *AuthController) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := ac.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return success even if user not found (for security)
			utils.SuccessResponse(c, 200, gin.H{
				"message": "If your email is registered, you will receive a password reset OTP",
			})
			return
		}
		utils.ErrorResponse(c, 500, gin.H{"message": "Internal server error"})
		return
	}

	// Check if user is active
	if user.Status != "active" {
		utils.ErrorResponse(c, 401, gin.H{"message": "Account is not active"})
		return
	}

	// Generate OTP for password reset
	otp, err := utils.GenerateOTP(ac.cfg.Security.OTPLength)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to generate OTP"})
		return
	}

	// Store OTP in Redis for password reset
	err = database.StorePasswordResetOTP(user.Email, otp, ac.cfg.Security.OTPExpiry)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to store OTP"})
		return
	}

	// Send password reset email
	err = utils.SendPasswordResetEmail(ac.cfg, user.Email, user.Name, otp, int(ac.cfg.Security.OTPExpiry.Minutes()))
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to send password reset email"})
		return
	}

	response := gin.H{
		"message": "Password reset OTP has been sent to your email",
	}

	utils.SuccessResponse(c, 200, response)
}

// ResetPassword - Reset password dengan OTP
func (ac *AuthController) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := ac.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, 404, gin.H{"message": "User not found"})
			return
		}
		utils.ErrorResponse(c, 500, gin.H{"message": "Internal server error"})
		return
	}

	// Get OTP from Redis
	storedOTP, err := database.GetPasswordResetOTP(req.Email)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Internal server error"})
		return
	}

	// Check if OTP exists
	if storedOTP == "" {
		utils.ErrorResponse(c, 400, gin.H{"message": "OTP has expired or not found"})
		return
	}

	// Verify OTP
	if storedOTP != req.OTP {
		utils.ErrorResponse(c, 400, gin.H{"message": "Invalid OTP"})
		return
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to hash password"})
		return
	}

	// Update password
	user.Password = hashedPassword
	user.IsVerified = true // Set as verified after password reset
	if err := ac.db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to update password"})
		return
	}

	// Delete OTP from Redis
	database.DeletePasswordResetOTP(req.Email)

	response := gin.H{
		"message": "Password has been reset successfully",
	}

	utils.SuccessResponse(c, 200, response)
}

// ChangePassword - Ubah password (butuh token JWT)
func (ac *AuthController) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, 401, gin.H{"message": "User not authenticated"})
		return
	}

	// Find user
	var user models.User
	if err := ac.db.First(&user, userID).Error; err != nil {
		utils.ErrorResponse(c, 404, gin.H{"message": "User not found"})
		return
	}

	// Verify old password
	if !utils.CheckPasswordHash(req.OldPassword, user.Password) {
		utils.ErrorResponse(c, 400, gin.H{"message": "Old password is incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to hash password"})
		return
	}

	// Update password
	user.Password = hashedPassword
	if err := ac.db.Save(&user).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to update password"})
		return
	}

	response := gin.H{
		"message": "Password has been changed successfully",
	}

	utils.SuccessResponse(c, 200, response)
}

// GetProfile - Get user profile
func (ac *AuthController) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, 401, gin.H{"message": "User not authenticated"})
		return
	}

	var user models.User
	if err := ac.db.First(&user, userID).Error; err != nil {
		utils.ErrorResponse(c, 404, gin.H{"message": "User not found"})
		return
	}

	var lastLoginStr *string
	if user.LastLogin != nil {
		str := user.LastLogin.Format(time.RFC3339)
		lastLoginStr = &str
	}

	response := gin.H{
		"id":          user.ID,
		"name":        user.Name,
		"email":       user.Email,
		"role":        user.Role,
		"customer_id": user.CustomerID,
		"status":      user.Status,
		"is_verified": user.IsVerified,
		"last_login":  lastLoginStr,
		"created_at":  user.CreatedAt,
		"updated_at":  user.UpdatedAt,
	}

	utils.SuccessResponse(c, 200, response)
}

// AdminGetUsers - Get all users (admin only)
func (ac *AuthController) AdminGetUsers(c *gin.Context) {
	var users []models.User
	if err := ac.db.Find(&users).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to fetch users"})
		return
	}

	// Hide passwords in response
	var response []gin.H
	for _, user := range users {
		var lastLoginStr *string
		if user.LastLogin != nil {
			str := user.LastLogin.Format(time.RFC3339)
			lastLoginStr = &str
		}

		response = append(response, gin.H{
			"id":          user.ID,
			"name":        user.Name,
			"email":       user.Email,
			"role":        user.Role,
			"customer_id": user.CustomerID,
			"status":      user.Status,
			"is_verified": user.IsVerified,
			"last_login":  lastLoginStr,
			"created_at":  user.CreatedAt,
			"updated_at":  user.UpdatedAt,
		})
	}

	utils.SuccessResponse(c, 200, gin.H{
		"users": response,
		"count": len(response),
	})
}

package main

import (
	"auth-api/config"
	"auth-api/controllers"
	"auth-api/database"
	"auth-api/middleware"
	"auth-api/utils"
	_ "fmt"
	"log"
	"time"
	_ "time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize MySQL database
	log.Println("🚀 Starting Authentication API...")
	log.Println("📦 Loading configuration...")

	if err := database.InitMySQL(cfg); err != nil {
		log.Fatalf("❌ Failed to connect to MySQL: %v", err)
	}

	// Initialize Redis
	if err := database.InitRedis(cfg); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode) // Use gin.DebugMode for development
	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Initialize controller
	authController := controllers.NewAuthController(cfg, database.DB)
	customerController := controllers.NewCustomerController(cfg, database.DB)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"service":   "auth-api",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// API Routes
	api := r.Group("/billapi/v2")
	{
		// Public routes
		api.POST("/register", authController.Register)
		api.POST("/login", authController.Login)
		api.POST("/verify-otp", authController.VerifyOTP)
		api.POST("/resend-otp", authController.ResendOTP)
		api.POST("/forgot-password", authController.ForgotPassword)
		api.POST("/reset-password", authController.ResetPassword)

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.JWTAuth(cfg))
		{
			protected.GET("/profile", authController.GetProfile)
			protected.POST("/change-password", authController.ChangePassword)

			// Customer routes
			customers := protected.Group("/customers")
			{
				customers.POST("", customerController.CreateCustomer)
				customers.GET("", customerController.GetCustomers)
				customers.GET("/stats", customerController.GetCustomerStats)
				customers.GET("/export", customerController.ExportCustomers)

				// Customer by ID routes
				customer := customers.Group("/:id")
				{
					customer.GET("", customerController.GetCustomerByID)
					customer.PUT("", customerController.UpdateCustomer)
					customer.DELETE("", customerController.DeleteCustomer)
					customer.PATCH("/balance", customerController.UpdateCustomerBalance)
					customer.GET("/history", customerController.GetCustomerHistory)
				}
			}

			// Admin routes
			admin := protected.Group("/admin")
			admin.Use(middleware.RoleMiddleware("admin"))
			{
				admin.GET("/users", authController.AdminGetUsers)
			}

			// Finance routes
			finance := protected.Group("/finance")
			finance.Use(middleware.RoleMiddleware("finance", "admin"))
			{
				// Add finance-specific routes here
				finance.GET("/dashboard", func(c *gin.Context) {
					utils.SuccessResponse(c, 200, gin.H{
						"message": "Welcome to Finance Dashboard",
					})
				})
			}
		}
	}

	// Print routes
	log.Println("🌐 Available Routes:")
	for _, route := range r.Routes() {
		log.Printf("   %-6s %s", route.Method, route.Path)
	}

	// Start server
	port := cfg.Server.Port
	log.Printf("✅ Server is running on http://localhost:%s", port)
	log.Println("📧 OTP Email Configuration:", cfg.SMTP.Username)
	log.Println("🔒 Security: OTP Length:", cfg.Security.OTPLength, "Expiry:", cfg.Security.OTPExpiry)
	log.Println("🔐 Max Login Attempts:", cfg.Security.MaxLoginAttempts)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}

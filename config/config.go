package config

import "time"

type Config struct {
	MySQL struct {
		Host     string
		Port     string
		User     string
		Password string
		Database string
	}
	Redis struct {
		Host     string
		Port     string
		Password string
		DB       int
	}
	JWT struct {
		Secret string
		Expiry time.Duration
	}
	Security struct {
		MaxLoginAttempts int
		BlockDuration    time.Duration
		OTPExpiry        time.Duration
		OTPLength        int
	}
	SMTP struct {
		Host     string
		Port     int
		Username string
		Password string
		From     string
	}
	Server struct {
		Port string
	}
}

func Load() *Config {
	cfg := &Config{}

	// MySQL Config
	cfg.MySQL.Host = "localhost"
	cfg.MySQL.Port = "3306"
	cfg.MySQL.User = "root"
	cfg.MySQL.Password = ""
	cfg.MySQL.Database = "billing_db"

	// Redis Config
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = "6379"
	cfg.Redis.Password = ""
	cfg.Redis.DB = 0

	// JWT Config
	cfg.JWT.Secret = "secret1029384756plmnjiuhbVGYTFCXZASDQWERZ"
	cfg.JWT.Expiry = 24 * time.Hour

	// Security Config
	cfg.Security.MaxLoginAttempts = 3
	cfg.Security.BlockDuration = 10 * time.Minute
	cfg.Security.OTPExpiry = 5 * time.Minute
	cfg.Security.OTPLength = 6

	// SMTP Config (sesuaikan dengan email provider Anda)
	cfg.SMTP.Host = "smtp.gmail.com"
	cfg.SMTP.Port = 587
	cfg.SMTP.Username = "devarf83@gmail.com"
	cfg.SMTP.Password = "brvezmypwnmsbgic"
	cfg.SMTP.From = "devarf83@gmail.com"

	// smtpHost := "smtp.gmail.com"
	// smtpPort := 587
	// smtpUsername := "devarf83@gmail.com" // Ganti dengan alamat email Anda
	// smtpPassword := "brvezmypwnmsbgic"          // Ganti dengan password email Anda

	// Server Config
	cfg.Server.Port = "8199"

	return cfg
}

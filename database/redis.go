package database

import (
	"auth-api/config"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client
var ctx = context.Background()

func InitRedis(cfg *config.Config) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		return err
	}

	log.Println("✅ Redis connected successfully")
	return nil
}

// Login attempts functions
func CheckLoginAttempts(email string, cfg *config.Config) (int, error) {
	key := fmt.Sprintf("login_attempts:%s", email)

	attempts, err := RedisClient.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return attempts, nil
}

func IncrementLoginAttempts(email string, cfg *config.Config) error {
	key := fmt.Sprintf("login_attempts:%s", email)

	attempts, err := RedisClient.Incr(ctx, key).Result()
	if err != nil {
		return err
	}

	if attempts == 1 {
		RedisClient.Expire(ctx, key, cfg.Security.BlockDuration)
	}

	if attempts >= int64(cfg.Security.MaxLoginAttempts) {
		blockKey := fmt.Sprintf("blocked:%s", email)
		RedisClient.Set(ctx, blockKey, "blocked", cfg.Security.BlockDuration)
	}

	return nil
}

func ResetLoginAttempts(email string) error {
	key := fmt.Sprintf("login_attempts:%s", email)
	blockKey := fmt.Sprintf("blocked:%s", email)

	// Delete both keys
	RedisClient.Del(ctx, key)
	RedisClient.Del(ctx, blockKey)

	return nil
}

func IsBlocked(email string) (bool, error) {
	key := fmt.Sprintf("blocked:%s", email)
	exists, err := RedisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// OTP functions
func StoreOTP(email, otp string, expiry time.Duration) error {
	key := fmt.Sprintf("otp:%s", email)
	return RedisClient.Set(ctx, key, otp, expiry).Err()
}

func GetOTP(email string) (string, error) {
	key := fmt.Sprintf("otp:%s", email)
	otp, err := RedisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return otp, err
}

func DeleteOTP(email string) error {
	key := fmt.Sprintf("otp:%s", email)
	return RedisClient.Del(ctx, key).Err()
}

func GetOTPTTL(email string) (time.Duration, error) {
	key := fmt.Sprintf("otp:%s", email)
	return RedisClient.TTL(ctx, key).Result()
}

// Password reset functions
func StorePasswordResetOTP(email, otp string, expiry time.Duration) error {
	key := fmt.Sprintf("pwd_reset:%s", email)
	return RedisClient.Set(ctx, key, otp, expiry).Err()
}

func GetPasswordResetOTP(email string) (string, error) {
	key := fmt.Sprintf("pwd_reset:%s", email)
	otp, err := RedisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return otp, err
}

func DeletePasswordResetOTP(email string) error {
	key := fmt.Sprintf("pwd_reset:%s", email)
	return RedisClient.Del(ctx, key).Err()
}

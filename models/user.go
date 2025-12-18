package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Name       string     `gorm:"size:100;not null" json:"name"`
	Email      string     `gorm:"size:100;uniqueIndex;not null" json:"email"`
	Password   string     `gorm:"size:255;not null" json:"-"`
	Role       string     `gorm:"type:ENUM('admin','finance','customer');default:'customer'" json:"role"`
	CustomerID *uint      `gorm:"null" json:"customer_id,omitempty"`
	Status     string     `gorm:"type:ENUM('active','inactive');default:'active'" json:"status"`
	IsVerified bool       `gorm:"default:false" json:"is_verified"`
	LastLogin  *time.Time `gorm:"null" json:"last_login,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	return nil
}

func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}

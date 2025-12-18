package models

import (
	"time"

	"gorm.io/gorm"
)

type Customer struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CustomerCode string    `gorm:"size:50;uniqueIndex;not null" json:"customer_code"`
	CompanyName  string    `gorm:"size:200;not null" json:"company_name"`
	ContactName  string    `gorm:"size:100" json:"contact_name"`
	Email        string    `gorm:"size:100" json:"email"`
	Phone        string    `gorm:"size:20" json:"phone"`
	Address      string    `gorm:"type:text" json:"address"`
	NPWP         string    `gorm:"size:25" json:"npwp"`
	Balance      float64   `gorm:"type:decimal(15,2);default:0" json:"balance"`
	Status       string    `gorm:"type:ENUM('active','suspended','terminated');default:'active'" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	UserID       uint      `gorm:"not null" json:"user_id"`
	User         User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user,omitempty"`
}

func (c *Customer) BeforeCreate(tx *gorm.DB) error {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	return nil
}

func (c *Customer) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

// CustomerHistory untuk tracking perubahan
type CustomerHistory struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CustomerID uint      `gorm:"not null" json:"customer_id"`
	Action     string    `gorm:"type:ENUM('create','update','delete','status_change','balance_update')" json:"action"`
	Changes    string    `gorm:"type:json" json:"changes"`
	ChangedBy  uint      `gorm:"not null" json:"changed_by"`
	CreatedAt  time.Time `json:"created_at"`
}

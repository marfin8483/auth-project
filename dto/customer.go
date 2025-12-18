package dto

import (
	"auth-api/models"
	"time"
)

type CustomerCreateRequest struct {
	CustomerCode string  `json:"customer_code" binding:"required,min=3,max=50"`
	CompanyName  string  `json:"company_name" binding:"required,min=2,max=200"`
	ContactName  string  `json:"contact_name" binding:"max=100"`
	Email        string  `json:"email" binding:"omitempty,email,max=100"`
	Phone        string  `json:"phone" binding:"omitempty,max=20"`
	Address      string  `json:"address" binding:"omitempty,max=500"`
	NPWP         string  `json:"npwp" binding:"omitempty,max=25"`
	Balance      float64 `json:"balance" binding:"omitempty,min=0"`
	Status       string  `json:"status" binding:"omitempty,oneof=active suspended terminated"`
}

type CustomerUpdateRequest struct {
	CompanyName string  `json:"company_name" binding:"omitempty,min=2,max=200"`
	ContactName string  `json:"contact_name" binding:"omitempty,max=100"`
	Email       string  `json:"email" binding:"omitempty,email,max=100"`
	Phone       string  `json:"phone" binding:"omitempty,max=20"`
	Address     string  `json:"address" binding:"omitempty,max=500"`
	NPWP        string  `json:"npwp" binding:"omitempty,max=25"`
	Balance     float64 `json:"balance" binding:"omitempty,min=0"`
	Status      string  `json:"status" binding:"omitempty,oneof=active suspended terminated"`
}

type CustomerResponse struct {
	ID           uint      `json:"id"`
	CustomerCode string    `json:"customer_code"`
	CompanyName  string    `json:"company_name"`
	ContactName  string    `json:"contact_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Address      string    `json:"address"`
	NPWP         string    `json:"npwp"`
	Balance      float64   `json:"balance"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	UserID       uint      `json:"user_id"`
	CreatedBy    string    `json:"created_by,omitempty"`
}

type CustomerListResponse struct {
	Customers []CustomerResponse `json:"customers"`
	Total     int64              `json:"total"`
	Page      int                `json:"page"`
	PageSize  int                `json:"page_size"`
	TotalPage int                `json:"total_page"`
}

type CustomerBalanceUpdateRequest struct {
	Amount float64 `json:"amount" binding:"required"`
	Type   string  `json:"type" binding:"required,oneof=deposit deduct"`
	Notes  string  `json:"notes" binding:"omitempty,max=255"`
}

type CustomerSearchRequest struct {
	Search    string `form:"search"`
	Status    string `form:"status"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=10"`
	SortBy    string `form:"sort_by,default=created_at"`
	SortOrder string `form:"sort_order,default=desc"`
}

// Helper function untuk convert model ke response
func ToCustomerResponse(customer models.Customer) CustomerResponse {
	return CustomerResponse{
		ID:           customer.ID,
		CustomerCode: customer.CustomerCode,
		CompanyName:  customer.CompanyName,
		ContactName:  customer.ContactName,
		Email:        customer.Email,
		Phone:        customer.Phone,
		Address:      customer.Address,
		NPWP:         customer.NPWP,
		Balance:      customer.Balance,
		Status:       customer.Status,
		CreatedAt:    customer.CreatedAt,
		UpdatedAt:    customer.UpdatedAt,
		UserID:       customer.UserID,
	}
}

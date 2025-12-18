package controllers

import (
	"auth-api/config"
	"auth-api/dto"
	"auth-api/models"
	"auth-api/utils"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CustomerController struct {
	cfg *config.Config
	db  *gorm.DB
}

func NewCustomerController(cfg *config.Config, db *gorm.DB) *CustomerController {
	return &CustomerController{cfg: cfg, db: db}
}

// CreateCustomer - Membuat customer baru
func (cc *CustomerController) CreateCustomer(c *gin.Context) {
	var req dto.CustomerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Get user ID from context (yang membuat customer)
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, 401, gin.H{"message": "User not authenticated"})
		return
	}

	// Check if customer code already exists
	var existingCustomer models.Customer
	if err := cc.db.Where("customer_code = ?", req.CustomerCode).First(&existingCustomer).Error; err == nil {
		utils.ErrorResponse(c, 400, gin.H{"message": "Customer code already exists"})
		return
	}

	// Create customer
	customer := models.Customer{
		CustomerCode: req.CustomerCode,
		CompanyName:  req.CompanyName,
		ContactName:  req.ContactName,
		Email:        req.Email,
		Phone:        req.Phone,
		Address:      req.Address,
		NPWP:         req.NPWP,
		Balance:      req.Balance,
		Status:       req.Status,
		UserID:       userID.(uint),
	}

	if customer.Status == "" {
		customer.Status = "active"
	}

	if err := cc.db.Create(&customer).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to create customer", "error": err.Error()})
		return
	}

	// Get user info for response
	var user models.User
	cc.db.First(&user, userID)

	// Create history record
	history := models.CustomerHistory{
		CustomerID: customer.ID,
		Action:     "create",
		Changes:    fmt.Sprintf(`{"customer_code":"%s","company_name":"%s"}`, customer.CustomerCode, customer.CompanyName),
		ChangedBy:  userID.(uint),
		CreatedAt:  time.Now(),
	}
	cc.db.Create(&history)

	response := dto.ToCustomerResponse(customer)
	response.CreatedBy = user.Name

	utils.SuccessResponse(c, 201, response)
}

// GetCustomers - Mendapatkan list customers dengan pagination dan filter
func (cc *CustomerController) GetCustomers(c *gin.Context) {
	var req dto.CustomerSearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Validasi pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}

	// Get user info for access control
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("role")

	// Build query
	query := cc.db.Model(&models.Customer{})

	// Apply role-based access control
	if userRole == "customer" {
		// Customer hanya bisa melihat data miliknya sendiri
		query = query.Where("user_id = ?", userID)
	} else if userRole == "finance" {
		// Finance bisa melihat semua kecuali yang di-terminated
		query = query.Where("status != ?", "terminated")
	}
	// Admin bisa melihat semua

	// Apply search filter
	if req.Search != "" {
		search := "%" + strings.ToLower(req.Search) + "%"
		query = query.Where("LOWER(customer_code) LIKE ? OR LOWER(company_name) LIKE ? OR LOWER(contact_name) LIKE ? OR LOWER(email) LIKE ?",
			search, search, search, search)
	}

	// Apply status filter
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Apply sorting
	sortOrder := "DESC"
	if req.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	validSortFields := map[string]bool{
		"customer_code": true,
		"company_name":  true,
		"balance":       true,
		"created_at":    true,
		"updated_at":    true,
	}

	sortField := "created_at"
	if validSortFields[req.SortBy] {
		sortField = req.SortBy
	}

	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	// Apply pagination
	offset := (req.Page - 1) * req.PageSize
	query = query.Offset(offset).Limit(req.PageSize)

	// Preload user data
	query = query.Preload("User")

	// Execute query
	var customers []models.Customer
	if err := query.Find(&customers).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to fetch customers", "error": err.Error()})
		return
	}

	// Convert to response
	var customerResponses []dto.CustomerResponse
	for _, customer := range customers {
		response := dto.ToCustomerResponse(customer)
		response.CreatedBy = customer.User.Name
		customerResponses = append(customerResponses, response)
	}

	totalPage := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPage++
	}

	response := dto.CustomerListResponse{
		Customers: customerResponses,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
		TotalPage: totalPage,
	}

	utils.SuccessResponse(c, 200, response)
}

// GetCustomerByID - Mendapatkan customer berdasarkan ID
func (cc *CustomerController) GetCustomerByID(c *gin.Context) {
	id := c.Param("id")
	customerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": "Invalid customer ID"})
		return
	}

	// Get user info for access control
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("role")

	var customer models.Customer
	query := cc.db.Preload("User")

	if userRole == "customer" {
		// Customer hanya bisa melihat data miliknya sendiri
		query = query.Where("id = ? AND user_id = ?", customerID, userID)
	} else {
		query = query.Where("id = ?", customerID)
	}

	if err := query.First(&customer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, 404, gin.H{"message": "Customer not found"})
			return
		}
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to fetch customer", "error": err.Error()})
		return
	}

	response := dto.ToCustomerResponse(customer)
	response.CreatedBy = customer.User.Name

	utils.SuccessResponse(c, 200, response)
}

// UpdateCustomer - Mengupdate data customer
func (cc *CustomerController) UpdateCustomer(c *gin.Context) {
	id := c.Param("id")
	customerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": "Invalid customer ID"})
		return
	}

	var req dto.CustomerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Get current user
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, 401, gin.H{"message": "User not authenticated"})
		return
	}

	// Find existing customer
	var customer models.Customer
	if err := cc.db.First(&customer, customerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, 404, gin.H{"message": "Customer not found"})
			return
		}
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to fetch customer", "error": err.Error()})
		return
	}

	// Check permission (admin/finance bisa update semua, customer hanya miliknya)
	userRole, _ := c.Get("role")
	if userRole == "customer" && customer.UserID != userID.(uint) {
		utils.ErrorResponse(c, 403, gin.H{"message": "Forbidden: You can only update your own customers"})
		return
	}

	// Track changes for history
	changes := make(map[string]interface{})
	oldValues := make(map[string]interface{})
	newValues := make(map[string]interface{})

	// Update fields if provided
	if req.CompanyName != "" && req.CompanyName != customer.CompanyName {
		oldValues["company_name"] = customer.CompanyName
		newValues["company_name"] = req.CompanyName
		customer.CompanyName = req.CompanyName
	}

	if req.ContactName != "" && req.ContactName != customer.ContactName {
		oldValues["contact_name"] = customer.ContactName
		newValues["contact_name"] = req.ContactName
		customer.ContactName = req.ContactName
	}

	if req.Email != "" && req.Email != customer.Email {
		oldValues["email"] = customer.Email
		newValues["email"] = req.Email
		customer.Email = req.Email
	}

	if req.Phone != "" && req.Phone != customer.Phone {
		oldValues["phone"] = customer.Phone
		newValues["phone"] = req.Phone
		customer.Phone = req.Phone
	}

	if req.Address != "" && req.Address != customer.Address {
		oldValues["address"] = customer.Address
		newValues["address"] = req.Address
		customer.Address = req.Address
	}

	if req.NPWP != "" && req.NPWP != customer.NPWP {
		oldValues["npwp"] = customer.NPWP
		newValues["npwp"] = req.NPWP
		customer.NPWP = req.NPWP
	}

	if req.Balance != 0 && req.Balance != customer.Balance {
		oldValues["balance"] = customer.Balance
		newValues["balance"] = req.Balance
		customer.Balance = req.Balance
	}

	if req.Status != "" && req.Status != customer.Status {
		oldValues["status"] = customer.Status
		newValues["status"] = req.Status
		customer.Status = req.Status
	}

	// Save changes
	if err := cc.db.Save(&customer).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to update customer", "error": err.Error()})
		return
	}

	// Create history record if there were changes
	if len(oldValues) > 0 {
		changes["old"] = oldValues
		changes["new"] = newValues

		history := models.CustomerHistory{
			CustomerID: customer.ID,
			Action:     "update",
			Changes:    fmt.Sprintf(`%s`, utils.ToJSON(changes)),
			ChangedBy:  userID.(uint),
			CreatedAt:  time.Now(),
		}
		cc.db.Create(&history)
	}

	response := dto.ToCustomerResponse(customer)
	utils.SuccessResponse(c, 200, response)
}

// DeleteCustomer - Menghapus customer (soft delete)
func (cc *CustomerController) DeleteCustomer(c *gin.Context) {
	id := c.Param("id")
	customerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": "Invalid customer ID"})
		return
	}

	// Get current user
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, 401, gin.H{"message": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("role")

	// Hanya admin yang bisa delete customer
	if userRole != "admin" {
		utils.ErrorResponse(c, 403, gin.H{"message": "Forbidden: Only admin can delete customers"})
		return
	}

	// Find customer
	var customer models.Customer
	if err := cc.db.First(&customer, customerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, 404, gin.H{"message": "Customer not found"})
			return
		}
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to fetch customer", "error": err.Error()})
		return
	}

	// Create history record before delete
	history := models.CustomerHistory{
		CustomerID: customer.ID,
		Action:     "delete",
		Changes:    fmt.Sprintf(`{"customer_code":"%s","company_name":"%s"}`, customer.CustomerCode, customer.CompanyName),
		ChangedBy:  userID.(uint),
		CreatedAt:  time.Now(),
	}
	cc.db.Create(&history)

	// Delete customer
	if err := cc.db.Delete(&customer).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to delete customer", "error": err.Error()})
		return
	}

	utils.SuccessResponse(c, 200, gin.H{
		"message":     "Customer deleted successfully",
		"customer_id": customer.ID,
	})
}

// UpdateCustomerBalance - Update balance customer (deposit/deduct)
func (cc *CustomerController) UpdateCustomerBalance(c *gin.Context) {
	id := c.Param("id")
	customerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": "Invalid customer ID"})
		return
	}

	var req dto.CustomerBalanceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": err.Error()})
		return
	}

	// Get current user
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, 401, gin.H{"message": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("role")

	// Hanya finance dan admin yang bisa update balance
	if userRole != "finance" && userRole != "admin" {
		utils.ErrorResponse(c, 403, gin.H{"message": "Forbidden: Only finance and admin can update balance"})
		return
	}

	// Find customer
	var customer models.Customer
	if err := cc.db.First(&customer, customerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, 404, gin.H{"message": "Customer not found"})
			return
		}
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to fetch customer", "error": err.Error()})
		return
	}

	// Calculate new balance
	oldBalance := customer.Balance
	var newBalance float64

	if req.Type == "deposit" {
		newBalance = customer.Balance + req.Amount
	} else if req.Type == "deduct" {
		// Check if balance is sufficient for deduction
		if customer.Balance < req.Amount {
			utils.ErrorResponse(c, 400, gin.H{"message": "Insufficient balance"})
			return
		}
		newBalance = customer.Balance - req.Amount
	}

	// Update balance
	customer.Balance = newBalance
	if err := cc.db.Save(&customer).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to update balance", "error": err.Error()})
		return
	}

	// Create history record
	history := models.CustomerHistory{
		CustomerID: customer.ID,
		Action:     "balance_update",
		Changes: fmt.Sprintf(`{"type":"%s","amount":%.2f,"old_balance":%.2f,"new_balance":%.2f,"notes":"%s"}`,
			req.Type, req.Amount, oldBalance, newBalance, req.Notes),
		ChangedBy: userID.(uint),
		CreatedAt: time.Now(),
	}
	cc.db.Create(&history)

	response := gin.H{
		"customer_id":   customer.ID,
		"customer_code": customer.CustomerCode,
		"company_name":  customer.CompanyName,
		"old_balance":   oldBalance,
		"new_balance":   newBalance,
		"amount":        req.Amount,
		"type":          req.Type,
		"notes":         req.Notes,
		"updated_at":    customer.UpdatedAt,
	}

	utils.SuccessResponse(c, 200, response)
}

// GetCustomerHistory - Mendapatkan history perubahan customer
func (cc *CustomerController) GetCustomerHistory(c *gin.Context) {
	id := c.Param("id")
	customerID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		utils.ErrorResponse(c, 400, gin.H{"message": "Invalid customer ID"})
		return
	}

	// Check if customer exists
	var customer models.Customer
	if err := cc.db.First(&customer, customerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, 404, gin.H{"message": "Customer not found"})
			return
		}
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to fetch customer", "error": err.Error()})
		return
	}

	// Get user info for access control
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("role")

	if userRole == "customer" && customer.UserID != userID.(uint) {
		utils.ErrorResponse(c, 403, gin.H{"message": "Forbidden: You can only view history of your own customers"})
		return
	}

	// Get query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Query history
	var history []models.CustomerHistory
	var total int64

	query := cc.db.Where("customer_id = ?", customerID)
	query.Count(&total)

	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&history).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to fetch history", "error": err.Error()})
		return
	}

	// Get user names for changed_by
	var historyResponses []gin.H
	for _, h := range history {
		var user models.User
		cc.db.Select("name").First(&user, h.ChangedBy)

		historyResponses = append(historyResponses, gin.H{
			"id":            h.ID,
			"action":        h.Action,
			"changes":       h.Changes,
			"changed_by":    user.Name,
			"changed_by_id": h.ChangedBy,
			"created_at":    h.CreatedAt,
		})
	}

	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}

	response := gin.H{
		"customer_id":   customerID,
		"customer_code": customer.CustomerCode,
		"company_name":  customer.CompanyName,
		"history":       historyResponses,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"total_page":    totalPage,
	}

	utils.SuccessResponse(c, 200, response)
}

// GetCustomerStats - Mendapatkan statistik customer
func (cc *CustomerController) GetCustomerStats(c *gin.Context) {
	// Get user info for access control
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("role")

	var stats gin.H

	if userRole == "admin" || userRole == "finance" {
		// Admin dan finance bisa melihat semua stats
		var totalCustomers int64
		var activeCustomers int64
		var suspendedCustomers int64
		var terminatedCustomers int64
		var totalBalance float64

		cc.db.Model(&models.Customer{}).Count(&totalCustomers)
		cc.db.Model(&models.Customer{}).Where("status = ?", "active").Count(&activeCustomers)
		cc.db.Model(&models.Customer{}).Where("status = ?", "suspended").Count(&suspendedCustomers)
		cc.db.Model(&models.Customer{}).Where("status = ?", "terminated").Count(&terminatedCustomers)
		cc.db.Model(&models.Customer{}).Select("COALESCE(SUM(balance), 0)").Row().Scan(&totalBalance)

		stats = gin.H{
			"total_customers":      totalCustomers,
			"active_customers":     activeCustomers,
			"suspended_customers":  suspendedCustomers,
			"terminated_customers": terminatedCustomers,
			"total_balance":        totalBalance,
			"average_balance":      totalBalance / float64(totalCustomers),
		}
	} else if userRole == "customer" {
		// Customer hanya bisa melihat stats miliknya sendiri
		var totalCustomers int64
		var activeCustomers int64
		var totalBalance float64

		cc.db.Model(&models.Customer{}).Where("user_id = ?", userID).Count(&totalCustomers)
		cc.db.Model(&models.Customer{}).Where("user_id = ? AND status = ?", userID, "active").Count(&activeCustomers)
		cc.db.Model(&models.Customer{}).Where("user_id = ?", userID).
			Select("COALESCE(SUM(balance), 0)").Row().Scan(&totalBalance)

		stats = gin.H{
			"total_customers":  totalCustomers,
			"active_customers": activeCustomers,
			"total_balance":    totalBalance,
			"average_balance":  totalBalance / float64(totalCustomers),
		}
	}

	utils.SuccessResponse(c, 200, stats)
}

// ExportCustomers - Export customers ke CSV
func (cc *CustomerController) ExportCustomers(c *gin.Context) {
	// Get user info for access control
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("role")

	var customers []models.Customer
	query := cc.db.Preload("User")

	if userRole == "customer" {
		query = query.Where("user_id = ?", userID)
	} else if userRole == "finance" {
		query = query.Where("status != ?", "terminated")
	}

	if err := query.Find(&customers).Error; err != nil {
		utils.ErrorResponse(c, 500, gin.H{"message": "Failed to fetch customers for export"})
		return
	}

	// Generate CSV
	csvData := "ID,Customer Code,Company Name,Contact Name,Email,Phone,Address,NPWP,Balance,Status,Created By,Created At\n"

	for _, customer := range customers {
		csvData += fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s,%s,%.2f,%s,%s,%s\n",
			customer.ID,
			customer.CustomerCode,
			customer.CompanyName,
			customer.ContactName,
			customer.Email,
			customer.Phone,
			strings.ReplaceAll(customer.Address, ",", ";"),
			customer.NPWP,
			customer.Balance,
			customer.Status,
			customer.User.Name,
			customer.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	// Set response headers for file download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=customers_export.csv")
	c.Header("Content-Length", fmt.Sprintf("%d", len(csvData)))

	c.String(200, csvData)
}

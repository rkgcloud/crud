package models

import "gorm.io/gorm"

// User represents a user in the database
type User struct {
	gorm.Model
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email" gorm:"unique"`
	Phone string `json:"phone" binding:"required"`
}

// Account represents an account in the database
type Account struct {
	gorm.Model
	UserID  uint    `json:"user_id" binding:"required"`
	Name    string  `json:"name" binding:"required"`
	Balance float64 `json:"balance" gorm:"default:0.00"`
}

package models

import "gorm.io/gorm"

// User represents a user in the database
type User struct {
	gorm.Model
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email" gorm:"unique"`
	Age   int    `json:"age" binding:"required"`
}

package models

import "gorm.io/gorm"

// User represents a user in the database
type User struct {
	gorm.Model
	Name  string `json:"name" binding:"required,max=100" gorm:"type:varchar(100);not null"`
	Email string `json:"email" binding:"required,email,max=255" gorm:"type:varchar(255);uniqueIndex;not null"`
	Phone string `json:"phone" binding:"required,max=20" gorm:"type:varchar(20);not null"`
}

// Account represents an account in the database
type Account struct {
	gorm.Model
	UserID  uint    `json:"user_id" binding:"required" gorm:"not null;index"`
	Name    string  `json:"name" binding:"required,max=100" gorm:"type:varchar(100);not null"`
	Balance float64 `json:"balance" binding:"min=0,max=999999999.99" gorm:"type:decimal(12,2);default:0.00;not null"`
	User    User    `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

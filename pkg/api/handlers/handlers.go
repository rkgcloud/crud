package handlers

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/rkgcloud/crud/pkg/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Index Renders the index page
func Index(c *gin.Context, db *gorm.DB) {

	var users []models.User
	pageData := gin.H{
		"title": "Users",
	}
	if err := db.Find(&users).Error; err != nil {
		c.String(http.StatusInternalServerError, "Could not retrieve users")
	}

	pageData["Records"] = users
	c.HTML(http.StatusOK, "index.html", pageData)
}

// CreateUser creates a new user in the database
func CreateUser(c *gin.Context, db *gorm.DB) {
	user := models.User{
		Name:  c.PostForm("name"),
		Email: c.PostForm("email"),
		Phone: c.PostForm("phone"),
	}
	user.ID = newAccountNumber()
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}

	c.Redirect(http.StatusFound, "/")
}

// GetUsers retrieves all users from the database
func GetUsers(c *gin.Context, db *gorm.DB) {
	users, err := getUsers(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve users"})
	}

	c.JSON(http.StatusOK, users)
}

func getUsers(db *gorm.DB) ([]models.User, error) {
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetUser retrieves a single user by ID
func GetUser(c *gin.Context, db *gorm.DB) {
	var user models.User
	id := c.Param("id")
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// UpdateUser updates a user's information
func UpdateUser(c *gin.Context, db *gorm.DB) {
	var user models.User
	id := c.Param("id")
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.Save(&user)
	c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user from the database
func DeleteUser(c *gin.Context, db *gorm.DB) {
	var user models.User
	id := c.Param("id")
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	db.Delete(&user)
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// CreateAccount creates a new account in the database
func CreateAccount(c *gin.Context, db *gorm.DB) {
	var account models.Account
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := db.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create account"})
		return
	}
}

// GetAccounts retrieves all accounts from the database
func GetAccounts(c *gin.Context, db *gorm.DB) {
	var accounts []models.Account
	if err := db.Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve accounts"})
	}
	c.JSON(http.StatusOK, accounts)
}

func newAccountNumber() uint {
	ns := rand.NewSource(time.Now().UnixNano())
	rand.New(ns)                             // Seed the random number generator
	randomNumber := 10000 + rand.Intn(90000) // Generates a random number between 10000 and 99999
	return uint(randomNumber)
}

package controllers

// https://gin-gonic.com/en/blog/news/how-to-build-one-effective-middleware/
// https://www.youtube.com/watch?v=2GSBlB8HFDw
import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rkgcloud/crud/pkg/auth"
	"github.com/rkgcloud/crud/pkg/models"
	"github.com/rkgcloud/crud/pkg/session"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"
)

const (
	imageLogoPath    = "./images/demo.png"
	maxNameLength    = 100
	maxEmailLength   = 255
	maxPhoneLength   = 20
	maxBalanceValue  = 999999999.99
	stateTokenLength = 32
)

var googleOauthConfig *oauth2.Config

// Input validation functions
func validateName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(name) > maxNameLength {
		return fmt.Errorf("name must be less than %d characters", maxNameLength)
	}
	return nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if len(email) == 0 {
		return fmt.Errorf("email is required")
	}
	if len(email) > maxEmailLength {
		return fmt.Errorf("email must be less than %d characters", maxEmailLength)
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func validatePhone(phone string) error {
	phone = strings.TrimSpace(phone)
	if len(phone) == 0 {
		return fmt.Errorf("phone is required")
	}
	if len(phone) > maxPhoneLength {
		return fmt.Errorf("phone must be less than %d characters", maxPhoneLength)
	}
	return nil
}

func validateBalance(balance float64) error {
	if balance < 0 {
		return fmt.Errorf("balance cannot be negative")
	}
	if balance > maxBalanceValue {
		return fmt.Errorf("balance cannot exceed %.2f", maxBalanceValue)
	}
	return nil
}

func init() {
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/auth/callback",
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),     // Get from Google Cloud Console
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"), // Get from Google Cloud Console
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.profile", "https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}
}

func LoginPage(c *gin.Context) {
	// Check if user is already logged in
	if session.IsLoggedIn(c) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":       "Login",
		"CompanyLogo": imageLogoPath,
	})
}

// generateStateToken creates a secure random state token
func generateStateToken() (string, error) {
	bytes := make([]byte, stateTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HandleGoogleLogin redirects to Google for authentication
func HandleGoogleLogin(c *gin.Context) {
	stateToken, err := generateStateToken()
	if err != nil {
		log.Printf("Failed to generate state token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Store state token in session for verification
	if err := session.SetStateToken(c, stateToken); err != nil {
		log.Printf("Failed to store state token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	url := googleOauthConfig.AuthCodeURL(stateToken, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusFound, url)
}

// HandleGoogleCallback handles the response from Google
func HandleGoogleCallback(c *gin.Context) {
	// Verify state token to prevent CSRF attacks
	receivedState := c.Query("state")
	storedState := session.GetStateToken(c)

	if receivedState == "" || storedState == "" || receivedState != storedState {
		log.Printf("OAuth state token mismatch or missing from IP: %s", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state token"})
		return
	}

	// Clean up state token
	if err := session.DeleteStateToken(c); err != nil {
		log.Printf("Failed to delete state token: %v", err)
	}

	code := c.Query("code")
	if code == "" {
		log.Printf("Missing authorization code from IP: %s", c.ClientIP())
		c.Redirect(http.StatusFound, "/login")
		return
	}

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("OAuth token exchange failed from IP %s: %v", c.ClientIP(), err)
		c.Redirect(http.StatusFound, "/login")
		return
	}

	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("Failed to get user info from Google from IP %s: %v", c.ClientIP(), err)
		c.Redirect(http.StatusFound, "/login")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Google API returned non-200 status: %d from IP: %s", resp.StatusCode, c.ClientIP())
		c.Redirect(http.StatusFound, "/login")
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body from IP %s: %v", c.ClientIP(), err)
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var profile auth.LoggedInUser
	if err := json.Unmarshal(data, &profile); err != nil {
		log.Printf("Failed to unmarshal user profile from IP %s: %v", c.ClientIP(), err)
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Validate required fields
	if profile.ID == "" || profile.Email == "" {
		log.Printf("Invalid user profile received from IP %s: missing required fields", c.ClientIP())
		c.Redirect(http.StatusFound, "/login")
		return
	}

	log.Printf("User %s (%s) logged in from IP: %s", profile.Name, profile.Email, c.ClientIP())

	// SetLoggedInUser user profile in the session
	if err := session.SetLoggedInUser(c, &profile); err != nil {
		log.Printf("Failed to save user profile in session: %s\n", err)
		c.Redirect(http.StatusFound, "/")
	}

	c.Redirect(http.StatusFound, "/")
}

func Logout(c *gin.Context) {
	if err := session.DeleteLoggedInUser(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete user"})
		return
	}
	c.Redirect(http.StatusFound, "/")
}

// Index Renders the index page
func Index(c *gin.Context, db *gorm.DB) {
	var profile auth.LoggedInUser
	user, exist := c.Get("loggedInUser")
	if exist {
		profile = user.(auth.LoggedInUser)
		log.Printf("User profile from login page: %s\n", profile.Name)
	}

	var users []models.User
	pageData := gin.H{
		"title": "Users",
	}
	if err := db.Find(&users).Error; err != nil {
		c.String(http.StatusInternalServerError, "Could not retrieve users")
	}

	pageData["Records"] = users
	c.HTML(http.StatusOK, "index.html", pageData)
	c.HTML(http.StatusOK, "layout.html", gin.H{
		"title":      "Users",
		"IsLoggedIn": exist,
		"Name":       profile.Name,
		"Email":      profile.Email,
		"Phone":      profile.Phone,
		"Picture":    profile.Picture,
	})
}

// CreateUser creates a new user in the database
func CreateUser(c *gin.Context, db *gorm.DB) {
	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(c.PostForm("email"))
	phone := strings.TrimSpace(c.PostForm("phone"))

	// Validate input
	if err := validateName(name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateEmail(email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validatePhone(phone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		Name:  name,
		Email: email,
		Phone: phone,
	}
	user.ID = newAccountNumber()

	if err := db.Create(&user).Error; err != nil {
		log.Printf("Failed to create user from IP %s: %v", c.ClientIP(), err)
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		}
		return
	}

	log.Printf("User created: %s (%s) by IP: %s", user.Name, user.Email, c.ClientIP())
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

	// Parse the update data into a temporary struct
	var updateData models.User
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input if provided
	if updateData.Name != "" {
		if err := validateName(updateData.Name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user.Name = strings.TrimSpace(updateData.Name)
	}

	if updateData.Email != "" {
		if err := validateEmail(updateData.Email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user.Email = strings.TrimSpace(updateData.Email)
	}

	if updateData.Phone != "" {
		if err := validatePhone(updateData.Phone); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user.Phone = strings.TrimSpace(updateData.Phone)
	}

	if err := db.Save(&user).Error; err != nil {
		log.Printf("Failed to update user %d from IP %s: %v", user.ID, c.ClientIP(), err)
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update user"})
		}
		return
	}

	log.Printf("User updated: %s (%s) by IP: %s", user.Name, user.Email, c.ClientIP())
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
	userid, err := strconv.ParseUint(c.PostForm("user-id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	account := models.Account{
		UserID: uint(userid),
		Name:   c.PostForm("name"),
	}
	if c.PostForm("balance") != "" {
		account.Balance, err = strconv.ParseFloat(c.PostForm("balance"), 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid balance data"})
			return
		}
	}
	if err := db.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create account"})
		return
	}
	c.Redirect(http.StatusFound, "/accounts")
}

// GetAccounts retrieves all accounts from the database
func GetAccounts(c *gin.Context, db *gorm.DB) {
	var profile auth.LoggedInUser
	user, exist := c.Get("loggedInUser")
	if exist {
		profile = user.(auth.LoggedInUser)
		log.Printf("User profile from login page: %s\n", profile.Name)
	}

	var accounts []models.Account
	pageData := gin.H{
		"title": "Users",
	}

	if err := db.Order("created_at DESC").Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve accounts"})
	}
	pageData["Accounts"] = accounts
	c.HTML(http.StatusOK, "accounts.html", pageData)
	c.HTML(http.StatusOK, "layout.html", gin.H{
		"title":      "Users",
		"IsLoggedIn": exist,
		"Name":       profile.Name,
		"Email":      profile.Email,
		"Phone":      profile.Phone,
		"Picture":    profile.Picture,
	})
}

func UpdateAccount(c *gin.Context, db *gorm.DB) {
	accountIDValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account id"})
		return
	}
	if accountIDValue == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account id"})
		return
	}
	userid, err := strconv.ParseUint(c.PostForm("user-id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	account := models.Account{
		UserID: uint(userid),
		Name:   c.PostForm("name"),
	}

	account.Balance, err = strconv.ParseFloat(c.PostForm("balance"), 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid balance data"})
		return
	}

	account.ID = uint(accountIDValue)
	if err := db.Save(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update account"})
		return
	}
	c.Redirect(http.StatusFound, "/accounts")
}

func newAccountNumber() uint {
	// Use crypto/rand for better security
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based approach if crypto/rand fails
		log.Printf("Failed to generate secure random number: %v", err)
		ns := time.Now().UnixNano()
		return uint(10000 + (ns % 90000))
	}

	// Convert bytes to uint and ensure it's in range 10000-99999
	randomNumber := uint(bytes[0])<<24 | uint(bytes[1])<<16 | uint(bytes[2])<<8 | uint(bytes[3])
	return 10000 + (randomNumber % 90000)
}

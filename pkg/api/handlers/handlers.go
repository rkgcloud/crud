package handlers

// https://gin-gonic.com/en/blog/news/how-to-build-one-effective-middleware/
// https://www.youtube.com/watch?v=2GSBlB8HFDw
import (
	"context"
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/rkgcloud/crud/pkg/api/session"
	"github.com/rkgcloud/crud/pkg/auth"
	"github.com/rkgcloud/crud/pkg/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	imageLogoPath = "./images/demo.png"
)

var googleOauthConfig *oauth2.Config

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
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":       "Login",
		"CompanyLogo": imageLogoPath,
	})
	user, exist := c.Get("loggedInUser")
	if exist {
		profile := user.(auth.LoggedInUser)
		log.Printf("User profile from login page: %s\n", profile.Name)
	}

	c.HTML(http.StatusOK, "layout.html", gin.H{
		"title": "Login",
	})
}

// HandleGoogleLogin redirects to Google for authentication
func HandleGoogleLogin(c *gin.Context) {
	url := googleOauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusFound, url)
}

// HandleGoogleCallback handles the response from Google
func HandleGoogleCallback(c *gin.Context) {
	code := c.Query("code")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("oauthConf.Exchange() failed: %s\n", err)
		c.Redirect(http.StatusFound, "/")
		return
	}

	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("client.Get() failed: %s\n", err)
		c.Redirect(http.StatusFound, "/")
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var profile auth.LoggedInUser
	if err := json.Unmarshal(data, &profile); err != nil {
		log.Printf("json.Unmarshal() failed: %s\n", err)
		c.Redirect(http.StatusFound, "/")
		return
	}

	log.Printf("Auth profile: %s\n", string(data))

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

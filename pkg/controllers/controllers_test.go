package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rkgcloud/crud/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.Account{})
	require.NoError(t, err)

	return db
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"valid name", "John Doe", false},
		{"empty name", "", true},
		{"whitespace only", "   ", true},
		{"too long name", strings.Repeat("a", 101), true},
		{"valid with spaces", "  John Doe  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"valid email", "test@example.com", false},
		{"empty email", "", true},
		{"invalid format - no @", "testexample.com", true},
		{"invalid format - no domain", "test@", true},
		{"too long email", strings.Repeat("a", 250) + "@example.com", true},
		{"valid with spaces", "  test@example.com  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"valid phone", "123-456-7890", false},
		{"empty phone", "", true},
		{"too long phone", strings.Repeat("1", 21), true},
		{"valid with spaces", "  123-456-7890  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePhone(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBalance(t *testing.T) {
	tests := []struct {
		name        string
		input       float64
		expectError bool
	}{
		{"valid balance", 100.50, false},
		{"zero balance", 0.0, false},
		{"negative balance", -10.0, true},
		{"too large balance", 1000000000.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBalance(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter()

	router.POST("/users", func(c *gin.Context) {
		CreateUser(c, db)
	})

	tests := []struct {
		name           string
		formData       url.Values
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name: "valid user creation",
			formData: url.Values{
				"name":  {"John Doe"},
				"email": {"john@example.com"},
				"phone": {"123-456-7890"},
			},
			expectedStatus: http.StatusFound, // Redirect
			checkResponse: func(t *testing.T, body string) {
				// Verify user was created in database
				var user models.User
				err := db.Where("email = ?", "john@example.com").First(&user).Error
				assert.NoError(t, err)
				assert.Equal(t, "John Doe", user.Name)
			},
		},
		{
			name: "invalid email",
			formData: url.Values{
				"name":  {"John Doe"},
				"email": {"invalid-email"},
				"phone": {"123-456-7890"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "invalid email format")
			},
		},
		{
			name: "missing name",
			formData: url.Values{
				"email": {"john@example.com"},
				"phone": {"123-456-7890"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "name is required")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean database for each test
			db.Exec("DELETE FROM users")

			body := strings.NewReader(tt.formData.Encode())
			req, _ := http.NewRequest("POST", "/users", body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String())
			}
		})
	}
}

func TestGetUsers(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter()

	router.GET("/users", func(c *gin.Context) {
		GetUsers(c, db)
	})

	// Create test users
	users := []models.User{
		{Name: "John Doe", Email: "john@example.com", Phone: "123-456-7890"},
		{Name: "Jane Smith", Email: "jane@example.com", Phone: "098-765-4321"},
	}

	for _, user := range users {
		user.ID = newAccountNumber()
		db.Create(&user)
	}

	req, _ := http.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseUsers []models.User
	err := json.Unmarshal(w.Body.Bytes(), &responseUsers)
	assert.NoError(t, err)
	assert.Len(t, responseUsers, 2)
}

func TestGetUser(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter()

	router.GET("/users/:id", func(c *gin.Context) {
		GetUser(c, db)
	})

	// Create test user
	user := models.User{
		Name:  "John Doe",
		Email: "john@example.com",
		Phone: "123-456-7890",
	}
	user.ID = 12345
	db.Create(&user)

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "existing user",
			userID:         "12345",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var responseUser models.User
				err := json.Unmarshal([]byte(body), &responseUser)
				assert.NoError(t, err)
				assert.Equal(t, "John Doe", responseUser.Name)
			},
		},
		{
			name:           "non-existing user",
			userID:         "99999",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "User not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/users/"+tt.userID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String())
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter()

	router.PUT("/users/:id", func(c *gin.Context) {
		UpdateUser(c, db)
	})

	// Create test user
	user := models.User{
		Name:  "John Doe",
		Email: "john@example.com",
		Phone: "123-456-7890",
	}
	user.ID = 12345
	db.Create(&user)

	updateData := models.User{
		Name:  "John Updated",
		Email: "john.updated@example.com",
		Phone: "111-222-3333",
	}

	jsonData, _ := json.Marshal(updateData)
	req, _ := http.NewRequest("PUT", "/users/12345", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify update in database
	var updatedUser models.User
	db.First(&updatedUser, 12345)
	assert.Equal(t, "John Updated", updatedUser.Name)
	assert.Equal(t, "john.updated@example.com", updatedUser.Email)
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter()

	router.DELETE("/users/:id", func(c *gin.Context) {
		DeleteUser(c, db)
	})

	// Create test user
	user := models.User{
		Name:  "John Doe",
		Email: "john@example.com",
		Phone: "123-456-7890",
	}
	user.ID = 12345
	db.Create(&user)

	req, _ := http.NewRequest("DELETE", "/users/12345", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify deletion in database
	var deletedUser models.User
	err := db.First(&deletedUser, 12345).Error
	assert.Error(t, err) // Should not find the user
}

func TestNewAccountNumber(t *testing.T) {
	// Test that newAccountNumber generates numbers in the expected range
	for i := 0; i < 100; i++ {
		num := newAccountNumber()
		assert.GreaterOrEqual(t, num, uint(10000))
		assert.LessOrEqual(t, num, uint(99999))
	}

	// Test that it generates different numbers (not a guarantee, but very likely)
	numbers := make(map[uint]bool)
	for i := 0; i < 50; i++ {
		num := newAccountNumber()
		numbers[num] = true
	}
	// Should have generated multiple different numbers
	assert.Greater(t, len(numbers), 1)
}

func BenchmarkNewAccountNumber(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newAccountNumber()
	}
}

func BenchmarkValidateEmail(b *testing.B) {
	email := "test@example.com"
	for i := 0; i < b.N; i++ {
		validateEmail(email)
	}
}

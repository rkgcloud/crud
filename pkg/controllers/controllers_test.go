package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/rkgcloud/crud/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test helper functions
func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&models.User{}, &models.Account{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test-session", store))
	return r
}

func createTestUser(db *gorm.DB) *models.User {
	user := &models.User{
		Name:  "Test User",
		Email: "test@example.com",
		Phone: "123-456-7890",
	}
	user.ID = 12345
	db.Create(user)
	return user
}

func createTestAccount(db *gorm.DB, userID uint) *models.Account {
	account := &models.Account{
		UserID:  userID,
		Name:    "Test Account",
		Balance: 100.50,
	}
	db.Create(account)
	return account
}

func createJSONRequest(method, url string, body interface{}) (*http.Request, error) {
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func createFormRequest(method, url string, formData map[string]string) (*http.Request, error) {
	var body bytes.Buffer
	for key, value := range formData {
		if body.Len() > 0 {
			body.WriteString("&")
		}
		body.WriteString(fmt.Sprintf("%s=%s", key, value))
	}
	req, err := http.NewRequest(method, url, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func performRequest(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateUser(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)

	r := setupTestRouter()
	r.POST("/users", func(c *gin.Context) {
		CreateUser(c, db)
	})

	t.Run("ValidUser", func(t *testing.T) {
		formData := map[string]string{
			"name":  "John Doe",
			"email": "john@example.com",
			"phone": "123-456-7890",
		}

		req, err := createFormRequest("POST", "/users", formData)
		require.NoError(t, err)

		w := performRequest(r, req)
		assert.Equal(t, http.StatusFound, w.Code)

		// Verify user was created in database
		var user models.User
		err = db.Where("email = ?", "john@example.com").First(&user).Error
		assert.NoError(t, err)
		assert.Equal(t, "John Doe", user.Name)
		assert.Equal(t, "john@example.com", user.Email)
		assert.Equal(t, "123-456-7890", user.Phone)
		assert.NotZero(t, user.ID)
	})

	t.Run("MissingFields", func(t *testing.T) {
		formData := map[string]string{
			"name": "Incomplete User",
			// Missing email and phone
		}

		req, err := createFormRequest("POST", "/users", formData)
		require.NoError(t, err)

		w := performRequest(r, req)

		// Should still redirect even with incomplete data
		// The actual validation would happen at the database level
		assert.Equal(t, http.StatusFound, w.Code)
	})
}

func TestGetUsers(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)

	r := setupTestRouter()
	r.GET("/users", func(c *gin.Context) {
		GetUsers(c, db)
	})

	// Create test users
	users := []models.User{
		{Name: "User 1", Email: "user1@example.com", Phone: "111-111-1111"},
		{Name: "User 2", Email: "user2@example.com", Phone: "222-222-2222"},
		{Name: "User 3", Email: "user3@example.com", Phone: "333-333-3333"},
	}

	for i, user := range users {
		user.ID = uint(1000 + i)
		err := db.Create(&user).Error
		require.NoError(t, err)
	}

	req := httptest.NewRequest("GET", "/users", nil)
	w := performRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.User
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(response), 3)
}

func TestGetUser(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)

	r := setupTestRouter()
	r.GET("/users/:id", func(c *gin.Context) {
		GetUser(c, db)
	})

	// Create a test user
	user := createTestUser(db)

	t.Run("UserExists", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/users/%d", user.ID), nil)
		w := performRequest(r, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.User
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, response.ID)
		assert.Equal(t, user.Name, response.Name)
		assert.Equal(t, user.Email, response.Email)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/99999", nil)
		w := performRequest(r, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User not found", response["error"])
	})

	t.Run("InvalidID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/invalid", nil)
		w := performRequest(r, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUpdateUser(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)

	r := setupTestRouter()
	r.PUT("/users/:id", func(c *gin.Context) {
		UpdateUser(c, db)
	})

	// Create a test user
	user := createTestUser(db)

	t.Run("ValidUpdate", func(t *testing.T) {
		updateData := models.User{
			Name:  "Updated Name",
			Email: "updated@example.com",
			Phone: "999-999-9999",
		}

		req, err := createJSONRequest("PUT", fmt.Sprintf("/users/%d", user.ID), updateData)
		require.NoError(t, err)

		w := performRequest(r, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var response models.User
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", response.Name)
		assert.Equal(t, "updated@example.com", response.Email)
		assert.Equal(t, "999-999-9999", response.Phone)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		updateData := models.User{Name: "Won't Update"}

		req, err := createJSONRequest("PUT", "/users/99999", updateData)
		require.NoError(t, err)

		w := performRequest(r, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("PUT", fmt.Sprintf("/users/%d", user.ID), nil)
		req.Header.Set("Content-Type", "application/json")

		w := performRequest(r, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeleteUser(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)

	r := setupTestRouter()
	r.DELETE("/users/:id", func(c *gin.Context) {
		DeleteUser(c, db)
	})

	t.Run("ValidDelete", func(t *testing.T) {
		// Create a user to delete
		user := createTestUser(db)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/users/%d", user.ID), nil)
		w := performRequest(r, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User deleted", response["message"])

		// Verify user is deleted (soft delete)
		var deletedUser models.User
		err = db.First(&deletedUser, user.ID).Error
		assert.Error(t, err) // Should not find the user
	})

	t.Run("UserNotFound", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/users/99999", nil)
		w := performRequest(r, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User not found", response["error"])
	})
}

func TestCreateAccount(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)

	r := setupTestRouter()
	r.POST("/accounts", func(c *gin.Context) {
		CreateAccount(c, db)
	})

	// Create a test user first
	user := createTestUser(db)

	t.Run("ValidAccount", func(t *testing.T) {
		formData := map[string]string{
			"user-id": strconv.Itoa(int(user.ID)),
			"name":    "Test Account",
			"balance": "500.75",
		}

		req, err := createFormRequest("POST", "/accounts", formData)
		require.NoError(t, err)

		w := performRequest(r, req)
		assert.Equal(t, http.StatusFound, w.Code)

		// Verify account was created
		var account models.Account
		err = db.Where("name = ?", "Test Account").First(&account).Error
		assert.NoError(t, err)
		assert.Equal(t, user.ID, account.UserID)
		assert.Equal(t, "Test Account", account.Name)
		assert.Equal(t, 500.75, account.Balance)
	})

	t.Run("AccountWithoutBalance", func(t *testing.T) {
		formData := map[string]string{
			"user-id": strconv.Itoa(int(user.ID)),
			"name":    "Zero Balance Account",
			// No balance specified
		}

		req, err := createFormRequest("POST", "/accounts", formData)
		require.NoError(t, err)

		w := performRequest(r, req)
		assert.Equal(t, http.StatusFound, w.Code)

		// Verify account was created with default balance
		var account models.Account
		err = db.Where("name = ?", "Zero Balance Account").First(&account).Error
		assert.NoError(t, err)
		assert.Equal(t, 0.0, account.Balance)
	})

	t.Run("InvalidUserID", func(t *testing.T) {
		formData := map[string]string{
			"user-id": "invalid",
			"name":    "Invalid Account",
		}

		req, err := createFormRequest("POST", "/accounts", formData)
		require.NoError(t, err)

		w := performRequest(r, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid user id", response["error"])
	})

	t.Run("InvalidBalance", func(t *testing.T) {
		formData := map[string]string{
			"user-id": strconv.Itoa(int(user.ID)),
			"name":    "Invalid Balance Account",
			"balance": "not-a-number",
		}

		req, err := createFormRequest("POST", "/accounts", formData)
		require.NoError(t, err)

		w := performRequest(r, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid balance data", response["error"])
	})
}

func TestUpdateAccount(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)

	r := setupTestRouter()
	r.POST("/accounts/update/:id", func(c *gin.Context) {
		UpdateAccount(c, db)
	})

	// Create test user and account
	user := createTestUser(db)
	account := createTestAccount(db, user.ID)

	t.Run("ValidUpdate", func(t *testing.T) {
		formData := map[string]string{
			"user-id": strconv.Itoa(int(user.ID)),
			"name":    "Updated Account Name",
			"balance": "999.99",
		}

		req, err := createFormRequest("POST", fmt.Sprintf("/accounts/update/%d", account.ID), formData)
		require.NoError(t, err)

		w := performRequest(r, req)
		assert.Equal(t, http.StatusFound, w.Code)

		// Verify account was updated
		var updatedAccount models.Account
		err = db.First(&updatedAccount, account.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "Updated Account Name", updatedAccount.Name)
		assert.Equal(t, 999.99, updatedAccount.Balance)
	})

	t.Run("InvalidAccountID", func(t *testing.T) {
		formData := map[string]string{
			"user-id": strconv.Itoa(int(user.ID)),
			"name":    "Won't Update",
			"balance": "100.00",
		}

		req, err := createFormRequest("POST", "/accounts/update/invalid", formData)
		require.NoError(t, err)

		w := performRequest(r, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ZeroAccountID", func(t *testing.T) {
		formData := map[string]string{
			"user-id": strconv.Itoa(int(user.ID)),
			"name":    "Won't Update",
			"balance": "100.00",
		}

		req, err := createFormRequest("POST", "/accounts/update/0", formData)
		require.NoError(t, err)

		w := performRequest(r, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestNewAccountNumber(t *testing.T) {
	// Test the random account number generation
	numbers := make(map[uint]bool)

	// Generate multiple account numbers
	for i := 0; i < 100; i++ {
		num := newAccountNumber()

		// Should be between 10000 and 99999
		assert.GreaterOrEqual(t, num, uint(10000))
		assert.LessOrEqual(t, num, uint(99999))

		// Track uniqueness (though random collisions are possible)
		numbers[num] = true
	}

	// We should have generated some variety (not all the same number)
	assert.Greater(t, len(numbers), 1)
}

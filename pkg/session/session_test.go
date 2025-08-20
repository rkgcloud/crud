package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/rkgcloud/crud/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSessionTest() (*gin.Engine, *gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test-session", store))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	// Initialize the session middleware
	sessions.Sessions("test-session", store)(c)

	return r, c, w
}

// createTestLoggedInUser creates a test logged-in user
func createTestLoggedInUser() *auth.LoggedInUser {
	return &auth.LoggedInUser{
		ID:      "test-user-id",
		Name:    "Test User",
		Email:   "test@example.com",
		Phone:   "123-456-7890",
		Picture: "https://example.com/picture.jpg",
	}
}

// setupTestRouter creates a test router with session middleware
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	r.Use(sessions.Sessions("test-session", store))

	return r
}

// setSessionUser sets a user in the session for testing
func setSessionUser(c *gin.Context, user *auth.LoggedInUser) {
	session := sessions.Default(c)
	session.Set("loggedInUser", user)
	session.Save()
}

// performRequest executes a request and returns the response
func performRequest(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSetLoggedInUser(t *testing.T) {
	_, c, _ := setupSessionTest()

	user := createTestLoggedInUser()

	err := SetLoggedInUser(c, user)
	assert.NoError(t, err)

	// Verify the user was set in session
	session := sessions.Default(c)
	sessionUser := session.Get(loggedUser)
	assert.NotNil(t, sessionUser)

	retrievedUser := sessionUser.(*auth.LoggedInUser)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Name, retrievedUser.Name)
	assert.Equal(t, user.Email, retrievedUser.Email)
	assert.Equal(t, user.Phone, retrievedUser.Phone)
	assert.Equal(t, user.Picture, retrievedUser.Picture)
}

func TestGetLoggedInUser(t *testing.T) {
	_, c, _ := setupSessionTest()

	t.Run("UserExists", func(t *testing.T) {
		// Set a user in session first
		user := createTestLoggedInUser()
		err := SetLoggedInUser(c, user)
		require.NoError(t, err)

		// Get the user from session
		retrievedUser := GetLoggedInUser(c)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, user.Name, retrievedUser.Name)
		assert.Equal(t, user.Email, retrievedUser.Email)
		assert.Equal(t, user.Phone, retrievedUser.Phone)
		assert.Equal(t, user.Picture, retrievedUser.Picture)
	})

	t.Run("UserDoesNotExist", func(t *testing.T) {
		_, c, _ := setupSessionTest() // Fresh context without user

		// Get user when none exists
		retrievedUser := GetLoggedInUser(c)
		assert.Empty(t, retrievedUser.ID)
		assert.Empty(t, retrievedUser.Name)
		assert.Empty(t, retrievedUser.Email)
		assert.Empty(t, retrievedUser.Phone)
		assert.Empty(t, retrievedUser.Picture)
	})
}

func TestDeleteLoggedInUser(t *testing.T) {
	_, c, _ := setupSessionTest()

	// Set a user in session first
	user := createTestLoggedInUser()
	err := SetLoggedInUser(c, user)
	require.NoError(t, err)

	// Verify user exists
	retrievedUser := GetLoggedInUser(c)
	assert.Equal(t, user.ID, retrievedUser.ID)

	// Delete the user
	err = DeleteLoggedInUser(c)
	assert.NoError(t, err)

	// Verify user is gone
	deletedUser := GetLoggedInUser(c)
	assert.Empty(t, deletedUser.ID)

	// Also verify directly from session
	session := sessions.Default(c)
	sessionUser := session.Get(loggedUser)
	assert.Nil(t, sessionUser)
}

func TestIsLoggedIn(t *testing.T) {
	_, c, _ := setupSessionTest()

	t.Run("UserLoggedIn", func(t *testing.T) {
		// Set a user in session
		user := createTestLoggedInUser()
		err := SetLoggedInUser(c, user)
		require.NoError(t, err)

		// Check if logged in
		loggedIn := IsLoggedIn(c)
		assert.True(t, loggedIn)
	})

	t.Run("UserNotLoggedIn", func(t *testing.T) {
		_, c, _ := setupSessionTest() // Fresh context without user

		// Check if logged in
		loggedIn := IsLoggedIn(c)
		assert.False(t, loggedIn)
	})

	t.Run("EmptyUserID", func(t *testing.T) {
		_, c, _ := setupSessionTest()

		// Set a user with empty ID
		user := &auth.LoggedInUser{
			ID:    "", // Empty ID
			Name:  "Test User",
			Email: "test@example.com",
		}
		err := SetLoggedInUser(c, user)
		require.NoError(t, err)

		// Should not be considered logged in
		loggedIn := IsLoggedIn(c)
		assert.False(t, loggedIn)
	})
}

func TestSessionIntegration(t *testing.T) {
	_, c, _ := setupSessionTest()

	// Test the complete flow
	user := createTestLoggedInUser()

	// Initially not logged in
	assert.False(t, IsLoggedIn(c))

	// Set user
	err := SetLoggedInUser(c, user)
	assert.NoError(t, err)

	// Now logged in
	assert.True(t, IsLoggedIn(c))

	// Can retrieve user
	retrievedUser := GetLoggedInUser(c)
	assert.Equal(t, user.ID, retrievedUser.ID)

	// Delete user
	err = DeleteLoggedInUser(c)
	assert.NoError(t, err)

	// No longer logged in
	assert.False(t, IsLoggedIn(c))
}

func TestSessionWithRealHTTPRequest(t *testing.T) {
	r := setupTestRouter()

	// Add test endpoint
	r.GET("/test", func(c *gin.Context) {
		user := createTestLoggedInUser()
		err := SetLoggedInUser(c, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		retrievedUser := GetLoggedInUser(c)
		c.JSON(http.StatusOK, gin.H{
			"logged_in": IsLoggedIn(c),
			"user":      retrievedUser,
		})
	})

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	w := performRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["logged_in"].(bool))

	userMap := response["user"].(map[string]interface{})
	assert.Equal(t, "test-user-id", userMap["id"])
	assert.Equal(t, "Test User", userMap["name"])
}

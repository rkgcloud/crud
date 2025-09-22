package session

// https://github.com/vaxilu/x-ui/blob/main/web/session/session.go#L20
import (
	"encoding/gob"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/rkgcloud/crud/pkg/auth"
)

const (
	loggedUser   = "loggedInUser"
	stateToken   = "stateToken"
	flashError   = "flashError"
	flashSuccess = "flashSuccess"
	flashWarning = "flashWarning"
	flashInfo    = "flashInfo"
)

func init() {
	gob.Register(auth.LoggedInUser{})
}

func SetLoggedInUser(c *gin.Context, user *auth.LoggedInUser) error {
	s := sessions.Default(c)
	s.Set(loggedUser, user)
	return s.Save()
}

func GetLoggedInUser(c *gin.Context) auth.LoggedInUser {
	s := sessions.Default(c)
	user := s.Get(loggedUser)
	if user == nil {
		return auth.LoggedInUser{}
	}
	return user.(auth.LoggedInUser)
}

func DeleteLoggedInUser(c *gin.Context) error {
	s := sessions.Default(c)
	s.Delete(loggedUser)
	return s.Save()
}

func IsLoggedIn(c *gin.Context) bool {
	return GetLoggedInUser(c).ID != ""
}

// SetStateToken stores the OAuth state token in the session
func SetStateToken(c *gin.Context, token string) error {
	s := sessions.Default(c)
	s.Set(stateToken, token)
	return s.Save()
}

// GetStateToken retrieves the OAuth state token from the session
func GetStateToken(c *gin.Context) string {
	s := sessions.Default(c)
	token := s.Get(stateToken)
	if token == nil {
		return ""
	}
	return token.(string)
}

// DeleteStateToken removes the OAuth state token from the session
func DeleteStateToken(c *gin.Context) error {
	s := sessions.Default(c)
	s.Delete(stateToken)
	return s.Save()
}

// Flash message functions for error/success notifications

// SetFlashError stores an error message in the session
func SetFlashError(c *gin.Context, message string) error {
	s := sessions.Default(c)
	s.Set(flashError, message)
	return s.Save()
}

// GetFlashError retrieves and removes an error message from the session
func GetFlashError(c *gin.Context) string {
	s := sessions.Default(c)
	message := s.Get(flashError)
	if message != nil {
		s.Delete(flashError)
		s.Save()
		return message.(string)
	}
	return ""
}

// SetFlashSuccess stores a success message in the session
func SetFlashSuccess(c *gin.Context, message string) error {
	s := sessions.Default(c)
	s.Set(flashSuccess, message)
	return s.Save()
}

// GetFlashSuccess retrieves and removes a success message from the session
func GetFlashSuccess(c *gin.Context) string {
	s := sessions.Default(c)
	message := s.Get(flashSuccess)
	if message != nil {
		s.Delete(flashSuccess)
		s.Save()
		return message.(string)
	}
	return ""
}

// SetFlashWarning stores a warning message in the session
func SetFlashWarning(c *gin.Context, message string) error {
	s := sessions.Default(c)
	s.Set(flashWarning, message)
	return s.Save()
}

// GetFlashWarning retrieves and removes a warning message from the session
func GetFlashWarning(c *gin.Context) string {
	s := sessions.Default(c)
	message := s.Get(flashWarning)
	if message != nil {
		s.Delete(flashWarning)
		s.Save()
		return message.(string)
	}
	return ""
}

// SetFlashInfo stores an info message in the session
func SetFlashInfo(c *gin.Context, message string) error {
	s := sessions.Default(c)
	s.Set(flashInfo, message)
	return s.Save()
}

// GetFlashInfo retrieves and removes an info message from the session
func GetFlashInfo(c *gin.Context) string {
	s := sessions.Default(c)
	message := s.Get(flashInfo)
	if message != nil {
		s.Delete(flashInfo)
		s.Save()
		return message.(string)
	}
	return ""
}

// FlashMessages represents all flash messages
type FlashMessages struct {
	Error   string
	Success string
	Warning string
	Info    string
}

// GetAllFlashMessages retrieves all flash messages at once
func GetAllFlashMessages(c *gin.Context) FlashMessages {
	return FlashMessages{
		Error:   GetFlashError(c),
		Success: GetFlashSuccess(c),
		Warning: GetFlashWarning(c),
		Info:    GetFlashInfo(c),
	}
}

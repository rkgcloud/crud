package session

// https://github.com/vaxilu/x-ui/blob/main/web/session/session.go#L20
import (
	"encoding/gob"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/rkgcloud/crud/pkg/auth"
)

const (
	loggedUser = "loggedInUser"
	stateToken = "stateToken"
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

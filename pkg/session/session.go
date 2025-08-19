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
	// Handle both pointer and value types
	if userPtr, ok := user.(*auth.LoggedInUser); ok {
		return *userPtr
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

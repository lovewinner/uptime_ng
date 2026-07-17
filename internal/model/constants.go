package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	StatusDown      uint16 = 0
	StatusUP        uint16 = 1
	StatusPending   uint16 = 2
	StatusMaintenance uint16 = 3

	RoleAdmin = "admin"
	RoleUser  = "user"

	MonitorTypeHTTP = "http"
	MonitorTypeTCP  = "tcp"
	MonitorTypePing = "ping"
	MonitorTypeDNS  = "dns"
	MonitorTypePush = "push"

	DefaultInterval     = 60
	MinInterval         = 3
	DefaultTimeout      = 30.0
	DefaultPingCount    = 4
	DefaultPingPacketSize = 56
	DefaultHTTPMaxRedirects = 10
	DefaultRetryInterval  = 60
	DefaultResponseMaxLen = 4096

	MinutelyKeepHours = 24
	HourlyKeepDays    = 30
	DailyKeepDays     = 365
)

func FlipStatus(status uint16) uint16 {
	if status == StatusUP {
		return StatusDown
	}
	if status == StatusDown {
		return StatusUP
	}
	return status
}

func FlatStatus(status uint16) uint16 {
	if status == StatusUP || status == StatusMaintenance {
		return StatusUP
	}
	return StatusDown
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateJWT(user *User, secret string, expireHours int) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func BeforeCreateUser(db *gorm.DB) error {
	if db.Statement.Schema != nil {
		if val, ok := db.Statement.Dest.(*User); ok {
			if val.Role == "" {
				val.Role = RoleUser
			}
		}
	}
	return nil
}
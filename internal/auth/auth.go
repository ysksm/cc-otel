// Package auth provides opt-in dashboard authentication: bcrypt password
// hashing and stateless HMAC-signed session tokens. It is enabled only when
// configured via the environment, so the default local experience needs no login.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// SessionCookie is the name of the session cookie.
const SessionCookie = "ccotel_session"

// SessionTTL is how long a session is valid.
const SessionTTL = 7 * 24 * time.Hour

// Config controls auth behaviour.
type Config struct {
	Enabled bool
	Secret  []byte
}

// ConfigFromEnv builds the auth config. Auth is enabled when CCOTEL_AUTH_ENABLED
// is "true" or CCOTEL_ADMIN_PASSWORD is set. The session secret comes from
// CCOTEL_SESSION_SECRET, or a random per-process secret (sessions then reset on
// restart).
func ConfigFromEnv() Config {
	enabled := os.Getenv("CCOTEL_AUTH_ENABLED") == "true" || os.Getenv("CCOTEL_ADMIN_PASSWORD") != ""
	secret := []byte(os.Getenv("CCOTEL_SESSION_SECRET"))
	if len(secret) == 0 {
		secret = make([]byte, 32)
		_, _ = rand.Read(secret)
	}
	return Config{Enabled: enabled, Secret: secret}
}

// HashPassword returns a bcrypt hash of the password.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword reports whether pw matches the bcrypt hash.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// SignSession returns a signed token encoding userID and an expiry.
func SignSession(secret []byte, userID string, expiry time.Time) string {
	msg := userID + "|" + strconv.FormatInt(expiry.Unix(), 10)
	sig := sign(secret, msg)
	return base64.RawURLEncoding.EncodeToString([]byte(msg + "|" + sig))
}

// VerifySession validates a token and returns the userID if valid and unexpired.
func VerifySession(secret []byte, token string) (string, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return "", false
	}
	userID, expStr, sig := parts[0], parts[1], parts[2]
	if !hmac.Equal([]byte(sig), []byte(sign(secret, userID+"|"+expStr))) {
		return "", false
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", false
	}
	return userID, true
}

func sign(secret []byte, msg string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

const (
	SessionCookieName = "session"
)

var (
	ErrSessionDecryption = errors.New("session decryption failed")
	ErrInvalidSession    = errors.New("invalid session")
)

type SessionConfig struct {
	Key            []byte
	CookieSecure   bool
	CookiePath     string
	CookieDomain   string
	CookieMaxAge   int
	CookieSameSite http.SameSite
}

type SessionData struct {
	UserId    string         `json:"user_id,omitempty"`
	Role      string         `json:"role,omitempty"`
	ExpiresAt time.Time      `json:"expires_at"`
	Data      map[string]any `json:"data,omitempty"`
}

type Session struct {
	config SessionConfig
}

func NewSession(config SessionConfig) (*Session, error) {
	// AES-256 requires a 32-byte key
	if len(config.Key) != 32 {
		return nil, errors.New("session encryption key should be 32 bytes")
	}
	if config.CookiePath == "" {
		config.CookiePath = "/"
	}
	if config.CookieMaxAge == 0 {
		config.CookieMaxAge = 86400 // 24 hours
	}
	if config.CookieSameSite == 0 {
		config.CookieSameSite = http.SameSiteLaxMode
	}

	return &Session{config: config}, nil
}

func (s *Session) Create(w http.ResponseWriter, sessionData SessionData) error {
	if sessionData.ExpiresAt.IsZero() {
		sessionData.ExpiresAt = time.Now().Add(time.Duration(s.config.CookieMaxAge) * time.Second)
	}

	jsonData, err := json.Marshal(sessionData)
	if err != nil {
		return err
	}

	encryptedData, err := s.encrypt(jsonData)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    encryptedData,
		Path:     s.config.CookiePath,
		Domain:   s.config.CookieDomain,
		MaxAge:   s.config.CookieMaxAge,
		Secure:   s.config.CookieSecure,
		HttpOnly: true,
		SameSite: s.config.CookieSameSite,
	})

	return nil

}

func (s *Session) Get(r *http.Request) (SessionData, error) {
	var sessionData SessionData

	// Get the session cookie
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return sessionData, err
	}

	// Decrypt the session data
	jsonData, err := s.decrypt(cookie.Value)
	if err != nil {
		return sessionData, ErrSessionDecryption
	}

	// Deserialize the JSON data
	if err := json.Unmarshal(jsonData, &sessionData); err != nil {
		return sessionData, ErrInvalidSession
	}

	// Check if the session has expired
	if time.Now().After(sessionData.ExpiresAt) {
		return sessionData, errors.New("session expired")
	}

	return sessionData, nil
}

// Clear removes the session
func (s *Session) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     s.config.CookiePath,
		Domain:   s.config.CookieDomain,
		MaxAge:   -1,
		Secure:   s.config.CookieSecure,
		HttpOnly: true,
		SameSite: s.config.CookieSameSite,
	})
}

// encrypt | decrypt helpers

func (s *Session) encrypt(plaintext []byte) (string, error) {
	// Create a new AES cipher block
	bloc, err := aes.NewCipher(s.config.Key)
	if err != nil {
		return "", err
	}

	// Create a new GCM cipher mode
	aesGCM, err := cipher.NewGCM(bloc)
	if err != nil {
		return "", err
	}

	// Create a nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the plaintext
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	// Base64 encode the ciphertext
	return base64.StdEncoding.EncodeToString(ciphertext), nil

}

func (s *Session) decrypt(encryptedData string) ([]byte, error) {
	// Base64 decode the ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	// Create a new AES cipher block
	block, err := aes.NewCipher(s.config.Key)
	if err != nil {
		return nil, err
	}

	// Create a new GCM cipher mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Extract the nonce
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt the ciphertext
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

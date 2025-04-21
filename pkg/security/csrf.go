package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"time"
)

const (
	CSRFTokenLength = 32
	CSRFCookieName  = "csrf_token"
	CSRFHeaderName  = "X-CSRF-Token"
	CSRFFormField   = "csrf_token"
)

var (
	ErrInvalidCSRFToken = errors.New("invalid CSRF token")
	ErrMissingCSRFToken = errors.New("missing CSRF token")
)

type CSRFConfig struct {
	Key            []byte
	CookieSecure   bool
	CookiePath     string
	CookieDomain   string
	CookieMaxAge   int
	CookieSameSite http.SameSite
}

type CSRFProtection struct {
	config CSRFConfig
}

func NewCSRFProtection(config CSRFConfig) *CSRFProtection {
	if len(config.Key) == 0 {
		panic("CSRF key cannot be empty")
	}

	if config.CookiePath == "" {
		config.CookiePath = "/"
	}
	if config.CookieMaxAge == 0 {
		config.CookieMaxAge = 86400 // 24 hours
	}
	if config.CookieSameSite == 0 {
		config.CookieSameSite = http.SameSiteStrictMode
	}

	return &CSRFProtection{
		config: config,
	}
}

func (c *CSRFProtection) Middleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet ||
			r.Method == http.MethodHead ||
			r.Method == http.MethodOptions || r.Method == http.MethodTrace {
			cookie, err := r.Cookie(CSRFCookieName)
			if err != nil || cookie.Value == "" {
				token, err := c.generateToken()
				if err != nil {
					http.Error(w, "failed to gen CSRF token", http.StatusInternalServerError)
					return
				}

				http.SetCookie(w, &http.Cookie{
					Name:     CSRFCookieName,
					Value:    token,
					Path:     c.config.CookiePath,
					Domain:   c.config.CookieDomain,
					MaxAge:   c.config.CookieMaxAge,
					Secure:   c.config.CookieSecure,
					HttpOnly: true,
					SameSite: c.config.CookieSameSite,
				})
			}
			next.ServeHTTP(w, r)
			return
		}

		// validation for other methods
		cookie, err := r.Cookie(CSRFCookieName)
		if err != nil || cookie.Value == "" {
			log.Error().Err(err).Msg("CSRF token cookie missing")
			http.Error(w, "CSRF token is missing", http.StatusForbidden)
			return
		}

		// check header or form for the tojen
		var token string
		if headerToken := r.Header.Get(CSRFHeaderName); headerToken != "" {
			token = headerToken
		} else if formToken := r.FormValue(CSRFFormField); formToken != "" {
			token = headerToken
		} else {
			log.Error().Msg("CSRF token is no in header | form")
			http.Error(w, "CSRF tokne is missing", http.StatusForbidden)
			return
		}

		//validation
		if err := c.validateToken(token); err != nil {
			log.Error().Err(err).Msg("CSRF token validation failed")
			http.Error(w, "invalid CSRF token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (c *CSRFProtection) generateToken() (string, error) {
	randomBytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	randomString := base64.StdEncoding.EncodeToString(randomBytes)

	//payload and time
	timeStamp := time.Now().Unix()
	payload := fmt.Sprintf("%s|%d", randomString, timeStamp)

	//sign
	h := hmac.New(sha256.New, c.config.Key)
	h.Write([]byte(payload))
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	//payload + sign
	token := fmt.Sprintf("%s|%s", payload, sign)

	return token, nil
}

func (c *CSRFProtection) validateToken(token string) error {
	if token == "" {
		return ErrMissingCSRFToken
	}

	// split token into signiture and payload
	parts := strings.Split(token, "|")
	if len(parts) != 3 {
		return ErrInvalidCSRFToken
	}

	//exctracion
	randomStr, timeStampStr, receiveSign := parts[0], parts[1], parts[2]
	payload := fmt.Sprintf("%s|%s", randomStr, timeStampStr)

	// sign payload with HMAC 256
	h := hmac.New(sha256.New, c.config.Key)
	h.Write([]byte(payload))
	expectedSign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	//compare
	if !hmac.Equal([]byte(receiveSign), []byte(expectedSign)) {
		return ErrInvalidCSRFToken
	}
	return nil
}

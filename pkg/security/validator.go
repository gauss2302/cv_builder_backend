package security

import (
	"errors"
	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/zerolog/log"
	"io"
	"mime"
	"net/http"
	"strings"
)

const (
	MaxBodySize = 1 * 1024 * 1024
)

var (
	ErrInvalidContentType = errors.New("invalid Content-Type")
	ErrBodyTooLarge       = errors.New("request body too large")
)

type ValidationConfig struct {
	AllowedContentTypes []string
	MaxBodySize         int64
	StrictPolicy        bool
}

type Validator struct {
	config        ValidationConfig
	htmlSanitizer *bluemonday.Policy
}

func NewValidator(config ValidationConfig) *Validator {
	if len(config.AllowedContentTypes) == 0 {
		config.AllowedContentTypes = []string{
			"application/json",
			"application/x-www-form-urlencoded",
			"multipart/form-data",
		}
	}
	if config.MaxBodySize <= 0 {
		config.MaxBodySize = MaxBodySize
	}
	var policy *bluemonday.Policy
	if config.StrictPolicy {
		policy = bluemonday.UGCPolicy()
	}
	return &Validator{
		config:        config,
		htmlSanitizer: policy,
	}
}

func (v *Validator) ValidateContentType(r *http.Request) error {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || r.Method == http.MethodTrace {
		return nil
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return nil
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ErrInvalidContentType
	}

	for _, allowed := range v.config.AllowedContentTypes {
		if mediaType == allowed {
			return nil
		}
		if strings.HasPrefix(allowed, "multipart/") && strings.HasPrefix(mediaType, allowed) {
			return nil
		}
	}
	return ErrInvalidContentType
}

func (v *Validator) LimitBodySize(r *http.Request) ([]byte, error) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead ||
		r.Method == http.MethodOptions || r.Method == http.MethodTrace {
		return nil, nil
	}

	r.Body = http.MaxBytesReader(nil, r.Body, v.config.MaxBodySize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			return nil, ErrBodyTooLarge
		}
		return nil, err
	}
	return body, err
}

func (v *Validator) SanitizeHTML(input string) string {
	return v.htmlSanitizer.Sanitize(input)
}

func (v *Validator) SanitizeMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for k, val := range input {
		result[k] = v.SanitizeHTML(val)
	}
	return result
}

func (v *Validator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := v.ValidateContentType(r); err != nil {
			log.Error().Err(err).
				Str("content_type", r.Header.Get("Content-Type")).
				Msg("Invalid Content-Type")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte(`{"error":"Unsupported Content-Type","status":415}`))
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead &&
			r.Method != http.MethodOptions && r.Method != http.MethodTrace {

			originalBody := r.Body
			limitedBody, err := v.LimitBodySize(r)
			if err != nil {
				if errors.Is(err, ErrBodyTooLarge) {
					log.Error().Err(err).Msg("Request body too large")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusRequestEntityTooLarge)
					w.Write([]byte(`{"error":"Request body too large","status":413}`))
					return
				}

				log.Error().Err(err).Msg("Error reading request body")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"Error reading request body","status":400}`))
				return
			}

			if limitedBody != nil {
				r.Body = io.NopCloser(strings.NewReader(string(limitedBody)))
			} else {
				r.Body = originalBody
			}
		}
		next.ServeHTTP(w, r)
	})
}

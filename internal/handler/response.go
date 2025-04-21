package handler

import (
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
)

type ErrorResponse struct {
	Status  int            `json:"-"`
	Error   string         `json:"error"`
	Code    string         `json:"code"`
	Details map[string]any `json:"details,omitempty"`
}

func RespondWithJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal JSON response")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(jsonData); err != nil {
		log.Error().Err(err).Msg("failed to write JSON response")
	}

}

func RespondWithError(w http.ResponseWriter, status int, message, code string) {
	response := ErrorResponse{
		Status: status,
		Error:  message,
		Code:   code,
	}
	RespondWithJSON(w, status, response)
}

func RespondWithValidationError(w http.ResponseWriter, validationErrors any) {
	// Create error details
	details := make(map[string]any)
	fieldErrors := make(map[string]string)

	// Check if it's the expected type
	if errs, ok := validationErrors.(validator.ValidationErrors); ok {
		for _, err := range errs {
			// Convert field name to JSON field name (usually lowercase)
			field := strings.ToLower(err.Field())
			fieldErrors[field] = GetValidationErrorMessage(err)
		}
	} else {
		// Handle unexpected validation error type
		fieldErrors["general"] = "Validation failed"
	}

	details["fields"] = fieldErrors

	// Create error response
	response := ErrorResponse{
		Status:  http.StatusBadRequest,
		Error:   "Validation failed",
		Code:    "VALIDATION_FAILED",
		Details: details,
	}

	RespondWithJSON(w, http.StatusBadRequest, response)
}

type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func GetValidationErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return "Must be at least " + err.Param() + " characters long"
	case "max":
		return "Must be at most " + err.Param() + " characters long"
	case "oneof":
		return "Must be one of: " + err.Param()
	default:
		return "Invalid value"

	}
}

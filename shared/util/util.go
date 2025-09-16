package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ride-sharing/shared/contracts"

	"github.com/go-playground/validator/v10"
)

// GetRandomAvatar returns a random avatar URL from the randomuser.me API
func GetRandomAvatar(index int) string {
	return fmt.Sprintf("https://randomuser.me/api/portraits/lego/%d.jpg", index)
}

func RespondWithError(w http.ResponseWriter, code int, message string, errors []string) {
	response := contracts.ErrorResponse{
		Status:  "error",
		Message: message,
		Errors:  errors,
	}
	respondWithJSON(w, code, response)
}

func RespondWithSuccess(w http.ResponseWriter, code int, message string, data interface{}) {
	response := contracts.SuccessResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}
	respondWithJSON(w, code, response)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func FormatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return err.Field() + " is required"
	case "min":
		return err.Field() + " must be at least " + err.Param() + " characters long"
	default:
		return err.Field() + " failed " + err.Tag() + " validation"
	}
}

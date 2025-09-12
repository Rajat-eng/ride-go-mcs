package main

import (
	"encoding/json"
	"net/http"
	"ride-sharing/shared/types"
	"ride-sharing/shared/util"

	"github.com/go-playground/validator/v10"
)

func HandleTripPreview(w http.ResponseWriter, r *http.Request) {
	var reqBody PreviewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// validation
	if err := types.Validate.Struct(reqBody); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		errors := make([]string, len(validationErrors))
		for i, err := range validationErrors {
			errors[i] = util.FormatValidationError(err)
		}
		util.RespondWithError(w, http.StatusBadRequest, "Validation failed", errors)
		return
	}
	response := map[string]interface{}{
		"estimatedPrice": 1000,
		"estimatedTime":  "15 minutes",
		"distance":       "5.2 km",
	}
	// interfaces give us flexibility to change response later
	util.RespondWithSuccess(w, http.StatusOK, "Trip Preview", response)
}

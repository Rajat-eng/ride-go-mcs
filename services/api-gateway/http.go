package main

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpc_clients"
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
	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	// call trip service using gRPC
	// new connection for every request is needed because
	// if tripservice is down, then only this request will fail
	// other requests will not be affected
	// if we use a single connection for all requests, then if tripservice is down
	// all requests will fail
	defer tripService.Close()

	tripPreview, err := tripService.Client.PreviewTrip(r.Context(), reqBody.toProto())
	// tripPreviwew is pb.PreviewTripResponse
	// it gives pb response
	// need to convert to json response
	if err != nil {
		log.Println("Error calling trip service:", err)
		http.Error(w, "Failed to get trip preview", http.StatusInternalServerError)
		return
	}

	util.RespondWithSuccess(w, http.StatusOK, "Trip Preview", tripPreview)
}

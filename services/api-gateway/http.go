package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpc_clients"

	"ride-sharing/shared/contracts"
	"ride-sharing/shared/tracing"
	"ride-sharing/shared/types"
	"ride-sharing/shared/util"

	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"

	"github.com/go-playground/validator/v10"
)

var tracer = tracing.GetTracer("api-gateway")

func HandleTripPreview(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleTripPreview")
	defer span.End() // any db or other operation
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

	tripPreview, err := tripService.Client.PreviewTrip(ctx, reqBody.toProto())
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

func HandleStartTrip(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleTripStart")
	defer span.End()
	var reqBody StartTripRequest
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
	trip, err := tripService.Client.CreateTrip(ctx, reqBody.toProto())
	if err != nil {
		log.Fatal(err)
	}

	util.RespondWithSuccess(w, http.StatusOK, "Trip Started", trip)
}

func handleStripeWebhook(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	ctx, span := tracer.Start(r.Context(), "handleStripeWebhook")
	defer span.End()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	webhookKey := env.GetString("STRIPE_WEBHOOK_KEY", "")
	if webhookKey == "" {
		log.Printf("Webhook key is required")
		return
	}

	event, err := webhook.ConstructEventWithOptions(
		body,
		r.Header.Get("Stripe-Signature"),
		webhookKey,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		log.Printf("Error verifying webhook signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}

	log.Printf("Received Stripe event: %v", event)

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession

		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		payload := messaging.PaymentStatusUpdateData{
			TripID:   session.Metadata["trip_id"],
			UserID:   session.Metadata["user_id"],
			DriverID: session.Metadata["driver_id"],
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshalling payload: %v", err)
			http.Error(w, "Failed to marshal payload", http.StatusInternalServerError)
			return
		}

		message := contracts.AmqpMessage{
			OwnerID: session.Metadata["user_id"],
			Data:    payloadBytes,
		}

		if err := rb.PublishMessage(
			ctx,
			contracts.PaymentEventSuccess,
			message,
		); err != nil {
			log.Printf("Error publishing payment event: %v", err)
			http.Error(w, "Failed to publish payment event", http.StatusInternalServerError)
			return
		}
	}
}

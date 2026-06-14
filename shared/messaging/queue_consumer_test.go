package messaging

import (
	"encoding/json"
	"testing"

	"ride-sharing/shared/contracts"
)

func TestCanonicalizeWSData_TripEnvelopeStripsIncompleteDriver(t *testing.T) {
	payload := []byte(`{
		"trip": {
			"id": "trip-1",
			"userID": "rider-1",
			"status": "pending",
			"driver": {}
		}
	}`)

	data, skip, err := canonicalizeWSData(contracts.TripEventCreated, payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if skip {
		t.Fatalf("expected skip=false")
	}

	raw, ok := data.(json.RawMessage)
	if !ok {
		t.Fatalf("expected json.RawMessage, got %T", data)
	}

	var trip map[string]any
	if err := json.Unmarshal(raw, &trip); err != nil {
		t.Fatalf("unmarshal canonical trip: %v", err)
	}

	if got, _ := trip["id"].(string); got != "trip-1" {
		t.Fatalf("unexpected trip id: %v", trip["id"])
	}
	if _, hasDriver := trip["driver"]; hasDriver {
		t.Fatalf("expected incomplete driver to be omitted, got %v", trip["driver"])
	}
}

func TestCanonicalizeWSData_SkipsChatDelivered(t *testing.T) {
	data, skip, err := canonicalizeWSData(contracts.ChatEventDelivered, []byte(`{"messageID":"m1","tripID":"t1"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !skip {
		t.Fatalf("expected skip=true")
	}
	if data != nil {
		t.Fatalf("expected nil data when skipped, got %T", data)
	}
}

func TestCanonicalizeWSData_RejectsInvalidPaymentPayload(t *testing.T) {
	_, skip, err := canonicalizeWSData(contracts.PaymentEventSessionCreated, []byte(`{"tripID":"","sessionID":""}`))
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if skip {
		t.Fatalf("expected skip=false for invalid payment payload")
	}
}

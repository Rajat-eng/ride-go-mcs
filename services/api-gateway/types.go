package main

import (
	"ride-sharing/shared/types"
)

type PreviewTripRequest struct {
	UserID      string           `json:"userID" validate:"required,min=1"`
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}

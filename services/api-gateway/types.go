package main

import (
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"
)

type PreviewTripRequest struct {
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}

type StartTripRequest struct {
	RideFare string `json:"rideFareID" validate:"required,min=1"`
}

func toProtoCoordinate(c types.Coordinate) *pb.Coordinate {
	return &pb.Coordinate{
		Latitude:  c.Latitude,
		Longitude: c.Longitude,
	}
}

func (p *PreviewTripRequest) toProto(userID string) *pb.PreviewTripRequest {
	return &pb.PreviewTripRequest{
		UserID:        userID,
		StartLocation: toProtoCoordinate(p.Pickup),
		EndLocation:   toProtoCoordinate(p.Destination),
	}
}

func (s *StartTripRequest) toProto(userID string) *pb.CreateTripRequest {
	return &pb.CreateTripRequest{
		UserID:     userID,
		RideFareID: s.RideFare,
	}
}

type CancelTripRequest struct {
	TripID string `json:"tripID" validate:"required,min=1"`
}

func (c *CancelTripRequest) toProto(userID string) *pb.CancelTripRequest {
	return &pb.CancelTripRequest{
		TripID: c.TripID,
		UserID: userID,
	}
}

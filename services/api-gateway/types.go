package main

import (
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"
)

type PreviewTripRequest struct {
	UserID      string           `json:"userID" validate:"required,min=1"`
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}

type StartTripRequest struct {
	UserID   string `json:"userID" validate:"required,min=1"`
	RideFare string `json:"rideFareID" validate:"required,min=1"`
}

func toProtoCoordinate(c types.Coordinate) *pb.Coordinate {
	return &pb.Coordinate{
		Latitude:  c.Latitude,
		Longitude: c.Longitude,
	}
}

type tripRequestFactory struct{}

type ProtoRequestFactory interface {
	toProto(req interface{}) (interface{}, error)
}

func NewTripRequestFactory() *tripRequestFactory {
	return &tripRequestFactory{}
}

func (p *PreviewTripRequest) toProto() *pb.PreviewTripRequest {
	return &pb.PreviewTripRequest{
		UserID:        p.UserID,
		StartLocation: toProtoCoordinate(p.Pickup),
		EndLocation:   toProtoCoordinate(p.Destination),
	}
}

func (s *StartTripRequest) toProto() *pb.CreateTripRequest {
	return &pb.CreateTripRequest{
		UserID:     s.UserID,
		RideFareID: s.RideFare,
	}
}

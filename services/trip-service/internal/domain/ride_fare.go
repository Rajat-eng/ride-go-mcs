package domain

import (
	"ride-sharing/services/trip-service/pkg/types"
	pb "ride-sharing/shared/proto/trip"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RideFareModel struct {
	ID                primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID            string                 `bson:"userID" json:"userID"`
	PackageSlug       string                 `bson:"packageSlug" json:"packageSlug"`
	TotalPriceInCents float64                `bson:"totalPriceInCents" json:"totalPriceInCents"`
	Route             *types.OSRMApiResponse `bson:"route" json:"route"`
}

func (r *RideFareModel) ToProto() *pb.RideFare {
	return &pb.RideFare{
		Id:                r.ID.Hex(),
		PackageSlug:       r.PackageSlug,
		TotalPriceInCents: r.TotalPriceInCents,
	}
}

type FarePreviewOption struct {
	ID                primitive.ObjectID `json:"id"`
	PackageSlug       string             `json:"packageSlug"`
	TotalPriceInCents float64            `json:"totalPriceInCents"`
}
type FarePreview struct {
	UserID string                 `json:"userID"`
	Route  *types.OSRMApiResponse `json:"route"`
	Fares  []*FarePreviewOption   `json:"fares"`
}

func (p *FarePreview) ToRideFaresProto() []*pb.RideFare {
	protoFares := make([]*pb.RideFare, len(p.Fares))
	for i, f := range p.Fares {
		protoFares[i] = &pb.RideFare{
			Id:                f.ID.Hex(),
			PackageSlug:       f.PackageSlug,
			TotalPriceInCents: f.TotalPriceInCents,
		}
	}
	return protoFares
}

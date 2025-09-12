package main

import (
	"context"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
)

func main() {
	log.Println("starting trip service")
	ctx := context.Background()
	InMemoryRepository := repository.NewInmemoryRepository()
	TripService := service.NewTripService(InMemoryRepository)
	fare := &domain.RideFareModel{
		UserID: "42",
	}
	// grpc handler wiil call orsm api to calucate fare
	TripService.CreateTrip(ctx, fare)
	log.Println("success")

	// for {
	// 	time.Sleep(time.Second)
	// }
}

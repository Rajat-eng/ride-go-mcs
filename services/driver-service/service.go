package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	driverGeoKeyPrefix = "drivers:geo:" // GEOADD key per package slug
	searchRadiusKM     = 10.0           // km radius for GEOSEARCH
)

type Service struct {
	rdb *redis.Client
}

func NewService(rdb *redis.Client) *Service {
	return &Service{rdb: rdb}
}

// UpdateDriverLocation upserts the driver's position in the Redis GEO index for the given package.
// Calling this on the first location message effectively makes the driver available for trip matching.
func (s *Service) UpdateDriverLocation(driverId, packageSlug string, lat, lng float64) error {
	return s.rdb.GeoAdd(context.Background(), driverGeoKeyPrefix+packageSlug, &redis.GeoLocation{
		Name:      driverId,
		Longitude: lng,
		Latitude:  lat,
	}).Err()
}

// RemoveDriverFromGeo removes the driver from the GEO index so they no longer receive trip requests.
func (s *Service) RemoveDriverFromGeo(driverId, packageSlug string) {
	s.rdb.ZRem(context.Background(), driverGeoKeyPrefix+packageSlug, driverId)
}

// FindAvailableDrivers returns driver IDs near pickupLat/Lng matching packageType.
func (s *Service) FindAvailableDrivers(packageType string, pickupLat, pickupLng float64) []string {
	ctx := context.Background()
	geoKey := driverGeoKeyPrefix + packageType

	results, err := s.rdb.GeoSearch(ctx, geoKey, &redis.GeoSearchQuery{
		Longitude:  pickupLng,
		Latitude:   pickupLat,
		Radius:     searchRadiusKM,
		RadiusUnit: "km",
		Sort:       "ASC",
		Count:      10,
	}).Result()

	if err != nil || len(results) == 0 {
		return []string{}
	}

	ids := make([]string, len(results))
	// notify nearby drivers of this incoming ride request so they can respond with accept/decline
	// this is done by the api-gateway consuming from the same GEO index and pushing trip requests to drivers via WS
	copy(ids, results)
	return ids
}

const activeRiderTTL = 2 * time.Hour

// SetActiveRider records that driverID is currently serving riderID (stored until trip ends or TTL).
func (s *Service) SetActiveRider(driverID, riderID string) error {
	return s.rdb.Set(context.Background(), "driver:"+driverID+":active_rider", riderID, activeRiderTTL).Err()
}

// GetActiveRider returns the riderID currently paired with this driver, or ("", redis.Nil) if none.
func (s *Service) GetActiveRider(driverID string) (string, error) {
	return s.rdb.Get(context.Background(), "driver:"+driverID+":active_rider").Result()
}

// ClearActiveRider removes the driver→rider mapping when the trip ends.
func (s *Service) ClearActiveRider(driverID string) {
	s.rdb.Del(context.Background(), "driver:"+driverID+":active_rider")
}

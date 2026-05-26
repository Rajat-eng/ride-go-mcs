package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	driverGeoKeyPrefix = "drivers:geo:" // GEOADD key per package slug
	searchRadiusKM     = 15.0           // km radius for GEOSEARCH
)

const activeDriverTTL = 2 * time.Hour

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
func (s *Service) RemoveDriverFromGeo(ctx context.Context, driverId, packageSlug string) {
	s.rdb.ZRem(ctx, driverGeoKeyPrefix+packageSlug, driverId)
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

func activeRiderKey(driverID string) string {
	return "driver:" + driverID + ":active_rider"
}

func activeDriverKey(riderID string) string {
	return "rider:" + riderID + ":active_driver"
}

func tripChatRiderKey(tripID string) string {
	return "trip:" + tripID + ":chat:rider"
}

func tripChatDriverKey(tripID string) string {
	return "trip:" + tripID + ":chat:driver"
}

// SetActiveRider records that driverID is currently serving riderID (stored until trip ends or TTL).
func (s *Service) SetActiveRider(driverID, riderID string) error {
	return s.rdb.Set(context.Background(), activeRiderKey(driverID), riderID, activeRiderTTL).Err()
}

// GetActiveRider returns the riderID currently paired with this driver, or ("", redis.Nil) if none.
func (s *Service) GetActiveRider(driverID string) (string, error) {
	return s.rdb.Get(context.Background(), activeRiderKey(driverID)).Result()
}

// SetTripChatPair stores the confirmed rider/driver trip pairing used by gateway chat authorization.
func (s *Service) SetTripChatPair(tripID, riderID, driverID string) error {
	ctx := context.Background()
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, activeRiderKey(driverID), riderID, activeRiderTTL)
	pipe.Set(ctx, activeDriverKey(riderID), driverID, activeDriverTTL)
	if tripID != "" {
		pipe.Set(ctx, tripChatRiderKey(tripID), riderID, activeRiderTTL)
		pipe.Set(ctx, tripChatDriverKey(tripID), driverID, activeRiderTTL)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// ClearActiveRider removes the driver→rider mapping when the trip ends.
func (s *Service) ClearActiveRider(driverID string) {
	s.rdb.Del(context.Background(), activeRiderKey(driverID))
}

package main

import (
	"encoding/json"
	"fmt"
	math "math/rand/v2"
	"net/http"
	"strconv"
)

// SeedDriver represents a fake driver seeded near a center point.
type SeedDriver struct {
	ID          string  `json:"id"`
	PackageSlug string  `json:"packageSlug"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

// HandleSeedDrivers seeds N fake drivers into the Redis GEO index around a given location.
// Query params:
//
//	lat    float  center latitude  (default: 12.9352 - Bengaluru)
//	lng    float  center longitude (default: 77.6245)
//	radius float  scatter radius in degrees (~0.01 ≈ 1 km)  (default: 0.05)
//	count  int    number of drivers per package  (default: 5)
func HandleSeedDrivers(svc *Service) http.HandlerFunc {
	packages := []string{"sedan", "suv", "van"}

	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		centerLat := parseFloat(q.Get("lat"), 12.9352)
		centerLng := parseFloat(q.Get("lng"), 77.6245)
		radius := parseFloat(q.Get("radius"), 0.05)
		count := parseInt(q.Get("count"), 5)

		var seeded []SeedDriver

		for _, pkg := range packages {
			for i := 0; i < count; i++ {
				id := fmt.Sprintf("seed-%s-%d", pkg, i)
				lat := centerLat + (math.Float64()-0.5)*2*radius
				lng := centerLng + (math.Float64()-0.5)*2*radius

				if err := svc.UpdateDriverLocation(id, pkg, lat, lng); err != nil {
					continue
				}

				seeded = append(seeded, SeedDriver{
					ID:          id,
					PackageSlug: pkg,
					Lat:         lat,
					Lng:         lng,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"seeded":  len(seeded),
			"drivers": seeded,
		})
	}
}

func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

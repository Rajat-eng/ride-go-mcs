package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	math "math/rand/v2"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var (
	wsURL       = flag.String("url", "ws://localhost:8080/ws/drivers", "api-gateway driver WS endpoint")
	jwtSecret   = flag.String("secret", "change-me-in-production", "JWT signing secret (must match JWT_SECRET env in gateway)")
	centerLat   = flag.Float64("lat", 12.9352, "center latitude (default: Bengaluru)")
	centerLng   = flag.Float64("lng", 77.6245, "center longitude")
	countPerPkg = flag.Int("count", 3, "number of drivers per package")
	packagesCSV = flag.String("packages", "sedan,suv,van", "comma-separated package slugs")
	interval    = flag.Duration("interval", 5*time.Second, "location update interval")
	response    = flag.String("response", "none", "auto response to trip request: none|accept|decline")
	location    = flag.String("location", "static", "location mode: static|walk|none")
)

// ── WS message contracts ─────────────────────────────────────────────────────

type wsMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

const (
	cmdLocation    = "driver.cmd.location"
	cmdTripRequest = "driver.cmd.trip_request"
	cmdTripAccept  = "driver.cmd.trip_accept"
	cmdTripDecline = "driver.cmd.trip_decline"
)

// ── JWT ───────────────────────────────────────────────────────────────────────

func makeToken(driverID, name, secret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": driverID,
		"name":    name,
		"email":   driverID + "@sim.local",
		"role":    "driver",
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ── random walk ───────────────────────────────────────────────────────────────

const (
	stepDeg   = 0.001 // ~111 m per step
	maxRadius = 0.09  // ~10 km cap
)

type position struct{ lat, lng float64 }

// step moves pos by a random delta, clamping to maxRadius from center.
func (p *position) step(centerLat, centerLng float64) {
	p.lat += (math.Float64() - 0.5) * 2 * stepDeg
	p.lng += (math.Float64() - 0.5) * 2 * stepDeg

	dLat := p.lat - centerLat
	dLng := p.lng - centerLng
	dist := dLat*dLat + dLng*dLng
	if cap := maxRadius * maxRadius; dist > cap {
		scale := maxRadius / (dist * dist) // nudge back
		p.lat = centerLat + dLat*scale
		p.lng = centerLng + dLng*scale
	}
}

// ── driver simulation ─────────────────────────────────────────────────────────

func runDriver(id, pkg, name, token string, wg *sync.WaitGroup) {
	defer wg.Done()

	u, _ := url.Parse(*wsURL)
	q := u.Query()
	q.Set("token", token)
	q.Set("packageSlug", pkg)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("[%s] connect failed: %v", id, err)
		return
	}
	defer conn.Close()
	log.Printf("[%s] connected as %s (package: %s)", id, name, pkg)

	pos := position{
		lat: *centerLat + (math.Float64()-0.5)*0.02,
		lng: *centerLng + (math.Float64()-0.5)*0.02,
	}

	var ticker *time.Ticker
	if *location == "walk" {
		ticker = time.NewTicker(*interval)
		defer ticker.Stop()
	}

	sendLocation := func() error {
		payload, _ := json.Marshal(map[string]any{
			"location": map[string]float64{
				"latitude":  pos.lat,
				"longitude": pos.lng,
			},
		})
		msg, _ := json.Marshal(wsMessage{Type: cmdLocation, Data: payload})
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return err
		}
		log.Printf("[%s] location → %.5f, %.5f", id, pos.lat, pos.lng)
		return nil
	}

	// Static mode registers the driver at a fixed location once.
	if *location == "static" {
		if err := sendLocation(); err != nil {
			log.Printf("[%s] initial location write error: %v", id, err)
			return
		}
	}

	// Channel to receive messages from the server
	incoming := make(chan wsMessage, 8)
	go func() {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				close(incoming)
				return
			}
			var msg wsMessage
			if err := json.Unmarshal(raw, &msg); err == nil {
				incoming <- msg
			}
		}
	}()

	for {
		select {
		case <-func() <-chan time.Time {
			if ticker != nil {
				return ticker.C
			}
			return nil
		}():
			if *location == "walk" {
				pos.step(*centerLat, *centerLng)
				if err := sendLocation(); err != nil {
					log.Printf("[%s] write error: %v", id, err)
					return
				}
			}

		case msg, ok := <-incoming:
			if !ok {
				log.Printf("[%s] disconnected", id)
				return
			}
			switch msg.Type {
			case cmdTripRequest:
				switch *response {
				case "accept":
					log.Printf("[%s] received trip request — auto-accepting", id)
					accept, _ := json.Marshal(wsMessage{Type: cmdTripAccept, Data: msg.Data})
					if err := conn.WriteMessage(websocket.TextMessage, accept); err != nil {
						log.Printf("[%s] accept write error: %v", id, err)
					}
				case "decline":
					log.Printf("[%s] received trip request — auto-declining", id)
					decline, _ := json.Marshal(wsMessage{Type: cmdTripDecline, Data: msg.Data})
					if err := conn.WriteMessage(websocket.TextMessage, decline); err != nil {
						log.Printf("[%s] decline write error: %v", id, err)
					}
				default:
					log.Printf("[%s] received trip request — no auto response (response=%s)", id, *response)
				}
			default:
				log.Printf("[%s] server msg: %s", id, msg.Type)
			}
		}
	}
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	flag.Parse()
	if *response != "none" && *response != "accept" && *response != "decline" {
		log.Fatalf("invalid -response value %q; expected one of: none, accept, decline", *response)
	}
	if *location != "none" && *location != "static" && *location != "walk" {
		log.Fatalf("invalid -location value %q; expected one of: none, static, walk", *location)
	}
	packages := strings.Split(*packagesCSV, ",")

	var wg sync.WaitGroup
	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		for i := 0; i < *countPerPkg; i++ {
			id := fmt.Sprintf("sim-%s-%d", pkg, i)
			name := fmt.Sprintf("Sim Driver %s %d", strings.ToUpper(pkg[:1])+pkg[1:], i+1)
			token, err := makeToken(id, name, *jwtSecret)
			if err != nil {
				log.Fatalf("failed to create token for %s: %v", id, err)
			}
			wg.Add(1)
			go runDriver(id, pkg, name, token, &wg)
		}
	}
	wg.Wait()
}

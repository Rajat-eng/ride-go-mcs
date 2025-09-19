package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-sharing/shared/env"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8081")
)

func main() {
	log.Println("Starting API Gateway")

	mux := http.NewServeMux() // create a new ServeMux for routing

	// Define a simple health check endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Route for trip preview
	mux.HandleFunc("POST /trip/preview", enableCORS(HandleTripPreview))
	mux.HandleFunc("POST /trip/start", enableCORS(HandleStartTrip))

	mux.HandleFunc("/ws/drivers", handleDriversWebSocket)
	mux.HandleFunc("/ws/riders", handleRidersWebSocket)

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	// graceful shutdown
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("HTTP server listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()
	// goroutine to start the server.This allows the main goroutine to continue executing and listen for shutdown signals

	shutdown := make(chan os.Signal, 1)                    // buffered channel to receive shutdown signals
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM) // relay os.Interrupt,SIGTERM to shutdown channel

	select {
	case err := <-serverErrors:
		log.Fatalf("could not start server: %v", err)
	case sig := <-shutdown:
		log.Printf("starting shutdown due to signal: %v", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		// wait for 10 seconds to finish ongoing requests before shutting down
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown did not complete in 10s : %v", err)
			server.Close() // force close
		}
	}
}

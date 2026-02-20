package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ayushbhandari/event-api/internal/db"
	httpapi "github.com/ayushbhandari/event-api/internal/http"
)

func main() {
	uri := os.Getenv("EVENT_API_MONGO_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017/infosys"
	}

	ctx := context.Background()

	client, err := db.OpenMongo(ctx, uri)
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}
	defer func() {
		_ = client.Disconnect(ctx)
	}()

	router := httpapi.NewRouter(client)

	srv := &http.Server{
		Addr:         ":3000",
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("event-api listening on :3000")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}



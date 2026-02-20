package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ayushbhandari/event-api/internal/db"
	"github.com/ayushbhandari/event-api/internal/events"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var repo *events.Repository

func BookTicket(eventID, userID primitive.ObjectID) error {
	return repo.RegisterForEvent(context.Background(), eventID, userID)
}

func SimulateConcurrentBooking(eventID primitive.ObjectID) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0

	totalUsers := 50

	start := time.Now()

	for i := 0; i < totalUsers; i++ {
		wg.Add(1)

		go func(userNum int) {
			defer wg.Done()

			userID := primitive.NewObjectID()

			err := BookTicket(eventID, userID)

			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				successCount++
				fmt.Printf("  goroutine %02d  ✅  BOOKED   (userID: %s)\n", userNum+1, userID.Hex())
			} else {
				fmt.Printf("  goroutine %02d  ❌  FAILED   (%v)\n", userNum+1, err)
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(start)

	fmt.Println("Total Attempts:    ", totalUsers)
	fmt.Println("Successful Bookings:", successCount)
	fmt.Println("Time Taken:        ", duration)

	if successCount == 1 {
		fmt.Println("\n✅  PASS — exactly 1 booking succeeded (no overbooking)")
	} else {
		fmt.Printf("\n❌  FAIL — expected 1 booking, got %d (overbooking!)\n", successCount)
	}
}

func main() {
	uri := os.Getenv("EVENT_API_MONGO_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017/infosys"
	}

	ctx := context.Background()

	client, err := db.OpenMongo(ctx, uri)
	if err != nil {
		log.Fatalf("OpenMongo: %v", err)
	}
	defer func() { _ = client.Disconnect(ctx) }()

	testDB := client.Database("conctest")
	repo = events.NewRepository(testDB)

	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)

	organizerID := primitive.NewObjectID()

	ev, err := repo.CreateEvent(
		ctx,
		organizerID,
		"Capacity-1 Stress Test",
		"Only one ticket available",
		1,
		startsAt,
		endsAt,
	)
	if err != nil {
		log.Fatalf("CreateEvent: %v", err)
	}

	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  Event.io — Concurrency Stress Test")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("Event ID : %s\n", ev.ID.Hex())
	fmt.Printf("Capacity : %d\n\n", ev.Capacity)

	SimulateConcurrentBooking(ev.ID)

	final, err := repo.GetEvent(ctx, ev.ID)
	if err != nil || final == nil {
		log.Fatalf("GetEvent: %v", err)
	}
	fmt.Printf("\nMongoDB final state  →  tickets_sold=%d  registration_ids=%d\n",
		final.TicketsSold, len(final.RegistrationIDs))

	_ = testDB.Drop(ctx)
}

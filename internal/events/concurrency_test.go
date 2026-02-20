package events

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ayushbhandari/event-api/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestConcurrentRegistrations(t *testing.T) {
	uri := os.Getenv("EVENT_API_MONGO_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017/infosys"
	}

	ctx := context.Background()

	client, err := db.OpenMongo(ctx, uri)
	if err != nil {
		t.Fatalf("OpenMongo: %v", err)
	}
	defer func() {
		_ = client.Disconnect(ctx)
	}()

	dbMongo := client.Database("infosys")

	if err := dbMongo.Collection("events").Drop(ctx); err != nil {
		t.Fatalf("drop events: %v", err)
	}
	if err := dbMongo.Collection("users").Drop(ctx); err != nil {
		t.Fatalf("drop users: %v", err)
	}

	repo := NewRepository(dbMongo)

	const userCount = 20
	users := make([]*User, 0, userCount)
	for i := 0; i < userCount; i++ {
		u, err := repo.CreateUser(ctx, "User"+string(rune('A'+i)),
			"user"+string(rune('A'+i))+"@example.com")
		if err != nil {
			t.Fatalf("CreateUser %d: %v", i, err)
		}
		users = append(users, u)
	}

	startsAt := time.Now().Add(1 * time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)
	capacity := 5

	ev, err := repo.CreateEvent(ctx, users[0].ID, "Concurrent Test Event", "Testing concurrent registrations", capacity, startsAt, endsAt)
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(userCount)

	var successCount int64
	var failCapacityCount int64

	for i := 0; i < userCount; i++ {
		u := users[i]
		go func(userID primitive.ObjectID) {
			defer wg.Done()
			err := repo.RegisterForEvent(ctx, ev.ID, userID)
			if err != nil {
				if err == ErrCapacityFull {
					atomic.AddInt64(&failCapacityCount, 1)
					return
				}
				if err == ErrAlreadyRegistered {
					return
				}
				t.Errorf("RegisterForEvent unexpected error: %v", err)
				return
			}
			atomic.AddInt64(&successCount, 1)
		}(u.ID)
	}

	wg.Wait()

	var stored Event
	if err := dbMongo.Collection("events").FindOne(ctx, bson.M{"_id": ev.ID}).Decode(&stored); err != nil {
		t.Fatalf("find event: %v", err)
	}

	if stored.TicketsSold > capacity {
		t.Fatalf("overbooking detected: tickets_sold=%d capacity=%d", stored.TicketsSold, capacity)
	}

	regCount := len(stored.RegistrationIDs)
	if regCount != capacity {
		t.Fatalf("expected exactly %d registrations, got %d (successCount=%d, failCapacityCount=%d)", capacity, regCount, successCount, failCapacityCount)
	}
}

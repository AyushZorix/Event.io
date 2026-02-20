package events

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrCapacityFull      = errors.New("event capacity reached")
	ErrAlreadyRegistered = errors.New("user already registered for event")
)

type Repository struct {
	db         *mongo.Database
	usersCol   *mongo.Collection
	eventsCol  *mongo.Collection
}

func NewRepository(db *mongo.Database) *Repository {
	r := &Repository{
		db:        db,
		usersCol:  db.Collection("users"),
		eventsCol: db.Collection("events"),
	}
	if err := r.EnsureIndexes(context.Background()); err != nil {
		log.Printf("[warn] EnsureIndexes: %v", err)
	}
	return r
}

func (r *Repository) EnsureIndexes(ctx context.Context) error {
	_, err := r.usersCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("users_email_unique"),
		},
	})
	if err != nil {
		return fmt.Errorf("users indexes: %w", err)
	}

	_, err = r.eventsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "starts_at", Value: 1}},
			Options: options.Index().SetName("events_starts_at"),
		},
		{
			Keys:    bson.D{{Key: "registration_ids", Value: 1}},
			Options: options.Index().SetName("events_reg_ids"),
		},
	})
	if err != nil {
		return fmt.Errorf("events indexes: %w", err)
	}
	return nil
}

func (r *Repository) CreateUser(ctx context.Context, name, email, password string) (*User, error) {
	u := &User{
		ID:        primitive.NilObjectID,
		Name:      name,
		Email:     email,
		Password:  password,
		CreatedAt: time.Now().UTC(),
	}
	res, err := r.usersCol.InsertOne(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	u.ID = res.InsertedID.(primitive.ObjectID)
	return u, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.usersCol.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &u, nil
}

func (r *Repository) CreateEvent(ctx context.Context, organizerID primitive.ObjectID, title, description string, capacity int, startsAt, endsAt time.Time) (*Event, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("capacity must be greater than 0")
	}
	e := &Event{
		ID:            primitive.NilObjectID,
		OrganizerID:   organizerID,
		Title:         title,
		Description:   description,
		Capacity:      capacity,
		TicketsSold:   0,
		StartsAt:      startsAt,
		EndsAt:        endsAt,
		CreatedAt:     time.Now().UTC(),
		RegistrationIDs: []primitive.ObjectID{},
	}
	res, err := r.eventsCol.InsertOne(ctx, e)
	if err != nil {
		return nil, fmt.Errorf("insert event: %w", err)
	}
	e.ID = res.InsertedID.(primitive.ObjectID)
	return e, nil
}

func (r *Repository) ListEvents(ctx context.Context) ([]Event, error) {
	findOpts := options.Find().SetSort(bson.D{{Key: "starts_at", Value: 1}})
	cur, err := r.eventsCol.Find(ctx, bson.D{}, findOpts)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer cur.Close(ctx)

	var result []Event
	for cur.Next(ctx) {
		var e Event
		if err := cur.Decode(&e); err != nil {
			return nil, fmt.Errorf("decode event: %w", err)
		}
		result = append(result, e)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("list events cursor: %w", err)
	}
	return result, nil
}

func (r *Repository) GetEvent(ctx context.Context, id primitive.ObjectID) (*Event, error) {
	var e Event
	if err := r.eventsCol.FindOne(ctx, bson.M{"_id": id}).Decode(&e); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("get event: %w", err)
	}
	return &e, nil
}

func (r *Repository) RegisterForEvent(ctx context.Context, eventID, userID primitive.ObjectID) error {
	filter := bson.M{
		"_id":              eventID,
		"registration_ids": bson.M{"$ne": userID},
		"$expr":            bson.M{"$lt": bson.A{"$tickets_sold", "$capacity"}},
	}
	update := bson.M{
		"$inc":      bson.M{"tickets_sold": 1},
		"$addToSet": bson.M{"registration_ids": userID},
	}

	res, err := r.eventsCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update event for registration: %w", err)
	}

	if res.MatchedCount == 0 {
		var e Event
		if err := r.eventsCol.FindOne(ctx, bson.M{"_id": eventID}).Decode(&e); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return fmt.Errorf("event not found")
			}
			return fmt.Errorf("post-check get event: %w", err)
		}

		for _, rid := range e.RegistrationIDs {
			if rid == userID {
				return ErrAlreadyRegistered
			}
		}
		return ErrCapacityFull
	}

	return nil
}

func (r *Repository) ListRegistrations(ctx context.Context, eventID primitive.ObjectID) ([]primitive.ObjectID, error) {
	var e Event
	if err := r.eventsCol.FindOne(ctx, bson.M{"_id": eventID}).Decode(&e); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("get event for registrations: %w", err)
	}
	return e.RegistrationIDs, nil
}


package events

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"password" json:"-"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type Event struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	OrganizerID   primitive.ObjectID   `bson:"organizer_id" json:"organizer_id"`
	Title         string               `bson:"title" json:"title"`
	Description   string               `bson:"description" json:"description"`
	Capacity      int                  `bson:"capacity" json:"capacity"`
	TicketsSold   int                  `bson:"tickets_sold" json:"tickets_sold"`
	StartsAt      time.Time            `bson:"starts_at" json:"starts_at"`
	EndsAt        time.Time            `bson:"ends_at" json:"ends_at"`
	CreatedAt     time.Time            `bson:"created_at" json:"created_at"`
	RegistrationIDs []primitive.ObjectID `bson:"registration_ids,omitempty" json:"registration_ids,omitempty"`
}

type Registration struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	EventID   primitive.ObjectID `bson:"event_id" json:"event_id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Status    string             `bson:"status" json:"status"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}


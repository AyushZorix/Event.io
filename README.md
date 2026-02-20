# 🎟 EVENT.IO

> Transform Events with Seamless, Reliable Engagement

Event.io is a high-performance event registration backend built with Go and MongoDB.  
It is designed to handle concurrent registrations safely while preventing overbooking and duplicate registrations.

---

##  Overview

Event.io provides a RESTful API for managing events and registrations.  
It ensures data integrity even under high traffic using atomic database operations.

Key highlights:

- Prevents overbooking during simultaneous registrations
- Ensures duplicate registration protection
- Implements proper REST API conventions
- Supports concurrency simulation testing
- Designed for horizontal scalability

---

##  Tech Stack

- Go (Golang)
- MongoDB
- HTML (UI Layer)
- Go Modules

---

##  Base URL
http://localhost:3000



---

##  API Routes

### 1️⃣ Create User  
**POST** `/api/users`

Response:
- `201 Created` → Returns the created user object with `id` (save this for future requests)

---

### 2️⃣ Login  
**POST** `/api/login`

Response:
- `200 OK` → Returns the user object with `id`

---

### 3️⃣ Create Event  
**POST** `/api/events`

Response:
- `201 Created` → Returns the created event object with `id` (save this)

---

### 4️⃣ List All Events  
**GET** `/api/events`

Response:
- `200 OK` → Returns an array of all events sorted by `starts_at`

---

### 5️⃣ Get Single Event  
**GET** `/api/events/<event_id>`

Response:
- `200 OK` → Returns the event object  
- `404 Not Found` → If event does not exist

---

### 6️⃣ Register for Event  
**POST** `/api/events/<event_id>/registrations`

Response:
- `201 Created` → Registration successful  
- `400 Bad Request` → User already registered  
- `409 Conflict` → Event capacity full  

---

### 7️⃣ List Registrations for Event  
**GET** `/api/events/<event_id>/registrations`

Response:
- `200 OK` → Returns array of user ObjectIDs registered to the event

---

##  Concurrency Strategy

The system prevents race conditions using MongoDB’s atomic conditional updates.

Instead of:
1. Checking ticket availability
2. Updating ticket count separately

The system performs both operations in a single atomic database call:

Increment ticketsSold ONLY IF ticketsSold < capacity


This guarantees:

- Only one request can update the event document at a time
- No overbooking even if multiple users register simultaneously
- Safe behavior in distributed environments
- No reliance on application-level mutex locking

MongoDB ensures document-level atomicity, meaning concurrent updates are serialized internally.

---

##  Concurrency Testing

Event.io includes a goroutine-based simulation that:

- Spawns 20–50 concurrent registration attempts
- Targets an event with capacity = 1
- Verifies that only one registration succeeds

Expected Result:

Total Attempts: 50
Successful Registrations: 1


This confirms the system is safe under concurrent load.

---

##  Database Schema

### Events Collection

{
    _id: ObjectId('69984dff943c75c5e975bedc'),
    organizer_id: ObjectId('69984de4943c75c5e975bedb'),
    title: 'boom',
    description: 'yes yes yes',
    capacity: 77,
    tickets_sold: 1,
    starts_at: ISODate('2026-02-12T14:06:00.000Z'),
    ends_at: ISODate('2026-02-26T04:04:00.000Z'),
    created_at: ISODate('2026-02-20T12:05:19.581Z'),
    registration_ids: [ ObjectId('6998c84a0e62c75be83197a8') ]
  }


---

### Users Collection

{
    _id: ObjectId('6998ae4d9d838f19acdf9b0c'),
    name: 'ayush',
    email: 'a@g.com',
    password: 'cnahnx44233njdnjnsajcn3qxx333-322',
    created_at: ISODate('2026-02-20T18:56:13.634Z')
}


##  Installation

Clone repository: git clone https://github.com/AyushZorix/Event.io


Navigate into project:
cd Event.io

Build project:
go build

---

## ▶ Running the Server
go run main.go

Server runs on:
http://localhost:3000

---

##  Design Principles

- RESTful endpoint structure
- Proper HTTP status codes
- JSON-based responses
- Optimistic concurrency control
- Horizontal scalability
- Clean modular architecture

---

##  Scalability

The system avoids application-level locks (like mutex) because they do not work across multiple server instances.

All concurrency safety is handled at the database level, making the system production-ready and scalable.

---

## 📜 License

MIT License

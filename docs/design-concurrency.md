# Concurrency & Race-Condition Strategy

This document explains how the API prevents overbooking and race conditions when multiple users attempt to register for the same event concurrently.

---

## The Problem

For an event with capacity $C$, we must guarantee that:

- No more than $C$ confirmed registrations are ever stored.
- Concurrent registration attempts cannot “sneak past” the capacity check.
- The solution is robust even under high contention and process crashes.

### Data Model

Relevant fields:

- **`events`**
  - `capacity INT NOT NULL CHECK (capacity > 0)`
  - `tickets_sold INT NOT NULL DEFAULT 0 CHECK (tickets_sold >= 0)`
  - `CHECK (tickets_sold <= capacity)`
- **`registrations`**
  - `event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE`
  - `user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE`
  - `UNIQUE (event_id, user_id)` to prevent duplicate registrations per user.

The `tickets_sold` field allows us to avoid a full `COUNT(*)` aggregation on every registration while still enforcing capacity constraints.

### Core Algorithm: Row-Level Locking + Transaction

The `RegisterForEvent` method in `internal/events/repository.go` performs registration using a **single SQL transaction**:

1. **Begin transaction**
2. **Lock the event row**

   ```sql
   SELECT capacity, tickets_sold
   FROM events
   WHERE id = $1
   FOR UPDATE;
   ```

   - `FOR UPDATE` acquires a row-level lock on the event.
   - Any concurrent transaction trying to lock the same row will block until the current transaction commits or rolls back.

3. **Check capacity inside the transaction**

   - If `tickets_sold >= capacity`, the transaction returns `ErrCapacityFull`.

4. **Insert registration**

   ```sql
   INSERT INTO registrations (event_id, user_id, status)
   VALUES ($1, $2, 'confirmed')
   RETURNING id, event_id, user_id, status, created_at;
   ```

   - If the `(event_id, user_id)` unique constraint is violated, we treat this as `ErrAlreadyRegistered`.

5. **Increment `tickets_sold`**

   ```sql
   UPDATE events
   SET tickets_sold = tickets_sold + 1
   WHERE id = $1;
   ```

6. **Commit transaction**

Only after the commit do other blocked transactions proceed, at which point they will see the updated `tickets_sold` value.

### Why This Prevents Overbooking

- **Serialization on the event row**
  - `SELECT ... FOR UPDATE` ensures that only one transaction at a time can mutate `tickets_sold` for a given event.
- **Capacity check and increment are atomic**
  - The capacity check and `tickets_sold++` happen in the same transaction on locked data.
  - No other transaction can read a stale `tickets_sold` and then increment it behind our back.
- **Database-level guard**
  - A `CHECK (tickets_sold <= capacity)` constraint provides an additional invariant in the database.
  - If a bug or mis-ordered sequence ever violated this rule, the database would reject the update.

The combination of **row-level locks**, **single-transaction mutation**, and **constraints** eliminates the classic read-modify-write race that leads to overbooking.

### Concurrent Booking Test

The integration test `internal/events/concurrency_test.go` simulates a realistic race:

- Creates many users.
- Creates a single event with a small capacity (e.g. 5).
- Spawns multiple goroutines (one per user).
- Each goroutine attempts to register its user for the event via `RegisterForEvent`.
- After all goroutines complete, the test asserts:
  - `tickets_sold` for the event equals the configured capacity.
  - The number of registration rows for the event equals the capacity.

Even with high contention, the test demonstrates that:

- Exactly `capacity` users are successfully registered.
- No additional registrations slip through once capacity is reached.

### Alternative Strategies (Not Implemented)

For completeness, other approaches that could be used (but are not implemented here):

- **Optimistic locking** with a version column and `WHERE id = $1 AND version = $2` in the update.
- **Advisory locks** in Postgres (e.g. `pg_advisory_xact_lock` on `event_id`).
- **Reservation queues** using a message broker where all registrations are serialized through a single consumer.

The chosen approach (row-level locking with `FOR UPDATE`) is:

- Simple to understand and implement.
- Fully enforced at the database layer.
- Sufficient for typical event-registration scale and correctness requirements.



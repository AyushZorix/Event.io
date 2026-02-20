# Concurrency: Goroutines + Atomic MongoDB Update

## The Problem

Multiple users can attempt to book the last ticket at the same time. Without protection, two goroutines can both read `tickets_sold < capacity`, both pass the check, and both write — resulting in overbooking.

## The Solution

A single atomic `UpdateOne` call in MongoDB. The filter and the increment happen inside WiredTiger's document-level write lock, so no two goroutines can satisfy the same condition simultaneously.

```go
filter := bson.M{
    "_id":              eventID,
    "registration_ids": bson.M{"$ne": userID},
    "$expr":            bson.M{"$lt": bson.A{"$tickets_sold", "$capacity"}},
}
update := bson.M{
    "$inc":      bson.M{"tickets_sold": 1},
    "$addToSet": bson.M{"registration_ids": userID},
}
res, err := eventsCol.UpdateOne(ctx, filter, update)
```

**Filter guards (both evaluated atomically with the write):**
- `registration_ids: {$ne: userID}` — user is not already registered
- `$expr: {$lt: ["$tickets_sold", "$capacity"]}` — there is still capacity

**Update operators:**
- `$inc` — atomically increments `tickets_sold` by 1
- `$addToSet` — adds the user ID to the array only if not already present (idempotent)

If `MatchedCount == 0` after the update, the document is re-read to determine whether the user was already registered (`ErrAlreadyRegistered`) or the event was full (`ErrCapacityFull`).

## Indexes

```js
db.events.createIndex({ starts_at: 1 },       { name: "events_starts_at" })
db.events.createIndex({ registration_ids: 1 }, { name: "events_reg_ids" })
```

The multikey index on `registration_ids` makes the `$ne` filter fast at scale.

## Stress Test Result

50 goroutines fired simultaneously at a capacity-1 event:

```
Total Attempts:     50
Successful Bookings: 1
Time Taken:         13ms

✅  PASS — exactly 1 booking succeeded (no overbooking)

MongoDB final state  →  tickets_sold=1  registration_ids=1
```

Only one goroutine wins the atomic race. All 49 others receive `event capacity reached`.

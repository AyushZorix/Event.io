Q1 — Can the same user register twice?

No. Duplicate registration is prevented at the database level.
When a user registers, the update query includes this condition:

{ "registration_ids": { "$ne": userID } }

This ensures the update only succeeds if the user’s ID is not already present in the registration_ids array.
If the user tries to register again:
The filter does not match
MatchedCount == 0
The system re-checks the document
If the user is already in registration_ids, it returns ErrAlreadyRegistered
Additionally, we use:
$addToSet
instead of $push.
This guarantees that even if two identical requests somehow race simultaneously, MongoDB will only store the user ID once.
HTTP Response:
400 Bad Request — "user already registered for event"


Q2 — What happens if capacity is 0?
Capacity is validated before the event is created.
In CreateEvent:
if capacity <= 0 {
    return nil, fmt.Errorf("capacity must be greater than 0")
}
This prevents invalid events from being stored.
Even if someone bypassed this check and a capacity-0 event somehow existed, registration would still fail because the atomic filter:
{ "$expr": { "$lt": ["$tickets_sold", "$capacity"] } }
would evaluate to:
0 < 0 → false
So no registration could ever succeed.
The validation exists to fail early and return a clear error instead of silently creating an unusable event.
HTTP Response:
400 Bad Request — "capacity must be greater than 0"


Q3 — What happens if the event doesn't exist?
If a registration request is made with a non-existent event ID:
The UpdateOne operation matches zero documents
The system performs a FindOne
If MongoDB returns ErrNoDocuments, we interpret it as "event not found"
For GET requests:
The router returns 404 Not Found
For POST registration:
The API returns a clear "event not found" error
This prevents silent failures and gives meaningful feedback.
HTTP Response:
404 Not Found
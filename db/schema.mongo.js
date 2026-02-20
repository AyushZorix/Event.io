db.createCollection("users");

db.users.createIndex(
  { email: 1 },
  { unique: true, name: "users_email_unique" }
);

db.createCollection("events");

db.events.createIndex(
  { starts_at: 1 },
  { name: "events_starts_at" }
);

db.events.createIndex(
  { registration_ids: 1 },
  { name: "events_reg_ids" }
);

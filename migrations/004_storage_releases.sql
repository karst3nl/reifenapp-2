CREATE TABLE storage_releases (
 id INTEGER PRIMARY KEY,
 claim_id INTEGER NOT NULL UNIQUE REFERENCES storage_claims(id),
 released_at TEXT NOT NULL
);

CREATE TABLE admin_account (
 id INTEGER PRIMARY KEY CHECK(id=1),
 username TEXT NOT NULL UNIQUE CHECK(username='admin'),
 password_hash TEXT NOT NULL,
 must_change INTEGER NOT NULL CHECK(must_change IN (0,1)),
 revision INTEGER NOT NULL CHECK(revision>0),
 updated_at TEXT NOT NULL
);

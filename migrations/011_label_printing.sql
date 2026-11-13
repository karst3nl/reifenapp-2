CREATE TABLE label_printer_settings (
 id INTEGER PRIMARY KEY CHECK(id=1),
 config TEXT NOT NULL CHECK(json_valid(config)),
 revision INTEGER NOT NULL DEFAULT 1
);
INSERT INTO label_printer_settings(id,config) VALUES(1,'{}');
CREATE TABLE label_print_jobs (
 id INTEGER PRIMARY KEY,
 event_id INTEGER UNIQUE REFERENCES history_events(id),
 receipt TEXT NOT NULL CHECK(json_valid(receipt)),
 config TEXT NOT NULL CHECK(json_valid(config)),
 status TEXT NOT NULL CHECK(status IN ('pending','processing','submitted','failed','uncertain','cancelled')),
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL,
 cups_id TEXT NOT NULL DEFAULT '',
 message TEXT NOT NULL DEFAULT '',
 attempts INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX label_jobs_status ON label_print_jobs(status,id);

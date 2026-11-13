CREATE TABLE csv_import_batches (
 id INTEGER PRIMARY KEY,
 file_sha256 TEXT NOT NULL UNIQUE,
 filename TEXT NOT NULL,
 imported_at TEXT NOT NULL,
 encoding TEXT NOT NULL,
 delimiter TEXT NOT NULL,
 headers TEXT NOT NULL CHECK(json_valid(headers)),
 row_count INTEGER NOT NULL,
 imported_count INTEGER NOT NULL,
 duplicate_count INTEGER NOT NULL,
 warning_count INTEGER NOT NULL,
 raw_csv BLOB NOT NULL
);
CREATE TABLE history_events_new (
 id INTEGER PRIMARY KEY,
 claim_id INTEGER REFERENCES storage_claims(id),
 operation TEXT NOT NULL CHECK(operation IN ('storage','wheel_change','reservation','release','imported')),
 occurred_at TEXT NOT NULL,
 place TEXT NOT NULL,
 name TEXT NOT NULL,
 plate TEXT NOT NULL,
 car TEXT NOT NULL,
 payload TEXT NOT NULL CHECK(json_valid(payload)),
 source TEXT NOT NULL CHECK(source IN ('recorded','legacy','csv')),
 import_key TEXT UNIQUE,
 UNIQUE(claim_id,operation),
 CHECK((source='csv' AND claim_id IS NULL AND import_key IS NOT NULL) OR
       (source!='csv' AND claim_id IS NOT NULL AND import_key IS NULL)),
 CHECK(operation!='imported' OR source='csv')
);
INSERT INTO history_events_new(id,claim_id,operation,occurred_at,place,name,plate,car,payload,source)
 SELECT id,claim_id,operation,occurred_at,place,name,plate,car,payload,source FROM history_events;
DROP TABLE history_events;
ALTER TABLE history_events_new RENAME TO history_events;
CREATE INDEX history_events_date ON history_events(occurred_at DESC,id DESC);
CREATE TRIGGER history_events_no_update BEFORE UPDATE ON history_events
 BEGIN SELECT RAISE(ABORT,'history events are immutable'); END;
CREATE TRIGGER history_events_no_delete BEFORE DELETE ON history_events
 BEGIN SELECT RAISE(ABORT,'history events are immutable'); END;
CREATE TABLE csv_import_rows (
 id INTEGER PRIMARY KEY,
 batch_id INTEGER NOT NULL REFERENCES csv_import_batches(id),
 event_id INTEGER NOT NULL REFERENCES history_events(id),
 line_number INTEGER NOT NULL,
 original_values TEXT NOT NULL CHECK(json_valid(original_values)),
 warnings TEXT NOT NULL CHECK(json_valid(warnings)),
 UNIQUE(batch_id,line_number)
);

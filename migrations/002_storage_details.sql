ALTER TABLE vehicles ADD COLUMN brand TEXT NOT NULL DEFAULT '';
ALTER TABLE vehicles ADD COLUMN model TEXT NOT NULL DEFAULT '';
CREATE TABLE storage_details (
 claim_id INTEGER PRIMARY KEY REFERENCES storage_claims(id),
 tire_type TEXT NOT NULL,
 rim_type TEXT NOT NULL,
 front_size TEXT NOT NULL CHECK(json_valid(front_size)),
 rear_size TEXT NOT NULL CHECK(json_valid(rear_size)),
 front_depth TEXT NOT NULL,
 rear_depth TEXT NOT NULL,
 accessories TEXT NOT NULL CHECK(json_valid(accessories)),
 mechanic TEXT NOT NULL,
 notes TEXT NOT NULL
);

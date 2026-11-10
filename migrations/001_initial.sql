CREATE TABLE customers (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
CREATE TABLE vehicles (
 id INTEGER PRIMARY KEY, customer_id INTEGER NOT NULL REFERENCES customers(id),
 plate TEXT NOT NULL, car TEXT NOT NULL
);
CREATE TABLE tire_sets (id INTEGER PRIMARY KEY, vehicle_id INTEGER NOT NULL REFERENCES vehicles(id));
CREATE TABLE storage_places (
 id TEXT PRIMARY KEY,
 area TEXT NOT NULL, number INTEGER NOT NULL,
 CHECK ((area='C1' AND number BETWEEN 1 AND 200)
 OR (area='C2' AND number BETWEEN 201 AND 400)
 OR (area IN ('C3','C4','OL','OR') AND number BETWEEN 1 AND 60)),
 CHECK (id = area || '.' || number), UNIQUE(area,number)
);
WITH RECURSIVE n(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM n WHERE x<400),
 areas(area,lo,hi) AS (VALUES('C1',1,200),('C2',201,400),('C3',1,60),('C4',1,60),('OL',1,60),('OR',1,60))
INSERT INTO storage_places SELECT area || '.' || x,area,x FROM areas JOIN n ON x BETWEEN lo AND hi;
-- Belegung und Reservierung teilen dieselbe exklusive Platzsperre.
CREATE TABLE storage_claims (
 id INTEGER PRIMARY KEY,
 place_id TEXT NOT NULL REFERENCES storage_places(id),
 tire_set_id INTEGER REFERENCES tire_sets(id),
 kind TEXT NOT NULL CHECK(kind IN ('occupancy','reservation')),
 status TEXT NOT NULL CHECK(status IN ('active','released','converted','cancelled','expired')),
 updated_at TEXT NOT NULL,
 CHECK(kind != 'occupancy' OR tire_set_id IS NOT NULL),
 CHECK((kind='occupancy' AND status IN ('active','released')) OR
       (kind='reservation' AND status IN ('active','converted','cancelled','expired')))
);
CREATE UNIQUE INDEX one_active_claim_per_place ON storage_claims(place_id) WHERE status='active';
CREATE UNIQUE INDEX one_active_claim_per_tire_set ON storage_claims(tire_set_id) WHERE status='active' AND tire_set_id IS NOT NULL;

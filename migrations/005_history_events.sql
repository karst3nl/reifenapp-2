CREATE TABLE history_events (
 id INTEGER PRIMARY KEY,
 claim_id INTEGER NOT NULL REFERENCES storage_claims(id),
 operation TEXT NOT NULL CHECK(operation IN ('storage','wheel_change','reservation','release')),
 occurred_at TEXT NOT NULL,
 place TEXT NOT NULL,
 name TEXT NOT NULL,
 plate TEXT NOT NULL,
 car TEXT NOT NULL,
 payload TEXT NOT NULL CHECK(json_valid(payload)),
 source TEXT NOT NULL CHECK(source IN ('recorded','legacy')),
 UNIQUE(claim_id,operation)
);
CREATE INDEX history_events_date ON history_events(occurred_at DESC,id DESC);
-- A shared snapshot source; searches read the immutable event copies.
CREATE VIEW claim_history_snapshot AS
 SELECT s.id AS claim_id,s.place_id AS place,COALESCE(c.name,'') AS name,
 COALESCE(v.plate,'') AS plate,COALESCE(v.car,'') AS car,s.updated_at AS initial_date,
 CASE WHEN s.kind='reservation' THEN 'reservation' ELSE COALESCE(d.operation,'storage') END AS operation,
 json_object('customer_id',c.id,'vehicle_id',v.id,'tire_set_id',t.id,
 'brand',v.brand,'model',v.model,'tire_type',d.tire_type,'rim_type',d.rim_type,
 'front',json(COALESCE(d.front_size,'{}')),'rear',json(COALESCE(d.rear_size,'{}')),
 'front_depth',d.front_depth,'rear_depth',d.rear_depth,
 'accessories',json(COALESCE(d.accessories,'{}')),'mechanic',d.mechanic,'notes',d.notes) AS payload
 FROM storage_claims s LEFT JOIN tire_sets t ON t.id=s.tire_set_id
 LEFT JOIN vehicles v ON v.id=t.vehicle_id LEFT JOIN customers c ON c.id=v.customer_id
 LEFT JOIN storage_details d ON d.claim_id=s.id;
-- Reconstruct only events supported by retained records; mark them as legacy.
INSERT INTO history_events(claim_id,operation,occurred_at,place,name,plate,car,payload,source)
 SELECT claim_id,operation,initial_date,place,name,plate,car,payload,'legacy' FROM claim_history_snapshot;
INSERT INTO history_events(claim_id,operation,occurred_at,place,name,plate,car,payload,source)
 SELECT r.claim_id,'release',r.released_at,s.place,s.name,s.plate,s.car,s.payload,'legacy'
 FROM storage_releases r JOIN claim_history_snapshot s ON s.claim_id=r.claim_id;
CREATE TRIGGER history_events_no_update BEFORE UPDATE ON history_events
 BEGIN SELECT RAISE(ABORT,'history events are immutable'); END;
CREATE TRIGGER history_events_no_delete BEFORE DELETE ON history_events
 BEGIN SELECT RAISE(ABORT,'history events are immutable'); END;

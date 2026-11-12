-- Rebuild only the place catalogue; retain every identifier and foreign key.
CREATE TABLE storage_areas (
 code TEXT PRIMARY KEY, first_number INTEGER NOT NULL, last_number INTEGER NOT NULL,
 revision INTEGER NOT NULL DEFAULT 1,
 CHECK(length(code) BETWEEN 1 AND 12 AND code NOT GLOB '*[^A-Z0-9_-]*'),
 CHECK(first_number>=1 AND last_number>=first_number AND last_number<=100000 AND last_number-first_number<10000)
);
INSERT INTO storage_areas(code,first_number,last_number)
 SELECT area,min(number),max(number) FROM storage_places GROUP BY area;
CREATE TABLE storage_places_new (
 id TEXT PRIMARY KEY, area TEXT NOT NULL REFERENCES storage_areas(code),
 number INTEGER NOT NULL CHECK(number BETWEEN 1 AND 100000),
 enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
 CHECK(id=area || '.' || number), UNIQUE(area,number)
);
INSERT INTO storage_places_new(id,area,number) SELECT id,area,number FROM storage_places;
DROP TABLE storage_places;
ALTER TABLE storage_places_new RENAME TO storage_places;
CREATE TRIGGER active_place_insert BEFORE INSERT ON storage_claims
 WHEN NEW.status='active' AND NOT EXISTS(SELECT 1 FROM storage_places WHERE id=NEW.place_id AND enabled=1)
 BEGIN SELECT RAISE(ABORT,'inactive storage place'); END;
CREATE TRIGGER active_place_update BEFORE UPDATE OF status,place_id ON storage_claims
 WHEN NEW.status='active' AND NOT EXISTS(SELECT 1 FROM storage_places WHERE id=NEW.place_id AND enabled=1)
 BEGIN SELECT RAISE(ABORT,'inactive storage place'); END;
CREATE TRIGGER protect_active_place BEFORE UPDATE OF enabled ON storage_places
 WHEN NEW.enabled=0 AND EXISTS(SELECT 1 FROM storage_claims WHERE place_id=OLD.id AND status='active')
 BEGIN SELECT RAISE(ABORT,'active storage claim'); END;
CREATE TRIGGER configured_place_insert BEFORE INSERT ON storage_places
 WHEN NEW.enabled=1 AND NOT EXISTS(SELECT 1 FROM storage_areas WHERE code=NEW.area AND NEW.number BETWEEN first_number AND last_number)
 BEGIN SELECT RAISE(ABORT,'outside storage range'); END;
CREATE TRIGGER configured_place_update BEFORE UPDATE ON storage_places
 WHEN NEW.enabled=1 AND NOT EXISTS(SELECT 1 FROM storage_areas WHERE code=NEW.area AND NEW.number BETWEEN first_number AND last_number)
 BEGIN SELECT RAISE(ABORT,'outside storage range'); END;

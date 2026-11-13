CREATE TABLE inactivity_settings (
 id INTEGER PRIMARY KEY CHECK(id=1),
 months INTEGER NOT NULL CHECK(months BETWEEN 1 AND 120)
);
INSERT INTO inactivity_settings(id,months) VALUES(1,12);

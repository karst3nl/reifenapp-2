CREATE TABLE accessory_options (
 id INTEGER PRIMARY KEY CHECK(id=1),
 revision INTEGER NOT NULL DEFAULT 1,
 locations TEXT NOT NULL CHECK(json_valid(locations))
);
INSERT INTO accessory_options(id,locations) VALUES(1,
 '{"hubcaps":["Am Radsatz","Beutel am Radsatz","Im Fahrzeug","Beim Kunden","Kofferraum"],"bolts":["Am Radsatz","Beutel am Radsatz","Im Fahrzeug","Beim Kunden","Kofferraum"],"lock":["Am Radsatz","Beutel am Radsatz","Im Fahrzeug","Beim Kunden","Kofferraum"]}');

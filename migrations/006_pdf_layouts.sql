CREATE TABLE pdf_layouts (
 operation TEXT PRIMARY KEY CHECK(operation IN ('storage','wheel_change','reservation','release')),
 layout TEXT NOT NULL CHECK(json_valid(layout)),
 enabled INTEGER NOT NULL CHECK(enabled IN (0,1)),
 revision INTEGER NOT NULL CHECK(revision>0),
 updated_at TEXT NOT NULL
);

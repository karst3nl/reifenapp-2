-- Existing records were created through the Neueinlagerung workflow.
ALTER TABLE storage_details ADD COLUMN operation TEXT NOT NULL DEFAULT 'storage'
 CHECK (operation IN ('storage','wheel_change'));

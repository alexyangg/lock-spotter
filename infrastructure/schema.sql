-- Ensure the spatial extension is active
CREATE EXTENSION IF NOT EXISTS postgis;

-- Drop tables if resetting
DROP TABLE IF EXISTS theft_incidents;
DROP TABLE IF EXISTS bike_racks;

CREATE TABLE bike_racks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    osm_id BIGINT UNIQUE,
    name VARCHAR(255),
    location GEOGRAPHY(Point, 4326),
    capacity INT DEFAULT 5
);

CREATE TABLE theft_incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rack_id UUID REFERENCES bike_racks(id) ON DELETE CASCADE,
    reported_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    severity VARCHAR(50) -- 'suspicious', 'part_stolen', 'bike_stolen'
);

-- Create a spatial index for fast geographical scanning
CREATE INDEX IF NOT EXISTS idx_bike_racks_location ON bike_racks USING GIST(location);
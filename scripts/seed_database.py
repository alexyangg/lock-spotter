import requests
import psycopg2
import redis
import random
import sys
import os
from datetime import datetime, timedelta
from dotenv import load_dotenv

load_dotenv()

POSTGRES_DSN = os.getenv("POSTGRES_DSN", "dbname=lockspotter_db user=lockspotter_admin password=secure_password_123 host=localhost port=5432")
REDIS_HOST = os.getenv("REDIS_HOST", "localhost")
REDIS_PORT = int(os.getenv("REDIS_PORT", 6379))

# Bounding Box coordinates for Palo Alto, CA
BBOX = "37.41,-122.17,37.46,-122.12"

def fetch_osm_bike_racks(bbox):
    print(f"[*] Querying OpenStreetMap Overpass API for bounding box: {bbox}...")
    url = "http://overpass-api.de/api/interpreter"
    query = f"""
    [out:json];
    node["amenity"="bicycle_parking"]({bbox});
    out body;
    """
    
    headers = {
        'User-Agent': 'LockSpotterApp/1.0 (local development project)',
        'Accept': 'application/json'
    }
    
    try:
        response = requests.get(url, params={'data': query}, headers=headers, timeout=30)
        response.raise_for_status()
        data = response.json()
        return data.get('elements', [])
    except Exception as e:
        print(f"[!] Error fetching data from OpenStreetMap: {e}")
        sys.exit(1)

def seed_storage_layers():
    elements = fetch_osm_bike_racks(BBOX)
    print(f"[+] Successfully retrieved {len(elements)} bike racks from OpenStreetMap.")

    if not elements:
        print("[!] No infrastructure elements found. Aborting seed.")
        return

    # Connect to databases
    try:
        pg_conn = psycopg2.connect(POSTGRES_DSN)
        pg_cursor = pg_conn.cursor()
        r_client = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    except Exception as e:
        print(f"[!] Database connection failure: {e}")
        sys.exit(1)

    print("[*] Flushing old Redis spatial keys...")
    r_client.delete("lockspotter_racks")

    print("[*] Ingesting infrastructure data into storage layers...")
    
    inserted_racks = []

    for node in elements:
        osm_id = node.get('id')
        lat = node.get('lat')
        lon = node.get('lon')
        tags = node.get('tags', {})
        name = tags.get('name', f"Rack {osm_id}")
        capacity = tags.get('capacity', random.randint(4, 10))

        try:
            # 1. Insert into PostgreSQL using PostGIS geography primitives
            pg_cursor.execute("""
                INSERT INTO bike_racks (osm_id, name, location, capacity)
                VALUES (%s, %s, ST_SetSRID(ST_MakePoint(%s, %s), 4326), %s)
                ON CONFLICT (osm_id) DO UPDATE 
                SET name = EXCLUDED.name, capacity = EXCLUDED.capacity
                RETURNING id;
            """, (osm_id, name, lon, lat, capacity))
            
            generated_uuid = pg_cursor.fetchone()[0]
            inserted_racks.append({'uuid': generated_uuid, 'lat': lat, 'lon': lon})

            # 2. Insert into Redis Geospatial Index (Longitude must precede Latitude)
            r_client.geoadd("lockspotter_racks", (lon, lat, str(generated_uuid)))

        except Exception as e:
            print(f"[!] Skipping node {osm_id} due to insertion error: {e}")
            pg_conn.rollback()
            continue

    pg_conn.commit()
    print(f"[+] Infrastructure seeding complete. Cached {len(inserted_racks)} nodes in Redis.")

    # 3. Generate Simulated Historical Theft Incidents
    print("[*] Generating historical crime baseline telemetry...")
    incident_severities = ['suspicious_activity', 'part_stolen', 'bike_stolen']
    total_incidents = 0

    for rack in inserted_racks:
        # Simulate higher incident clusters at random intervals (hotspots like transit zones)
        # TODO: Swap out for real municipal data
        if random.random() > 0.6:
            num_thefts = random.randint(1, 5)
            for _ in range(num_thefts):
                severity = random.choice(incident_severities)
                days_ago = random.randint(1, 90)
                timestamp = datetime.now() - timedelta(days=days_ago)

                pg_cursor.execute("""
                    INSERT INTO theft_incidents (rack_id, reported_at, severity)
                    VALUES (%s, %s, %s);
                """, (rack['uuid'], timestamp, severity))
                total_incidents += 1

    pg_conn.commit()
    print(f"[+] Ingestion complete. Seeded {total_incidents} structural theft records into PostgreSQL.")

    # Close handles
    pg_cursor.close()
    pg_conn.close()

if __name__ == "__main__":
    seed_storage_layers()
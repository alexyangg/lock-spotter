package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"lockspotter-backend/database"
	"lockspotter-backend/models"

	"github.com/redis/go-redis/v9"
)

// RackHandler structural wrapper exposes database pools cleanly to our HTTP handlers
type RackHandler struct {
	Storage *database.StorageClient
}

// NewRackHandler initializes the structural dependency layout
func NewRackHandler(storage *database.StorageClient) *RackHandler {
	return &RackHandler{Storage: storage}
}

// HandleGetNearbyRacks handles GET /api/racks/nearby?lat=X&lng=Y&radius=1000
func (h *RackHandler) HandleGetNearbyRacks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Parse URL parameters manually to avoid framework dependencies
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	radStr := r.URL.Query().Get("radius")

	lat, errLat := strconv.ParseFloat(latStr, 64)
	lng, errLng := strconv.ParseFloat(lngStr, 64)
	if errLat != nil || errLng != nil {
		http.Error(w, `{"error": "Missing or invalid 'lat' and 'lng' parameters"}`, http.StatusBadRequest)
		return
	}

	radius := 1000.0 // Default radius parameter set to 1 kilometer
	if radStr != "" {
		if parsedRad, err := strconv.ParseFloat(radStr, 64); err == nil {
			radius = parsedRad
		}
	}

	// 2. Query Redis In-Memory Geospatial Index via GEORADIUS primitive
	// Redis expects parameters ordered: Key, Longitude, Latitude, Query Radius, Unit
	redisQuery := h.Storage.Redis.GeoRadius(ctx, "lockspotter_racks", lng, lat, &redis.GeoRadiusQuery{
		Radius: radius,
		Unit:   "m",
	})

	locations, err := redisQuery.Result()
	if err != nil {
		http.Error(w, `{"error": "Failed to scan in-memory geospatial index"}`, http.StatusInternalServerError)
		return
	}

	// If no structural IDs fall within bounds, exit early to protect the relational layer
	if len(locations) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	// Extract target primary key strings from Redis response structures
	var rackUUIDs []string
	for _, loc := range locations {
		rackUUIDs = append(rackUUIDs, loc.Name)
	}

	// 3. Resolve metadata and calculate live risk aggregates from PostgreSQL
	// PostGIS ST_X and ST_Y function macros extract native floats from geography types
	query := `
		SELECT 
			r.id, r.osm_id, r.name, ST_Y(r.location::geometry) as lat, ST_X(r.location::geometry) as lng, r.capacity,
			COUNT(t.id) as theft_count,
			COALESCE(SUM(CASE 
				WHEN t.severity = 'bike_stolen' THEN 3.0
				WHEN t.severity = 'part_stolen' THEN 1.5
				WHEN t.severity = 'suspicious_activity' THEN 0.5
				ELSE 0.0
			END), 0.0) as risk_score
		FROM bike_racks r
		LEFT JOIN theft_incidents t ON r.id = t.rack_id
		WHERE r.id = ANY($1)
		GROUP BY r.id;
	`

	rows, err := h.Storage.DB.Query(ctx, query, rackUUIDs)
	if err != nil {
		http.Error(w, `{"error": "Relational resolution sequence failed"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var responsePayload []models.RackDTO
	for rows.Next() {
		var rack models.RackDTO
		err := rows.Scan(
			&rack.ID, &rack.OSMID, &rack.Name, &rack.Latitude, &rack.Longitude, 
			&rack.Capacity, &rack.TheftCount, &rack.RiskScore,
		)
		if err != nil {
			continue
		}
		responsePayload = append(responsePayload, rack)
	}

	// 4. Stream structured JSON back down to client
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Handle local CORS configuration
	
	// Fast custom JSON marshaller fallback
	// In production, use "encoding/json" or "github.com/bytedance/sonic"
	writeJSONResponse(w, responsePayload)
}

func writeJSONResponse(w http.ResponseWriter, payload []models.RackDTO) {
	// Simple manual JSON marshalling array fallback loop
	w.Write([]byte("["))
	for i, r := range payload {
		item := `{"id":"` + r.ID + `","osm_id":` + strconv.FormatInt(r.OSMID, 10) + 
			`,"name":"` + r.Name + `","latitude":` + strconv.FormatFloat(r.Latitude, 'f', 6, 64) + 
			`,"longitude":` + strconv.FormatFloat(r.Longitude, 'f', 6, 64) + 
			`,"capacity":` + strconv.Itoa(r.Capacity) + `,"theft_count":` + strconv.Itoa(r.TheftCount) + 
			`,"risk_score":` + strconv.FormatFloat(r.RiskScore, 'f', 2, 64) + `}`
		w.Write([]byte(item))
		if i < len(payload)-1 {
			w.Write([]byte(","))
		}
	}
	w.Write([]byte("]"))
}
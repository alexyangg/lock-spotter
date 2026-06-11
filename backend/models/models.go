package models

// RackDTO represents the client-facing payload for a bicycle parking facility
type RackDTO struct {
	ID        string    `json:"id"`
	OSMID     int64     `json:"osm_id"`
	Name      string    `json:"name"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Capacity  int       `json:"capacity"`
	RiskScore float64   `json:"risk_score"`
	TheftCount int      `json:"theft_count"`
}

// NearbyRequest binds incoming spatial radius filter queries
type NearbyRequest struct {
	Latitude  float64 `form:"lat" binding:"required"`
	Longitude float64 `form:"lng" binding:"required"`
	RadiusM   float64 `form:"radius"` // Defaulted if missing
}

// IncidentReportRequest binds incoming user crowdsourced data payloads
type IncidentReportRequest struct {
	RackID   string `json:"rack_id" binding:"required"`
	Severity string `json:"severity" binding:"required"` // 'suspicious', 'part_stolen', 'bike_stolen'
}
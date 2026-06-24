package handlers

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed data/world.json
var worldDataFS embed.FS

// worldEntry represents a single cluster entry in world.json.
type worldEntry struct {
	Index string `json:"Index"`
	Name  string `json:"Name"`
	Tier  string `json:"Tier"`
	Area  string `json:"Area"`
}

// clusterOverrides takes precedence over world.json for entries where we want
// a cleaner display name (e.g. "Caerleon" instead of "Caerleon Market").
var clusterOverrides = map[string]string{
	"3005": "Caerleon",
	"3013": "Caerleon",
}

// zoneType classifies a zone by its PvP risk level, derived from the area
// code and tier using heuristics that match the Albion Online map layout.
func zoneType(area, tier string) string {
	switch area {
	case "CTY":
		return "" // City — safe, no label needed
	case "OUT":
		return "Black" // Outlands — full-loot black zones
	case "TNL":
		return "Roads" // Avalonian Roads
	case "ROY", "":
		// Royal continent (or unknown area): tier determines PvP risk level.
		// T1 = Blue (safe), T2-T4 = Yellow (knockdown), T5+ = Red (full-loot).
		switch tier {
		case "1":
			return "Blue"
		case "2", "3", "4":
			return "Yellow"
		case "5", "6", "7", "8":
			return "Red"
		}
	}
	return ""
}

// clusterData holds the parsed world data, keyed by cluster Index.
var clusterData map[string]worldEntry

func init() {
	raw, err := worldDataFS.ReadFile("data/world.json")
	if err != nil {
		panic("failed to read embedded world.json: " + err.Error())
	}

	var entries []worldEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		panic("failed to parse world.json: " + err.Error())
	}

	clusterData = make(map[string]worldEntry, len(entries))
	for _, e := range entries {
		if e.Index == "" {
			continue
		}
		if override, ok := clusterOverrides[e.Index]; ok {
			e.Name = override
		}
		clusterData[e.Index] = e
	}
}

// ClusterName returns the human-readable name for a cluster index, or the raw
// index if no mapping is known.
func ClusterName(index string) string {
	if e, ok := clusterData[index]; ok && e.Name != "" {
		return e.Name
	}
	return index
}

// ClusterDisplay returns a formatted string with the map name, tier, and area
// type for display in the UI. For cities only the name is shown; for open-world
// zones the tier and area type are appended (e.g. "Battlebrae Grassland · T7 · Black").
func ClusterDisplay(index string) string {
	e, ok := clusterData[index]
	if !ok || e.Name == "" {
		return index
	}

	parts := []string{e.Name}
	if e.Tier != "" && e.Tier != "1" && e.Area != "CTY" {
		parts = append(parts, "T"+e.Tier)
	}
	if zt := zoneType(e.Area, e.Tier); zt != "" {
		parts = append(parts, zt)
	}

	if len(parts) == 1 {
		return parts[0]
	}
	// Join with middle dot separator
	result := parts[0]
	for _, p := range parts[1:] {
		result = fmt.Sprintf("%s · %s", result, p)
	}
	return result
}

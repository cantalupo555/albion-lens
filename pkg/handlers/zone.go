package handlers

import "strings"

// MapType identifies the kind of map/cluster the local player is currently in,
// derived by substring matching on the cluster string (param[0] of the
// ChangeCluster response). The values and matching order mirror the reference
// implementation (StatisticsAnalysisTool WorldData.GetMapType).
type MapType int

const (
	MapTypeUnknown          MapType = 0
	MapTypeRandomDungeon    MapType = 1
	MapTypeHellGate         MapType = 2
	MapTypeCorruptedDungeon MapType = 3
	MapTypeIsland           MapType = 4
	MapTypeHideout          MapType = 5
	MapTypeExpedition       MapType = 6
	MapTypeArena            MapType = 7
	MapTypeMists            MapType = 8
	MapTypeMistsDungeon     MapType = 9
	MapTypeAbyssalDepths    MapType = 10
)

// String returns a human-readable name for the map type.
func (m MapType) String() string {
	switch m {
	case MapTypeRandomDungeon:
		return "Random Dungeon"
	case MapTypeHellGate:
		return "Hell Gate"
	case MapTypeCorruptedDungeon:
		return "Corrupted Dungeon"
	case MapTypeIsland:
		return "Island"
	case MapTypeHideout:
		return "Hideout"
	case MapTypeExpedition:
		return "Expedition"
	case MapTypeArena:
		return "Arena"
	case MapTypeMists:
		return "Mists"
	case MapTypeMistsDungeon:
		return "Mists Dungeon"
	case MapTypeAbyssalDepths:
		return "Abyssal Depths"
	default:
		return "Unknown"
	}
}

// ParseMapType derives the MapType from a cluster string by substring matching.
// The matching order is significant: longer/more-specific keys must be checked
// before shorter ones (e.g. MISTSDUNGEON before MISTS). Mirrors
// WorldData.GetMapType from the reference project.
func ParseMapType(clusterString string) MapType {
	upper := strings.ToUpper(clusterString)
	switch {
	case strings.Contains(upper, "HELLCLUSTER"):
		return MapTypeHellGate
	case strings.Contains(upper, "RANDOMDUNGEON"):
		return MapTypeRandomDungeon
	case strings.Contains(upper, "CORRUPTEDDUNGEON"):
		return MapTypeCorruptedDungeon
	case strings.Contains(upper, "ISLAND"):
		return MapTypeIsland
	case strings.Contains(upper, "HIDEOUT"):
		return MapTypeHideout
	case strings.Contains(upper, "EXPEDITION"):
		return MapTypeExpedition
	case strings.Contains(upper, "ARENA"):
		return MapTypeArena
	case strings.Contains(upper, "MISTSDUNGEON"):
		return MapTypeMistsDungeon
	case strings.Contains(upper, "MISTS"):
		return MapTypeMists
	case strings.Contains(upper, "HELLDUNGEON"):
		return MapTypeAbyssalDepths
	default:
		return MapTypeUnknown
	}
}

// ZoneInfo holds the parsed identity of the player's current map/cluster,
// extracted from the ChangeCluster operation response (Op #41).
type ZoneInfo struct {
	// MapType is the derived category of the current map.
	MapType MapType
	// ClusterIndex is the raw cluster string from param[0] (e.g. "4000" for
	// open-world cities, "@ISLAND@<guid>" for islands). Used as the display
	// label for open-world/city maps where MapType is Unknown.
	ClusterIndex string
	// IslandName is param[2]; non-empty only on personal/guild islands.
	IslandName string
	// MainClusterIndex is the parent cluster of a hideout (parsed from the
	// hideout cluster string). Empty for non-hideout maps.
	MainClusterIndex string
	// HasDungeonInfo reports whether param[3] (DungeonInformation byte[]) was
	// present. The payload itself is not deeply parsed.
	HasDungeonInfo bool
}

// DisplayString returns a human-friendly label for the current zone. For typed
// maps it returns the MapType name; for islands the island name is appended.
// For open-world/city maps (MapTypeUnknown) it resolves the cluster index to
// a known name (e.g. "4000" → "Fort Sterling"), falling back to the raw index.
// Returns an empty string only when the zone has never been set (zero-value
// ZoneInfo).
func (z ZoneInfo) DisplayString() string {
	if z.MapType == MapTypeUnknown {
		if z.ClusterIndex == "" {
			return ""
		}
		return ClusterDisplay(z.ClusterIndex)
	}
	label := z.MapType.String()
	if z.IslandName != "" {
		label += " — " + z.IslandName
	}
	return label
}

// parseChangeClusterResponse extracts zone identity from the ChangeCluster
// operation response parameters, mirroring ChangeClusterResponse.cs from the
// reference project. Parameter mapping:
//
//	param[0] = cluster string (e.g. "@ISLAND@<guid>", "4000")
//	param[1] = WorldMapDataType (string, not displayed)
//	param[2] = IslandName (string)
//	param[3] = DungeonInformation (byte[], presence tracked only)
//
// A zero-value ZoneInfo is returned on malformed input without panicking.
func parseChangeClusterResponse(params map[byte]interface{}) ZoneInfo {
	info := ZoneInfo{}

	clusterString := getString(params, 0)
	info.ClusterIndex = clusterString

	if clusterString != "" && strings.Contains(strings.ToLower(clusterString), "@") {
		// Split on "@" and filter empty parts to match RemoveEmptyEntries semantics.
		rawParts := strings.Split(clusterString, "@")
		parts := make([]string, 0, len(rawParts))
		for _, p := range rawParts {
			if p != "" {
				parts = append(parts, p)
			}
		}

		if len(parts) > 1 {
			info.MapType = ParseMapType(clusterString)

			if info.MapType == MapTypeHideout && len(parts) >= 3 {
				info.MainClusterIndex = parts[1]
			}
		} else {
			info.MapType = MapTypeUnknown
		}
	}

	info.IslandName = getString(params, 2)

	if _, ok := params[3]; ok {
		info.HasDungeonInfo = true
	}

	return info
}

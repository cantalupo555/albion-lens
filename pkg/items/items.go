// Package items provides item ID to name translation using ao-bin-dumps data
package items

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// ItemDatabase holds the loaded items data
type ItemDatabase struct {
	items     map[string]ItemInfo // key: uniquename (e.g., "T4_BAG")
	itemsByID map[int]ItemInfo    // key: numeric index (if available)
	mu        sync.RWMutex
	loaded    bool
}

// ItemInfo contains item information
type ItemInfo struct {
	UniqueName  string `json:"@uniquename"`
	Index       int    // Numeric index based on position
	Tier        int    // Parsed tier (1-8)
	Enchantment int    // Enchantment level (0-4)
	Category    string // Shop category
	SubCategory string // Shop subcategory
}

// Global database instance
var db *ItemDatabase
var once sync.Once

// GetDatabase returns the global item database
func GetDatabase() *ItemDatabase {
	once.Do(func() {
		db = &ItemDatabase{
			items:     make(map[string]ItemInfo),
			itemsByID: make(map[int]ItemInfo),
		}
	})
	return db
}

// LoadFromFile loads items from an items.json file
func (d *ItemDatabase) LoadFromFile(filePath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read items file: %w", err)
	}

	return d.parseItemsJSON(data)
}

// LoadFromPath tries to find and load items.json from common paths
func (d *ItemDatabase) LoadFromPath(basePath string) error {
	// Try common locations
	paths := []string{
		filepath.Join(basePath, "items.json"),
		filepath.Join(basePath, "ao-bin-dumps", "items.json"),
		filepath.Join(basePath, "..", "ao-bin-dumps", "items.json"),
		"items.json",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return d.LoadFromFile(path)
		}
	}

	return fmt.Errorf("items.json not found in any of the expected locations")
}

// parseItemsJSON parses the complex items.json structure
func (d *ItemDatabase) parseItemsJSON(data []byte) error {
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	items, ok := root["items"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid items.json structure: missing 'items' key")
	}

	// Process different item categories
	itemIndex := 0
	categoryProcessors := []string{
		"hideoutitem",
		"farmableitem",
		"simpleitem",
		"consumableitem",
		"consumablefrominventoryitem",
		"equipmentitem",
		"weapon",
		"mount",
		"furnitureitem",
		"mountskin",
		"journalitem",
		"labourercontract",
		"crystalleagueitem",
		"killtrophy",
		"trackingitem",
	}

	for _, category := range categoryProcessors {
		if categoryData, exists := items[category]; exists {
			itemIndex = d.processCategory(categoryData, category, itemIndex)
		}
	}

	d.loaded = true
	return nil
}

// processCategory processes items from a specific category
func (d *ItemDatabase) processCategory(data interface{}, category string, startIndex int) int {
	index := startIndex

	switch items := data.(type) {
	case []interface{}:
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if info := d.extractItemInfo(itemMap, category, index); info != nil {
					d.items[info.UniqueName] = *info
					d.itemsByID[index] = *info
					index++
				}
			}
		}
	case map[string]interface{}:
		if info := d.extractItemInfo(items, category, index); info != nil {
			d.items[info.UniqueName] = *info
			d.itemsByID[index] = *info
			index++
		}
	}

	return index
}

// extractItemInfo extracts item information from a JSON object
func (d *ItemDatabase) extractItemInfo(itemMap map[string]interface{}, category string, index int) *ItemInfo {
	uniqueName, ok := itemMap["@uniquename"].(string)
	if !ok || uniqueName == "" {
		return nil
	}

	info := &ItemInfo{
		UniqueName: uniqueName,
		Index:      index,
		Category:   category,
	}

	// Parse tier from name (e.g., "T4_BAG" -> tier 4)
	info.Tier, info.Enchantment = parseTierAndEnchantment(uniqueName)

	// Get subcategory if available
	if subcat, ok := itemMap["@shopcategory"].(string); ok {
		info.SubCategory = subcat
	}

	return info
}

// parseTierAndEnchantment extracts tier and enchantment from item name
func parseTierAndEnchantment(name string) (tier int, enchantment int) {
	// Format examples: T4_BAG, T4_BAG@1, T8_LEATHER@3
	parts := strings.Split(name, "@")
	baseName := parts[0]

	if len(parts) > 1 {
		if e, err := strconv.Atoi(parts[1]); err == nil {
			enchantment = e
		}
	}

	if len(baseName) >= 2 && baseName[0] == 'T' {
		if t, err := strconv.Atoi(string(baseName[1])); err == nil {
			tier = t
		}
	}

	return
}

// GetByUniqueName returns item info by unique name
func (d *ItemDatabase) GetByUniqueName(name string) (ItemInfo, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	info, ok := d.items[name]
	return info, ok
}

// GetByID returns item info by numeric ID
func (d *ItemDatabase) GetByID(id int) (ItemInfo, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	info, ok := d.itemsByID[id]
	return info, ok
}

// GetItemName returns a human-readable name for an item
// It accepts either a numeric ID or a string unique name
func (d *ItemDatabase) GetItemName(itemID interface{}) string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	switch id := itemID.(type) {
	case int:
		if info, ok := d.itemsByID[id]; ok {
			return formatItemName(info.UniqueName)
		}
		return fmt.Sprintf("Item#%d", id)
	case int32:
		if info, ok := d.itemsByID[int(id)]; ok {
			return formatItemName(info.UniqueName)
		}
		return fmt.Sprintf("Item#%d", id)
	case int64:
		if info, ok := d.itemsByID[int(id)]; ok {
			return formatItemName(info.UniqueName)
		}
		return fmt.Sprintf("Item#%d", id)
	case string:
		return formatItemName(id)
	default:
		return fmt.Sprintf("Item<%v>", itemID)
	}
}

// formatItemName converts internal name to readable format
// T4_BAG -> "T4 Bag"
// T8_LEATHER@3 -> "T8.3 Leather"
func formatItemName(name string) string {
	if name == "" {
		return "Unknown"
	}

	// Handle enchantment suffix
	parts := strings.Split(name, "@")
	baseName := parts[0]
	enchant := ""
	if len(parts) > 1 {
		enchant = "." + parts[1]
	}

	// Split by underscore and format
	nameParts := strings.Split(baseName, "_")
	if len(nameParts) == 0 {
		return name
	}

	// First part is typically tier (T4, T5, etc.)
	if len(nameParts[0]) >= 2 && nameParts[0][0] == 'T' {
		tier := nameParts[0] + enchant
		rest := strings.Join(nameParts[1:], " ")
		rest = strings.Title(strings.ToLower(rest))
		return fmt.Sprintf("%s %s", tier, rest)
	}

	return strings.Title(strings.ToLower(strings.ReplaceAll(name, "_", " ")))
}

// IsLoaded returns whether the database has been loaded
func (d *ItemDatabase) IsLoaded() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.loaded
}

// ItemCount returns the number of loaded items
func (d *ItemDatabase) ItemCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.items)
}

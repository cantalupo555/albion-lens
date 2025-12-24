package items

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// resetDatabase resets the global database for testing
func resetDatabase() {
	db = nil
	once = sync.Once{}
}

// TestGetDatabase tests singleton database creation
func TestGetDatabase(t *testing.T) {
	resetDatabase()

	db1 := GetDatabase()
	if db1 == nil {
		t.Fatal("GetDatabase returned nil")
	}

	db2 := GetDatabase()
	if db1 != db2 {
		t.Error("GetDatabase should return the same instance")
	}

	if db1.items == nil {
		t.Error("items map not initialized")
	}

	if db1.itemsByID == nil {
		t.Error("itemsByID map not initialized")
	}
}

// TestIsLoaded tests the loaded state
func TestIsLoaded(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	if db.IsLoaded() {
		t.Error("database should not be loaded initially")
	}
}

// TestItemCount tests item counting
func TestItemCount(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	if db.ItemCount() != 0 {
		t.Error("initial item count should be 0")
	}
}

// TestParseTierAndEnchantment tests tier and enchantment parsing
func TestParseTierAndEnchantment(t *testing.T) {
	testCases := []struct {
		name              string
		expectedTier      int
		expectedEnchant   int
	}{
		{"T4_BAG", 4, 0},
		{"T8_LEATHER", 8, 0},
		{"T4_BAG@1", 4, 1},
		{"T8_LEATHER@3", 8, 3},
		{"T6_SWORD@2", 6, 2},
		{"T1_WOOD", 1, 0},
		{"SOME_ITEM", 0, 0},        // No tier prefix
		{"T4", 4, 0},               // Just tier
		{"@2", 0, 2},               // Just enchantment
		{"", 0, 0},                 // Empty string
		{"NOTIER_ITEM@1", 0, 1},    // No tier but has enchantment
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tier, enchant := parseTierAndEnchantment(tc.name)
			if tier != tc.expectedTier {
				t.Errorf("tier: expected %d, got %d", tc.expectedTier, tier)
			}
			if enchant != tc.expectedEnchant {
				t.Errorf("enchantment: expected %d, got %d", tc.expectedEnchant, enchant)
			}
		})
	}
}

// TestFormatItemName tests item name formatting
func TestFormatItemName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"T4_BAG", "T4 Bag"},
		{"T8_LEATHER", "T8 Leather"},
		{"T4_BAG@1", "T4.1 Bag"},
		{"T8_LEATHER@3", "T8.3 Leather"},
		{"T6_LONG_SWORD", "T6 Long Sword"},
		{"SOME_RANDOM_ITEM", "Some Random Item"},
		{"", "Unknown"},
		{"T4", "T4 "},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := formatItemName(tc.input)
			if result != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestGetByUniqueName tests retrieval by unique name
func TestGetByUniqueName(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	// Add test item directly
	db.mu.Lock()
	db.items["T4_TEST_ITEM"] = ItemInfo{
		UniqueName: "T4_TEST_ITEM",
		Index:      1,
		Tier:       4,
	}
	db.mu.Unlock()

	// Test retrieval
	info, ok := db.GetByUniqueName("T4_TEST_ITEM")
	if !ok {
		t.Fatal("item not found")
	}

	if info.UniqueName != "T4_TEST_ITEM" {
		t.Errorf("expected 'T4_TEST_ITEM', got '%s'", info.UniqueName)
	}

	if info.Tier != 4 {
		t.Errorf("expected tier 4, got %d", info.Tier)
	}

	// Test missing item
	_, ok = db.GetByUniqueName("NONEXISTENT")
	if ok {
		t.Error("should return false for missing item")
	}
}

// TestGetByID tests retrieval by numeric ID
func TestGetByID(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	// Add test item directly
	db.mu.Lock()
	db.itemsByID[42] = ItemInfo{
		UniqueName: "T5_SWORD",
		Index:      42,
		Tier:       5,
	}
	db.mu.Unlock()

	// Test retrieval
	info, ok := db.GetByID(42)
	if !ok {
		t.Fatal("item not found")
	}

	if info.UniqueName != "T5_SWORD" {
		t.Errorf("expected 'T5_SWORD', got '%s'", info.UniqueName)
	}

	// Test missing item
	_, ok = db.GetByID(9999)
	if ok {
		t.Error("should return false for missing item")
	}
}

// TestGetItemName tests the GetItemName method with various input types
func TestGetItemName(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	// Add test items
	db.mu.Lock()
	db.itemsByID[100] = ItemInfo{UniqueName: "T4_BAG", Index: 100}
	db.itemsByID[200] = ItemInfo{UniqueName: "T8_LEATHER@3", Index: 200}
	db.mu.Unlock()

	// Test int
	name := db.GetItemName(100)
	if name != "T4 Bag" {
		t.Errorf("int: expected 'T4 Bag', got '%s'", name)
	}

	// Test int32
	name = db.GetItemName(int32(100))
	if name != "T4 Bag" {
		t.Errorf("int32: expected 'T4 Bag', got '%s'", name)
	}

	// Test int64
	name = db.GetItemName(int64(200))
	if name != "T8.3 Leather" {
		t.Errorf("int64: expected 'T8.3 Leather', got '%s'", name)
	}

	// Test string
	name = db.GetItemName("T6_SWORD")
	if name != "T6 Sword" {
		t.Errorf("string: expected 'T6 Sword', got '%s'", name)
	}

	// Test missing ID
	name = db.GetItemName(9999)
	if name != "Item#9999" {
		t.Errorf("missing: expected 'Item#9999', got '%s'", name)
	}

	// Test unknown type
	name = db.GetItemName(3.14)
	if name != "Item<3.14>" {
		t.Errorf("unknown type: expected 'Item<3.14>', got '%s'", name)
	}
}

// TestLoadFromFile tests loading from a JSON file
func TestLoadFromFile(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	// Create temp JSON file
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "items.json")

	jsonContent := `{
		"items": {
			"simpleitem": [
				{"@uniquename": "T4_BAG"},
				{"@uniquename": "T5_CAPE"}
			],
			"equipmentitem": [
				{"@uniquename": "T6_ARMOR", "@shopcategory": "armor"}
			]
		}
	}`

	err := os.WriteFile(jsonPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Load the file
	err = db.LoadFromFile(jsonPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if !db.IsLoaded() {
		t.Error("database should be loaded after LoadFromFile")
	}

	if db.ItemCount() != 3 {
		t.Errorf("expected 3 items, got %d", db.ItemCount())
	}

	// Verify items
	info, ok := db.GetByUniqueName("T4_BAG")
	if !ok {
		t.Error("T4_BAG not found")
	}
	if info.Tier != 4 {
		t.Errorf("T4_BAG tier: expected 4, got %d", info.Tier)
	}

	info, ok = db.GetByUniqueName("T6_ARMOR")
	if !ok {
		t.Error("T6_ARMOR not found")
	}
	if info.SubCategory != "armor" {
		t.Errorf("T6_ARMOR subcategory: expected 'armor', got '%s'", info.SubCategory)
	}
}

// TestLoadFromFileNotFound tests loading from nonexistent file
func TestLoadFromFileNotFound(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	err := db.LoadFromFile("/nonexistent/path/items.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// TestLoadFromFileInvalidJSON tests loading invalid JSON
func TestLoadFromFileInvalidJSON(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "items.json")

	err := os.WriteFile(jsonPath, []byte("not valid json"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = db.LoadFromFile(jsonPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestLoadFromFileMissingItemsKey tests loading JSON without items key
func TestLoadFromFileMissingItemsKey(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "items.json")

	err := os.WriteFile(jsonPath, []byte(`{"other": "data"}`), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = db.LoadFromFile(jsonPath)
	if err == nil {
		t.Error("expected error for missing 'items' key")
	}
}

// TestLoadFromPath tests path auto-detection
func TestLoadFromPath(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	// Create temp directory structure
	tmpDir := t.TempDir()

	jsonContent := `{"items": {"simpleitem": [{"@uniquename": "T1_TEST"}]}}`

	// Test direct path
	jsonPath := filepath.Join(tmpDir, "items.json")
	err := os.WriteFile(jsonPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = db.LoadFromPath(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromPath failed: %v", err)
	}

	if db.ItemCount() != 1 {
		t.Errorf("expected 1 item, got %d", db.ItemCount())
	}
}

// TestLoadFromPathNotFound tests path not found
func TestLoadFromPathNotFound(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	err := db.LoadFromPath("/nonexistent/base/path")
	if err == nil {
		t.Error("expected error for nonexistent paths")
	}
}

// TestProcessCategorySingleItem tests processing a single item (not array)
func TestProcessCategorySingleItem(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "items.json")

	// Single item instead of array
	jsonContent := `{
		"items": {
			"simpleitem": {"@uniquename": "T4_SINGLE_ITEM"}
		}
	}`

	err := os.WriteFile(jsonPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = db.LoadFromFile(jsonPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if db.ItemCount() != 1 {
		t.Errorf("expected 1 item, got %d", db.ItemCount())
	}

	_, ok := db.GetByUniqueName("T4_SINGLE_ITEM")
	if !ok {
		t.Error("T4_SINGLE_ITEM not found")
	}
}

// TestExtractItemInfoMissingName tests item extraction with missing name
func TestExtractItemInfoMissingName(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "items.json")

	// Item without @uniquename
	jsonContent := `{
		"items": {
			"simpleitem": [
				{"other": "data"},
				{"@uniquename": "T4_VALID"}
			]
		}
	}`

	err := os.WriteFile(jsonPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = db.LoadFromFile(jsonPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Should only have the valid item
	if db.ItemCount() != 1 {
		t.Errorf("expected 1 item (invalid should be skipped), got %d", db.ItemCount())
	}
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	// Add some items
	db.mu.Lock()
	for i := 0; i < 100; i++ {
		db.itemsByID[i] = ItemInfo{UniqueName: "T4_ITEM", Index: i}
	}
	db.mu.Unlock()

	// Concurrent reads
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			db.GetByID(id)
			db.IsLoaded()
			db.ItemCount()
		}(i)
	}

	wg.Wait()
}

// TestItemInfoStructure tests ItemInfo struct fields
func TestItemInfoStructure(t *testing.T) {
	info := ItemInfo{
		UniqueName:  "T4_BAG@2",
		Index:       42,
		Tier:        4,
		Enchantment: 2,
		Category:    "simpleitem",
		SubCategory: "bag",
	}

	if info.UniqueName != "T4_BAG@2" {
		t.Error("UniqueName field incorrect")
	}
	if info.Index != 42 {
		t.Error("Index field incorrect")
	}
	if info.Tier != 4 {
		t.Error("Tier field incorrect")
	}
	if info.Enchantment != 2 {
		t.Error("Enchantment field incorrect")
	}
	if info.Category != "simpleitem" {
		t.Error("Category field incorrect")
	}
	if info.SubCategory != "bag" {
		t.Error("SubCategory field incorrect")
	}
}

// TestAllCategoryProcessors tests that all category processors work
func TestAllCategoryProcessors(t *testing.T) {
	resetDatabase()
	db := GetDatabase()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "items.json")

	// JSON with multiple categories
	jsonContent := `{
		"items": {
			"hideoutitem": [{"@uniquename": "T1_HIDEOUT"}],
			"farmableitem": [{"@uniquename": "T2_FARM"}],
			"simpleitem": [{"@uniquename": "T3_SIMPLE"}],
			"consumableitem": [{"@uniquename": "T4_CONSUME"}],
			"consumablefrominventoryitem": [{"@uniquename": "T5_INV_CONSUME"}],
			"equipmentitem": [{"@uniquename": "T6_EQUIP"}],
			"weapon": [{"@uniquename": "T7_WEAPON"}],
			"mount": [{"@uniquename": "T8_MOUNT"}],
			"furnitureitem": [{"@uniquename": "T4_FURNITURE"}],
			"mountskin": [{"@uniquename": "T4_MOUNTSKIN"}],
			"journalitem": [{"@uniquename": "T4_JOURNAL"}],
			"labourercontract": [{"@uniquename": "T4_CONTRACT"}],
			"crystalleagueitem": [{"@uniquename": "T4_CRYSTAL"}],
			"killtrophy": [{"@uniquename": "T4_TROPHY"}],
			"trackingitem": [{"@uniquename": "T4_TRACKING"}]
		}
	}`

	err := os.WriteFile(jsonPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = db.LoadFromFile(jsonPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	expectedCount := 15
	if db.ItemCount() != expectedCount {
		t.Errorf("expected %d items from all categories, got %d", expectedCount, db.ItemCount())
	}
}

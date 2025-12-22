// Package events contains event code definitions and event structures for Albion Online
package events

// Event codes from Albion Online
// Sources: AlbionOnline-StatisticsAnalysis EventCodes.cs and ao-loot-logger events-v9.0.0.json
const (
	// Core events
	EventUnused       = 0
	EventLeave        = 1
	EventJoinFinished = 2
	EventMove         = 3
	EventTeleport     = 4

	// Combat events
	EventHealthUpdate  = 6
	EventHealthUpdates = 7
	EventEnergyUpdate  = 8
	EventAttack        = 13
	EventCastStart     = 14
	EventCastHit       = 21
	EventKilledPlayer  = 170
	EventDied          = 171
	EventKnockedDown   = 172

	// Inventory events
	EventInventoryPutItem    = 26
	EventInventoryDeleteItem = 27
	EventNewCharacter        = 29
	EventNewEquipmentItem    = 30
	EventNewSiegeBannerItem  = 31
	EventNewSimpleItem       = 32
	EventNewFurnitureItem    = 33

	// Harvesting events
	EventHarvestStart    = 52
	EventHarvestCancel   = 53
	EventHarvestFinished = 54
	EventTakeSilver      = 55

	// Economy events (confirmed via discovery mode)
	EventUpdateMoney       = 80
	EventUpdateFame        = 81 // Simple fame update (only total)
	EventUpdateFameDetails = 82 // Detailed fame update (total, gained, zone)

	// Loot events
	EventNewLoot             = 98
	EventAttachItemContainer = 99
	EventDetachItemContainer = 100

	// Character stats
	EventCharacterStats = 143

	// Party events
	EventPartyInvitation   = 210
	EventPartyJoinRequest  = 211
	EventPartyJoined       = 212
	EventPartyDisbanded    = 213
	EventPartyPlayerJoined = 214
	EventPartyPlayerLeft   = 216

	// Combat state
	EventOtherGrabbedLoot = 275
	EventInCombatState    = 257
)

// Special parameter keys
const (
	ParamEventCode     = 252 // Event code is always in parameter 252
	ParamOperationCode = 253 // Operation code is always in parameter 253
)

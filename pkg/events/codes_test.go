package events

import (
	"strings"
	"testing"
)

// TestCriticalEventCodes verifies that event codes critical for detection
// have the exact numeric values expected by the current Albion Online protocol.
// These values are derived from the upstream EventCodes.cs enum positions
// (commit 2026-04-13) and MUST NOT drift. If this test fails after editing
// codes.go, the const block order is wrong and packets will be misidentified.
//
// The central fix for the "stopped detecting loot/silver" bug is
// EventOtherGrabbedLoot == 277 (was 275 before adding MatchNewCombatRound
// and MatchEndCombatRound).
func TestCriticalEventCodes(t *testing.T) {
	cases := []struct {
		name     string
		code     EventCode
		expected int
	}{
		// Events before the insertion point (unchanged by the fix)
		{"UpdateMoney", EventUpdateMoney, 81},
		{"UpdateFame", EventUpdateFame, 82},
		{"NewLoot", EventNewLoot, 98},
		{"KilledPlayer", EventKilledPlayer, 164},
		{"Died", EventDied, 165},
		{"MatchTimeLineEventEvent", EventMatchTimeLineEventEvent, 171},

		// New events inserted in this fix
		{"MatchNewCombatRound", EventMatchNewCombatRound, 172},
		{"MatchEndCombatRound", EventMatchEndCombatRound, 173},

		// Events shifted by +2 from MatchTimeLine insertion
		{"OtherGrabbedLoot (CENTRAL FIX)", EventOtherGrabbedLoot, 277},
		{"InCombatStateUpdate", EventInCombatStateUpdate, 276},

		// RedZone block reorganization effects
		{"RedZonePlayerNotification", EventRedZonePlayerNotification, 474},
		{"RedZoneEventCheatCleanup", EventRedZoneEventCheatCleanup, 475},
		{"RedZoneFortressEventChestOpened", EventRedZoneFortressEventChestOpened, 476},
		{"RedZoneWorldEvent", EventRedZoneWorldEvent, 477},

		// Last event before the appended block (shifted by +3 total)
		{"FactionFortressCutoffFightCancelledByClusterOwnerChangeEvent",
			EventFactionFortressCutoffFightCancelledByClusterOwnerChangeEvent, 664},

		// Last event overall (newly appended)
		{"LosingCarriableObjectFinished (last)", EventLosingCarriableObjectFinished, 682},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if int(tc.code) != tc.expected {
				t.Errorf("%s = %d, want %d (if this fails, the const block order is wrong)",
					tc.name, int(tc.code), tc.expected)
			}
		})
	}
}

// TestAllEventCodesGolden locks in the exact iota position AND the String()
// name for every single event code (683 entries). This is the comprehensive
// regression test: if ANY two codes are swapped, inserted, or removed in the
// wrong position in the const block, or if any map entry has a typo, this
// test will catch it. TestCriticalEventCodes above highlights key boundary
// points for documentation; this test provides full coverage.
func TestAllEventCodesGolden(t *testing.T) {
	golden := []struct {
		code EventCode
		name string
	}{
		{EventUnused, "Unused"},                                                             // 0
		{EventLeave, "Leave"},                                                               // 1
		{EventJoinFinished, "JoinFinished"},                                                 // 2
		{EventMove, "Move"},                                                                 // 3
		{EventTeleport, "Teleport"},                                                         // 4
		{EventChangeEquipment, "ChangeEquipment"},                                           // 5
		{EventHealthUpdate, "HealthUpdate"},                                                 // 6
		{EventHealthUpdates, "HealthUpdates"},                                               // 7
		{EventEnergyUpdate, "EnergyUpdate"},                                                 // 8
		{EventDamageShieldUpdate, "DamageShieldUpdate"},                                     // 9
		{EventCraftingFocusUpdate, "CraftingFocusUpdate"},                                   // 10
		{EventActiveSpellEffectsUpdate, "ActiveSpellEffectsUpdate"},                         // 11
		{EventResetCooldowns, "ResetCooldowns"},                                             // 12
		{EventAttack, "Attack"},                                                             // 13
		{EventCastStart, "CastStart"},                                                       // 14
		{EventChannelingUpdate, "ChannelingUpdate"},                                         // 15
		{EventCastCancel, "CastCancel"},                                                     // 16
		{EventCastTimeUpdate, "CastTimeUpdate"},                                             // 17
		{EventCastFinished, "CastFinished"},                                                 // 18
		{EventCastSpell, "CastSpell"},                                                       // 19
		{EventCastSpells, "CastSpells"},                                                     // 20
		{EventCastHit, "CastHit"},                                                           // 21
		{EventCastHits, "CastHits"},                                                         // 22
		{EventStoredTargetsUpdate, "StoredTargetsUpdate"},                                   // 23
		{EventChannelingEnded, "ChannelingEnded"},                                           // 24
		{EventAttackBuilding, "AttackBuilding"},                                             // 25
		{EventInventoryPutItem, "InventoryPutItem"},                                         // 26
		{EventInventoryDeleteItem, "InventoryDeleteItem"},                                   // 27
		{EventInventoryState, "InventoryState"},                                             // 28
		{EventNewCharacter, "NewCharacter"},                                                 // 29
		{EventNewEquipmentItem, "NewEquipmentItem"},                                         // 30
		{EventNewSiegeBannerItem, "NewSiegeBannerItem"},                                     // 31
		{EventNewSimpleItem, "NewSimpleItem"},                                               // 32
		{EventNewFurnitureItem, "NewFurnitureItem"},                                         // 33
		{EventNewKillTrophyItem, "NewKillTrophyItem"},                                       // 34
		{EventNewJournalItem, "NewJournalItem"},                                             // 35
		{EventNewLaborerItem, "NewLaborerItem"},                                             // 36
		{EventNewEquipmentItemLegendarySoul, "NewEquipmentItemLegendarySoul"},               // 37
		{EventNewSimpleHarvestableObject, "NewSimpleHarvestableObject"},                     // 38
		{EventNewSimpleHarvestableObjectList, "NewSimpleHarvestableObjectList"},             // 39
		{EventNewHarvestableObject, "NewHarvestableObject"},                                 // 40
		{EventNewTreasureDestinationObject, "NewTreasureDestinationObject"},                 // 41
		{EventTreasureDestinationObjectStatus, "TreasureDestinationObjectStatus"},           // 42
		{EventCloseTreasureDestinationObject, "CloseTreasureDestinationObject"},             // 43
		{EventNewSilverObject, "NewSilverObject"},                                           // 44
		{EventNewBuilding, "NewBuilding"},                                                   // 45
		{EventHarvestableChangeState, "HarvestableChangeState"},                             // 46
		{EventMobChangeState, "MobChangeState"},                                             // 47
		{EventFactionBuildingInfo, "FactionBuildingInfo"},                                   // 48
		{EventCraftBuildingInfo, "CraftBuildingInfo"},                                       // 49
		{EventRepairBuildingInfo, "RepairBuildingInfo"},                                     // 50
		{EventMeldBuildingInfo, "MeldBuildingInfo"},                                         // 51
		{EventConstructionSiteInfo, "ConstructionSiteInfo"},                                 // 52
		{EventPlayerBuildingInfo, "PlayerBuildingInfo"},                                     // 53
		{EventFarmBuildingInfo, "FarmBuildingInfo"},                                         // 54
		{EventTutorialBuildingInfo, "TutorialBuildingInfo"},                                 // 55
		{EventLaborerObjectInfo, "LaborerObjectInfo"},                                       // 56
		{EventLaborerObjectJobInfo, "LaborerObjectJobInfo"},                                 // 57
		{EventMarketPlaceBuildingInfo, "MarketPlaceBuildingInfo"},                           // 58
		{EventHarvestStart, "HarvestStart"},                                                 // 59
		{EventHarvestCancel, "HarvestCancel"},                                               // 60
		{EventHarvestFinished, "HarvestFinished"},                                           // 61
		{EventTakeSilver, "TakeSilver"},                                                     // 62
		{EventRemoveSilver, "RemoveSilver"},                                                 // 63
		{EventActionOnBuildingStart, "ActionOnBuildingStart"},                               // 64
		{EventActionOnBuildingCancel, "ActionOnBuildingCancel"},                             // 65
		{EventActionOnBuildingFinished, "ActionOnBuildingFinished"},                         // 66
		{EventItemRerollQualityFinished, "ItemRerollQualityFinished"},                       // 67
		{EventInstallResourceStart, "InstallResourceStart"},                                 // 68
		{EventInstallResourceCancel, "InstallResourceCancel"},                               // 69
		{EventInstallResourceFinished, "InstallResourceFinished"},                           // 70
		{EventCraftItemFinished, "CraftItemFinished"},                                       // 71
		{EventLogoutCancel, "LogoutCancel"},                                                 // 72
		{EventChatMessage, "ChatMessage"},                                                   // 73
		{EventChatSay, "ChatSay"},                                                           // 74
		{EventChatWhisper, "ChatWhisper"},                                                   // 75
		{EventChatMuted, "ChatMuted"},                                                       // 76
		{EventPlayEmote, "PlayEmote"},                                                       // 77
		{EventStopEmote, "StopEmote"},                                                       // 78
		{EventSystemMessage, "SystemMessage"},                                               // 79
		{EventUtilityTextMessage, "UtilityTextMessage"},                                     // 80
		{EventUpdateMoney, "UpdateMoney"},                                                   // 81
		{EventUpdateFame, "UpdateFame"},                                                     // 82
		{EventUpdateLearningPoints, "UpdateLearningPoints"},                                 // 83
		{EventUpdateReSpecPoints, "UpdateReSpecPoints"},                                     // 84
		{EventUpdateCurrency, "UpdateCurrency"},                                             // 85
		{EventUpdateFactionStanding, "UpdateFactionStanding"},                               // 86
		{EventUpdateStanding, "UpdateStanding"},                                             // 87
		{EventRespawn, "Respawn"},                                                           // 88
		{EventServerDebugLog, "ServerDebugLog"},                                             // 89
		{EventCharacterEquipmentChanged, "CharacterEquipmentChanged"},                       // 90
		{EventRegenerationHealthChanged, "RegenerationHealthChanged"},                       // 91
		{EventRegenerationEnergyChanged, "RegenerationEnergyChanged"},                       // 92
		{EventRegenerationMountHealthChanged, "RegenerationMountHealthChanged"},             // 93
		{EventRegenerationCraftingChanged, "RegenerationCraftingChanged"},                   // 94
		{EventRegenerationHealthEnergyComboChanged, "RegenerationHealthEnergyComboChanged"}, // 95
		{EventRegenerationPlayerComboChanged, "RegenerationPlayerComboChanged"},             // 96
		{EventDurabilityChanged, "DurabilityChanged"},                                       // 97
		{EventNewLoot, "NewLoot"},                                                           // 98
		{EventAttachItemContainer, "AttachItemContainer"},                                   // 99
		{EventDetachItemContainer, "DetachItemContainer"},                                   // 100
		{EventInvalidateItemContainer, "InvalidateItemContainer"},                           // 101
		{EventLockItemContainer, "LockItemContainer"},                                       // 102
		{EventGuildUpdate, "GuildUpdate"},                                                   // 103
		{EventGuildPlayerUpdated, "GuildPlayerUpdated"},                                     // 104
		{EventInvitedToGuild, "InvitedToGuild"},                                             // 105
		{EventGuildMemberWorldUpdate, "GuildMemberWorldUpdate"},                             // 106
		{EventUpdateMatchDetails, "UpdateMatchDetails"},                                     // 107
		{EventObjectEvent, "ObjectEvent"},                                                   // 108
		{EventNewMonolithObject, "NewMonolithObject"},                                       // 109
		{EventMonolithHasBannersPlacedUpdate, "MonolithHasBannersPlacedUpdate"},             // 110
		{EventNewOrbObject, "NewOrbObject"},                                                 // 111
		{EventNewCastleObject, "NewCastleObject"},                                           // 112
		{EventNewSpellEffectArea, "NewSpellEffectArea"},                                     // 113
		{EventUpdateSpellEffectArea, "UpdateSpellEffectArea"},                               // 114
		{EventNewChainSpell, "NewChainSpell"},                                               // 115
		{EventUpdateChainSpell, "UpdateChainSpell"},                                         // 116
		{EventNewTreasureChest, "NewTreasureChest"},                                         // 117
		{EventStartMatch, "StartMatch"},                                                     // 118
		{EventStartArenaMatchInfos, "StartArenaMatchInfos"},                                 // 119
		{EventEndArenaMatch, "EndArenaMatch"},                                               // 120
		{EventMatchUpdate, "MatchUpdate"},                                                   // 121
		{EventActiveMatchUpdate, "ActiveMatchUpdate"},                                       // 122
		{EventNewMob, "NewMob"},                                                             // 123
		{EventDebugAggroInfo, "DebugAggroInfo"},                                             // 124
		{EventDebugVariablesInfo, "DebugVariablesInfo"},                                     // 125
		{EventDebugReputationInfo, "DebugReputationInfo"},                                   // 126
		{EventDebugDiminishingReturnInfo, "DebugDiminishingReturnInfo"},                     // 127
		{EventDebugSmartClusterQueueInfo, "DebugSmartClusterQueueInfo"},                     // 128
		{EventClaimOrbStart, "ClaimOrbStart"},                                               // 129
		{EventClaimOrbFinished, "ClaimOrbFinished"},                                         // 130
		{EventClaimOrbCancel, "ClaimOrbCancel"},                                             // 131
		{EventOrbUpdate, "OrbUpdate"},                                                       // 132
		{EventOrbClaimed, "OrbClaimed"},                                                     // 133
		{EventOrbReset, "OrbReset"},                                                         // 134
		{EventNewWarCampObject, "NewWarCampObject"},                                         // 135
		{EventNewMatchLootChestObject, "NewMatchLootChestObject"},                           // 136
		{EventNewArenaExit, "NewArenaExit"},                                                 // 137
		{EventGuildMemberTerritoryUpdate, "GuildMemberTerritoryUpdate"},                     // 138
		{EventInvitedMercenaryToMatch, "InvitedMercenaryToMatch"},                           // 139
		{EventClusterInfoUpdate, "ClusterInfoUpdate"},                                       // 140
		{EventForcedMovement, "ForcedMovement"},                                             // 141
		{EventForcedMovementCancel, "ForcedMovementCancel"},                                 // 142
		{EventCharacterStats, "CharacterStats"},                                             // 143
		{EventCharacterStatsKillHistory, "CharacterStatsKillHistory"},                       // 144
		{EventCharacterStatsDeathHistory, "CharacterStatsDeathHistory"},                     // 145
		{EventCharacterStatsKnockDownHistory, "CharacterStatsKnockDownHistory"},             // 146
		{EventCharacterStatsKnockedDownHistory, "CharacterStatsKnockedDownHistory"},         // 147
		{EventGuildStats, "GuildStats"},                                                     // 148
		{EventKillHistoryDetails, "KillHistoryDetails"},                                     // 149
		{EventItemKillHistoryDetails, "ItemKillHistoryDetails"},                             // 150
		{EventFullAchievementInfo, "FullAchievementInfo"},                                   // 151
		{EventFinishedAchievement, "FinishedAchievement"},                                   // 152
		{EventAchievementProgressInfo, "AchievementProgressInfo"},                           // 153
		{EventFullAchievementProgressInfo, "FullAchievementProgressInfo"},                   // 154
		{EventFullTrackedAchievementInfo, "FullTrackedAchievementInfo"},                     // 155
		{EventFullAutoLearnAchievementInfo, "FullAutoLearnAchievementInfo"},                 // 156
		{EventQuestGiverQuestOffered, "QuestGiverQuestOffered"},                             // 157
		{EventQuestGiverDebugInfo, "QuestGiverDebugInfo"},                                   // 158
		{EventConsoleEvent, "ConsoleEvent"},                                                 // 159
		{EventTimeSync, "TimeSync"},                                                         // 160
		{EventChangeAvatar, "ChangeAvatar"},                                                 // 161
		{EventChangeMountSkin, "ChangeMountSkin"},                                           // 162
		{EventGameEvent, "GameEvent"},                                                       // 163
		{EventKilledPlayer, "KilledPlayer"},                                                 // 164
		{EventDied, "Died"},                                                                 // 165
		{EventKnockedDown, "KnockedDown"},                                                   // 166
		{EventUnconcious, "Unconcious"},                                                     // 167
		{EventMatchPlayerJoinedEvent, "MatchPlayerJoinedEvent"},                             // 168
		{EventMatchPlayerStatsEvent, "MatchPlayerStatsEvent"},                               // 169
		{EventMatchPlayerStatsCompleteEvent, "MatchPlayerStatsCompleteEvent"},               // 170
		{EventMatchTimeLineEventEvent, "MatchTimeLineEventEvent"},                           // 171
		{EventMatchNewCombatRound, "MatchNewCombatRound"},                                   // 172
		{EventMatchEndCombatRound, "MatchEndCombatRound"},                                   // 173
		{EventMatchPlayerMainGearStatsEvent, "MatchPlayerMainGearStatsEvent"},               // 174
		{EventMatchPlayerChangedAvatarEvent, "MatchPlayerChangedAvatarEvent"},               // 175
		{EventInvitationPlayerTrade, "InvitationPlayerTrade"},                               // 176
		{EventPlayerTradeStart, "PlayerTradeStart"},                                         // 177
		{EventPlayerTradeCancel, "PlayerTradeCancel"},                                       // 178
		{EventPlayerTradeUpdate, "PlayerTradeUpdate"},                                       // 179
		{EventPlayerTradeFinished, "PlayerTradeFinished"},                                   // 180
		{EventPlayerTradeAcceptChange, "PlayerTradeAcceptChange"},                           // 181
		{EventMiniMapPing, "MiniMapPing"},                                                   // 182
		{EventMarketPlaceNotification, "MarketPlaceNotification"},                           // 183
		{EventDuellingChallengePlayer, "DuellingChallengePlayer"},                           // 184
		{EventNewDuellingPost, "NewDuellingPost"},                                           // 185
		{EventDuelStarted, "DuelStarted"},                                                   // 186
		{EventDuelEnded, "DuelEnded"},                                                       // 187
		{EventDuelDenied, "DuelDenied"},                                                     // 188
		{EventDuelRequestCanceled, "DuelRequestCanceled"},                                   // 189
		{EventDuelLeftArea, "DuelLeftArea"},                                                 // 190
		{EventDuelReEnteredArea, "DuelReEnteredArea"},                                       // 191
		{EventNewRealEstate, "NewRealEstate"},                                               // 192
		{EventMiniMapOwnedBuildingsPositions, "MiniMapOwnedBuildingsPositions"},             // 193
		{EventRealEstateListUpdate, "RealEstateListUpdate"},                                 // 194
		{EventGuildLogoUpdate, "GuildLogoUpdate"},                                           // 195
		{EventGuildLogoChanged, "GuildLogoChanged"},                                         // 196
		{EventPlaceableObjectPlace, "PlaceableObjectPlace"},                                 // 197
		{EventPlaceableObjectPlaceCancel, "PlaceableObjectPlaceCancel"},                     // 198
		{EventFurnitureObjectBuffProviderInfo, "FurnitureObjectBuffProviderInfo"},           // 199
		{EventFurnitureObjectCheatProviderInfo, "FurnitureObjectCheatProviderInfo"},         // 200
		{EventFarmableObjectInfo, "FarmableObjectInfo"},                                     // 201
		{EventNewUnreadMails, "NewUnreadMails"},                                             // 202
		{EventMailOperationPossible, "MailOperationPossible"},                               // 203
		{EventGuildLogoObjectUpdate, "GuildLogoObjectUpdate"},                               // 204
		{EventStartLogout, "StartLogout"},                                                   // 205
		{EventNewChatChannels, "NewChatChannels"},                                           // 206
		{EventJoinedChatChannel, "JoinedChatChannel"},                                       // 207
		{EventLeftChatChannel, "LeftChatChannel"},                                           // 208
		{EventRemovedChatChannel, "RemovedChatChannel"},                                     // 209
		{EventAccessStatus, "AccessStatus"},                                                 // 210
		{EventMounted, "Mounted"},                                                           // 211
		{EventMountStart, "MountStart"},                                                     // 212
		{EventMountCancel, "MountCancel"},                                                   // 213
		{EventNewTravelpoint, "NewTravelpoint"},                                             // 214
		{EventNewIslandAccessPoint, "NewIslandAccessPoint"},                                 // 215
		{EventNewExit, "NewExit"},                                                           // 216
		{EventUpdateHome, "UpdateHome"},                                                     // 217
		{EventUpdateChatSettings, "UpdateChatSettings"},                                     // 218
		{EventResurrectionOffer, "ResurrectionOffer"},                                       // 219
		{EventResurrectionReply, "ResurrectionReply"},                                       // 220
		{EventLootEquipmentChanged, "LootEquipmentChanged"},                                 // 221
		{EventUpdateUnlockedGuildLogos, "UpdateUnlockedGuildLogos"},                         // 222
		{EventUpdateUnlockedAvatars, "UpdateUnlockedAvatars"},                               // 223
		{EventUpdateUnlockedAvatarRings, "UpdateUnlockedAvatarRings"},                       // 224
		{EventUpdateUnlockedBuildings, "UpdateUnlockedBuildings"},                           // 225
		{EventNewIslandManagement, "NewIslandManagement"},                                   // 226
		{EventNewTeleportStone, "NewTeleportStone"},                                         // 227
		{EventCloak, "Cloak"},                                                               // 228
		{EventPartyInvitation, "PartyInvitation"},                                           // 229
		{EventPartyJoinRequest, "PartyJoinRequest"},                                         // 230
		{EventPartyJoined, "PartyJoined"},                                                   // 231
		{EventPartyDisbanded, "PartyDisbanded"},                                             // 232
		{EventPartyPlayerJoined, "PartyPlayerJoined"},                                       // 233
		{EventPartyChangedOrder, "PartyChangedOrder"},                                       // 234
		{EventPartyPlayerLeft, "PartyPlayerLeft"},                                           // 235
		{EventPartyLeaderChanged, "PartyLeaderChanged"},                                     // 236
		{EventPartyLootSettingChangedPlayer, "PartyLootSettingChangedPlayer"},               // 237
		{EventPartySilverGained, "PartySilverGained"},                                       // 238
		{EventPartyPlayerUpdated, "PartyPlayerUpdated"},                                     // 239
		{EventPartyInvitationAnswer, "PartyInvitationAnswer"},                               // 240
		{EventPartyJoinRequestAnswer, "PartyJoinRequestAnswer"},                             // 241
		{EventPartyMarkedObjectsUpdated, "PartyMarkedObjectsUpdated"},                       // 242
		{EventPartyOnClusterPartyJoined, "PartyOnClusterPartyJoined"},                       // 243
		{EventPartySetRoleFlag, "PartySetRoleFlag"},                                         // 244
		{EventPartyInviteOrJoinPlayerEquipmentInfo, "PartyInviteOrJoinPlayerEquipmentInfo"}, // 245
		{EventPartyReadyCheckUpdate, "PartyReadyCheckUpdate"},                               // 246
		{EventPartyFactionWarfareReinforcementSettingChangedPlayer, "PartyFactionWarfareReinforcementSettingChangedPlayer"},                 // 247
		{EventSpellCooldownUpdate, "SpellCooldownUpdate"},                                                                                   // 248
		{EventNewHellgateExitPortal, "NewHellgateExitPortal"},                                                                               // 249
		{EventNewExpeditionExit, "NewExpeditionExit"},                                                                                       // 250
		{EventNewExpeditionNarrator, "NewExpeditionNarrator"},                                                                               // 251
		{EventExitEnterStart, "ExitEnterStart"},                                                                                             // 252
		{EventExitEnterCancel, "ExitEnterCancel"},                                                                                           // 253
		{EventExitEnterFinished, "ExitEnterFinished"},                                                                                       // 254
		{EventNewQuestGiverObject, "NewQuestGiverObject"},                                                                                   // 255
		{EventFullQuestInfo, "FullQuestInfo"},                                                                                               // 256
		{EventQuestProgressInfo, "QuestProgressInfo"},                                                                                       // 257
		{EventQuestGiverInfoForPlayer, "QuestGiverInfoForPlayer"},                                                                           // 258
		{EventFullExpeditionInfo, "FullExpeditionInfo"},                                                                                     // 259
		{EventExpeditionQuestProgressInfo, "ExpeditionQuestProgressInfo"},                                                                   // 260
		{EventInvitedToExpedition, "InvitedToExpedition"},                                                                                   // 261
		{EventExpeditionRegistrationInfo, "ExpeditionRegistrationInfo"},                                                                     // 262
		{EventEnteringExpeditionStart, "EnteringExpeditionStart"},                                                                           // 263
		{EventEnteringExpeditionCancel, "EnteringExpeditionCancel"},                                                                         // 264
		{EventRewardGranted, "RewardGranted"},                                                                                               // 265
		{EventArenaRegistrationInfo, "ArenaRegistrationInfo"},                                                                               // 266
		{EventEnteringArenaStart, "EnteringArenaStart"},                                                                                     // 267
		{EventEnteringArenaCancel, "EnteringArenaCancel"},                                                                                   // 268
		{EventEnteringArenaLockStart, "EnteringArenaLockStart"},                                                                             // 269
		{EventEnteringArenaLockCancel, "EnteringArenaLockCancel"},                                                                           // 270
		{EventInvitedToArenaMatch, "InvitedToArenaMatch"},                                                                                   // 271
		{EventUsingHellgateShrine, "UsingHellgateShrine"},                                                                                   // 272
		{EventEnteringHellgateLockStart, "EnteringHellgateLockStart"},                                                                       // 273
		{EventEnteringHellgateLockCancel, "EnteringHellgateLockCancel"},                                                                     // 274
		{EventPlayerCounts, "PlayerCounts"},                                                                                                 // 275
		{EventInCombatStateUpdate, "InCombatStateUpdate"},                                                                                   // 276
		{EventOtherGrabbedLoot, "OtherGrabbedLoot"},                                                                                         // 277
		{EventTreasureChestUsingStart, "TreasureChestUsingStart"},                                                                           // 278
		{EventTreasureChestUsingFinished, "TreasureChestUsingFinished"},                                                                     // 279
		{EventTreasureChestUsingCancel, "TreasureChestUsingCancel"},                                                                         // 280
		{EventTreasureChestUsingOpeningComplete, "TreasureChestUsingOpeningComplete"},                                                       // 281
		{EventTreasureChestForceCloseInventory, "TreasureChestForceCloseInventory"},                                                         // 282
		{EventLocalTreasuresUpdate, "LocalTreasuresUpdate"},                                                                                 // 283
		{EventLootChestSpawnpointsUpdate, "LootChestSpawnpointsUpdate"},                                                                     // 284
		{EventPremiumChanged, "PremiumChanged"},                                                                                             // 285
		{EventPremiumExtended, "PremiumExtended"},                                                                                           // 286
		{EventPremiumLifeTimeRewardGained, "PremiumLifeTimeRewardGained"},                                                                   // 287
		{EventGoldPurchased, "GoldPurchased"},                                                                                               // 288
		{EventLaborerGotUpgraded, "LaborerGotUpgraded"},                                                                                     // 289
		{EventJournalGotFull, "JournalGotFull"},                                                                                             // 290
		{EventJournalFillError, "JournalFillError"},                                                                                         // 291
		{EventFriendRequest, "FriendRequest"},                                                                                               // 292
		{EventFriendRequestInfos, "FriendRequestInfos"},                                                                                     // 293
		{EventFriendInfos, "FriendInfos"},                                                                                                   // 294
		{EventFriendRequestAnswered, "FriendRequestAnswered"},                                                                               // 295
		{EventFriendOnlineStatus, "FriendOnlineStatus"},                                                                                     // 296
		{EventFriendRequestCanceled, "FriendRequestCanceled"},                                                                               // 297
		{EventFriendRemoved, "FriendRemoved"},                                                                                               // 298
		{EventFriendUpdated, "FriendUpdated"},                                                                                               // 299
		{EventPartyLootItems, "PartyLootItems"},                                                                                             // 300
		{EventPartyLootItemsRemoved, "PartyLootItemsRemoved"},                                                                               // 301
		{EventPartyLootItemTypesRemoved, "PartyLootItemTypesRemoved"},                                                                       // 302
		{EventReputationUpdate, "ReputationUpdate"},                                                                                         // 303
		{EventDefenseUnitAttackBegin, "DefenseUnitAttackBegin"},                                                                             // 304
		{EventDefenseUnitAttackEnd, "DefenseUnitAttackEnd"},                                                                                 // 305
		{EventDefenseUnitAttackDamage, "DefenseUnitAttackDamage"},                                                                           // 306
		{EventUnrestrictedPvpZoneUpdate, "UnrestrictedPvpZoneUpdate"},                                                                       // 307
		{EventUnrestrictedPvpZoneStatus, "UnrestrictedPvpZoneStatus"},                                                                       // 308
		{EventReputationImplicationUpdate, "ReputationImplicationUpdate"},                                                                   // 309
		{EventNewMountObject, "NewMountObject"},                                                                                             // 310
		{EventMountHealthUpdate, "MountHealthUpdate"},                                                                                       // 311
		{EventMountCooldownUpdate, "MountCooldownUpdate"},                                                                                   // 312
		{EventNewExpeditionAgent, "NewExpeditionAgent"},                                                                                     // 313
		{EventNewExpeditionCheckPoint, "NewExpeditionCheckPoint"},                                                                           // 314
		{EventExpeditionStartEvent, "ExpeditionStartEvent"},                                                                                 // 315
		{EventVoteEvent, "VoteEvent"},                                                                                                       // 316
		{EventRatingEvent, "RatingEvent"},                                                                                                   // 317
		{EventNewArenaAgent, "NewArenaAgent"},                                                                                               // 318
		{EventBoostFarmable, "BoostFarmable"},                                                                                               // 319
		{EventUseFunction, "UseFunction"},                                                                                                   // 320
		{EventNewPortalEntrance, "NewPortalEntrance"},                                                                                       // 321
		{EventNewPortalExit, "NewPortalExit"},                                                                                               // 322
		{EventNewRandomDungeonExit, "NewRandomDungeonExit"},                                                                                 // 323
		{EventWaitingQueueUpdate, "WaitingQueueUpdate"},                                                                                     // 324
		{EventPlayerMovementRateUpdate, "PlayerMovementRateUpdate"},                                                                         // 325
		{EventObserveStart, "ObserveStart"},                                                                                                 // 326
		{EventMinimapZergs, "MinimapZergs"},                                                                                                 // 327
		{EventMinimapSmartClusterZergs, "MinimapSmartClusterZergs"},                                                                         // 328
		{EventPaymentTransactions, "PaymentTransactions"},                                                                                   // 329
		{EventPerformanceStatsUpdate, "PerformanceStatsUpdate"},                                                                             // 330
		{EventOverloadModeUpdate, "OverloadModeUpdate"},                                                                                     // 331
		{EventDebugDrawEvent, "DebugDrawEvent"},                                                                                             // 332
		{EventRecordCameraMove, "RecordCameraMove"},                                                                                         // 333
		{EventRecordStart, "RecordStart"},                                                                                                   // 334
		{EventClaimPowerCrystalStart, "ClaimPowerCrystalStart"},                                                                             // 335
		{EventClaimPowerCrystalCancel, "ClaimPowerCrystalCancel"},                                                                           // 336
		{EventClaimPowerCrystalReset, "ClaimPowerCrystalReset"},                                                                             // 337
		{EventClaimPowerCrystalFinished, "ClaimPowerCrystalFinished"},                                                                       // 338
		{EventTerritoryClaimStart, "TerritoryClaimStart"},                                                                                   // 339
		{EventTerritoryClaimCancel, "TerritoryClaimCancel"},                                                                                 // 340
		{EventTerritoryClaimFinished, "TerritoryClaimFinished"},                                                                             // 341
		{EventTerritoryScheduleResult, "TerritoryScheduleResult"},                                                                           // 342
		{EventTerritoryUpgradeWithPowerCrystalResult, "TerritoryUpgradeWithPowerCrystalResult"},                                             // 343
		{EventReturningPowerCrystalStart, "ReturningPowerCrystalStart"},                                                                     // 344
		{EventReturningPowerCrystalFinished, "ReturningPowerCrystalFinished"},                                                               // 345
		{EventUpdateAccountState, "UpdateAccountState"},                                                                                     // 346
		{EventStartDeterministicRoam, "StartDeterministicRoam"},                                                                             // 347
		{EventGuildFullAccessTagsUpdated, "GuildFullAccessTagsUpdated"},                                                                     // 348
		{EventGuildAccessTagUpdated, "GuildAccessTagUpdated"},                                                                               // 349
		{EventGvgSeasonUpdate, "GvgSeasonUpdate"},                                                                                           // 350
		{EventGvgSeasonCheatCommand, "GvgSeasonCheatCommand"},                                                                               // 351
		{EventSeasonPointsByKillingBooster, "SeasonPointsByKillingBooster"},                                                                 // 352
		{EventFishingStart, "FishingStart"},                                                                                                 // 353
		{EventFishingCast, "FishingCast"},                                                                                                   // 354
		{EventFishingCatch, "FishingCatch"},                                                                                                 // 355
		{EventFishingFinished, "FishingFinished"},                                                                                           // 356
		{EventFishingCancel, "FishingCancel"},                                                                                               // 357
		{EventNewFloatObject, "NewFloatObject"},                                                                                             // 358
		{EventNewFishingZoneObject, "NewFishingZoneObject"},                                                                                 // 359
		{EventFishingMiniGame, "FishingMiniGame"},                                                                                           // 360
		{EventSteamAchievementCompleted, "SteamAchievementCompleted"},                                                                       // 361
		{EventUpdatePuppet, "UpdatePuppet"},                                                                                                 // 362
		{EventChangeFlaggingFinished, "ChangeFlaggingFinished"},                                                                             // 363
		{EventNewOutpostObject, "NewOutpostObject"},                                                                                         // 364
		{EventOutpostUpdate, "OutpostUpdate"},                                                                                               // 365
		{EventOutpostClaimed, "OutpostClaimed"},                                                                                             // 366
		{EventOverChargeEnd, "OverChargeEnd"},                                                                                               // 367
		{EventOverChargeStatus, "OverChargeStatus"},                                                                                         // 368
		{EventPartyFinderFullUpdate, "PartyFinderFullUpdate"},                                                                               // 369
		{EventPartyFinderUpdate, "PartyFinderUpdate"},                                                                                       // 370
		{EventPartyFinderApplicantsUpdate, "PartyFinderApplicantsUpdate"},                                                                   // 371
		{EventPartyFinderEquipmentSnapshot, "PartyFinderEquipmentSnapshot"},                                                                 // 372
		{EventPartyFinderJoinRequestDeclined, "PartyFinderJoinRequestDeclined"},                                                             // 373
		{EventNewUnlockedPersonalSeasonRewards, "NewUnlockedPersonalSeasonRewards"},                                                         // 374
		{EventPersonalSeasonPointsGained, "PersonalSeasonPointsGained"},                                                                     // 375
		{EventPersonalSeasonPastSeasonDataEvent, "PersonalSeasonPastSeasonDataEvent"},                                                       // 376
		{EventMatchLootChestOpeningStart, "MatchLootChestOpeningStart"},                                                                     // 377
		{EventMatchLootChestOpeningFinished, "MatchLootChestOpeningFinished"},                                                               // 378
		{EventMatchLootChestOpeningCancel, "MatchLootChestOpeningCancel"},                                                                   // 379
		{EventNotifyCrystalMatchReward, "NotifyCrystalMatchReward"},                                                                         // 380
		{EventCrystalRealmFeedback, "CrystalRealmFeedback"},                                                                                 // 381
		{EventNewLocationMarker, "NewLocationMarker"},                                                                                       // 382
		{EventNewTutorialBlocker, "NewTutorialBlocker"},                                                                                     // 383
		{EventNewTileSwitch, "NewTileSwitch"},                                                                                               // 384
		{EventNewInformationProvider, "NewInformationProvider"},                                                                             // 385
		{EventNewDynamicGuildLogo, "NewDynamicGuildLogo"},                                                                                   // 386
		{EventNewDecoration, "NewDecoration"},                                                                                               // 387
		{EventTutorialUpdate, "TutorialUpdate"},                                                                                             // 388
		{EventTriggerHintBox, "TriggerHintBox"},                                                                                             // 389
		{EventRandomDungeonPositionInfo, "RandomDungeonPositionInfo"},                                                                       // 390
		{EventNewLootChest, "NewLootChest"},                                                                                                 // 391
		{EventUpdateLootChest, "UpdateLootChest"},                                                                                           // 392
		{EventLootChestOpened, "LootChestOpened"},                                                                                           // 393
		{EventUpdateLootProtectedByMobsWithMinimapDisplay, "UpdateLootProtectedByMobsWithMinimapDisplay"},                                   // 394
		{EventNewShrine, "NewShrine"},                                                                                                       // 395
		{EventUpdateShrine, "UpdateShrine"},                                                                                                 // 396
		{EventUpdateRoom, "UpdateRoom"},                                                                                                     // 397
		{EventNewMobSoul, "NewMobSoul"},                                                                                                     // 398
		{EventNewHellgateShrine, "NewHellgateShrine"},                                                                                       // 399
		{EventUpdateHellgateShrine, "UpdateHellgateShrine"},                                                                                 // 400
		{EventActivateHellgateExit, "ActivateHellgateExit"},                                                                                 // 401
		{EventMutePlayerUpdate, "MutePlayerUpdate"},                                                                                         // 402
		{EventShopTileUpdate, "ShopTileUpdate"},                                                                                             // 403
		{EventShopUpdate, "ShopUpdate"},                                                                                                     // 404
		{EventEasyAntiCheatKick, "EasyAntiCheatKick"},                                                                                       // 405
		{EventBattlEyeServerMessage, "BattlEyeServerMessage"},                                                                               // 406
		{EventUnlockVanityUnlock, "UnlockVanityUnlock"},                                                                                     // 407
		{EventAvatarUnlocked, "AvatarUnlocked"},                                                                                             // 408
		{EventCustomizationChanged, "CustomizationChanged"},                                                                                 // 409
		{EventBaseVaultInfo, "BaseVaultInfo"},                                                                                               // 410
		{EventGuildVaultInfo, "GuildVaultInfo"},                                                                                             // 411
		{EventBankVaultInfo, "BankVaultInfo"},                                                                                               // 412
		{EventRecoveryVaultPlayerInfo, "RecoveryVaultPlayerInfo"},                                                                           // 413
		{EventRecoveryVaultGuildInfo, "RecoveryVaultGuildInfo"},                                                                             // 414
		{EventUpdateWardrobe, "UpdateWardrobe"},                                                                                             // 415
		{EventCastlePhaseChanged, "CastlePhaseChanged"},                                                                                     // 416
		{EventGuildAccountLogEvent, "GuildAccountLogEvent"},                                                                                 // 417
		{EventNewHideoutObject, "NewHideoutObject"},                                                                                         // 418
		{EventNewHideoutManagement, "NewHideoutManagement"},                                                                                 // 419
		{EventNewHideoutExit, "NewHideoutExit"},                                                                                             // 420
		{EventInitHideoutAttackStart, "InitHideoutAttackStart"},                                                                             // 421
		{EventInitHideoutAttackCancel, "InitHideoutAttackCancel"},                                                                           // 422
		{EventInitHideoutAttackFinished, "InitHideoutAttackFinished"},                                                                       // 423
		{EventHideoutManagementUpdate, "HideoutManagementUpdate"},                                                                           // 424
		{EventHideoutUpgradeWithPowerCrystalResult, "HideoutUpgradeWithPowerCrystalResult"},                                                 // 425
		{EventIpChanged, "IpChanged"},                                                                                                       // 426
		{EventSmartClusterQueueUpdateInfo, "SmartClusterQueueUpdateInfo"},                                                                   // 427
		{EventSmartClusterQueueActiveInfo, "SmartClusterQueueActiveInfo"},                                                                   // 428
		{EventSmartClusterQueueKickWarning, "SmartClusterQueueKickWarning"},                                                                 // 429
		{EventSmartClusterQueueInvite, "SmartClusterQueueInvite"},                                                                           // 430
		{EventReceivedGvgSeasonPoints, "ReceivedGvgSeasonPoints"},                                                                           // 431
		{EventTowerPowerPointUpdate, "TowerPowerPointUpdate"},                                                                               // 432
		{EventOpenWorldAttackScheduleStart, "OpenWorldAttackScheduleStart"},                                                                 // 433
		{EventOpenWorldAttackScheduleFinished, "OpenWorldAttackScheduleFinished"},                                                           // 434
		{EventOpenWorldAttackScheduleCancel, "OpenWorldAttackScheduleCancel"},                                                               // 435
		{EventOpenWorldAttackConquerStart, "OpenWorldAttackConquerStart"},                                                                   // 436
		{EventOpenWorldAttackConquerFinished, "OpenWorldAttackConquerFinished"},                                                             // 437
		{EventOpenWorldAttackConquerCancel, "OpenWorldAttackConquerCancel"},                                                                 // 438
		{EventOpenWorldAttackConquerStatus, "OpenWorldAttackConquerStatus"},                                                                 // 439
		{EventOpenWorldAttackStart, "OpenWorldAttackStart"},                                                                                 // 440
		{EventOpenWorldAttackEnd, "OpenWorldAttackEnd"},                                                                                     // 441
		{EventNewRandomResourceBlocker, "NewRandomResourceBlocker"},                                                                         // 442
		{EventNewHomeObject, "NewHomeObject"},                                                                                               // 443
		{EventHideoutObjectUpdate, "HideoutObjectUpdate"},                                                                                   // 444
		{EventUpdateInfamy, "UpdateInfamy"},                                                                                                 // 445
		{EventMinimapPositionMarkers, "MinimapPositionMarkers"},                                                                             // 446
		{EventNewTunnelExit, "NewTunnelExit"},                                                                                               // 447
		{EventCorruptedDungeonUpdate, "CorruptedDungeonUpdate"},                                                                             // 448
		{EventCorruptedDungeonStatus, "CorruptedDungeonStatus"},                                                                             // 449
		{EventCorruptedDungeonInfamy, "CorruptedDungeonInfamy"},                                                                             // 450
		{EventHellgateRestrictedAreaUpdate, "HellgateRestrictedAreaUpdate"},                                                                 // 451
		{EventHellgateInfamy, "HellgateInfamy"},                                                                                             // 452
		{EventHellgateStatus, "HellgateStatus"},                                                                                             // 453
		{EventHellgateStatusUpdate, "HellgateStatusUpdate"},                                                                                 // 454
		{EventHellgateSuspense, "HellgateSuspense"},                                                                                         // 455
		{EventReplaceSpellSlotWithMultiSpell, "ReplaceSpellSlotWithMultiSpell"},                                                             // 456
		{EventNewCorruptedShrine, "NewCorruptedShrine"},                                                                                     // 457
		{EventUpdateCorruptedShrine, "UpdateCorruptedShrine"},                                                                               // 458
		{EventCorruptedShrineUsageStart, "CorruptedShrineUsageStart"},                                                                       // 459
		{EventCorruptedShrineUsageCancel, "CorruptedShrineUsageCancel"},                                                                     // 460
		{EventExitUsed, "ExitUsed"},                                                                                                         // 461
		{EventLinkedToObject, "LinkedToObject"},                                                                                             // 462
		{EventLinkToObjectBroken, "LinkToObjectBroken"},                                                                                     // 463
		{EventEstimatedMarketValueUpdate, "EstimatedMarketValueUpdate"},                                                                     // 464
		{EventStuckCancel, "StuckCancel"},                                                                                                   // 465
		{EventDungonEscapeReady, "DungonEscapeReady"},                                                                                       // 466
		{EventFactionWarfareClusterState, "FactionWarfareClusterState"},                                                                     // 467
		{EventFactionWarfareHasUnclaimedWeeklyReportsEvent, "FactionWarfareHasUnclaimedWeeklyReportsEvent"},                                 // 468
		{EventSimpleFeedback, "SimpleFeedback"},                                                                                             // 469
		{EventSmartClusterQueueSkipClusterError, "SmartClusterQueueSkipClusterError"},                                                       // 470
		{EventXignCodeEvent, "XignCodeEvent"},                                                                                               // 471
		{EventBatchUseItemStart, "BatchUseItemStart"},                                                                                       // 472
		{EventBatchUseItemEnd, "BatchUseItemEnd"},                                                                                           // 473
		{EventRedZonePlayerNotification, "RedZonePlayerNotification"},                                                                       // 474
		{EventRedZoneEventCheatCleanup, "RedZoneEventCheatCleanup"},                                                                         // 475
		{EventRedZoneFortressEventChestOpened, "RedZoneFortressEventChestOpened"},                                                           // 476
		{EventRedZoneWorldEvent, "RedZoneWorldEvent"},                                                                                       // 477
		{EventFactionWarfareStats, "FactionWarfareStats"},                                                                                   // 478
		{EventUpdateFactionBalanceFactors, "UpdateFactionBalanceFactors"},                                                                   // 479
		{EventFactionEnlistmentChanged, "FactionEnlistmentChanged"},                                                                         // 480
		{EventUpdateFactionRank, "UpdateFactionRank"},                                                                                       // 481
		{EventFactionWarfareCampaignRewardsUnlocked, "FactionWarfareCampaignRewardsUnlocked"},                                               // 482
		{EventFeaturedFeatureUpdate, "FeaturedFeatureUpdate"},                                                                               // 483
		{EventNewPowerCrystalObject, "NewPowerCrystalObject"},                                                                               // 484
		{EventMinimapCrystalPositionMarker, "MinimapCrystalPositionMarker"},                                                                 // 485
		{EventCarryPowerCrystalUpdate, "CarryPowerCrystalUpdate"},                                                                           // 486
		{EventPickupPowerCrystalStart, "PickupPowerCrystalStart"},                                                                           // 487
		{EventPickupPowerCrystalCancel, "PickupPowerCrystalCancel"},                                                                         // 488
		{EventPickupPowerCrystalFinished, "PickupPowerCrystalFinished"},                                                                     // 489
		{EventDoSimpleActionStart, "DoSimpleActionStart"},                                                                                   // 490
		{EventDoSimpleActionCancel, "DoSimpleActionCancel"},                                                                                 // 491
		{EventDoSimpleActionFinished, "DoSimpleActionFinished"},                                                                             // 492
		{EventNotifyGuestAccountVerified, "NotifyGuestAccountVerified"},                                                                     // 493
		{EventMightAndFavorReceivedEvent, "MightAndFavorReceivedEvent"},                                                                     // 494
		{EventWeeklyPvpChallengeRewardStateUpdate, "WeeklyPvpChallengeRewardStateUpdate"},                                                   // 495
		{EventNewUnlockedPvpSeasonChallengeRewards, "NewUnlockedPvpSeasonChallengeRewards"},                                                 // 496
		{EventStaticDungeonEntrancesDungeonEventStatusUpdates, "StaticDungeonEntrancesDungeonEventStatusUpdates"},                           // 497
		{EventStaticDungeonDungeonValueUpdate, "StaticDungeonDungeonValueUpdate"},                                                           // 498
		{EventStaticDungeonEntranceDungeonEventsAborted, "StaticDungeonEntranceDungeonEventsAborted"},                                       // 499
		{EventInAppPurchaseConfirmedGooglePlay, "InAppPurchaseConfirmedGooglePlay"},                                                         // 500
		{EventFeatureSwitchInfo, "FeatureSwitchInfo"},                                                                                       // 501
		{EventPartyJoinRequestAborted, "PartyJoinRequestAborted"},                                                                           // 502
		{EventPartyInviteAborted, "PartyInviteAborted"},                                                                                     // 503
		{EventPartyStartHuntRequest, "PartyStartHuntRequest"},                                                                               // 504
		{EventPartyStartHuntRequested, "PartyStartHuntRequested"},                                                                           // 505
		{EventPartyStartHuntRequestAnswer, "PartyStartHuntRequestAnswer"},                                                                   // 506
		{EventPartyPlayerLeaveScheduled, "PartyPlayerLeaveScheduled"},                                                                       // 507
		{EventGuildInviteDeclined, "GuildInviteDeclined"},                                                                                   // 508
		{EventCancelMultiSpellSlots, "CancelMultiSpellSlots"},                                                                               // 509
		{EventNewVisualEventObject, "NewVisualEventObject"},                                                                                 // 510
		{EventCastleClaimProgress, "CastleClaimProgress"},                                                                                   // 511
		{EventCastleClaimProgressLogo, "CastleClaimProgressLogo"},                                                                           // 512
		{EventTownPortalUpdateState, "TownPortalUpdateState"},                                                                               // 513
		{EventTownPortalFailed, "TownPortalFailed"},                                                                                         // 514
		{EventConsumableVanityChargesAdded, "ConsumableVanityChargesAdded"},                                                                 // 515
		{EventFestivitiesUpdate, "FestivitiesUpdate"},                                                                                       // 516
		{EventNewBannerObject, "NewBannerObject"},                                                                                           // 517
		{EventNewMistsImmediateReturnExit, "NewMistsImmediateReturnExit"},                                                                   // 518
		{EventMistsPlayerJoinedInfo, "MistsPlayerJoinedInfo"},                                                                               // 519
		{EventNewMistsStaticEntrance, "NewMistsStaticEntrance"},                                                                             // 520
		{EventNewMistsOpenWorldExit, "NewMistsOpenWorldExit"},                                                                               // 521
		{EventNewTunnelExitTemp, "NewTunnelExitTemp"},                                                                                       // 522
		{EventNewMistsWispSpawn, "NewMistsWispSpawn"},                                                                                       // 523
		{EventMistsWispSpawnStateChange, "MistsWispSpawnStateChange"},                                                                       // 524
		{EventNewMistsCityEntrance, "NewMistsCityEntrance"},                                                                                 // 525
		{EventNewMistsCityRoadsEntrance, "NewMistsCityRoadsEntrance"},                                                                       // 526
		{EventMistsCityRoadsEntrancePartyStateUpdate, "MistsCityRoadsEntrancePartyStateUpdate"},                                             // 527
		{EventMistsCityRoadsEntranceClearStateForParty, "MistsCityRoadsEntranceClearStateForParty"},                                         // 528
		{EventMistsEntranceDataChanged, "MistsEntranceDataChanged"},                                                                         // 529
		{EventNewCagedObject, "NewCagedObject"},                                                                                             // 530
		{EventCagedObjectStateUpdated, "CagedObjectStateUpdated"},                                                                           // 531
		{EventEntrancePartyBindingCreated, "EntrancePartyBindingCreated"},                                                                   // 532
		{EventEntrancePartyBindingCleared, "EntrancePartyBindingCleared"},                                                                   // 533
		{EventEntrancePartyBindingInfos, "EntrancePartyBindingInfos"},                                                                       // 534
		{EventNewMistsBorderExit, "NewMistsBorderExit"},                                                                                     // 535
		{EventNewMistsDungeonExit, "NewMistsDungeonExit"},                                                                                   // 536
		{EventLocalQuestInfos, "LocalQuestInfos"},                                                                                           // 537
		{EventLocalQuestStarted, "LocalQuestStarted"},                                                                                       // 538
		{EventLocalQuestActive, "LocalQuestActive"},                                                                                         // 539
		{EventLocalQuestInactive, "LocalQuestInactive"},                                                                                     // 540
		{EventLocalQuestProgressUpdate, "LocalQuestProgressUpdate"},                                                                         // 541
		{EventNewUnrestrictedPvpZone, "NewUnrestrictedPvpZone"},                                                                             // 542
		{EventTemporaryFlaggingStatusUpdate, "TemporaryFlaggingStatusUpdate"},                                                               // 543
		{EventSpellTestPerformanceUpdate, "SpellTestPerformanceUpdate"},                                                                     // 544
		{EventTransformation, "Transformation"},                                                                                             // 545
		{EventTransformationEnd, "TransformationEnd"},                                                                                       // 546
		{EventUpdateTrustlevel, "UpdateTrustlevel"},                                                                                         // 547
		{EventRevealHiddenTimeStamps, "RevealHiddenTimeStamps"},                                                                             // 548
		{EventModifyItemTraitFinished, "ModifyItemTraitFinished"},                                                                           // 549
		{EventRerollItemTraitValueFinished, "RerollItemTraitValueFinished"},                                                                 // 550
		{EventHuntQuestProgressInfo, "HuntQuestProgressInfo"},                                                                               // 551
		{EventHuntStarted, "HuntStarted"},                                                                                                   // 552
		{EventHuntFinished, "HuntFinished"},                                                                                                 // 553
		{EventHuntAborted, "HuntAborted"},                                                                                                   // 554
		{EventHuntMissionStepStateUpdate, "HuntMissionStepStateUpdate"},                                                                     // 555
		{EventNewHuntTrack, "NewHuntTrack"},                                                                                                 // 556
		{EventHuntMissionUpdate, "HuntMissionUpdate"},                                                                                       // 557
		{EventHuntQuestMissionProgressUpdate, "HuntQuestMissionProgressUpdate"},                                                             // 558
		{EventHuntTrackUsed, "HuntTrackUsed"},                                                                                               // 559
		{EventHuntTrackUseableAgain, "HuntTrackUseableAgain"},                                                                               // 560
		{EventMinimapHuntTrackMarkers, "MinimapHuntTrackMarkers"},                                                                           // 561
		{EventNoTracksFound, "NoTracksFound"},                                                                                               // 562
		{EventHuntQuestAborted, "HuntQuestAborted"},                                                                                         // 563
		{EventInteractWithTrackStart, "InteractWithTrackStart"},                                                                             // 564
		{EventInteractWithTrackCancel, "InteractWithTrackCancel"},                                                                           // 565
		{EventInteractWithTrackFinished, "InteractWithTrackFinished"},                                                                       // 566
		{EventNewDynamicCompound, "NewDynamicCompound"},                                                                                     // 567
		{EventLegendaryItemDestroyed, "LegendaryItemDestroyed"},                                                                             // 568
		{EventAttunementInfo, "AttunementInfo"},                                                                                             // 569
		{EventTerritoryClaimRaidedRawEnergyCrystalResult, "TerritoryClaimRaidedRawEnergyCrystalResult"},                                     // 570
		{EventCarriedObjectExpiryWarning, "CarriedObjectExpiryWarning"},                                                                     // 571
		{EventCarriedObjectExpired, "CarriedObjectExpired"},                                                                                 // 572
		{EventTerritoryRaidStart, "TerritoryRaidStart"},                                                                                     // 573
		{EventTerritoryRaidCancel, "TerritoryRaidCancel"},                                                                                   // 574
		{EventTerritoryRaidFinished, "TerritoryRaidFinished"},                                                                               // 575
		{EventTerritoryRaidResult, "TerritoryRaidResult"},                                                                                   // 576
		{EventTerritoryMonolithActiveRaidStatus, "TerritoryMonolithActiveRaidStatus"},                                                       // 577
		{EventTerritoryMonolithActiveRaidCancelled, "TerritoryMonolithActiveRaidCancelled"},                                                 // 578
		{EventMonolithEnergyStorageUpdate, "MonolithEnergyStorageUpdate"},                                                                   // 579
		{EventMonolithNextScheduledOpenWorldAttackUpdate, "MonolithNextScheduledOpenWorldAttackUpdate"},                                     // 580
		{EventMonolithProtectedBuildingsDamageReductionUpdate, "MonolithProtectedBuildingsDamageReductionUpdate"},                           // 581
		{EventNewBuildingBaseEvent, "NewBuildingBaseEvent"},                                                                                 // 582
		{EventNewFortificationBuilding, "NewFortificationBuilding"},                                                                         // 583
		{EventNewCastleGateBuilding, "NewCastleGateBuilding"},                                                                               // 584
		{EventBuildingDurabilityUpdate, "BuildingDurabilityUpdate"},                                                                         // 585
		{EventMonolithFortificationPointsUpdate, "MonolithFortificationPointsUpdate"},                                                       // 586
		{EventFortificationBuildingUpgradeInfo, "FortificationBuildingUpgradeInfo"},                                                         // 587
		{EventFortificationBuildingsDamageStateUpdate, "FortificationBuildingsDamageStateUpdate"},                                           // 588
		{EventSiegeNotificationEvent, "SiegeNotificationEvent"},                                                                             // 589
		{EventUpdateEnemyWarBannerActive, "UpdateEnemyWarBannerActive"},                                                                     // 590
		{EventTerritoryAnnouncePlayerEjection, "TerritoryAnnouncePlayerEjection"},                                                           // 591
		{EventCastleGateSwitchUseStarted, "CastleGateSwitchUseStarted"},                                                                     // 592
		{EventCastleGateSwitchUseFinished, "CastleGateSwitchUseFinished"},                                                                   // 593
		{EventFortificationBuildingWillDowngrade, "FortificationBuildingWillDowngrade"},                                                     // 594
		{EventBotCommand, "BotCommand"},                                                                                                     // 595
		{EventJournalAchievementProgressUpdate, "JournalAchievementProgressUpdate"},                                                         // 596
		{EventJournalClaimableRewardUpdate, "JournalClaimableRewardUpdate"},                                                                 // 597
		{EventKeySync, "KeySync"},                                                                                                           // 598
		{EventLocalQuestAreaGone, "LocalQuestAreaGone"},                                                                                     // 599
		{EventDynamicTemplate, "DynamicTemplate"},                                                                                           // 600
		{EventDynamicTemplateForcedStateChange, "DynamicTemplateForcedStateChange"},                                                         // 601
		{EventNewOutlandsTeleportationPortal, "NewOutlandsTeleportationPortal"},                                                             // 602
		{EventNewOutlandsTeleportationReturnPortal, "NewOutlandsTeleportationReturnPortal"},                                                 // 603
		{EventOutlandsTeleportationBindingCleared, "OutlandsTeleportationBindingCleared"},                                                   // 604
		{EventOutlandsTeleportationReturnPortalUpdateEvent, "OutlandsTeleportationReturnPortalUpdateEvent"},                                 // 605
		{EventPlayerUsedOutlandsTeleportationPortal, "PlayerUsedOutlandsTeleportationPortal"},                                               // 606
		{EventEncumberedRestricted, "EncumberedRestricted"},                                                                                 // 607
		{EventNewPiledObject, "NewPiledObject"},                                                                                             // 608
		{EventPiledObjectStateChanged, "PiledObjectStateChanged"},                                                                           // 609
		{EventNewSmugglerCrateDeliveryStation, "NewSmugglerCrateDeliveryStation"},                                                           // 610
		{EventKillRewardedNoFame, "KillRewardedNoFame"},                                                                                     // 611
		{EventPickupFromPiledObjectStart, "PickupFromPiledObjectStart"},                                                                     // 612
		{EventPickupFromPiledObjectCancel, "PickupFromPiledObjectCancel"},                                                                   // 613
		{EventPickupFromPiledObjectReset, "PickupFromPiledObjectReset"},                                                                     // 614
		{EventPickupFromPiledObjectFinished, "PickupFromPiledObjectFinished"},                                                               // 615
		{EventArmoryActivityChange, "ArmoryActivityChange"},                                                                                 // 616
		{EventNewKillTrophyFurnitureBuilding, "NewKillTrophyFurnitureBuilding"},                                                             // 617
		{EventHellDungeonsPlayerJoinedInfo, "HellDungeonsPlayerJoinedInfo"},                                                                 // 618
		{EventNewTileSwitchTrigger, "NewTileSwitchTrigger"},                                                                                 // 619
		{EventNewMultiRewardObject, "NewMultiRewardObject"},                                                                                 // 620
		{EventNewHellDungeonSoulShrineObject, "NewHellDungeonSoulShrineObject"},                                                             // 621
		{EventHellDungeonSoulShrineStateUpdate, "HellDungeonSoulShrineStateUpdate"},                                                         // 622
		{EventNewResurrectionShrine, "NewResurrectionShrine"},                                                                               // 623
		{EventUpdateResurrectionShrine, "UpdateResurrectionShrine"},                                                                         // 624
		{EventStandTimeFinished, "StandTimeFinished"},                                                                                       // 625
		{EventEpicAchievementAndStatsUpdate, "EpicAchievementAndStatsUpdate"},                                                               // 626
		{EventSpectateTargetAfterDeathUpdate, "SpectateTargetAfterDeathUpdate"},                                                             // 627
		{EventSpectateTargetAfterDeathEnded, "SpectateTargetAfterDeathEnded"},                                                               // 628
		{EventNewHellDungeonUpwardExit, "NewHellDungeonUpwardExit"},                                                                         // 629
		{EventNewHellDungeonSoulExit, "NewHellDungeonSoulExit"},                                                                             // 630
		{EventNewHellDungeonDownwardExit, "NewHellDungeonDownwardExit"},                                                                     // 631
		{EventNewHellDungeonChestExit, "NewHellDungeonChestExit"},                                                                           // 632
		{EventNewCorruptedStaticEntrance, "NewCorruptedStaticEntrance"},                                                                     // 633
		{EventNewHellDungeonStaticEntrance, "NewHellDungeonStaticEntrance"},                                                                 // 634
		{EventUpdateHellDungeonStaticEntranceState, "UpdateHellDungeonStaticEntranceState"},                                                 // 635
		{EventDebugTriggerHellDungeonShutdownStart, "DebugTriggerHellDungeonShutdownStart"},                                                 // 636
		{EventFullJournalQuestInfo, "FullJournalQuestInfo"},                                                                                 // 637
		{EventJournalQuestProgressInfo, "JournalQuestProgressInfo"},                                                                         // 638
		{EventNewHellDungeonRoomShrineObject, "NewHellDungeonRoomShrineObject"},                                                             // 639
		{EventHellDungeonRoomShrineStateUpdate, "HellDungeonRoomShrineStateUpdate"},                                                         // 640
		{EventSimpleBehaviourBuildingStateUpdate, "SimpleBehaviourBuildingStateUpdate"},                                                     // 641
		{EventSetTimeScaling, "SetTimeScaling"},                                                                                             // 642
		{EventStopTimeScaling, "StopTimeScaling"},                                                                                           // 643
		{EventKeyValidation, "KeyValidation"},                                                                                               // 644
		{EventPlayerJoinMapMarkerTimerStates, "PlayerJoinMapMarkerTimerStates"},                                                             // 645
		{EventNewMapMarkerTimer, "NewMapMarkerTimer"},                                                                                       // 646
		{EventRemoveMapMarkerTimer, "RemoveMapMarkerTimer"},                                                                                 // 647
		{EventNewFactionFortressObject, "NewFactionFortressObject"},                                                                         // 648
		{EventFactionFortressAnnouncePlayerEjection, "FactionFortressAnnouncePlayerEjection"},                                               // 649
		{EventRewardFactionWarfareSupply, "RewardFactionWarfareSupply"},                                                                     // 650
		{EventFactionCaptureAreaProgressUpdate, "FactionCaptureAreaProgressUpdate"},                                                         // 651
		{EventFactionFortressClaimed, "FactionFortressClaimed"},                                                                             // 652
		{EventFactionFortressWeaponCachesSpawned, "FactionFortressWeaponCachesSpawned"},                                                     // 653
		{EventFactionFortressWeaponCacheClaimed, "FactionFortressWeaponCacheClaimed"},                                                       // 654
		{EventFactionFortressFightStateUpdate, "FactionFortressFightStateUpdate"},                                                           // 655
		{EventFactionFortressCutoffFightStateUpdate, "FactionFortressCutoffFightStateUpdate"},                                               // 656
		{EventFactionFortressFightEnded, "FactionFortressFightEnded"},                                                                       // 657
		{EventNewFactionWarfarePortal, "NewFactionWarfarePortal"},                                                                           // 658
		{EventFactionPortalTargetUpdate, "FactionPortalTargetUpdate"},                                                                       // 659
		{EventFactionFortressFightStartedInRemoteClusterEvent, "FactionFortressFightStartedInRemoteClusterEvent"},                           // 660
		{EventFactionFortressFightFinishedInRemoteClusterEvent, "FactionFortressFightFinishedInRemoteClusterEvent"},                         // 661
		{EventFactionDuchySupplyWarDefensiveVictoryEvent, "FactionDuchySupplyWarDefensiveVictoryEvent"},                                     // 662
		{EventFactionDuchyReconnectedFromCutoffEvent, "FactionDuchyReconnectedFromCutoffEvent"},                                             // 663
		{EventFactionFortressCutoffFightCancelledByClusterOwnerChangeEvent, "FactionFortressCutoffFightCancelledByClusterOwnerChangeEvent"}, // 664
		{EventFactionDuchyEnteredCutoffStateEvent, "FactionDuchyEnteredCutoffStateEvent"},                                                   // 665
		{EventLeaveProtectionStateUpdate, "LeaveProtectionStateUpdate"},                                                                     // 666
		{EventRedZoneEventStandings, "RedZoneEventStandings"},                                                                               // 667
		{EventNewFactionBattleStandardDeliveryStation, "NewFactionBattleStandardDeliveryStation"},                                           // 668
		{EventNewLoreSnippetObject, "NewLoreSnippetObject"},                                                                                 // 669
		{EventLoreSnippetObjectStateUpdate, "LoreSnippetObjectStateUpdate"},                                                                 // 670
		{EventLoreSnippedClaimed, "LoreSnippedClaimed"},                                                                                     // 671
		{EventLoreSnippetStatesChangedByCheat, "LoreSnippetStatesChangedByCheat"},                                                           // 672
		{EventNewTeleporterNode, "NewTeleporterNode"},                                                                                       // 673
		{EventTeleporterNodeStateChanged, "TeleporterNodeStateChanged"},                                                                     // 674
		{EventTeleporterConnectionsFullStateUpdate, "TeleporterConnectionsFullStateUpdate"},                                                 // 675
		{EventTeleporterConnectionStateChanged, "TeleporterConnectionStateChanged"},                                                         // 676
		{EventRetrieveCarriableObjectStart, "RetrieveCarriableObjectStart"},                                                                 // 677
		{EventRetrieveCarriableObjectCancel, "RetrieveCarriableObjectCancel"},                                                               // 678
		{EventRetrieveCarriableObjectReset, "RetrieveCarriableObjectReset"},                                                                 // 679
		{EventRetrieveCarriableObjectFinished, "RetrieveCarriableObjectFinished"},                                                           // 680
		{EventLosingCarriableObjectStart, "LosingCarriableObjectStart"},                                                                     // 681
		{EventLosingCarriableObjectFinished, "LosingCarriableObjectFinished"},                                                               // 682
	}

	expected := int(EventLosingCarriableObjectFinished) + 1
	if len(golden) != expected {
		t.Fatalf("golden list has %d entries, expected %d", len(golden), expected)
	}

	for i, g := range golden {
		if int(g.code) != i {
			t.Errorf("golden[%d]: %s has value %d, expected %d (position drift in const block)",
				i, g.name, int(g.code), i)
		}
		if got := g.code.String(); got != g.name {
			t.Errorf("golden[%d]: String() = %q, expected %q (typo in EventCodeNames map)",
				i, got, g.name)
		}
	}
}

// TestEventCodeNamesCompleteness verifies that every Event* constant in the
// iota block has a corresponding entry in EventCodeNames. This catches
// copy-paste mistakes where a new const is added but the map is not updated.
func TestEventCodeNamesCompleteness(t *testing.T) {
	// Enumerate all EventCode values from EventUnused (0) to the last known
	// constant. Every value in [0, maxCode] should either map to a name or
	// be explicitly absent — but since we use iota with no gaps, every value
	// in range should have a name.
	maxCode := int(EventLosingCarriableObjectFinished)
	if got := len(EventCodeNames); got != maxCode+1 {
		t.Errorf("len(EventCodeNames) = %d, want %d (every iota value 0..%d should have a name)",
			got, maxCode+1, maxCode)
	}

	// Verify no EventCode in range resolves to "Unknown(...)"
	for i := 0; i <= maxCode; i++ {
		name := EventCode(i).String()
		if strings.HasPrefix(name, "Unknown(") {
			t.Errorf("EventCode(%d).String() = %q — missing entry in EventCodeNames", i, name)
		}
	}
}

// TestRedZoneEventClusterStatusRemoved verifies that the removed event code
// is no longer present in the names map (it was deprecated upstream).
func TestRedZoneEventClusterStatusRemoved(t *testing.T) {
	for _, name := range EventCodeNames {
		if name == "RedZoneEventClusterStatus" {
			t.Error("RedZoneEventClusterStatus should have been removed from EventCodeNames")
		}
	}
}

// Package events contains event code definitions and event structures for Albion Online.
//
// Event codes are defined by the Albion Online game protocol (Photon Engine).
// These values are extracted from network traffic analysis and are consistent
// across all client implementations that interact with the game servers.
package events

import "fmt"

// EventCode represents the event type for Albion Online network packets
type EventCode int16

// String returns the name of the event code
func (e EventCode) String() string {
	if name, ok := EventCodeNames[e]; ok {
		return name
	}
	return fmt.Sprintf("Unknown(%d)", e)
}

// Event codes from Albion Online
const (
	EventUnused EventCode = iota
	EventLeave
	EventJoinFinished
	EventMove
	EventTeleport
	EventChangeEquipment
	EventHealthUpdate
	EventHealthUpdates
	EventEnergyUpdate
	EventDamageShieldUpdate
	EventCraftingFocusUpdate
	EventActiveSpellEffectsUpdate
	EventResetCooldowns
	EventAttack
	EventCastStart
	EventChannelingUpdate
	EventCastCancel
	EventCastTimeUpdate
	EventCastFinished
	EventCastSpell
	EventCastSpells
	EventCastHit
	EventCastHits
	EventStoredTargetsUpdate
	EventChannelingEnded
	EventAttackBuilding
	EventInventoryPutItem
	EventInventoryDeleteItem
	EventInventoryState
	EventNewCharacter
	EventNewEquipmentItem
	EventNewSiegeBannerItem
	EventNewSimpleItem
	EventNewFurnitureItem
	EventNewKillTrophyItem
	EventNewJournalItem
	EventNewLaborerItem
	EventNewEquipmentItemLegendarySoul
	EventNewSimpleHarvestableObject
	EventNewSimpleHarvestableObjectList
	EventNewHarvestableObject
	EventNewTreasureDestinationObject
	EventTreasureDestinationObjectStatus
	EventCloseTreasureDestinationObject
	EventNewSilverObject
	EventNewBuilding
	EventHarvestableChangeState
	EventMobChangeState
	EventFactionBuildingInfo
	EventCraftBuildingInfo
	EventRepairBuildingInfo
	EventMeldBuildingInfo
	EventConstructionSiteInfo
	EventPlayerBuildingInfo
	EventFarmBuildingInfo
	EventTutorialBuildingInfo
	EventLaborerObjectInfo
	EventLaborerObjectJobInfo
	EventMarketPlaceBuildingInfo
	EventHarvestStart
	EventHarvestCancel
	EventHarvestFinished
	EventTakeSilver
	EventRemoveSilver
	EventActionOnBuildingStart
	EventActionOnBuildingCancel
	EventActionOnBuildingFinished
	EventItemRerollQualityFinished
	EventInstallResourceStart
	EventInstallResourceCancel
	EventInstallResourceFinished
	EventCraftItemFinished
	EventLogoutCancel
	EventChatMessage
	EventChatSay
	EventChatWhisper
	EventChatMuted
	EventPlayEmote
	EventStopEmote
	EventSystemMessage
	EventUtilityTextMessage
	EventUpdateMoney
	EventUpdateFame
	EventUpdateLearningPoints
	EventUpdateReSpecPoints
	EventUpdateCurrency
	EventUpdateFactionStanding
	EventUpdateStanding
	EventRespawn
	EventServerDebugLog
	EventCharacterEquipmentChanged
	EventRegenerationHealthChanged
	EventRegenerationEnergyChanged
	EventRegenerationMountHealthChanged
	EventRegenerationCraftingChanged
	EventRegenerationHealthEnergyComboChanged
	EventRegenerationPlayerComboChanged
	EventDurabilityChanged
	EventNewLoot
	EventAttachItemContainer
	EventDetachItemContainer
	EventInvalidateItemContainer
	EventLockItemContainer
	EventGuildUpdate
	EventGuildPlayerUpdated
	EventInvitedToGuild
	EventGuildMemberWorldUpdate
	EventUpdateMatchDetails
	EventObjectEvent
	EventNewMonolithObject
	EventMonolithHasBannersPlacedUpdate
	EventNewOrbObject
	EventNewCastleObject
	EventNewSpellEffectArea
	EventUpdateSpellEffectArea
	EventNewChainSpell
	EventUpdateChainSpell
	EventNewTreasureChest
	EventStartMatch
	EventStartArenaMatchInfos
	EventEndArenaMatch
	EventMatchUpdate
	EventActiveMatchUpdate
	EventNewMob
	EventDebugAggroInfo
	EventDebugVariablesInfo
	EventDebugReputationInfo
	EventDebugDiminishingReturnInfo
	EventDebugSmartClusterQueueInfo
	EventClaimOrbStart
	EventClaimOrbFinished
	EventClaimOrbCancel
	EventOrbUpdate
	EventOrbClaimed
	EventOrbReset
	EventNewWarCampObject
	EventNewMatchLootChestObject
	EventNewArenaExit
	EventGuildMemberTerritoryUpdate
	EventInvitedMercenaryToMatch
	EventClusterInfoUpdate
	EventForcedMovement
	EventForcedMovementCancel
	EventCharacterStats
	EventCharacterStatsKillHistory
	EventCharacterStatsDeathHistory
	EventCharacterStatsKnockDownHistory
	EventCharacterStatsKnockedDownHistory
	EventGuildStats
	EventKillHistoryDetails
	EventItemKillHistoryDetails
	EventFullAchievementInfo
	EventFinishedAchievement
	EventAchievementProgressInfo
	EventFullAchievementProgressInfo
	EventFullTrackedAchievementInfo
	EventFullAutoLearnAchievementInfo
	EventQuestGiverQuestOffered
	EventQuestGiverDebugInfo
	EventConsoleEvent
	EventTimeSync
	EventChangeAvatar
	EventChangeMountSkin
	EventGameEvent
	EventKilledPlayer
	EventDied
	EventKnockedDown
	EventUnconcious
	EventMatchPlayerJoinedEvent
	EventMatchPlayerStatsEvent
	EventMatchPlayerStatsCompleteEvent
	EventMatchTimeLineEventEvent
	EventMatchPlayerMainGearStatsEvent
	EventMatchPlayerChangedAvatarEvent
	EventInvitationPlayerTrade
	EventPlayerTradeStart
	EventPlayerTradeCancel
	EventPlayerTradeUpdate
	EventPlayerTradeFinished
	EventPlayerTradeAcceptChange
	EventMiniMapPing
	EventMarketPlaceNotification
	EventDuellingChallengePlayer
	EventNewDuellingPost
	EventDuelStarted
	EventDuelEnded
	EventDuelDenied
	EventDuelRequestCanceled
	EventDuelLeftArea
	EventDuelReEnteredArea
	EventNewRealEstate
	EventMiniMapOwnedBuildingsPositions
	EventRealEstateListUpdate
	EventGuildLogoUpdate
	EventGuildLogoChanged
	EventPlaceableObjectPlace
	EventPlaceableObjectPlaceCancel
	EventFurnitureObjectBuffProviderInfo
	EventFurnitureObjectCheatProviderInfo
	EventFarmableObjectInfo
	EventNewUnreadMails
	EventMailOperationPossible
	EventGuildLogoObjectUpdate
	EventStartLogout
	EventNewChatChannels
	EventJoinedChatChannel
	EventLeftChatChannel
	EventRemovedChatChannel
	EventAccessStatus
	EventMounted
	EventMountStart
	EventMountCancel
	EventNewTravelpoint
	EventNewIslandAccessPoint
	EventNewExit
	EventUpdateHome
	EventUpdateChatSettings
	EventResurrectionOffer
	EventResurrectionReply
	EventLootEquipmentChanged
	EventUpdateUnlockedGuildLogos
	EventUpdateUnlockedAvatars
	EventUpdateUnlockedAvatarRings
	EventUpdateUnlockedBuildings
	EventNewIslandManagement
	EventNewTeleportStone
	EventCloak
	EventPartyInvitation
	EventPartyJoinRequest
	EventPartyJoined
	EventPartyDisbanded
	EventPartyPlayerJoined
	EventPartyChangedOrder
	EventPartyPlayerLeft
	EventPartyLeaderChanged
	EventPartyLootSettingChangedPlayer
	EventPartySilverGained
	EventPartyPlayerUpdated
	EventPartyInvitationAnswer
	EventPartyJoinRequestAnswer
	EventPartyMarkedObjectsUpdated
	EventPartyOnClusterPartyJoined
	EventPartySetRoleFlag
	EventPartyInviteOrJoinPlayerEquipmentInfo
	EventPartyReadyCheckUpdate
	EventPartyFactionWarfareReinforcementSettingChangedPlayer
	EventSpellCooldownUpdate
	EventNewHellgateExitPortal
	EventNewExpeditionExit
	EventNewExpeditionNarrator
	EventExitEnterStart
	EventExitEnterCancel
	EventExitEnterFinished
	EventNewQuestGiverObject
	EventFullQuestInfo
	EventQuestProgressInfo
	EventQuestGiverInfoForPlayer
	EventFullExpeditionInfo
	EventExpeditionQuestProgressInfo
	EventInvitedToExpedition
	EventExpeditionRegistrationInfo
	EventEnteringExpeditionStart
	EventEnteringExpeditionCancel
	EventRewardGranted
	EventArenaRegistrationInfo
	EventEnteringArenaStart
	EventEnteringArenaCancel
	EventEnteringArenaLockStart
	EventEnteringArenaLockCancel
	EventInvitedToArenaMatch
	EventUsingHellgateShrine
	EventEnteringHellgateLockStart
	EventEnteringHellgateLockCancel
	EventPlayerCounts
	EventInCombatStateUpdate
	EventOtherGrabbedLoot
	EventTreasureChestUsingStart
	EventTreasureChestUsingFinished
	EventTreasureChestUsingCancel
	EventTreasureChestUsingOpeningComplete
	EventTreasureChestForceCloseInventory
	EventLocalTreasuresUpdate
	EventLootChestSpawnpointsUpdate
	EventPremiumChanged
	EventPremiumExtended
	EventPremiumLifeTimeRewardGained
	EventGoldPurchased
	EventLaborerGotUpgraded
	EventJournalGotFull
	EventJournalFillError
	EventFriendRequest
	EventFriendRequestInfos
	EventFriendInfos
	EventFriendRequestAnswered
	EventFriendOnlineStatus
	EventFriendRequestCanceled
	EventFriendRemoved
	EventFriendUpdated
	EventPartyLootItems
	EventPartyLootItemsRemoved
	EventPartyLootItemTypesRemoved
	EventReputationUpdate
	EventDefenseUnitAttackBegin
	EventDefenseUnitAttackEnd
	EventDefenseUnitAttackDamage
	EventUnrestrictedPvpZoneUpdate
	EventUnrestrictedPvpZoneStatus
	EventReputationImplicationUpdate
	EventNewMountObject
	EventMountHealthUpdate
	EventMountCooldownUpdate
	EventNewExpeditionAgent
	EventNewExpeditionCheckPoint
	EventExpeditionStartEvent
	EventVoteEvent
	EventRatingEvent
	EventNewArenaAgent
	EventBoostFarmable
	EventUseFunction
	EventNewPortalEntrance
	EventNewPortalExit
	EventNewRandomDungeonExit
	EventWaitingQueueUpdate
	EventPlayerMovementRateUpdate
	EventObserveStart
	EventMinimapZergs
	EventMinimapSmartClusterZergs
	EventPaymentTransactions
	EventPerformanceStatsUpdate
	EventOverloadModeUpdate
	EventDebugDrawEvent
	EventRecordCameraMove
	EventRecordStart
	EventClaimPowerCrystalStart
	EventClaimPowerCrystalCancel
	EventClaimPowerCrystalReset
	EventClaimPowerCrystalFinished
	EventTerritoryClaimStart
	EventTerritoryClaimCancel
	EventTerritoryClaimFinished
	EventTerritoryScheduleResult
	EventTerritoryUpgradeWithPowerCrystalResult
	EventReturningPowerCrystalStart
	EventReturningPowerCrystalFinished
	EventUpdateAccountState
	EventStartDeterministicRoam
	EventGuildFullAccessTagsUpdated
	EventGuildAccessTagUpdated
	EventGvgSeasonUpdate
	EventGvgSeasonCheatCommand
	EventSeasonPointsByKillingBooster
	EventFishingStart
	EventFishingCast
	EventFishingCatch
	EventFishingFinished
	EventFishingCancel
	EventNewFloatObject
	EventNewFishingZoneObject
	EventFishingMiniGame
	EventSteamAchievementCompleted
	EventUpdatePuppet
	EventChangeFlaggingFinished
	EventNewOutpostObject
	EventOutpostUpdate
	EventOutpostClaimed
	EventOverChargeEnd
	EventOverChargeStatus
	EventPartyFinderFullUpdate
	EventPartyFinderUpdate
	EventPartyFinderApplicantsUpdate
	EventPartyFinderEquipmentSnapshot
	EventPartyFinderJoinRequestDeclined
	EventNewUnlockedPersonalSeasonRewards
	EventPersonalSeasonPointsGained
	EventPersonalSeasonPastSeasonDataEvent
	EventMatchLootChestOpeningStart
	EventMatchLootChestOpeningFinished
	EventMatchLootChestOpeningCancel
	EventNotifyCrystalMatchReward
	EventCrystalRealmFeedback
	EventNewLocationMarker
	EventNewTutorialBlocker
	EventNewTileSwitch
	EventNewInformationProvider
	EventNewDynamicGuildLogo
	EventNewDecoration
	EventTutorialUpdate
	EventTriggerHintBox
	EventRandomDungeonPositionInfo
	EventNewLootChest
	EventUpdateLootChest
	EventLootChestOpened
	EventUpdateLootProtectedByMobsWithMinimapDisplay
	EventNewShrine
	EventUpdateShrine
	EventUpdateRoom
	EventNewMobSoul
	EventNewHellgateShrine
	EventUpdateHellgateShrine
	EventActivateHellgateExit
	EventMutePlayerUpdate
	EventShopTileUpdate
	EventShopUpdate
	EventEasyAntiCheatKick
	EventBattlEyeServerMessage
	EventUnlockVanityUnlock
	EventAvatarUnlocked
	EventCustomizationChanged
	EventBaseVaultInfo
	EventGuildVaultInfo
	EventBankVaultInfo
	EventRecoveryVaultPlayerInfo
	EventRecoveryVaultGuildInfo
	EventUpdateWardrobe
	EventCastlePhaseChanged
	EventGuildAccountLogEvent
	EventNewHideoutObject
	EventNewHideoutManagement
	EventNewHideoutExit
	EventInitHideoutAttackStart
	EventInitHideoutAttackCancel
	EventInitHideoutAttackFinished
	EventHideoutManagementUpdate
	EventHideoutUpgradeWithPowerCrystalResult
	EventIpChanged
	EventSmartClusterQueueUpdateInfo
	EventSmartClusterQueueActiveInfo
	EventSmartClusterQueueKickWarning
	EventSmartClusterQueueInvite
	EventReceivedGvgSeasonPoints
	EventTowerPowerPointUpdate
	EventOpenWorldAttackScheduleStart
	EventOpenWorldAttackScheduleFinished
	EventOpenWorldAttackScheduleCancel
	EventOpenWorldAttackConquerStart
	EventOpenWorldAttackConquerFinished
	EventOpenWorldAttackConquerCancel
	EventOpenWorldAttackConquerStatus
	EventOpenWorldAttackStart
	EventOpenWorldAttackEnd
	EventNewRandomResourceBlocker
	EventNewHomeObject
	EventHideoutObjectUpdate
	EventUpdateInfamy
	EventMinimapPositionMarkers
	EventNewTunnelExit
	EventCorruptedDungeonUpdate
	EventCorruptedDungeonStatus
	EventCorruptedDungeonInfamy
	EventHellgateRestrictedAreaUpdate
	EventHellgateInfamy
	EventHellgateStatus
	EventHellgateStatusUpdate
	EventHellgateSuspense
	EventReplaceSpellSlotWithMultiSpell
	EventNewCorruptedShrine
	EventUpdateCorruptedShrine
	EventCorruptedShrineUsageStart
	EventCorruptedShrineUsageCancel
	EventExitUsed
	EventLinkedToObject
	EventLinkToObjectBroken
	EventEstimatedMarketValueUpdate
	EventStuckCancel
	EventDungonEscapeReady
	EventFactionWarfareClusterState
	EventFactionWarfareHasUnclaimedWeeklyReportsEvent
	EventSimpleFeedback
	EventSmartClusterQueueSkipClusterError
	EventXignCodeEvent
	EventBatchUseItemStart
	EventBatchUseItemEnd
	EventRedZoneEventClusterStatus
	EventRedZonePlayerNotification
	EventRedZoneWorldEvent
	EventFactionWarfareStats
	EventUpdateFactionBalanceFactors
	EventFactionEnlistmentChanged
	EventUpdateFactionRank
	EventFactionWarfareCampaignRewardsUnlocked
	EventFeaturedFeatureUpdate
	EventNewPowerCrystalObject
	EventMinimapCrystalPositionMarker
	EventCarryPowerCrystalUpdate
	EventPickupPowerCrystalStart
	EventPickupPowerCrystalCancel
	EventPickupPowerCrystalFinished
	EventDoSimpleActionStart
	EventDoSimpleActionCancel
	EventDoSimpleActionFinished
	EventNotifyGuestAccountVerified
	EventMightAndFavorReceivedEvent
	EventWeeklyPvpChallengeRewardStateUpdate
	EventNewUnlockedPvpSeasonChallengeRewards
	EventStaticDungeonEntrancesDungeonEventStatusUpdates
	EventStaticDungeonDungeonValueUpdate
	EventStaticDungeonEntranceDungeonEventsAborted
	EventInAppPurchaseConfirmedGooglePlay
	EventFeatureSwitchInfo
	EventPartyJoinRequestAborted
	EventPartyInviteAborted
	EventPartyStartHuntRequest
	EventPartyStartHuntRequested
	EventPartyStartHuntRequestAnswer
	EventPartyPlayerLeaveScheduled
	EventGuildInviteDeclined
	EventCancelMultiSpellSlots
	EventNewVisualEventObject
	EventCastleClaimProgress
	EventCastleClaimProgressLogo
	EventTownPortalUpdateState
	EventTownPortalFailed
	EventConsumableVanityChargesAdded
	EventFestivitiesUpdate
	EventNewBannerObject
	EventNewMistsImmediateReturnExit
	EventMistsPlayerJoinedInfo
	EventNewMistsStaticEntrance
	EventNewMistsOpenWorldExit
	EventNewTunnelExitTemp
	EventNewMistsWispSpawn
	EventMistsWispSpawnStateChange
	EventNewMistsCityEntrance
	EventNewMistsCityRoadsEntrance
	EventMistsCityRoadsEntrancePartyStateUpdate
	EventMistsCityRoadsEntranceClearStateForParty
	EventMistsEntranceDataChanged
	EventNewCagedObject
	EventCagedObjectStateUpdated
	EventEntrancePartyBindingCreated
	EventEntrancePartyBindingCleared
	EventEntrancePartyBindingInfos
	EventNewMistsBorderExit
	EventNewMistsDungeonExit
	EventLocalQuestInfos
	EventLocalQuestStarted
	EventLocalQuestActive
	EventLocalQuestInactive
	EventLocalQuestProgressUpdate
	EventNewUnrestrictedPvpZone
	EventTemporaryFlaggingStatusUpdate
	EventSpellTestPerformanceUpdate
	EventTransformation
	EventTransformationEnd
	EventUpdateTrustlevel
	EventRevealHiddenTimeStamps
	EventModifyItemTraitFinished
	EventRerollItemTraitValueFinished
	EventHuntQuestProgressInfo
	EventHuntStarted
	EventHuntFinished
	EventHuntAborted
	EventHuntMissionStepStateUpdate
	EventNewHuntTrack
	EventHuntMissionUpdate
	EventHuntQuestMissionProgressUpdate
	EventHuntTrackUsed
	EventHuntTrackUseableAgain
	EventMinimapHuntTrackMarkers
	EventNoTracksFound
	EventHuntQuestAborted
	EventInteractWithTrackStart
	EventInteractWithTrackCancel
	EventInteractWithTrackFinished
	EventNewDynamicCompound
	EventLegendaryItemDestroyed
	EventAttunementInfo
	EventTerritoryClaimRaidedRawEnergyCrystalResult
	EventCarriedObjectExpiryWarning
	EventCarriedObjectExpired
	EventTerritoryRaidStart
	EventTerritoryRaidCancel
	EventTerritoryRaidFinished
	EventTerritoryRaidResult
	EventTerritoryMonolithActiveRaidStatus
	EventTerritoryMonolithActiveRaidCancelled
	EventMonolithEnergyStorageUpdate
	EventMonolithNextScheduledOpenWorldAttackUpdate
	EventMonolithProtectedBuildingsDamageReductionUpdate
	EventNewBuildingBaseEvent
	EventNewFortificationBuilding
	EventNewCastleGateBuilding
	EventBuildingDurabilityUpdate
	EventMonolithFortificationPointsUpdate
	EventFortificationBuildingUpgradeInfo
	EventFortificationBuildingsDamageStateUpdate
	EventSiegeNotificationEvent
	EventUpdateEnemyWarBannerActive
	EventTerritoryAnnouncePlayerEjection
	EventCastleGateSwitchUseStarted
	EventCastleGateSwitchUseFinished
	EventFortificationBuildingWillDowngrade
	EventBotCommand
	EventJournalAchievementProgressUpdate
	EventJournalClaimableRewardUpdate
	EventKeySync
	EventLocalQuestAreaGone
	EventDynamicTemplate
	EventDynamicTemplateForcedStateChange
	EventNewOutlandsTeleportationPortal
	EventNewOutlandsTeleportationReturnPortal
	EventOutlandsTeleportationBindingCleared
	EventOutlandsTeleportationReturnPortalUpdateEvent
	EventPlayerUsedOutlandsTeleportationPortal
	EventEncumberedRestricted
	EventNewPiledObject
	EventPiledObjectStateChanged
	EventNewSmugglerCrateDeliveryStation
	EventKillRewardedNoFame
	EventPickupFromPiledObjectStart
	EventPickupFromPiledObjectCancel
	EventPickupFromPiledObjectReset
	EventPickupFromPiledObjectFinished
	EventArmoryActivityChange
	EventNewKillTrophyFurnitureBuilding
	EventHellDungeonsPlayerJoinedInfo
	EventNewTileSwitchTrigger
	EventNewMultiRewardObject
	EventNewHellDungeonSoulShrineObject
	EventHellDungeonSoulShrineStateUpdate
	EventNewResurrectionShrine
	EventUpdateResurrectionShrine
	EventStandTimeFinished
	EventEpicAchievementAndStatsUpdate
	EventSpectateTargetAfterDeathUpdate
	EventSpectateTargetAfterDeathEnded
	EventNewHellDungeonUpwardExit
	EventNewHellDungeonSoulExit
	EventNewHellDungeonDownwardExit
	EventNewHellDungeonChestExit
	EventNewCorruptedStaticEntrance
	EventNewHellDungeonStaticEntrance
	EventUpdateHellDungeonStaticEntranceState
	EventDebugTriggerHellDungeonShutdownStart
	EventFullJournalQuestInfo
	EventJournalQuestProgressInfo
	EventNewHellDungeonRoomShrineObject
	EventHellDungeonRoomShrineStateUpdate
	EventSimpleBehaviourBuildingStateUpdate
	EventSetTimeScaling
	EventStopTimeScaling
	EventKeyValidation
	EventPlayerJoinMapMarkerTimerStates
	EventNewMapMarkerTimer
	EventRemoveMapMarkerTimer
	EventNewFactionFortressObject
	EventFactionFortressAnnouncePlayerEjection
	EventRewardFactionWarfareSupply
	EventFactionCaptureAreaProgressUpdate
	EventFactionFortressClaimed
	EventFactionFortressWeaponCachesSpawned
	EventFactionFortressWeaponCacheClaimed
	EventFactionFortressFightStateUpdate
	EventFactionFortressCutoffFightStateUpdate
	EventFactionFortressFightEnded
	EventNewFactionWarfarePortal
	EventFactionPortalTargetUpdate
	EventFactionFortressFightStartedInRemoteClusterEvent
	EventFactionFortressFightFinishedInRemoteClusterEvent
	EventFactionDuchySupplyWarDefensiveVictoryEvent
	EventFactionDuchyReconnectedFromCutoffEvent
	EventFactionFortressCutoffFightCancelledByClusterOwnerChangeEvent
)

// Special parameter keys
const (
	ParamEventCode     = 252 // 252 - Event code parameter key
	ParamOperationCode = 253 // 253 - Operation code parameter key
)

// EventCodeNames maps event codes to their string representation
var EventCodeNames = map[EventCode]string{
	EventUnused: "Unused",
	EventLeave: "Leave",
	EventJoinFinished: "JoinFinished",
	EventMove: "Move",
	EventTeleport: "Teleport",
	EventChangeEquipment: "ChangeEquipment",
	EventHealthUpdate: "HealthUpdate",
	EventHealthUpdates: "HealthUpdates",
	EventEnergyUpdate: "EnergyUpdate",
	EventDamageShieldUpdate: "DamageShieldUpdate",
	EventCraftingFocusUpdate: "CraftingFocusUpdate",
	EventActiveSpellEffectsUpdate: "ActiveSpellEffectsUpdate",
	EventResetCooldowns: "ResetCooldowns",
	EventAttack: "Attack",
	EventCastStart: "CastStart",
	EventChannelingUpdate: "ChannelingUpdate",
	EventCastCancel: "CastCancel",
	EventCastTimeUpdate: "CastTimeUpdate",
	EventCastFinished: "CastFinished",
	EventCastSpell: "CastSpell",
	EventCastSpells: "CastSpells",
	EventCastHit: "CastHit",
	EventCastHits: "CastHits",
	EventStoredTargetsUpdate: "StoredTargetsUpdate",
	EventChannelingEnded: "ChannelingEnded",
	EventAttackBuilding: "AttackBuilding",
	EventInventoryPutItem: "InventoryPutItem",
	EventInventoryDeleteItem: "InventoryDeleteItem",
	EventInventoryState: "InventoryState",
	EventNewCharacter: "NewCharacter",
	EventNewEquipmentItem: "NewEquipmentItem",
	EventNewSiegeBannerItem: "NewSiegeBannerItem",
	EventNewSimpleItem: "NewSimpleItem",
	EventNewFurnitureItem: "NewFurnitureItem",
	EventNewKillTrophyItem: "NewKillTrophyItem",
	EventNewJournalItem: "NewJournalItem",
	EventNewLaborerItem: "NewLaborerItem",
	EventNewEquipmentItemLegendarySoul: "NewEquipmentItemLegendarySoul",
	EventNewSimpleHarvestableObject: "NewSimpleHarvestableObject",
	EventNewSimpleHarvestableObjectList: "NewSimpleHarvestableObjectList",
	EventNewHarvestableObject: "NewHarvestableObject",
	EventNewTreasureDestinationObject: "NewTreasureDestinationObject",
	EventTreasureDestinationObjectStatus: "TreasureDestinationObjectStatus",
	EventCloseTreasureDestinationObject: "CloseTreasureDestinationObject",
	EventNewSilverObject: "NewSilverObject",
	EventNewBuilding: "NewBuilding",
	EventHarvestableChangeState: "HarvestableChangeState",
	EventMobChangeState: "MobChangeState",
	EventFactionBuildingInfo: "FactionBuildingInfo",
	EventCraftBuildingInfo: "CraftBuildingInfo",
	EventRepairBuildingInfo: "RepairBuildingInfo",
	EventMeldBuildingInfo: "MeldBuildingInfo",
	EventConstructionSiteInfo: "ConstructionSiteInfo",
	EventPlayerBuildingInfo: "PlayerBuildingInfo",
	EventFarmBuildingInfo: "FarmBuildingInfo",
	EventTutorialBuildingInfo: "TutorialBuildingInfo",
	EventLaborerObjectInfo: "LaborerObjectInfo",
	EventLaborerObjectJobInfo: "LaborerObjectJobInfo",
	EventMarketPlaceBuildingInfo: "MarketPlaceBuildingInfo",
	EventHarvestStart: "HarvestStart",
	EventHarvestCancel: "HarvestCancel",
	EventHarvestFinished: "HarvestFinished",
	EventTakeSilver: "TakeSilver",
	EventRemoveSilver: "RemoveSilver",
	EventActionOnBuildingStart: "ActionOnBuildingStart",
	EventActionOnBuildingCancel: "ActionOnBuildingCancel",
	EventActionOnBuildingFinished: "ActionOnBuildingFinished",
	EventItemRerollQualityFinished: "ItemRerollQualityFinished",
	EventInstallResourceStart: "InstallResourceStart",
	EventInstallResourceCancel: "InstallResourceCancel",
	EventInstallResourceFinished: "InstallResourceFinished",
	EventCraftItemFinished: "CraftItemFinished",
	EventLogoutCancel: "LogoutCancel",
	EventChatMessage: "ChatMessage",
	EventChatSay: "ChatSay",
	EventChatWhisper: "ChatWhisper",
	EventChatMuted: "ChatMuted",
	EventPlayEmote: "PlayEmote",
	EventStopEmote: "StopEmote",
	EventSystemMessage: "SystemMessage",
	EventUtilityTextMessage: "UtilityTextMessage",
	EventUpdateMoney: "UpdateMoney",
	EventUpdateFame: "UpdateFame",
	EventUpdateLearningPoints: "UpdateLearningPoints",
	EventUpdateReSpecPoints: "UpdateReSpecPoints",
	EventUpdateCurrency: "UpdateCurrency",
	EventUpdateFactionStanding: "UpdateFactionStanding",
	EventUpdateStanding: "UpdateStanding",
	EventRespawn: "Respawn",
	EventServerDebugLog: "ServerDebugLog",
	EventCharacterEquipmentChanged: "CharacterEquipmentChanged",
	EventRegenerationHealthChanged: "RegenerationHealthChanged",
	EventRegenerationEnergyChanged: "RegenerationEnergyChanged",
	EventRegenerationMountHealthChanged: "RegenerationMountHealthChanged",
	EventRegenerationCraftingChanged: "RegenerationCraftingChanged",
	EventRegenerationHealthEnergyComboChanged: "RegenerationHealthEnergyComboChanged",
	EventRegenerationPlayerComboChanged: "RegenerationPlayerComboChanged",
	EventDurabilityChanged: "DurabilityChanged",
	EventNewLoot: "NewLoot",
	EventAttachItemContainer: "AttachItemContainer",
	EventDetachItemContainer: "DetachItemContainer",
	EventInvalidateItemContainer: "InvalidateItemContainer",
	EventLockItemContainer: "LockItemContainer",
	EventGuildUpdate: "GuildUpdate",
	EventGuildPlayerUpdated: "GuildPlayerUpdated",
	EventInvitedToGuild: "InvitedToGuild",
	EventGuildMemberWorldUpdate: "GuildMemberWorldUpdate",
	EventUpdateMatchDetails: "UpdateMatchDetails",
	EventObjectEvent: "ObjectEvent",
	EventNewMonolithObject: "NewMonolithObject",
	EventMonolithHasBannersPlacedUpdate: "MonolithHasBannersPlacedUpdate",
	EventNewOrbObject: "NewOrbObject",
	EventNewCastleObject: "NewCastleObject",
	EventNewSpellEffectArea: "NewSpellEffectArea",
	EventUpdateSpellEffectArea: "UpdateSpellEffectArea",
	EventNewChainSpell: "NewChainSpell",
	EventUpdateChainSpell: "UpdateChainSpell",
	EventNewTreasureChest: "NewTreasureChest",
	EventStartMatch: "StartMatch",
	EventStartArenaMatchInfos: "StartArenaMatchInfos",
	EventEndArenaMatch: "EndArenaMatch",
	EventMatchUpdate: "MatchUpdate",
	EventActiveMatchUpdate: "ActiveMatchUpdate",
	EventNewMob: "NewMob",
	EventDebugAggroInfo: "DebugAggroInfo",
	EventDebugVariablesInfo: "DebugVariablesInfo",
	EventDebugReputationInfo: "DebugReputationInfo",
	EventDebugDiminishingReturnInfo: "DebugDiminishingReturnInfo",
	EventDebugSmartClusterQueueInfo: "DebugSmartClusterQueueInfo",
	EventClaimOrbStart: "ClaimOrbStart",
	EventClaimOrbFinished: "ClaimOrbFinished",
	EventClaimOrbCancel: "ClaimOrbCancel",
	EventOrbUpdate: "OrbUpdate",
	EventOrbClaimed: "OrbClaimed",
	EventOrbReset: "OrbReset",
	EventNewWarCampObject: "NewWarCampObject",
	EventNewMatchLootChestObject: "NewMatchLootChestObject",
	EventNewArenaExit: "NewArenaExit",
	EventGuildMemberTerritoryUpdate: "GuildMemberTerritoryUpdate",
	EventInvitedMercenaryToMatch: "InvitedMercenaryToMatch",
	EventClusterInfoUpdate: "ClusterInfoUpdate",
	EventForcedMovement: "ForcedMovement",
	EventForcedMovementCancel: "ForcedMovementCancel",
	EventCharacterStats: "CharacterStats",
	EventCharacterStatsKillHistory: "CharacterStatsKillHistory",
	EventCharacterStatsDeathHistory: "CharacterStatsDeathHistory",
	EventCharacterStatsKnockDownHistory: "CharacterStatsKnockDownHistory",
	EventCharacterStatsKnockedDownHistory: "CharacterStatsKnockedDownHistory",
	EventGuildStats: "GuildStats",
	EventKillHistoryDetails: "KillHistoryDetails",
	EventItemKillHistoryDetails: "ItemKillHistoryDetails",
	EventFullAchievementInfo: "FullAchievementInfo",
	EventFinishedAchievement: "FinishedAchievement",
	EventAchievementProgressInfo: "AchievementProgressInfo",
	EventFullAchievementProgressInfo: "FullAchievementProgressInfo",
	EventFullTrackedAchievementInfo: "FullTrackedAchievementInfo",
	EventFullAutoLearnAchievementInfo: "FullAutoLearnAchievementInfo",
	EventQuestGiverQuestOffered: "QuestGiverQuestOffered",
	EventQuestGiverDebugInfo: "QuestGiverDebugInfo",
	EventConsoleEvent: "ConsoleEvent",
	EventTimeSync: "TimeSync",
	EventChangeAvatar: "ChangeAvatar",
	EventChangeMountSkin: "ChangeMountSkin",
	EventGameEvent: "GameEvent",
	EventKilledPlayer: "KilledPlayer",
	EventDied: "Died",
	EventKnockedDown: "KnockedDown",
	EventUnconcious: "Unconcious",
	EventMatchPlayerJoinedEvent: "MatchPlayerJoinedEvent",
	EventMatchPlayerStatsEvent: "MatchPlayerStatsEvent",
	EventMatchPlayerStatsCompleteEvent: "MatchPlayerStatsCompleteEvent",
	EventMatchTimeLineEventEvent: "MatchTimeLineEventEvent",
	EventMatchPlayerMainGearStatsEvent: "MatchPlayerMainGearStatsEvent",
	EventMatchPlayerChangedAvatarEvent: "MatchPlayerChangedAvatarEvent",
	EventInvitationPlayerTrade: "InvitationPlayerTrade",
	EventPlayerTradeStart: "PlayerTradeStart",
	EventPlayerTradeCancel: "PlayerTradeCancel",
	EventPlayerTradeUpdate: "PlayerTradeUpdate",
	EventPlayerTradeFinished: "PlayerTradeFinished",
	EventPlayerTradeAcceptChange: "PlayerTradeAcceptChange",
	EventMiniMapPing: "MiniMapPing",
	EventMarketPlaceNotification: "MarketPlaceNotification",
	EventDuellingChallengePlayer: "DuellingChallengePlayer",
	EventNewDuellingPost: "NewDuellingPost",
	EventDuelStarted: "DuelStarted",
	EventDuelEnded: "DuelEnded",
	EventDuelDenied: "DuelDenied",
	EventDuelRequestCanceled: "DuelRequestCanceled",
	EventDuelLeftArea: "DuelLeftArea",
	EventDuelReEnteredArea: "DuelReEnteredArea",
	EventNewRealEstate: "NewRealEstate",
	EventMiniMapOwnedBuildingsPositions: "MiniMapOwnedBuildingsPositions",
	EventRealEstateListUpdate: "RealEstateListUpdate",
	EventGuildLogoUpdate: "GuildLogoUpdate",
	EventGuildLogoChanged: "GuildLogoChanged",
	EventPlaceableObjectPlace: "PlaceableObjectPlace",
	EventPlaceableObjectPlaceCancel: "PlaceableObjectPlaceCancel",
	EventFurnitureObjectBuffProviderInfo: "FurnitureObjectBuffProviderInfo",
	EventFurnitureObjectCheatProviderInfo: "FurnitureObjectCheatProviderInfo",
	EventFarmableObjectInfo: "FarmableObjectInfo",
	EventNewUnreadMails: "NewUnreadMails",
	EventMailOperationPossible: "MailOperationPossible",
	EventGuildLogoObjectUpdate: "GuildLogoObjectUpdate",
	EventStartLogout: "StartLogout",
	EventNewChatChannels: "NewChatChannels",
	EventJoinedChatChannel: "JoinedChatChannel",
	EventLeftChatChannel: "LeftChatChannel",
	EventRemovedChatChannel: "RemovedChatChannel",
	EventAccessStatus: "AccessStatus",
	EventMounted: "Mounted",
	EventMountStart: "MountStart",
	EventMountCancel: "MountCancel",
	EventNewTravelpoint: "NewTravelpoint",
	EventNewIslandAccessPoint: "NewIslandAccessPoint",
	EventNewExit: "NewExit",
	EventUpdateHome: "UpdateHome",
	EventUpdateChatSettings: "UpdateChatSettings",
	EventResurrectionOffer: "ResurrectionOffer",
	EventResurrectionReply: "ResurrectionReply",
	EventLootEquipmentChanged: "LootEquipmentChanged",
	EventUpdateUnlockedGuildLogos: "UpdateUnlockedGuildLogos",
	EventUpdateUnlockedAvatars: "UpdateUnlockedAvatars",
	EventUpdateUnlockedAvatarRings: "UpdateUnlockedAvatarRings",
	EventUpdateUnlockedBuildings: "UpdateUnlockedBuildings",
	EventNewIslandManagement: "NewIslandManagement",
	EventNewTeleportStone: "NewTeleportStone",
	EventCloak: "Cloak",
	EventPartyInvitation: "PartyInvitation",
	EventPartyJoinRequest: "PartyJoinRequest",
	EventPartyJoined: "PartyJoined",
	EventPartyDisbanded: "PartyDisbanded",
	EventPartyPlayerJoined: "PartyPlayerJoined",
	EventPartyChangedOrder: "PartyChangedOrder",
	EventPartyPlayerLeft: "PartyPlayerLeft",
	EventPartyLeaderChanged: "PartyLeaderChanged",
	EventPartyLootSettingChangedPlayer: "PartyLootSettingChangedPlayer",
	EventPartySilverGained: "PartySilverGained",
	EventPartyPlayerUpdated: "PartyPlayerUpdated",
	EventPartyInvitationAnswer: "PartyInvitationAnswer",
	EventPartyJoinRequestAnswer: "PartyJoinRequestAnswer",
	EventPartyMarkedObjectsUpdated: "PartyMarkedObjectsUpdated",
	EventPartyOnClusterPartyJoined: "PartyOnClusterPartyJoined",
	EventPartySetRoleFlag: "PartySetRoleFlag",
	EventPartyInviteOrJoinPlayerEquipmentInfo: "PartyInviteOrJoinPlayerEquipmentInfo",
	EventPartyReadyCheckUpdate: "PartyReadyCheckUpdate",
	EventPartyFactionWarfareReinforcementSettingChangedPlayer: "PartyFactionWarfareReinforcementSettingChangedPlayer",
	EventSpellCooldownUpdate: "SpellCooldownUpdate",
	EventNewHellgateExitPortal: "NewHellgateExitPortal",
	EventNewExpeditionExit: "NewExpeditionExit",
	EventNewExpeditionNarrator: "NewExpeditionNarrator",
	EventExitEnterStart: "ExitEnterStart",
	EventExitEnterCancel: "ExitEnterCancel",
	EventExitEnterFinished: "ExitEnterFinished",
	EventNewQuestGiverObject: "NewQuestGiverObject",
	EventFullQuestInfo: "FullQuestInfo",
	EventQuestProgressInfo: "QuestProgressInfo",
	EventQuestGiverInfoForPlayer: "QuestGiverInfoForPlayer",
	EventFullExpeditionInfo: "FullExpeditionInfo",
	EventExpeditionQuestProgressInfo: "ExpeditionQuestProgressInfo",
	EventInvitedToExpedition: "InvitedToExpedition",
	EventExpeditionRegistrationInfo: "ExpeditionRegistrationInfo",
	EventEnteringExpeditionStart: "EnteringExpeditionStart",
	EventEnteringExpeditionCancel: "EnteringExpeditionCancel",
	EventRewardGranted: "RewardGranted",
	EventArenaRegistrationInfo: "ArenaRegistrationInfo",
	EventEnteringArenaStart: "EnteringArenaStart",
	EventEnteringArenaCancel: "EnteringArenaCancel",
	EventEnteringArenaLockStart: "EnteringArenaLockStart",
	EventEnteringArenaLockCancel: "EnteringArenaLockCancel",
	EventInvitedToArenaMatch: "InvitedToArenaMatch",
	EventUsingHellgateShrine: "UsingHellgateShrine",
	EventEnteringHellgateLockStart: "EnteringHellgateLockStart",
	EventEnteringHellgateLockCancel: "EnteringHellgateLockCancel",
	EventPlayerCounts: "PlayerCounts",
	EventInCombatStateUpdate: "InCombatStateUpdate",
	EventOtherGrabbedLoot: "OtherGrabbedLoot",
	EventTreasureChestUsingStart: "TreasureChestUsingStart",
	EventTreasureChestUsingFinished: "TreasureChestUsingFinished",
	EventTreasureChestUsingCancel: "TreasureChestUsingCancel",
	EventTreasureChestUsingOpeningComplete: "TreasureChestUsingOpeningComplete",
	EventTreasureChestForceCloseInventory: "TreasureChestForceCloseInventory",
	EventLocalTreasuresUpdate: "LocalTreasuresUpdate",
	EventLootChestSpawnpointsUpdate: "LootChestSpawnpointsUpdate",
	EventPremiumChanged: "PremiumChanged",
	EventPremiumExtended: "PremiumExtended",
	EventPremiumLifeTimeRewardGained: "PremiumLifeTimeRewardGained",
	EventGoldPurchased: "GoldPurchased",
	EventLaborerGotUpgraded: "LaborerGotUpgraded",
	EventJournalGotFull: "JournalGotFull",
	EventJournalFillError: "JournalFillError",
	EventFriendRequest: "FriendRequest",
	EventFriendRequestInfos: "FriendRequestInfos",
	EventFriendInfos: "FriendInfos",
	EventFriendRequestAnswered: "FriendRequestAnswered",
	EventFriendOnlineStatus: "FriendOnlineStatus",
	EventFriendRequestCanceled: "FriendRequestCanceled",
	EventFriendRemoved: "FriendRemoved",
	EventFriendUpdated: "FriendUpdated",
	EventPartyLootItems: "PartyLootItems",
	EventPartyLootItemsRemoved: "PartyLootItemsRemoved",
	EventPartyLootItemTypesRemoved: "PartyLootItemTypesRemoved",
	EventReputationUpdate: "ReputationUpdate",
	EventDefenseUnitAttackBegin: "DefenseUnitAttackBegin",
	EventDefenseUnitAttackEnd: "DefenseUnitAttackEnd",
	EventDefenseUnitAttackDamage: "DefenseUnitAttackDamage",
	EventUnrestrictedPvpZoneUpdate: "UnrestrictedPvpZoneUpdate",
	EventUnrestrictedPvpZoneStatus: "UnrestrictedPvpZoneStatus",
	EventReputationImplicationUpdate: "ReputationImplicationUpdate",
	EventNewMountObject: "NewMountObject",
	EventMountHealthUpdate: "MountHealthUpdate",
	EventMountCooldownUpdate: "MountCooldownUpdate",
	EventNewExpeditionAgent: "NewExpeditionAgent",
	EventNewExpeditionCheckPoint: "NewExpeditionCheckPoint",
	EventExpeditionStartEvent: "ExpeditionStartEvent",
	EventVoteEvent: "VoteEvent",
	EventRatingEvent: "RatingEvent",
	EventNewArenaAgent: "NewArenaAgent",
	EventBoostFarmable: "BoostFarmable",
	EventUseFunction: "UseFunction",
	EventNewPortalEntrance: "NewPortalEntrance",
	EventNewPortalExit: "NewPortalExit",
	EventNewRandomDungeonExit: "NewRandomDungeonExit",
	EventWaitingQueueUpdate: "WaitingQueueUpdate",
	EventPlayerMovementRateUpdate: "PlayerMovementRateUpdate",
	EventObserveStart: "ObserveStart",
	EventMinimapZergs: "MinimapZergs",
	EventMinimapSmartClusterZergs: "MinimapSmartClusterZergs",
	EventPaymentTransactions: "PaymentTransactions",
	EventPerformanceStatsUpdate: "PerformanceStatsUpdate",
	EventOverloadModeUpdate: "OverloadModeUpdate",
	EventDebugDrawEvent: "DebugDrawEvent",
	EventRecordCameraMove: "RecordCameraMove",
	EventRecordStart: "RecordStart",
	EventClaimPowerCrystalStart: "ClaimPowerCrystalStart",
	EventClaimPowerCrystalCancel: "ClaimPowerCrystalCancel",
	EventClaimPowerCrystalReset: "ClaimPowerCrystalReset",
	EventClaimPowerCrystalFinished: "ClaimPowerCrystalFinished",
	EventTerritoryClaimStart: "TerritoryClaimStart",
	EventTerritoryClaimCancel: "TerritoryClaimCancel",
	EventTerritoryClaimFinished: "TerritoryClaimFinished",
	EventTerritoryScheduleResult: "TerritoryScheduleResult",
	EventTerritoryUpgradeWithPowerCrystalResult: "TerritoryUpgradeWithPowerCrystalResult",
	EventReturningPowerCrystalStart: "ReturningPowerCrystalStart",
	EventReturningPowerCrystalFinished: "ReturningPowerCrystalFinished",
	EventUpdateAccountState: "UpdateAccountState",
	EventStartDeterministicRoam: "StartDeterministicRoam",
	EventGuildFullAccessTagsUpdated: "GuildFullAccessTagsUpdated",
	EventGuildAccessTagUpdated: "GuildAccessTagUpdated",
	EventGvgSeasonUpdate: "GvgSeasonUpdate",
	EventGvgSeasonCheatCommand: "GvgSeasonCheatCommand",
	EventSeasonPointsByKillingBooster: "SeasonPointsByKillingBooster",
	EventFishingStart: "FishingStart",
	EventFishingCast: "FishingCast",
	EventFishingCatch: "FishingCatch",
	EventFishingFinished: "FishingFinished",
	EventFishingCancel: "FishingCancel",
	EventNewFloatObject: "NewFloatObject",
	EventNewFishingZoneObject: "NewFishingZoneObject",
	EventFishingMiniGame: "FishingMiniGame",
	EventSteamAchievementCompleted: "SteamAchievementCompleted",
	EventUpdatePuppet: "UpdatePuppet",
	EventChangeFlaggingFinished: "ChangeFlaggingFinished",
	EventNewOutpostObject: "NewOutpostObject",
	EventOutpostUpdate: "OutpostUpdate",
	EventOutpostClaimed: "OutpostClaimed",
	EventOverChargeEnd: "OverChargeEnd",
	EventOverChargeStatus: "OverChargeStatus",
	EventPartyFinderFullUpdate: "PartyFinderFullUpdate",
	EventPartyFinderUpdate: "PartyFinderUpdate",
	EventPartyFinderApplicantsUpdate: "PartyFinderApplicantsUpdate",
	EventPartyFinderEquipmentSnapshot: "PartyFinderEquipmentSnapshot",
	EventPartyFinderJoinRequestDeclined: "PartyFinderJoinRequestDeclined",
	EventNewUnlockedPersonalSeasonRewards: "NewUnlockedPersonalSeasonRewards",
	EventPersonalSeasonPointsGained: "PersonalSeasonPointsGained",
	EventPersonalSeasonPastSeasonDataEvent: "PersonalSeasonPastSeasonDataEvent",
	EventMatchLootChestOpeningStart: "MatchLootChestOpeningStart",
	EventMatchLootChestOpeningFinished: "MatchLootChestOpeningFinished",
	EventMatchLootChestOpeningCancel: "MatchLootChestOpeningCancel",
	EventNotifyCrystalMatchReward: "NotifyCrystalMatchReward",
	EventCrystalRealmFeedback: "CrystalRealmFeedback",
	EventNewLocationMarker: "NewLocationMarker",
	EventNewTutorialBlocker: "NewTutorialBlocker",
	EventNewTileSwitch: "NewTileSwitch",
	EventNewInformationProvider: "NewInformationProvider",
	EventNewDynamicGuildLogo: "NewDynamicGuildLogo",
	EventNewDecoration: "NewDecoration",
	EventTutorialUpdate: "TutorialUpdate",
	EventTriggerHintBox: "TriggerHintBox",
	EventRandomDungeonPositionInfo: "RandomDungeonPositionInfo",
	EventNewLootChest: "NewLootChest",
	EventUpdateLootChest: "UpdateLootChest",
	EventLootChestOpened: "LootChestOpened",
	EventUpdateLootProtectedByMobsWithMinimapDisplay: "UpdateLootProtectedByMobsWithMinimapDisplay",
	EventNewShrine: "NewShrine",
	EventUpdateShrine: "UpdateShrine",
	EventUpdateRoom: "UpdateRoom",
	EventNewMobSoul: "NewMobSoul",
	EventNewHellgateShrine: "NewHellgateShrine",
	EventUpdateHellgateShrine: "UpdateHellgateShrine",
	EventActivateHellgateExit: "ActivateHellgateExit",
	EventMutePlayerUpdate: "MutePlayerUpdate",
	EventShopTileUpdate: "ShopTileUpdate",
	EventShopUpdate: "ShopUpdate",
	EventEasyAntiCheatKick: "EasyAntiCheatKick",
	EventBattlEyeServerMessage: "BattlEyeServerMessage",
	EventUnlockVanityUnlock: "UnlockVanityUnlock",
	EventAvatarUnlocked: "AvatarUnlocked",
	EventCustomizationChanged: "CustomizationChanged",
	EventBaseVaultInfo: "BaseVaultInfo",
	EventGuildVaultInfo: "GuildVaultInfo",
	EventBankVaultInfo: "BankVaultInfo",
	EventRecoveryVaultPlayerInfo: "RecoveryVaultPlayerInfo",
	EventRecoveryVaultGuildInfo: "RecoveryVaultGuildInfo",
	EventUpdateWardrobe: "UpdateWardrobe",
	EventCastlePhaseChanged: "CastlePhaseChanged",
	EventGuildAccountLogEvent: "GuildAccountLogEvent",
	EventNewHideoutObject: "NewHideoutObject",
	EventNewHideoutManagement: "NewHideoutManagement",
	EventNewHideoutExit: "NewHideoutExit",
	EventInitHideoutAttackStart: "InitHideoutAttackStart",
	EventInitHideoutAttackCancel: "InitHideoutAttackCancel",
	EventInitHideoutAttackFinished: "InitHideoutAttackFinished",
	EventHideoutManagementUpdate: "HideoutManagementUpdate",
	EventHideoutUpgradeWithPowerCrystalResult: "HideoutUpgradeWithPowerCrystalResult",
	EventIpChanged: "IpChanged",
	EventSmartClusterQueueUpdateInfo: "SmartClusterQueueUpdateInfo",
	EventSmartClusterQueueActiveInfo: "SmartClusterQueueActiveInfo",
	EventSmartClusterQueueKickWarning: "SmartClusterQueueKickWarning",
	EventSmartClusterQueueInvite: "SmartClusterQueueInvite",
	EventReceivedGvgSeasonPoints: "ReceivedGvgSeasonPoints",
	EventTowerPowerPointUpdate: "TowerPowerPointUpdate",
	EventOpenWorldAttackScheduleStart: "OpenWorldAttackScheduleStart",
	EventOpenWorldAttackScheduleFinished: "OpenWorldAttackScheduleFinished",
	EventOpenWorldAttackScheduleCancel: "OpenWorldAttackScheduleCancel",
	EventOpenWorldAttackConquerStart: "OpenWorldAttackConquerStart",
	EventOpenWorldAttackConquerFinished: "OpenWorldAttackConquerFinished",
	EventOpenWorldAttackConquerCancel: "OpenWorldAttackConquerCancel",
	EventOpenWorldAttackConquerStatus: "OpenWorldAttackConquerStatus",
	EventOpenWorldAttackStart: "OpenWorldAttackStart",
	EventOpenWorldAttackEnd: "OpenWorldAttackEnd",
	EventNewRandomResourceBlocker: "NewRandomResourceBlocker",
	EventNewHomeObject: "NewHomeObject",
	EventHideoutObjectUpdate: "HideoutObjectUpdate",
	EventUpdateInfamy: "UpdateInfamy",
	EventMinimapPositionMarkers: "MinimapPositionMarkers",
	EventNewTunnelExit: "NewTunnelExit",
	EventCorruptedDungeonUpdate: "CorruptedDungeonUpdate",
	EventCorruptedDungeonStatus: "CorruptedDungeonStatus",
	EventCorruptedDungeonInfamy: "CorruptedDungeonInfamy",
	EventHellgateRestrictedAreaUpdate: "HellgateRestrictedAreaUpdate",
	EventHellgateInfamy: "HellgateInfamy",
	EventHellgateStatus: "HellgateStatus",
	EventHellgateStatusUpdate: "HellgateStatusUpdate",
	EventHellgateSuspense: "HellgateSuspense",
	EventReplaceSpellSlotWithMultiSpell: "ReplaceSpellSlotWithMultiSpell",
	EventNewCorruptedShrine: "NewCorruptedShrine",
	EventUpdateCorruptedShrine: "UpdateCorruptedShrine",
	EventCorruptedShrineUsageStart: "CorruptedShrineUsageStart",
	EventCorruptedShrineUsageCancel: "CorruptedShrineUsageCancel",
	EventExitUsed: "ExitUsed",
	EventLinkedToObject: "LinkedToObject",
	EventLinkToObjectBroken: "LinkToObjectBroken",
	EventEstimatedMarketValueUpdate: "EstimatedMarketValueUpdate",
	EventStuckCancel: "StuckCancel",
	EventDungonEscapeReady: "DungonEscapeReady",
	EventFactionWarfareClusterState: "FactionWarfareClusterState",
	EventFactionWarfareHasUnclaimedWeeklyReportsEvent: "FactionWarfareHasUnclaimedWeeklyReportsEvent",
	EventSimpleFeedback: "SimpleFeedback",
	EventSmartClusterQueueSkipClusterError: "SmartClusterQueueSkipClusterError",
	EventXignCodeEvent: "XignCodeEvent",
	EventBatchUseItemStart: "BatchUseItemStart",
	EventBatchUseItemEnd: "BatchUseItemEnd",
	EventRedZoneEventClusterStatus: "RedZoneEventClusterStatus",
	EventRedZonePlayerNotification: "RedZonePlayerNotification",
	EventRedZoneWorldEvent: "RedZoneWorldEvent",
	EventFactionWarfareStats: "FactionWarfareStats",
	EventUpdateFactionBalanceFactors: "UpdateFactionBalanceFactors",
	EventFactionEnlistmentChanged: "FactionEnlistmentChanged",
	EventUpdateFactionRank: "UpdateFactionRank",
	EventFactionWarfareCampaignRewardsUnlocked: "FactionWarfareCampaignRewardsUnlocked",
	EventFeaturedFeatureUpdate: "FeaturedFeatureUpdate",
	EventNewPowerCrystalObject: "NewPowerCrystalObject",
	EventMinimapCrystalPositionMarker: "MinimapCrystalPositionMarker",
	EventCarryPowerCrystalUpdate: "CarryPowerCrystalUpdate",
	EventPickupPowerCrystalStart: "PickupPowerCrystalStart",
	EventPickupPowerCrystalCancel: "PickupPowerCrystalCancel",
	EventPickupPowerCrystalFinished: "PickupPowerCrystalFinished",
	EventDoSimpleActionStart: "DoSimpleActionStart",
	EventDoSimpleActionCancel: "DoSimpleActionCancel",
	EventDoSimpleActionFinished: "DoSimpleActionFinished",
	EventNotifyGuestAccountVerified: "NotifyGuestAccountVerified",
	EventMightAndFavorReceivedEvent: "MightAndFavorReceivedEvent",
	EventWeeklyPvpChallengeRewardStateUpdate: "WeeklyPvpChallengeRewardStateUpdate",
	EventNewUnlockedPvpSeasonChallengeRewards: "NewUnlockedPvpSeasonChallengeRewards",
	EventStaticDungeonEntrancesDungeonEventStatusUpdates: "StaticDungeonEntrancesDungeonEventStatusUpdates",
	EventStaticDungeonDungeonValueUpdate: "StaticDungeonDungeonValueUpdate",
	EventStaticDungeonEntranceDungeonEventsAborted: "StaticDungeonEntranceDungeonEventsAborted",
	EventInAppPurchaseConfirmedGooglePlay: "InAppPurchaseConfirmedGooglePlay",
	EventFeatureSwitchInfo: "FeatureSwitchInfo",
	EventPartyJoinRequestAborted: "PartyJoinRequestAborted",
	EventPartyInviteAborted: "PartyInviteAborted",
	EventPartyStartHuntRequest: "PartyStartHuntRequest",
	EventPartyStartHuntRequested: "PartyStartHuntRequested",
	EventPartyStartHuntRequestAnswer: "PartyStartHuntRequestAnswer",
	EventPartyPlayerLeaveScheduled: "PartyPlayerLeaveScheduled",
	EventGuildInviteDeclined: "GuildInviteDeclined",
	EventCancelMultiSpellSlots: "CancelMultiSpellSlots",
	EventNewVisualEventObject: "NewVisualEventObject",
	EventCastleClaimProgress: "CastleClaimProgress",
	EventCastleClaimProgressLogo: "CastleClaimProgressLogo",
	EventTownPortalUpdateState: "TownPortalUpdateState",
	EventTownPortalFailed: "TownPortalFailed",
	EventConsumableVanityChargesAdded: "ConsumableVanityChargesAdded",
	EventFestivitiesUpdate: "FestivitiesUpdate",
	EventNewBannerObject: "NewBannerObject",
	EventNewMistsImmediateReturnExit: "NewMistsImmediateReturnExit",
	EventMistsPlayerJoinedInfo: "MistsPlayerJoinedInfo",
	EventNewMistsStaticEntrance: "NewMistsStaticEntrance",
	EventNewMistsOpenWorldExit: "NewMistsOpenWorldExit",
	EventNewTunnelExitTemp: "NewTunnelExitTemp",
	EventNewMistsWispSpawn: "NewMistsWispSpawn",
	EventMistsWispSpawnStateChange: "MistsWispSpawnStateChange",
	EventNewMistsCityEntrance: "NewMistsCityEntrance",
	EventNewMistsCityRoadsEntrance: "NewMistsCityRoadsEntrance",
	EventMistsCityRoadsEntrancePartyStateUpdate: "MistsCityRoadsEntrancePartyStateUpdate",
	EventMistsCityRoadsEntranceClearStateForParty: "MistsCityRoadsEntranceClearStateForParty",
	EventMistsEntranceDataChanged: "MistsEntranceDataChanged",
	EventNewCagedObject: "NewCagedObject",
	EventCagedObjectStateUpdated: "CagedObjectStateUpdated",
	EventEntrancePartyBindingCreated: "EntrancePartyBindingCreated",
	EventEntrancePartyBindingCleared: "EntrancePartyBindingCleared",
	EventEntrancePartyBindingInfos: "EntrancePartyBindingInfos",
	EventNewMistsBorderExit: "NewMistsBorderExit",
	EventNewMistsDungeonExit: "NewMistsDungeonExit",
	EventLocalQuestInfos: "LocalQuestInfos",
	EventLocalQuestStarted: "LocalQuestStarted",
	EventLocalQuestActive: "LocalQuestActive",
	EventLocalQuestInactive: "LocalQuestInactive",
	EventLocalQuestProgressUpdate: "LocalQuestProgressUpdate",
	EventNewUnrestrictedPvpZone: "NewUnrestrictedPvpZone",
	EventTemporaryFlaggingStatusUpdate: "TemporaryFlaggingStatusUpdate",
	EventSpellTestPerformanceUpdate: "SpellTestPerformanceUpdate",
	EventTransformation: "Transformation",
	EventTransformationEnd: "TransformationEnd",
	EventUpdateTrustlevel: "UpdateTrustlevel",
	EventRevealHiddenTimeStamps: "RevealHiddenTimeStamps",
	EventModifyItemTraitFinished: "ModifyItemTraitFinished",
	EventRerollItemTraitValueFinished: "RerollItemTraitValueFinished",
	EventHuntQuestProgressInfo: "HuntQuestProgressInfo",
	EventHuntStarted: "HuntStarted",
	EventHuntFinished: "HuntFinished",
	EventHuntAborted: "HuntAborted",
	EventHuntMissionStepStateUpdate: "HuntMissionStepStateUpdate",
	EventNewHuntTrack: "NewHuntTrack",
	EventHuntMissionUpdate: "HuntMissionUpdate",
	EventHuntQuestMissionProgressUpdate: "HuntQuestMissionProgressUpdate",
	EventHuntTrackUsed: "HuntTrackUsed",
	EventHuntTrackUseableAgain: "HuntTrackUseableAgain",
	EventMinimapHuntTrackMarkers: "MinimapHuntTrackMarkers",
	EventNoTracksFound: "NoTracksFound",
	EventHuntQuestAborted: "HuntQuestAborted",
	EventInteractWithTrackStart: "InteractWithTrackStart",
	EventInteractWithTrackCancel: "InteractWithTrackCancel",
	EventInteractWithTrackFinished: "InteractWithTrackFinished",
	EventNewDynamicCompound: "NewDynamicCompound",
	EventLegendaryItemDestroyed: "LegendaryItemDestroyed",
	EventAttunementInfo: "AttunementInfo",
	EventTerritoryClaimRaidedRawEnergyCrystalResult: "TerritoryClaimRaidedRawEnergyCrystalResult",
	EventCarriedObjectExpiryWarning: "CarriedObjectExpiryWarning",
	EventCarriedObjectExpired: "CarriedObjectExpired",
	EventTerritoryRaidStart: "TerritoryRaidStart",
	EventTerritoryRaidCancel: "TerritoryRaidCancel",
	EventTerritoryRaidFinished: "TerritoryRaidFinished",
	EventTerritoryRaidResult: "TerritoryRaidResult",
	EventTerritoryMonolithActiveRaidStatus: "TerritoryMonolithActiveRaidStatus",
	EventTerritoryMonolithActiveRaidCancelled: "TerritoryMonolithActiveRaidCancelled",
	EventMonolithEnergyStorageUpdate: "MonolithEnergyStorageUpdate",
	EventMonolithNextScheduledOpenWorldAttackUpdate: "MonolithNextScheduledOpenWorldAttackUpdate",
	EventMonolithProtectedBuildingsDamageReductionUpdate: "MonolithProtectedBuildingsDamageReductionUpdate",
	EventNewBuildingBaseEvent: "NewBuildingBaseEvent",
	EventNewFortificationBuilding: "NewFortificationBuilding",
	EventNewCastleGateBuilding: "NewCastleGateBuilding",
	EventBuildingDurabilityUpdate: "BuildingDurabilityUpdate",
	EventMonolithFortificationPointsUpdate: "MonolithFortificationPointsUpdate",
	EventFortificationBuildingUpgradeInfo: "FortificationBuildingUpgradeInfo",
	EventFortificationBuildingsDamageStateUpdate: "FortificationBuildingsDamageStateUpdate",
	EventSiegeNotificationEvent: "SiegeNotificationEvent",
	EventUpdateEnemyWarBannerActive: "UpdateEnemyWarBannerActive",
	EventTerritoryAnnouncePlayerEjection: "TerritoryAnnouncePlayerEjection",
	EventCastleGateSwitchUseStarted: "CastleGateSwitchUseStarted",
	EventCastleGateSwitchUseFinished: "CastleGateSwitchUseFinished",
	EventFortificationBuildingWillDowngrade: "FortificationBuildingWillDowngrade",
	EventBotCommand: "BotCommand",
	EventJournalAchievementProgressUpdate: "JournalAchievementProgressUpdate",
	EventJournalClaimableRewardUpdate: "JournalClaimableRewardUpdate",
	EventKeySync: "KeySync",
	EventLocalQuestAreaGone: "LocalQuestAreaGone",
	EventDynamicTemplate: "DynamicTemplate",
	EventDynamicTemplateForcedStateChange: "DynamicTemplateForcedStateChange",
	EventNewOutlandsTeleportationPortal: "NewOutlandsTeleportationPortal",
	EventNewOutlandsTeleportationReturnPortal: "NewOutlandsTeleportationReturnPortal",
	EventOutlandsTeleportationBindingCleared: "OutlandsTeleportationBindingCleared",
	EventOutlandsTeleportationReturnPortalUpdateEvent: "OutlandsTeleportationReturnPortalUpdateEvent",
	EventPlayerUsedOutlandsTeleportationPortal: "PlayerUsedOutlandsTeleportationPortal",
	EventEncumberedRestricted: "EncumberedRestricted",
	EventNewPiledObject: "NewPiledObject",
	EventPiledObjectStateChanged: "PiledObjectStateChanged",
	EventNewSmugglerCrateDeliveryStation: "NewSmugglerCrateDeliveryStation",
	EventKillRewardedNoFame: "KillRewardedNoFame",
	EventPickupFromPiledObjectStart: "PickupFromPiledObjectStart",
	EventPickupFromPiledObjectCancel: "PickupFromPiledObjectCancel",
	EventPickupFromPiledObjectReset: "PickupFromPiledObjectReset",
	EventPickupFromPiledObjectFinished: "PickupFromPiledObjectFinished",
	EventArmoryActivityChange: "ArmoryActivityChange",
	EventNewKillTrophyFurnitureBuilding: "NewKillTrophyFurnitureBuilding",
	EventHellDungeonsPlayerJoinedInfo: "HellDungeonsPlayerJoinedInfo",
	EventNewTileSwitchTrigger: "NewTileSwitchTrigger",
	EventNewMultiRewardObject: "NewMultiRewardObject",
	EventNewHellDungeonSoulShrineObject: "NewHellDungeonSoulShrineObject",
	EventHellDungeonSoulShrineStateUpdate: "HellDungeonSoulShrineStateUpdate",
	EventNewResurrectionShrine: "NewResurrectionShrine",
	EventUpdateResurrectionShrine: "UpdateResurrectionShrine",
	EventStandTimeFinished: "StandTimeFinished",
	EventEpicAchievementAndStatsUpdate: "EpicAchievementAndStatsUpdate",
	EventSpectateTargetAfterDeathUpdate: "SpectateTargetAfterDeathUpdate",
	EventSpectateTargetAfterDeathEnded: "SpectateTargetAfterDeathEnded",
	EventNewHellDungeonUpwardExit: "NewHellDungeonUpwardExit",
	EventNewHellDungeonSoulExit: "NewHellDungeonSoulExit",
	EventNewHellDungeonDownwardExit: "NewHellDungeonDownwardExit",
	EventNewHellDungeonChestExit: "NewHellDungeonChestExit",
	EventNewCorruptedStaticEntrance: "NewCorruptedStaticEntrance",
	EventNewHellDungeonStaticEntrance: "NewHellDungeonStaticEntrance",
	EventUpdateHellDungeonStaticEntranceState: "UpdateHellDungeonStaticEntranceState",
	EventDebugTriggerHellDungeonShutdownStart: "DebugTriggerHellDungeonShutdownStart",
	EventFullJournalQuestInfo: "FullJournalQuestInfo",
	EventJournalQuestProgressInfo: "JournalQuestProgressInfo",
	EventNewHellDungeonRoomShrineObject: "NewHellDungeonRoomShrineObject",
	EventHellDungeonRoomShrineStateUpdate: "HellDungeonRoomShrineStateUpdate",
	EventSimpleBehaviourBuildingStateUpdate: "SimpleBehaviourBuildingStateUpdate",
	EventSetTimeScaling: "SetTimeScaling",
	EventStopTimeScaling: "StopTimeScaling",
	EventKeyValidation: "KeyValidation",
	EventPlayerJoinMapMarkerTimerStates: "PlayerJoinMapMarkerTimerStates",
	EventNewMapMarkerTimer: "NewMapMarkerTimer",
	EventRemoveMapMarkerTimer: "RemoveMapMarkerTimer",
	EventNewFactionFortressObject: "NewFactionFortressObject",
	EventFactionFortressAnnouncePlayerEjection: "FactionFortressAnnouncePlayerEjection",
	EventRewardFactionWarfareSupply: "RewardFactionWarfareSupply",
	EventFactionCaptureAreaProgressUpdate: "FactionCaptureAreaProgressUpdate",
	EventFactionFortressClaimed: "FactionFortressClaimed",
	EventFactionFortressWeaponCachesSpawned: "FactionFortressWeaponCachesSpawned",
	EventFactionFortressWeaponCacheClaimed: "FactionFortressWeaponCacheClaimed",
	EventFactionFortressFightStateUpdate: "FactionFortressFightStateUpdate",
	EventFactionFortressCutoffFightStateUpdate: "FactionFortressCutoffFightStateUpdate",
	EventFactionFortressFightEnded: "FactionFortressFightEnded",
	EventNewFactionWarfarePortal: "NewFactionWarfarePortal",
	EventFactionPortalTargetUpdate: "FactionPortalTargetUpdate",
	EventFactionFortressFightStartedInRemoteClusterEvent: "FactionFortressFightStartedInRemoteClusterEvent",
	EventFactionFortressFightFinishedInRemoteClusterEvent: "FactionFortressFightFinishedInRemoteClusterEvent",
	EventFactionDuchySupplyWarDefensiveVictoryEvent: "FactionDuchySupplyWarDefensiveVictoryEvent",
	EventFactionDuchyReconnectedFromCutoffEvent: "FactionDuchyReconnectedFromCutoffEvent",
	EventFactionFortressCutoffFightCancelledByClusterOwnerChangeEvent: "FactionFortressCutoffFightCancelledByClusterOwnerChangeEvent",
}

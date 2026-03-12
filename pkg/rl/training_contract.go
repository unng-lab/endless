package rl

import "time"

// TrainingTransitionRecord documents the stable logical shape that external trainers are
// expected to read. ClickHouse exposes this contract through the generated transitions view,
// which aliases the internal step table columns into explicit obs/next_obs names.
type TrainingTransitionRecord struct {
	EpisodeID uint64 `ch:"episode_id" json:"episode_id"`
	Tick      uint32 `ch:"tick" json:"tick"`
	Scenario  string `ch:"scenario" json:"scenario"`
	Outcome   string `ch:"outcome" json:"outcome"`

	ObsPatchRadius                  int16   `ch:"obs_patch_radius" json:"obs_patch_radius"`
	ObsShooterX                     float32 `ch:"obs_shooter_x" json:"obs_shooter_x"`
	ObsShooterY                     float32 `ch:"obs_shooter_y" json:"obs_shooter_y"`
	ObsShooterHP                    int16   `ch:"obs_shooter_hp" json:"obs_shooter_hp"`
	ObsTargetX                      float32 `ch:"obs_target_x" json:"obs_target_x"`
	ObsTargetY                      float32 `ch:"obs_target_y" json:"obs_target_y"`
	ObsTargetHP                     int16   `ch:"obs_target_hp" json:"obs_target_hp"`
	ObsRelativeTargetX              float32 `ch:"obs_relative_target_x" json:"obs_relative_target_x"`
	ObsRelativeTargetY              float32 `ch:"obs_relative_target_y" json:"obs_relative_target_y"`
	ObsDistanceToTarget             float32 `ch:"obs_distance_to_target" json:"obs_distance_to_target"`
	ObsProjectileCount              uint16  `ch:"obs_projectile_count" json:"obs_projectile_count"`
	ObsShooterWeaponReady           uint8   `ch:"obs_shooter_weapon_ready" json:"obs_shooter_weapon_ready"`
	ObsShooterCooldownRemaining     uint16  `ch:"obs_shooter_cooldown_remaining" json:"obs_shooter_cooldown_remaining"`
	ObsShooterHasActiveFireOrder    uint8   `ch:"obs_shooter_has_active_fire_order" json:"obs_shooter_has_active_fire_order"`
	ObsShooterHasQueuedFireOrder    uint8   `ch:"obs_shooter_has_queued_fire_order" json:"obs_shooter_has_queued_fire_order"`
	ObsShooterHasActiveMoveOrder    uint8   `ch:"obs_shooter_has_active_move_order" json:"obs_shooter_has_active_move_order"`
	ObsShooterHasQueuedMoveOrder    uint8   `ch:"obs_shooter_has_queued_move_order" json:"obs_shooter_has_queued_move_order"`
	ObsShooterHasDestination        uint8   `ch:"obs_shooter_has_destination" json:"obs_shooter_has_destination"`
	ObsShooterDestinationX          float32 `ch:"obs_shooter_destination_x" json:"obs_shooter_destination_x"`
	ObsShooterDestinationY          float32 `ch:"obs_shooter_destination_y" json:"obs_shooter_destination_y"`
	ObsShooterDistanceToDestination float32 `ch:"obs_shooter_distance_to_destination" json:"obs_shooter_distance_to_destination"`
	ObsShooterRecentMoveFailure     uint8   `ch:"obs_shooter_recent_move_failure" json:"obs_shooter_recent_move_failure"`
	ObsLocalTerrainPatch            []int16 `ch:"obs_local_terrain_patch" json:"obs_local_terrain_patch"`
	ObsLocalOccupancyPatch          []int16 `ch:"obs_local_occupancy_patch" json:"obs_local_occupancy_patch"`
	ObsNearestFriendlyShotExists    uint8   `ch:"obs_nearest_friendly_shot_exists" json:"obs_nearest_friendly_shot_exists"`
	ObsNearestFriendlyShotX         float32 `ch:"obs_nearest_friendly_shot_x" json:"obs_nearest_friendly_shot_x"`
	ObsNearestFriendlyShotY         float32 `ch:"obs_nearest_friendly_shot_y" json:"obs_nearest_friendly_shot_y"`
	ObsNearestFriendlyShotDist      float32 `ch:"obs_nearest_friendly_shot_dist" json:"obs_nearest_friendly_shot_dist"`
	ObsNearestHostileShotExists     uint8   `ch:"obs_nearest_hostile_shot_exists" json:"obs_nearest_hostile_shot_exists"`
	ObsNearestHostileShotX          float32 `ch:"obs_nearest_hostile_shot_x" json:"obs_nearest_hostile_shot_x"`
	ObsNearestHostileShotY          float32 `ch:"obs_nearest_hostile_shot_y" json:"obs_nearest_hostile_shot_y"`
	ObsNearestHostileShotDist       float32 `ch:"obs_nearest_hostile_shot_dist" json:"obs_nearest_hostile_shot_dist"`

	ActionType        string  `ch:"action_type" json:"action_type"`
	ActionAccepted    uint8   `ch:"action_accepted" json:"action_accepted"`
	ActionMoveTargetX float32 `ch:"action_move_target_x" json:"action_move_target_x"`
	ActionMoveTargetY float32 `ch:"action_move_target_y" json:"action_move_target_y"`
	ActionDirX        float32 `ch:"action_dir_x" json:"action_dir_x"`
	ActionDirY        float32 `ch:"action_dir_y" json:"action_dir_y"`
	Reward            float32 `ch:"reward" json:"reward"`
	Done              uint8   `ch:"done" json:"done"`

	NextObsPatchRadius                  int16   `ch:"next_obs_patch_radius" json:"next_obs_patch_radius"`
	NextObsShooterX                     float32 `ch:"next_obs_shooter_x" json:"next_obs_shooter_x"`
	NextObsShooterY                     float32 `ch:"next_obs_shooter_y" json:"next_obs_shooter_y"`
	NextObsShooterHP                    int16   `ch:"next_obs_shooter_hp" json:"next_obs_shooter_hp"`
	NextObsTargetX                      float32 `ch:"next_obs_target_x" json:"next_obs_target_x"`
	NextObsTargetY                      float32 `ch:"next_obs_target_y" json:"next_obs_target_y"`
	NextObsTargetHP                     int16   `ch:"next_obs_target_hp" json:"next_obs_target_hp"`
	NextObsRelativeTargetX              float32 `ch:"next_obs_relative_target_x" json:"next_obs_relative_target_x"`
	NextObsRelativeTargetY              float32 `ch:"next_obs_relative_target_y" json:"next_obs_relative_target_y"`
	NextObsDistanceToTarget             float32 `ch:"next_obs_distance_to_target" json:"next_obs_distance_to_target"`
	NextObsProjectileCount              uint16  `ch:"next_obs_projectile_count" json:"next_obs_projectile_count"`
	NextObsShooterWeaponReady           uint8   `ch:"next_obs_shooter_weapon_ready" json:"next_obs_shooter_weapon_ready"`
	NextObsShooterCooldownRemaining     uint16  `ch:"next_obs_shooter_cooldown_remaining" json:"next_obs_shooter_cooldown_remaining"`
	NextObsShooterHasActiveFireOrder    uint8   `ch:"next_obs_shooter_has_active_fire_order" json:"next_obs_shooter_has_active_fire_order"`
	NextObsShooterHasQueuedFireOrder    uint8   `ch:"next_obs_shooter_has_queued_fire_order" json:"next_obs_shooter_has_queued_fire_order"`
	NextObsShooterHasActiveMoveOrder    uint8   `ch:"next_obs_shooter_has_active_move_order" json:"next_obs_shooter_has_active_move_order"`
	NextObsShooterHasQueuedMoveOrder    uint8   `ch:"next_obs_shooter_has_queued_move_order" json:"next_obs_shooter_has_queued_move_order"`
	NextObsShooterHasDestination        uint8   `ch:"next_obs_shooter_has_destination" json:"next_obs_shooter_has_destination"`
	NextObsShooterDestinationX          float32 `ch:"next_obs_shooter_destination_x" json:"next_obs_shooter_destination_x"`
	NextObsShooterDestinationY          float32 `ch:"next_obs_shooter_destination_y" json:"next_obs_shooter_destination_y"`
	NextObsShooterDistanceToDestination float32 `ch:"next_obs_shooter_distance_to_destination" json:"next_obs_shooter_distance_to_destination"`
	NextObsShooterRecentMoveFailure     uint8   `ch:"next_obs_shooter_recent_move_failure" json:"next_obs_shooter_recent_move_failure"`
	NextObsLocalTerrainPatch            []int16 `ch:"next_obs_local_terrain_patch" json:"next_obs_local_terrain_patch"`
	NextObsLocalOccupancyPatch          []int16 `ch:"next_obs_local_occupancy_patch" json:"next_obs_local_occupancy_patch"`
	NextObsNearestFriendlyShotExists    uint8   `ch:"next_obs_nearest_friendly_shot_exists" json:"next_obs_nearest_friendly_shot_exists"`
	NextObsNearestFriendlyShotX         float32 `ch:"next_obs_nearest_friendly_shot_x" json:"next_obs_nearest_friendly_shot_x"`
	NextObsNearestFriendlyShotY         float32 `ch:"next_obs_nearest_friendly_shot_y" json:"next_obs_nearest_friendly_shot_y"`
	NextObsNearestFriendlyShotDist      float32 `ch:"next_obs_nearest_friendly_shot_dist" json:"next_obs_nearest_friendly_shot_dist"`
	NextObsNearestHostileShotExists     uint8   `ch:"next_obs_nearest_hostile_shot_exists" json:"next_obs_nearest_hostile_shot_exists"`
	NextObsNearestHostileShotX          float32 `ch:"next_obs_nearest_hostile_shot_x" json:"next_obs_nearest_hostile_shot_x"`
	NextObsNearestHostileShotY          float32 `ch:"next_obs_nearest_hostile_shot_y" json:"next_obs_nearest_hostile_shot_y"`
	NextObsNearestHostileShotDist       float32 `ch:"next_obs_nearest_hostile_shot_dist" json:"next_obs_nearest_hostile_shot_dist"`

	CreatedAt time.Time `ch:"created_at" json:"created_at"`
}

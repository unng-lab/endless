package rl

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const (
	defaultBatchSize   = 512
	defaultTablePrefix = "endless_rl"
)

// ClickHouseConfig groups every connection and schema setting required by the RL dataset sink.
type ClickHouseConfig struct {
	Addr        []string
	Database    string
	Username    string
	Password    string
	TablePrefix string
	BatchSize   int
	DialTimeout time.Duration
}

// LoadClickHouseConfigFromEnv keeps secrets out of the repository while still giving the new
// headless launcher one deterministic place to read its dataset sink configuration from.
func LoadClickHouseConfigFromEnv(lookup func(string) string) ClickHouseConfig {
	if lookup == nil {
		lookup = func(string) string { return "" }
	}

	cfg := ClickHouseConfig{
		Addr:        splitAddrList(lookup("ENDLESS_CLICKHOUSE_ADDR")),
		Database:    strings.TrimSpace(lookup("ENDLESS_CLICKHOUSE_DATABASE")),
		Username:    strings.TrimSpace(lookup("ENDLESS_CLICKHOUSE_USERNAME")),
		Password:    lookup("ENDLESS_CLICKHOUSE_PASSWORD"),
		TablePrefix: strings.TrimSpace(lookup("ENDLESS_CLICKHOUSE_TABLE_PREFIX")),
		BatchSize:   parsePositiveInt(lookup("ENDLESS_CLICKHOUSE_BATCH_SIZE"), defaultBatchSize),
		DialTimeout: parseDuration(lookup("ENDLESS_CLICKHOUSE_DIAL_TIMEOUT"), 5*time.Second),
	}
	if cfg.Database == "" {
		cfg.Database = "default"
	}
	if cfg.Username == "" {
		cfg.Username = "default"
	}
	if cfg.TablePrefix == "" {
		cfg.TablePrefix = defaultTablePrefix
	}
	return cfg
}

// ClickHouseRecorder batches step, event and episode rows into MergeTree tables.
type ClickHouseRecorder struct {
	conn driver.Conn
	cfg  ClickHouseConfig

	mu       sync.Mutex
	steps    []StepRecord
	events   []EventRecord
	episodes []EpisodeRecord
}

// NewClickHouseRecorder establishes one connection, verifies it with Ping and ensures the
// required MergeTree tables exist before any training episode starts.
func NewClickHouseRecorder(ctx context.Context, cfg ClickHouseConfig) (*ClickHouseRecorder, error) {
	conn, cfg, err := openClickHouseConn(ctx, cfg)
	if err != nil {
		return nil, err
	}

	recorder := &ClickHouseRecorder{
		conn:     conn,
		cfg:      cfg,
		steps:    make([]StepRecord, 0, cfg.BatchSize),
		events:   make([]EventRecord, 0, cfg.BatchSize),
		episodes: make([]EpisodeRecord, 0, cfg.BatchSize),
	}
	if err := recorder.ensureSchema(ctx); err != nil {
		return nil, err
	}
	return recorder, nil
}

func (c *ClickHouseRecorder) RecordStep(ctx context.Context, step StepRecord) error {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.steps = append(c.steps, step)
	if len(c.steps) < c.cfg.BatchSize {
		return nil
	}

	return c.flushStepsLocked(ctx)
}

func (c *ClickHouseRecorder) RecordEvents(ctx context.Context, events []EventRecord) error {
	if c == nil || len(events) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.events = append(c.events, events...)
	if len(c.events) < c.cfg.BatchSize {
		return nil
	}

	return c.flushEventsLocked(ctx)
}

func (c *ClickHouseRecorder) RecordEpisode(ctx context.Context, episode EpisodeRecord) error {
	if c == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.episodes = append(c.episodes, episode)
	if len(c.episodes) < c.cfg.BatchSize {
		return nil
	}

	return c.flushEpisodesLocked(ctx)
}

// Close flushes every remaining buffered row category before returning control to the caller.
func (c *ClickHouseRecorder) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}

	var flushErr error
	c.mu.Lock()
	if err := c.flushEpisodesLocked(ctx); err != nil && flushErr == nil {
		flushErr = err
	}
	if err := c.flushEventsLocked(ctx); err != nil && flushErr == nil {
		flushErr = err
	}
	if err := c.flushStepsLocked(ctx); err != nil && flushErr == nil {
		flushErr = err
	}
	c.mu.Unlock()

	if c.conn != nil {
		if err := c.conn.Close(); err != nil && flushErr == nil {
			flushErr = fmt.Errorf("close clickhouse connection: %w", err)
		}
	}
	return flushErr
}

func (c *ClickHouseRecorder) ensureSchema(ctx context.Context) error {
	if c == nil {
		return nil
	}

	statements := []string{
		fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	episode_id UInt64,
	scenario LowCardinality(String),
	seed Int64,
	started_at DateTime64(3),
	ended_at DateTime64(3),
	ticks_total UInt32,
	outcome LowCardinality(String),
	total_reward Float32
) ENGINE = MergeTree
ORDER BY (episode_id)
`, c.episodesTable()),
		fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	episode_id UInt64,
	tick UInt32,
	shooter_id Int64,
	target_id Int64,
	obs_patch_radius Int16,
	obs_shooter_x Float32,
	obs_shooter_y Float32,
	obs_shooter_hp Int16,
	obs_target_x Float32,
	obs_target_y Float32,
	obs_target_hp Int16,
	obs_relative_target_x Float32,
	obs_relative_target_y Float32,
	obs_distance_to_target Float32,
	obs_projectile_count UInt16,
	obs_shooter_weapon_ready UInt8,
	obs_shooter_cooldown_remaining UInt16,
	obs_shooter_has_active_fire_order UInt8,
	obs_shooter_has_queued_fire_order UInt8,
	obs_shooter_has_active_move_order UInt8,
	obs_shooter_has_queued_move_order UInt8,
	obs_shooter_has_destination UInt8,
	obs_shooter_destination_x Float32,
	obs_shooter_destination_y Float32,
	obs_shooter_distance_to_destination Float32,
	obs_shooter_recent_move_failure UInt8,
	obs_local_terrain_patch Array(Int16),
	obs_local_occupancy_patch Array(Int16),
	obs_nearest_friendly_shot_exists UInt8,
	obs_nearest_friendly_shot_x Float32,
	obs_nearest_friendly_shot_y Float32,
	obs_nearest_friendly_shot_dist Float32,
	obs_nearest_hostile_shot_exists UInt8,
	obs_nearest_hostile_shot_x Float32,
	obs_nearest_hostile_shot_y Float32,
	obs_nearest_hostile_shot_dist Float32,
	patch_radius Int16,
	shooter_x Float32,
	shooter_y Float32,
	shooter_hp Int16,
	target_x Float32,
	target_y Float32,
	target_hp Int16,
	relative_target_x Float32,
	relative_target_y Float32,
	distance_to_target Float32,
	projectile_count UInt16,
	shooter_weapon_ready UInt8,
	shooter_cooldown_remaining UInt16,
	shooter_has_active_fire_order UInt8,
	shooter_has_queued_fire_order UInt8,
	shooter_has_active_move_order UInt8,
	shooter_has_queued_move_order UInt8,
	shooter_has_destination UInt8,
	shooter_destination_x Float32,
	shooter_destination_y Float32,
	shooter_distance_to_destination Float32,
	shooter_recent_move_failure UInt8,
	local_terrain_patch Array(Int16),
	local_occupancy_patch Array(Int16),
	nearest_friendly_shot_exists UInt8,
	nearest_friendly_shot_x Float32,
	nearest_friendly_shot_y Float32,
	nearest_friendly_shot_dist Float32,
	nearest_hostile_shot_exists UInt8,
	nearest_hostile_shot_x Float32,
	nearest_hostile_shot_y Float32,
	nearest_hostile_shot_dist Float32,
	action_type LowCardinality(String),
	action_accepted UInt8,
	action_move_target_x Float32,
	action_move_target_y Float32,
	action_dir_x Float32,
	action_dir_y Float32,
	reward Float32,
	done UInt8,
	created_at DateTime64(3)
) ENGINE = MergeTree
ORDER BY (episode_id, tick)
`, c.stepsTable()),
		fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	episode_id UInt64,
	tick UInt32,
	category LowCardinality(String),
	event_type LowCardinality(String),
	unit_id Int64,
	source_unit_id Int64,
	target_unit_id Int64,
	projectile_unit_id Int64,
	order_id Int64,
	order_kind LowCardinality(String),
	order_status LowCardinality(String),
	position_x Float32,
	position_y Float32,
	direction_x Float32,
	direction_y Float32,
	damage Int16,
	killed UInt8,
	created_at DateTime64(3)
) ENGINE = MergeTree
ORDER BY (episode_id, tick, category, event_type)
`, c.eventsTable()),
		c.transitionsViewStatement(),
	}

	for _, statement := range statements {
		if err := c.conn.Exec(ctx, statement); err != nil {
			return fmt.Errorf("ensure clickhouse schema: %w", err)
		}
	}
	if err := c.ensureStepColumns(ctx); err != nil {
		return err
	}
	return nil
}

// ensureStepColumns upgrades older step tables in place so the recorder may add richer RL
// transition fields without forcing callers to drop existing datasets manually.
func (c *ClickHouseRecorder) ensureStepColumns(ctx context.Context) error {
	if c == nil {
		return nil
	}

	statements := []string{
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_patch_radius Int16", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_x Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_y Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_hp Int16", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_target_x Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_target_y Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_target_hp Int16", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_relative_target_x Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_relative_target_y Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_distance_to_target Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_projectile_count UInt16", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_weapon_ready UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_cooldown_remaining UInt16", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_has_active_fire_order UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_has_queued_fire_order UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_has_active_move_order UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_has_queued_move_order UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_has_destination UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_destination_x Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_destination_y Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_distance_to_destination Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_shooter_recent_move_failure UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_local_terrain_patch Array(Int16)", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_local_occupancy_patch Array(Int16)", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_nearest_friendly_shot_exists UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_nearest_friendly_shot_x Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_nearest_friendly_shot_y Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_nearest_friendly_shot_dist Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_nearest_hostile_shot_exists UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_nearest_hostile_shot_x Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_nearest_hostile_shot_y Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS obs_nearest_hostile_shot_dist Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS patch_radius Int16", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS shooter_has_active_move_order UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS shooter_has_queued_move_order UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS shooter_has_destination UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS shooter_destination_x Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS shooter_destination_y Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS shooter_distance_to_destination Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS shooter_recent_move_failure UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS local_terrain_patch Array(Int16)", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS local_occupancy_patch Array(Int16)", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS nearest_friendly_shot_exists UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS nearest_friendly_shot_x Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS nearest_friendly_shot_y Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS nearest_friendly_shot_dist Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS nearest_hostile_shot_exists UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS nearest_hostile_shot_x Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS nearest_hostile_shot_y Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS nearest_hostile_shot_dist Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS action_accepted UInt8", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS action_move_target_x Float32", c.stepsTable()),
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS action_move_target_y Float32", c.stepsTable()),
	}

	for _, statement := range statements {
		if err := c.conn.Exec(ctx, statement); err != nil {
			return fmt.Errorf("ensure clickhouse step columns: %w", err)
		}
	}
	return nil
}

func (c *ClickHouseRecorder) flushStepsLocked(ctx context.Context) error {
	if len(c.steps) == 0 {
		return nil
	}

	pending := append([]StepRecord(nil), c.steps...)
	batch, err := c.conn.PrepareBatch(ctx, fmt.Sprintf(
		"INSERT INTO %s (episode_id, tick, shooter_id, target_id, obs_patch_radius, obs_shooter_x, obs_shooter_y, obs_shooter_hp, obs_target_x, obs_target_y, obs_target_hp, obs_relative_target_x, obs_relative_target_y, obs_distance_to_target, obs_projectile_count, obs_shooter_weapon_ready, obs_shooter_cooldown_remaining, obs_shooter_has_active_fire_order, obs_shooter_has_queued_fire_order, obs_shooter_has_active_move_order, obs_shooter_has_queued_move_order, obs_shooter_has_destination, obs_shooter_destination_x, obs_shooter_destination_y, obs_shooter_distance_to_destination, obs_shooter_recent_move_failure, obs_local_terrain_patch, obs_local_occupancy_patch, obs_nearest_friendly_shot_exists, obs_nearest_friendly_shot_x, obs_nearest_friendly_shot_y, obs_nearest_friendly_shot_dist, obs_nearest_hostile_shot_exists, obs_nearest_hostile_shot_x, obs_nearest_hostile_shot_y, obs_nearest_hostile_shot_dist, patch_radius, shooter_x, shooter_y, shooter_hp, target_x, target_y, target_hp, relative_target_x, relative_target_y, distance_to_target, projectile_count, shooter_weapon_ready, shooter_cooldown_remaining, shooter_has_active_fire_order, shooter_has_queued_fire_order, shooter_has_active_move_order, shooter_has_queued_move_order, shooter_has_destination, shooter_destination_x, shooter_destination_y, shooter_distance_to_destination, shooter_recent_move_failure, local_terrain_patch, local_occupancy_patch, nearest_friendly_shot_exists, nearest_friendly_shot_x, nearest_friendly_shot_y, nearest_friendly_shot_dist, nearest_hostile_shot_exists, nearest_hostile_shot_x, nearest_hostile_shot_y, nearest_hostile_shot_dist, action_type, action_accepted, action_move_target_x, action_move_target_y, action_dir_x, action_dir_y, reward, done, created_at)",
		c.stepsTable(),
	))
	if err != nil {
		return fmt.Errorf("prepare steps batch: %w", err)
	}

	for _, step := range pending {
		if err := batch.Append(
			step.EpisodeID,
			step.Tick,
			step.ShooterID,
			step.TargetID,
			step.ObsPatchRadius,
			step.ObsShooterX,
			step.ObsShooterY,
			step.ObsShooterHP,
			step.ObsTargetX,
			step.ObsTargetY,
			step.ObsTargetHP,
			step.ObsRelativeTargetX,
			step.ObsRelativeTargetY,
			step.ObsDistanceToTarget,
			step.ObsProjectileCount,
			step.ObsShooterWeaponReady,
			step.ObsShooterCooldownRemaining,
			step.ObsShooterHasActiveFireOrder,
			step.ObsShooterHasQueuedFireOrder,
			step.ObsShooterHasActiveMoveOrder,
			step.ObsShooterHasQueuedMoveOrder,
			step.ObsShooterHasDestination,
			step.ObsShooterDestinationX,
			step.ObsShooterDestinationY,
			step.ObsShooterDistanceToDestination,
			step.ObsShooterRecentMoveFailure,
			step.ObsLocalTerrainPatch,
			step.ObsLocalOccupancyPatch,
			step.ObsNearestFriendlyShotExists,
			step.ObsNearestFriendlyShotX,
			step.ObsNearestFriendlyShotY,
			step.ObsNearestFriendlyShotDist,
			step.ObsNearestHostileShotExists,
			step.ObsNearestHostileShotX,
			step.ObsNearestHostileShotY,
			step.ObsNearestHostileShotDist,
			step.PatchRadius,
			step.ShooterX,
			step.ShooterY,
			step.ShooterHP,
			step.TargetX,
			step.TargetY,
			step.TargetHP,
			step.RelativeTargetX,
			step.RelativeTargetY,
			step.DistanceToTarget,
			step.ProjectileCount,
			step.ShooterWeaponReady,
			step.ShooterCooldownRemaining,
			step.ShooterHasActiveFireOrder,
			step.ShooterHasQueuedFireOrder,
			step.ShooterHasActiveMoveOrder,
			step.ShooterHasQueuedMoveOrder,
			step.ShooterHasDestination,
			step.ShooterDestinationX,
			step.ShooterDestinationY,
			step.ShooterDistanceToDestination,
			step.ShooterRecentMoveFailure,
			step.LocalTerrainPatch,
			step.LocalOccupancyPatch,
			step.NearestFriendlyShotExists,
			step.NearestFriendlyShotX,
			step.NearestFriendlyShotY,
			step.NearestFriendlyShotDist,
			step.NearestHostileShotExists,
			step.NearestHostileShotX,
			step.NearestHostileShotY,
			step.NearestHostileShotDist,
			step.ActionType,
			step.ActionAccepted,
			step.ActionMoveTargetX,
			step.ActionMoveTargetY,
			step.ActionDirX,
			step.ActionDirY,
			step.Reward,
			step.Done,
			step.CreatedAt,
		); err != nil {
			return fmt.Errorf("append step row: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send steps batch: %w", err)
	}

	c.steps = c.steps[:0]
	return nil
}

func (c *ClickHouseRecorder) flushEventsLocked(ctx context.Context) error {
	if len(c.events) == 0 {
		return nil
	}

	pending := append([]EventRecord(nil), c.events...)
	batch, err := c.conn.PrepareBatch(ctx, fmt.Sprintf(
		"INSERT INTO %s (episode_id, tick, category, event_type, unit_id, source_unit_id, target_unit_id, projectile_unit_id, order_id, order_kind, order_status, position_x, position_y, direction_x, direction_y, damage, killed, created_at)",
		c.eventsTable(),
	))
	if err != nil {
		return fmt.Errorf("prepare events batch: %w", err)
	}

	for _, event := range pending {
		if err := batch.Append(
			event.EpisodeID,
			event.Tick,
			event.Category,
			event.EventType,
			event.UnitID,
			event.SourceUnitID,
			event.TargetUnitID,
			event.ProjectileUnitID,
			event.OrderID,
			event.OrderKind,
			event.OrderStatus,
			event.PositionX,
			event.PositionY,
			event.DirectionX,
			event.DirectionY,
			event.Damage,
			event.Killed,
			event.CreatedAt,
		); err != nil {
			return fmt.Errorf("append event row: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send events batch: %w", err)
	}

	c.events = c.events[:0]
	return nil
}

func (c *ClickHouseRecorder) flushEpisodesLocked(ctx context.Context) error {
	if len(c.episodes) == 0 {
		return nil
	}

	pending := append([]EpisodeRecord(nil), c.episodes...)
	batch, err := c.conn.PrepareBatch(ctx, fmt.Sprintf(
		"INSERT INTO %s (episode_id, scenario, seed, started_at, ended_at, ticks_total, outcome, total_reward)",
		c.episodesTable(),
	))
	if err != nil {
		return fmt.Errorf("prepare episodes batch: %w", err)
	}

	for _, episode := range pending {
		if err := batch.Append(
			episode.EpisodeID,
			episode.Scenario,
			episode.Seed,
			episode.StartedAt,
			episode.EndedAt,
			episode.TicksTotal,
			episode.Outcome,
			episode.TotalReward,
		); err != nil {
			return fmt.Errorf("append episode row: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send episodes batch: %w", err)
	}

	c.episodes = c.episodes[:0]
	return nil
}

func (c *ClickHouseRecorder) episodesTable() string {
	return fmt.Sprintf("%s.%s_episodes", c.cfg.Database, c.cfg.TablePrefix)
}

func (c *ClickHouseRecorder) stepsTable() string {
	return fmt.Sprintf("%s.%s_steps", c.cfg.Database, c.cfg.TablePrefix)
}

func (c *ClickHouseRecorder) eventsTable() string {
	return fmt.Sprintf("%s.%s_events", c.cfg.Database, c.cfg.TablePrefix)
}

func (c *ClickHouseRecorder) transitionsView() string {
	return fmt.Sprintf("%s.%s_transitions", c.cfg.Database, c.cfg.TablePrefix)
}

// transitionsViewStatement exposes one stable trainer-facing read contract with explicit
// obs_* and next_obs_* aliases. External training code should prefer this view over the raw
// step table so future internal column renames stay isolated inside the recorder.
func (c *ClickHouseRecorder) transitionsViewStatement() string {
	return fmt.Sprintf(`
CREATE VIEW IF NOT EXISTS %s AS
SELECT
	s.episode_id,
	s.tick,
	e.scenario,
	e.outcome,
	s.obs_patch_radius,
	s.obs_shooter_x,
	s.obs_shooter_y,
	s.obs_shooter_hp,
	s.obs_target_x,
	s.obs_target_y,
	s.obs_target_hp,
	s.obs_relative_target_x,
	s.obs_relative_target_y,
	s.obs_distance_to_target,
	s.obs_projectile_count,
	s.obs_shooter_weapon_ready,
	s.obs_shooter_cooldown_remaining,
	s.obs_shooter_has_active_fire_order,
	s.obs_shooter_has_queued_fire_order,
	s.obs_shooter_has_active_move_order,
	s.obs_shooter_has_queued_move_order,
	s.obs_shooter_has_destination,
	s.obs_shooter_destination_x,
	s.obs_shooter_destination_y,
	s.obs_shooter_distance_to_destination,
	s.obs_shooter_recent_move_failure,
	s.obs_local_terrain_patch,
	s.obs_local_occupancy_patch,
	s.obs_nearest_friendly_shot_exists,
	s.obs_nearest_friendly_shot_x,
	s.obs_nearest_friendly_shot_y,
	s.obs_nearest_friendly_shot_dist,
	s.obs_nearest_hostile_shot_exists,
	s.obs_nearest_hostile_shot_x,
	s.obs_nearest_hostile_shot_y,
	s.obs_nearest_hostile_shot_dist,
	s.action_type,
	s.action_accepted,
	s.action_move_target_x,
	s.action_move_target_y,
	s.action_dir_x,
	s.action_dir_y,
	s.reward,
	s.done,
	s.patch_radius AS next_obs_patch_radius,
	s.shooter_x AS next_obs_shooter_x,
	s.shooter_y AS next_obs_shooter_y,
	s.shooter_hp AS next_obs_shooter_hp,
	s.target_x AS next_obs_target_x,
	s.target_y AS next_obs_target_y,
	s.target_hp AS next_obs_target_hp,
	s.relative_target_x AS next_obs_relative_target_x,
	s.relative_target_y AS next_obs_relative_target_y,
	s.distance_to_target AS next_obs_distance_to_target,
	s.projectile_count AS next_obs_projectile_count,
	s.shooter_weapon_ready AS next_obs_shooter_weapon_ready,
	s.shooter_cooldown_remaining AS next_obs_shooter_cooldown_remaining,
	s.shooter_has_active_fire_order AS next_obs_shooter_has_active_fire_order,
	s.shooter_has_queued_fire_order AS next_obs_shooter_has_queued_fire_order,
	s.shooter_has_active_move_order AS next_obs_shooter_has_active_move_order,
	s.shooter_has_queued_move_order AS next_obs_shooter_has_queued_move_order,
	s.shooter_has_destination AS next_obs_shooter_has_destination,
	s.shooter_destination_x AS next_obs_shooter_destination_x,
	s.shooter_destination_y AS next_obs_shooter_destination_y,
	s.shooter_distance_to_destination AS next_obs_shooter_distance_to_destination,
	s.shooter_recent_move_failure AS next_obs_shooter_recent_move_failure,
	s.local_terrain_patch AS next_obs_local_terrain_patch,
	s.local_occupancy_patch AS next_obs_local_occupancy_patch,
	s.nearest_friendly_shot_exists AS next_obs_nearest_friendly_shot_exists,
	s.nearest_friendly_shot_x AS next_obs_nearest_friendly_shot_x,
	s.nearest_friendly_shot_y AS next_obs_nearest_friendly_shot_y,
	s.nearest_friendly_shot_dist AS next_obs_nearest_friendly_shot_dist,
	s.nearest_hostile_shot_exists AS next_obs_nearest_hostile_shot_exists,
	s.nearest_hostile_shot_x AS next_obs_nearest_hostile_shot_x,
	s.nearest_hostile_shot_y AS next_obs_nearest_hostile_shot_y,
	s.nearest_hostile_shot_dist AS next_obs_nearest_hostile_shot_dist,
	s.created_at
FROM %s AS s
LEFT JOIN %s AS e ON s.episode_id = e.episode_id
`, c.transitionsView(), c.stepsTable(), c.episodesTable())
}

func (c ClickHouseConfig) normalized() ClickHouseConfig {
	if c.Database == "" {
		c.Database = "default"
	}
	if c.Username == "" {
		c.Username = "default"
	}
	if c.TablePrefix == "" {
		c.TablePrefix = defaultTablePrefix
	}
	if c.BatchSize <= 0 {
		c.BatchSize = defaultBatchSize
	}
	if c.DialTimeout <= 0 {
		c.DialTimeout = 5 * time.Second
	}
	return c
}

// openClickHouseConn centralizes connection setup so recorders and trainer-facing readers use
// the same transport, authentication and validation rules.
func openClickHouseConn(ctx context.Context, cfg ClickHouseConfig) (driver.Conn, ClickHouseConfig, error) {
	cfg = cfg.normalized()
	if len(cfg.Addr) == 0 {
		return nil, ClickHouseConfig{}, fmt.Errorf("clickhouse addr is empty; set ENDLESS_CLICKHOUSE_ADDR")
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr:     cfg.Addr,
		Protocol: clickhouse.HTTP,
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		DialTimeout:  cfg.DialTimeout,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, ClickHouseConfig{}, fmt.Errorf("open clickhouse: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, ClickHouseConfig{}, fmt.Errorf("ping clickhouse: %w", err)
	}
	return conn, cfg, nil
}

func splitAddrList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	addrs := make([]string, 0, len(parts))
	for _, part := range parts {
		addr := strings.TrimSpace(part)
		if addr == "" {
			continue
		}
		addrs = append(addrs, addr)
	}
	return addrs
}

func parsePositiveInt(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseDuration(value string, fallback time.Duration) time.Duration {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

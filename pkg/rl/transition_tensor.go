package rl

import (
	"fmt"
	"math"
)

const (
	defaultObservationPositionScale = 4096
	defaultObservationDistanceScale = 4096
	defaultObservationHealthScale   = 3
	defaultProjectileCountScale     = 8
	defaultCooldownScale            = 10
	actionParameterFeatureCount     = 5
)

var (
	defaultTerrainVocabulary   = []int16{-1, 0, 1, 2, 3, 4}
	defaultOccupancyVocabulary = []int16{-1, 0, 1, 2, 3, 4, 5}
	defaultActionVocabulary    = []ActionType{ActionTypeNone, ActionTypeMove, ActionTypeFire}
)

// TransitionNormalizationSpec freezes how trainer-facing transition rows are transformed into
// dense float32 tensors. Scalar numeric features are clipped into bounded ranges, binary flags
// become explicit 0/1 floats, and terrain / occupancy patches are expanded into one-hot blocks.
type TransitionNormalizationSpec struct {
	PatchRadius          int
	PositionScale        float32
	DistanceScale        float32
	HealthScale          float32
	ProjectileCountScale float32
	CooldownScale        float32
	TerrainVocabulary    []int16
	OccupancyVocabulary  []int16
	ActionVocabulary     []ActionType
}

// VectorizedTransition contains one fully normalized trainer sample with fixed tensor blocks
// for observation, action payload, reward, done flag and next observation.
type VectorizedTransition struct {
	EpisodeID uint64
	Tick      uint32
	Scenario  string
	Outcome   string

	Obs     []float32
	Action  []float32
	Reward  float32
	Done    float32
	NextObs []float32
}

// TransitionBatch keeps one contiguous batch layout so trainer-side code can hand slices
// directly to tensor libraries without re-packing per-sample arrays.
type TransitionBatch struct {
	BatchSize int
	ObsDim    int
	ActionDim int
	Obs       []float32
	Action    []float32
	Reward    []float32
	Done      []float32
	NextObs   []float32
}

// TransitionBatchBuilder incrementally packs normalized transition rows into fixed-size
// batches while preserving the frozen observation and action tensor dimensions.
type TransitionBatchBuilder struct {
	spec      TransitionNormalizationSpec
	batchSize int
	current   *TransitionBatch
}

// TransitionTensorInspection reports the shapes and value ranges observed while vectorizing
// trainer-facing transition rows so CLI tools can detect schema drifts and bad scales quickly.
type TransitionTensorInspection struct {
	Rows                int
	CompletedBatches    int
	TailBatchSize       int
	ObsDim              int
	ActionDim           int
	ActionAcceptedCount int
	DoneCount           int
	RewardMin           float32
	RewardMax           float32
	ObsMin              float32
	ObsMax              float32
	ActionMin           float32
	ActionMax           float32
	NextObsMin          float32
	NextObsMax          float32
}

type transitionObservationProjection struct {
	PatchRadius                  int16
	ShooterX                     float32
	ShooterY                     float32
	ShooterHP                    int16
	TargetX                      float32
	TargetY                      float32
	TargetHP                     int16
	RelativeTargetX              float32
	RelativeTargetY              float32
	DistanceToTarget             float32
	ProjectileCount              uint16
	ShooterWeaponReady           uint8
	ShooterCooldownRemaining     uint16
	ShooterHasActiveFireOrder    uint8
	ShooterHasQueuedFireOrder    uint8
	ShooterHasActiveMoveOrder    uint8
	ShooterHasQueuedMoveOrder    uint8
	ShooterHasDestination        uint8
	ShooterDestinationX          float32
	ShooterDestinationY          float32
	ShooterDistanceToDestination float32
	ShooterRecentMoveFailure     uint8
	LocalTerrainPatch            []int16
	LocalOccupancyPatch          []int16
	NearestFriendlyShotExists    uint8
	NearestFriendlyShotX         float32
	NearestFriendlyShotY         float32
	NearestFriendlyShotDist      float32
	NearestHostileShotExists     uint8
	NearestHostileShotX          float32
	NearestHostileShotY          float32
	NearestHostileShotDist       float32
}

// DefaultTransitionNormalizationSpec returns the first frozen tensor contract for the duel
// dataset. The defaults match the current duel rules: health 3, fire cooldown 10 ticks, and
// one-hot patch vocabularies for the known terrain / occupancy codes.
func DefaultTransitionNormalizationSpec() TransitionNormalizationSpec {
	return TransitionNormalizationSpec{
		PatchRadius:          duelObservationPatchRadius,
		PositionScale:        defaultObservationPositionScale,
		DistanceScale:        defaultObservationDistanceScale,
		HealthScale:          defaultObservationHealthScale,
		ProjectileCountScale: defaultProjectileCountScale,
		CooldownScale:        defaultCooldownScale,
		TerrainVocabulary:    append([]int16(nil), defaultTerrainVocabulary...),
		OccupancyVocabulary:  append([]int16(nil), defaultOccupancyVocabulary...),
		ActionVocabulary:     append([]ActionType(nil), defaultActionVocabulary...),
	}
}

// Normalized fills any omitted field in the spec so every trainer-side component consumes one
// complete contract even when callers override only a subset of the defaults.
func (s TransitionNormalizationSpec) Normalized() TransitionNormalizationSpec {
	if s.PatchRadius <= 0 {
		s.PatchRadius = duelObservationPatchRadius
	}
	if s.PositionScale <= 0 {
		s.PositionScale = defaultObservationPositionScale
	}
	if s.DistanceScale <= 0 {
		s.DistanceScale = defaultObservationDistanceScale
	}
	if s.HealthScale <= 0 {
		s.HealthScale = defaultObservationHealthScale
	}
	if s.ProjectileCountScale <= 0 {
		s.ProjectileCountScale = defaultProjectileCountScale
	}
	if s.CooldownScale <= 0 {
		s.CooldownScale = defaultCooldownScale
	}
	if len(s.TerrainVocabulary) == 0 {
		s.TerrainVocabulary = append([]int16(nil), defaultTerrainVocabulary...)
	} else {
		s.TerrainVocabulary = append([]int16(nil), s.TerrainVocabulary...)
	}
	if len(s.OccupancyVocabulary) == 0 {
		s.OccupancyVocabulary = append([]int16(nil), defaultOccupancyVocabulary...)
	} else {
		s.OccupancyVocabulary = append([]int16(nil), s.OccupancyVocabulary...)
	}
	if len(s.ActionVocabulary) == 0 {
		s.ActionVocabulary = append([]ActionType(nil), defaultActionVocabulary...)
	} else {
		s.ActionVocabulary = append([]ActionType(nil), s.ActionVocabulary...)
	}
	return s
}

// PatchWidth reports the square edge length implied by the configured patch radius.
func (s TransitionNormalizationSpec) PatchWidth() int {
	s = s.Normalized()
	return s.PatchRadius*2 + 1
}

// ExpectedPatchLength reports how many cells every terrain and occupancy patch must contain.
func (s TransitionNormalizationSpec) ExpectedPatchLength() int {
	patchWidth := s.PatchWidth()
	return patchWidth * patchWidth
}

// ObservationDim reports the final dense observation tensor length after scalar normalization
// and one-hot expansion of both local patches.
func (s TransitionNormalizationSpec) ObservationDim() int {
	s = s.Normalized()
	return len(s.observationScalarFeatureNames()) +
		s.ExpectedPatchLength()*len(s.TerrainVocabulary) +
		s.ExpectedPatchLength()*len(s.OccupancyVocabulary)
}

// ActionDim reports the fixed action tensor length for the configured action vocabulary plus
// the accepted flag and continuous move / fire payload parameters.
func (s TransitionNormalizationSpec) ActionDim() int {
	s = s.Normalized()
	return len(s.ActionVocabulary) + actionParameterFeatureCount
}

// ObservationFeatureNames documents the deterministic observation tensor layout so any trainer
// or diagnostics tool can map tensor positions back to logical fields without guesswork.
func (s TransitionNormalizationSpec) ObservationFeatureNames() []string {
	s = s.Normalized()
	names := append([]string(nil), s.observationScalarFeatureNames()...)
	for patchIndex := 0; patchIndex < s.ExpectedPatchLength(); patchIndex++ {
		for _, code := range s.TerrainVocabulary {
			names = append(names, fmt.Sprintf("terrain_patch[%d]==%d", patchIndex, code))
		}
	}
	for patchIndex := 0; patchIndex < s.ExpectedPatchLength(); patchIndex++ {
		for _, code := range s.OccupancyVocabulary {
			names = append(names, fmt.Sprintf("occupancy_patch[%d]==%d", patchIndex, code))
		}
	}
	return names
}

// ActionFeatureNames documents the deterministic action tensor layout so trainer-side code can
// interpret one-hot action ids and continuous parameters consistently.
func (s TransitionNormalizationSpec) ActionFeatureNames() []string {
	s = s.Normalized()
	names := make([]string, 0, s.ActionDim())
	for _, actionType := range s.ActionVocabulary {
		names = append(names, fmt.Sprintf("action_type==%s", actionType))
	}
	names = append(names, s.actionParameterFeatureNames()...)
	return names
}

// VectorizeTransition converts one trainer-facing transition row into the frozen tensor layout.
// Any schema drift, unknown categorical value or wrong patch length returns an explicit error.
func VectorizeTransition(record TrainingTransitionRecord, spec TransitionNormalizationSpec) (VectorizedTransition, error) {
	spec = spec.Normalized()

	obs, err := vectorizeObservationProjection(spec, currentObservationProjection(record))
	if err != nil {
		return VectorizedTransition{}, fmt.Errorf("vectorize current observation: %w", err)
	}
	action, err := vectorizeAction(spec, record)
	if err != nil {
		return VectorizedTransition{}, fmt.Errorf("vectorize action: %w", err)
	}
	nextObs, err := vectorizeObservationProjection(spec, nextObservationProjection(record))
	if err != nil {
		return VectorizedTransition{}, fmt.Errorf("vectorize next observation: %w", err)
	}

	return VectorizedTransition{
		EpisodeID: record.EpisodeID,
		Tick:      record.Tick,
		Scenario:  record.Scenario,
		Outcome:   record.Outcome,
		Obs:       obs,
		Action:    action,
		Reward:    record.Reward,
		Done:      normalizeBinary(record.Done),
		NextObs:   nextObs,
	}, nil
}

// NewTransitionBatchBuilder creates one incremental packer for fixed-size transition batches.
func NewTransitionBatchBuilder(spec TransitionNormalizationSpec, batchSize int) (*TransitionBatchBuilder, error) {
	if batchSize <= 0 {
		return nil, fmt.Errorf("batch size must be positive")
	}
	spec = spec.Normalized()
	return &TransitionBatchBuilder{
		spec:      spec,
		batchSize: batchSize,
		current:   newTransitionBatch(batchSize, spec.ObservationDim(), spec.ActionDim()),
	}, nil
}

// AppendRecord vectorizes one raw transition row and appends it to the current batch. When the
// batch reaches capacity, the completed batch is returned and the builder starts a fresh one.
func (b *TransitionBatchBuilder) AppendRecord(record TrainingTransitionRecord) (*TransitionBatch, error) {
	if b == nil {
		return nil, fmt.Errorf("transition batch builder is nil")
	}

	transition, err := VectorizeTransition(record, b.spec)
	if err != nil {
		return nil, err
	}
	return b.AppendTransition(transition)
}

// AppendTransition appends one already vectorized transition to the current batch.
func (b *TransitionBatchBuilder) AppendTransition(transition VectorizedTransition) (*TransitionBatch, error) {
	if b == nil {
		return nil, fmt.Errorf("transition batch builder is nil")
	}
	if b.current == nil {
		b.current = newTransitionBatch(b.batchSize, b.spec.ObservationDim(), b.spec.ActionDim())
	}
	if len(transition.Obs) != b.current.ObsDim {
		return nil, fmt.Errorf("observation dim = %d, want %d", len(transition.Obs), b.current.ObsDim)
	}
	if len(transition.Action) != b.current.ActionDim {
		return nil, fmt.Errorf("action dim = %d, want %d", len(transition.Action), b.current.ActionDim)
	}
	if len(transition.NextObs) != b.current.ObsDim {
		return nil, fmt.Errorf("next observation dim = %d, want %d", len(transition.NextObs), b.current.ObsDim)
	}
	if b.current.BatchSize >= b.batchSize {
		return nil, fmt.Errorf("current batch is already full")
	}

	b.current.Obs = append(b.current.Obs, transition.Obs...)
	b.current.Action = append(b.current.Action, transition.Action...)
	b.current.Reward = append(b.current.Reward, transition.Reward)
	b.current.Done = append(b.current.Done, transition.Done)
	b.current.NextObs = append(b.current.NextObs, transition.NextObs...)
	b.current.BatchSize++
	if b.current.BatchSize < b.batchSize {
		return nil, nil
	}

	completed := b.current
	b.current = newTransitionBatch(b.batchSize, b.spec.ObservationDim(), b.spec.ActionDim())
	return completed, nil
}

// Flush returns the final partial batch, if any samples are buffered, and resets the builder.
func (b *TransitionBatchBuilder) Flush() *TransitionBatch {
	if b == nil || b.current == nil || b.current.BatchSize == 0 {
		return nil
	}

	completed := b.current
	b.current = newTransitionBatch(b.batchSize, b.spec.ObservationDim(), b.spec.ActionDim())
	return completed
}

// ObserveTransition updates the inspection summary from one normalized sample.
func (i *TransitionTensorInspection) ObserveTransition(transition VectorizedTransition) {
	if i == nil {
		return
	}

	i.Rows++
	i.ObsDim = len(transition.Obs)
	i.ActionDim = len(transition.Action)
	if i.Rows == 1 {
		i.RewardMin = transition.Reward
		i.RewardMax = transition.Reward
		i.ObsMin, i.ObsMax = minMaxSlice(transition.Obs)
		i.ActionMin, i.ActionMax = minMaxSlice(transition.Action)
		i.NextObsMin, i.NextObsMax = minMaxSlice(transition.NextObs)
	} else {
		i.RewardMin = minFloat32(i.RewardMin, transition.Reward)
		i.RewardMax = maxFloat32(i.RewardMax, transition.Reward)
		obsMin, obsMax := minMaxSlice(transition.Obs)
		actionMin, actionMax := minMaxSlice(transition.Action)
		nextObsMin, nextObsMax := minMaxSlice(transition.NextObs)
		i.ObsMin = minFloat32(i.ObsMin, obsMin)
		i.ObsMax = maxFloat32(i.ObsMax, obsMax)
		i.ActionMin = minFloat32(i.ActionMin, actionMin)
		i.ActionMax = maxFloat32(i.ActionMax, actionMax)
		i.NextObsMin = minFloat32(i.NextObsMin, nextObsMin)
		i.NextObsMax = maxFloat32(i.NextObsMax, nextObsMax)
	}
	if transition.Done >= 0.5 {
		i.DoneCount++
	}
	actionAcceptedIndex := len(transition.Action) - actionParameterFeatureCount
	if actionAcceptedIndex >= 0 && transition.Action[actionAcceptedIndex] >= 0.5 {
		i.ActionAcceptedCount++
	}
}

// ObserveCompletedBatch increments the batch counters after the caller receives one full batch.
func (i *TransitionTensorInspection) ObserveCompletedBatch(batch *TransitionBatch) {
	if i == nil || batch == nil {
		return
	}
	i.CompletedBatches++
	i.TailBatchSize = batch.BatchSize
}

// ObserveTailBatch records the final partially filled batch size after the stream ends.
func (i *TransitionTensorInspection) ObserveTailBatch(batch *TransitionBatch) {
	if i == nil {
		return
	}
	if batch == nil {
		i.TailBatchSize = 0
		return
	}
	i.TailBatchSize = batch.BatchSize
}

func (s TransitionNormalizationSpec) observationScalarFeatureNames() []string {
	return []string{
		"patch_radius",
		"shooter_x",
		"shooter_y",
		"shooter_hp",
		"target_x",
		"target_y",
		"target_hp",
		"relative_target_x",
		"relative_target_y",
		"distance_to_target",
		"projectile_count",
		"shooter_weapon_ready",
		"shooter_cooldown_remaining",
		"shooter_has_active_fire_order",
		"shooter_has_queued_fire_order",
		"shooter_has_active_move_order",
		"shooter_has_queued_move_order",
		"shooter_has_destination",
		"shooter_destination_x",
		"shooter_destination_y",
		"shooter_distance_to_destination",
		"shooter_recent_move_failure",
		"nearest_friendly_shot_exists",
		"nearest_friendly_shot_x",
		"nearest_friendly_shot_y",
		"nearest_friendly_shot_dist",
		"nearest_hostile_shot_exists",
		"nearest_hostile_shot_x",
		"nearest_hostile_shot_y",
		"nearest_hostile_shot_dist",
	}
}

func (s TransitionNormalizationSpec) actionParameterFeatureNames() []string {
	return []string{
		"action_accepted",
		"action_move_target_x",
		"action_move_target_y",
		"action_dir_x",
		"action_dir_y",
	}
}

func currentObservationProjection(record TrainingTransitionRecord) transitionObservationProjection {
	return transitionObservationProjection{
		PatchRadius:                  record.ObsPatchRadius,
		ShooterX:                     record.ObsShooterX,
		ShooterY:                     record.ObsShooterY,
		ShooterHP:                    record.ObsShooterHP,
		TargetX:                      record.ObsTargetX,
		TargetY:                      record.ObsTargetY,
		TargetHP:                     record.ObsTargetHP,
		RelativeTargetX:              record.ObsRelativeTargetX,
		RelativeTargetY:              record.ObsRelativeTargetY,
		DistanceToTarget:             record.ObsDistanceToTarget,
		ProjectileCount:              record.ObsProjectileCount,
		ShooterWeaponReady:           record.ObsShooterWeaponReady,
		ShooterCooldownRemaining:     record.ObsShooterCooldownRemaining,
		ShooterHasActiveFireOrder:    record.ObsShooterHasActiveFireOrder,
		ShooterHasQueuedFireOrder:    record.ObsShooterHasQueuedFireOrder,
		ShooterHasActiveMoveOrder:    record.ObsShooterHasActiveMoveOrder,
		ShooterHasQueuedMoveOrder:    record.ObsShooterHasQueuedMoveOrder,
		ShooterHasDestination:        record.ObsShooterHasDestination,
		ShooterDestinationX:          record.ObsShooterDestinationX,
		ShooterDestinationY:          record.ObsShooterDestinationY,
		ShooterDistanceToDestination: record.ObsShooterDistanceToDestination,
		ShooterRecentMoveFailure:     record.ObsShooterRecentMoveFailure,
		LocalTerrainPatch:            record.ObsLocalTerrainPatch,
		LocalOccupancyPatch:          record.ObsLocalOccupancyPatch,
		NearestFriendlyShotExists:    record.ObsNearestFriendlyShotExists,
		NearestFriendlyShotX:         record.ObsNearestFriendlyShotX,
		NearestFriendlyShotY:         record.ObsNearestFriendlyShotY,
		NearestFriendlyShotDist:      record.ObsNearestFriendlyShotDist,
		NearestHostileShotExists:     record.ObsNearestHostileShotExists,
		NearestHostileShotX:          record.ObsNearestHostileShotX,
		NearestHostileShotY:          record.ObsNearestHostileShotY,
		NearestHostileShotDist:       record.ObsNearestHostileShotDist,
	}
}

func nextObservationProjection(record TrainingTransitionRecord) transitionObservationProjection {
	return transitionObservationProjection{
		PatchRadius:                  record.NextObsPatchRadius,
		ShooterX:                     record.NextObsShooterX,
		ShooterY:                     record.NextObsShooterY,
		ShooterHP:                    record.NextObsShooterHP,
		TargetX:                      record.NextObsTargetX,
		TargetY:                      record.NextObsTargetY,
		TargetHP:                     record.NextObsTargetHP,
		RelativeTargetX:              record.NextObsRelativeTargetX,
		RelativeTargetY:              record.NextObsRelativeTargetY,
		DistanceToTarget:             record.NextObsDistanceToTarget,
		ProjectileCount:              record.NextObsProjectileCount,
		ShooterWeaponReady:           record.NextObsShooterWeaponReady,
		ShooterCooldownRemaining:     record.NextObsShooterCooldownRemaining,
		ShooterHasActiveFireOrder:    record.NextObsShooterHasActiveFireOrder,
		ShooterHasQueuedFireOrder:    record.NextObsShooterHasQueuedFireOrder,
		ShooterHasActiveMoveOrder:    record.NextObsShooterHasActiveMoveOrder,
		ShooterHasQueuedMoveOrder:    record.NextObsShooterHasQueuedMoveOrder,
		ShooterHasDestination:        record.NextObsShooterHasDestination,
		ShooterDestinationX:          record.NextObsShooterDestinationX,
		ShooterDestinationY:          record.NextObsShooterDestinationY,
		ShooterDistanceToDestination: record.NextObsShooterDistanceToDestination,
		ShooterRecentMoveFailure:     record.NextObsShooterRecentMoveFailure,
		LocalTerrainPatch:            record.NextObsLocalTerrainPatch,
		LocalOccupancyPatch:          record.NextObsLocalOccupancyPatch,
		NearestFriendlyShotExists:    record.NextObsNearestFriendlyShotExists,
		NearestFriendlyShotX:         record.NextObsNearestFriendlyShotX,
		NearestFriendlyShotY:         record.NextObsNearestFriendlyShotY,
		NearestFriendlyShotDist:      record.NextObsNearestFriendlyShotDist,
		NearestHostileShotExists:     record.NextObsNearestHostileShotExists,
		NearestHostileShotX:          record.NextObsNearestHostileShotX,
		NearestHostileShotY:          record.NextObsNearestHostileShotY,
		NearestHostileShotDist:       record.NextObsNearestHostileShotDist,
	}
}

func vectorizeObservationProjection(spec TransitionNormalizationSpec, projection transitionObservationProjection) ([]float32, error) {
	if int(projection.PatchRadius) != spec.PatchRadius {
		return nil, fmt.Errorf("patch radius = %d, want %d", projection.PatchRadius, spec.PatchRadius)
	}
	if len(projection.LocalTerrainPatch) != spec.ExpectedPatchLength() {
		return nil, fmt.Errorf("terrain patch length = %d, want %d", len(projection.LocalTerrainPatch), spec.ExpectedPatchLength())
	}
	if len(projection.LocalOccupancyPatch) != spec.ExpectedPatchLength() {
		return nil, fmt.Errorf("occupancy patch length = %d, want %d", len(projection.LocalOccupancyPatch), spec.ExpectedPatchLength())
	}

	features := make([]float32, 0, spec.ObservationDim())
	features = append(features,
		normalizeNonNegative(float32(projection.PatchRadius), float32(spec.PatchRadius)),
		normalizeNonNegative(projection.ShooterX, spec.PositionScale),
		normalizeNonNegative(projection.ShooterY, spec.PositionScale),
		normalizeNonNegative(float32(projection.ShooterHP), spec.HealthScale),
		normalizeNonNegative(projection.TargetX, spec.PositionScale),
		normalizeNonNegative(projection.TargetY, spec.PositionScale),
		normalizeNonNegative(float32(projection.TargetHP), spec.HealthScale),
		normalizeSymmetric(projection.RelativeTargetX, spec.PositionScale),
		normalizeSymmetric(projection.RelativeTargetY, spec.PositionScale),
		normalizeNonNegative(projection.DistanceToTarget, spec.DistanceScale),
		normalizeNonNegative(float32(projection.ProjectileCount), spec.ProjectileCountScale),
		normalizeBinary(projection.ShooterWeaponReady),
		normalizeNonNegative(float32(projection.ShooterCooldownRemaining), spec.CooldownScale),
		normalizeBinary(projection.ShooterHasActiveFireOrder),
		normalizeBinary(projection.ShooterHasQueuedFireOrder),
		normalizeBinary(projection.ShooterHasActiveMoveOrder),
		normalizeBinary(projection.ShooterHasQueuedMoveOrder),
		normalizeBinary(projection.ShooterHasDestination),
		normalizeSymmetric(projection.ShooterDestinationX, spec.PositionScale),
		normalizeSymmetric(projection.ShooterDestinationY, spec.PositionScale),
		normalizeNonNegative(projection.ShooterDistanceToDestination, spec.DistanceScale),
		normalizeBinary(projection.ShooterRecentMoveFailure),
		normalizeBinary(projection.NearestFriendlyShotExists),
		normalizeSymmetric(projection.NearestFriendlyShotX, spec.PositionScale),
		normalizeSymmetric(projection.NearestFriendlyShotY, spec.PositionScale),
		normalizeNonNegative(projection.NearestFriendlyShotDist, spec.DistanceScale),
		normalizeBinary(projection.NearestHostileShotExists),
		normalizeSymmetric(projection.NearestHostileShotX, spec.PositionScale),
		normalizeSymmetric(projection.NearestHostileShotY, spec.PositionScale),
		normalizeNonNegative(projection.NearestHostileShotDist, spec.DistanceScale),
	)

	encodedTerrainPatch, err := encodeCategoricalPatch(projection.LocalTerrainPatch, spec.TerrainVocabulary, "terrain")
	if err != nil {
		return nil, err
	}
	features = append(features, encodedTerrainPatch...)

	encodedOccupancyPatch, err := encodeCategoricalPatch(projection.LocalOccupancyPatch, spec.OccupancyVocabulary, "occupancy")
	if err != nil {
		return nil, err
	}
	features = append(features, encodedOccupancyPatch...)
	return features, nil
}

func vectorizeAction(spec TransitionNormalizationSpec, record TrainingTransitionRecord) ([]float32, error) {
	action := make([]float32, 0, spec.ActionDim())
	actionType, err := encodeActionType(spec.ActionVocabulary, record.ActionType)
	if err != nil {
		return nil, err
	}
	action = append(action, actionType...)
	action = append(action,
		normalizeBinary(record.ActionAccepted),
		normalizeNonNegative(record.ActionMoveTargetX, spec.PositionScale),
		normalizeNonNegative(record.ActionMoveTargetY, spec.PositionScale),
		normalizeSymmetric(record.ActionDirX, 1),
		normalizeSymmetric(record.ActionDirY, 1),
	)
	return action, nil
}

func encodeActionType(vocabulary []ActionType, actionType string) ([]float32, error) {
	encoded := make([]float32, len(vocabulary))
	for index, candidate := range vocabulary {
		if string(candidate) != actionType {
			continue
		}
		encoded[index] = 1
		return encoded, nil
	}
	return nil, fmt.Errorf("unknown action type %q", actionType)
}

func encodeCategoricalPatch(values, vocabulary []int16, patchName string) ([]float32, error) {
	encoded := make([]float32, 0, len(values)*len(vocabulary))
	for patchIndex, value := range values {
		matched := false
		for _, candidate := range vocabulary {
			if value == candidate {
				encoded = append(encoded, 1)
				matched = true
			} else {
				encoded = append(encoded, 0)
			}
		}
		if matched {
			continue
		}
		return nil, fmt.Errorf("%s patch value %d at index %d is outside vocabulary", patchName, value, patchIndex)
	}
	return encoded, nil
}

func newTransitionBatch(batchSize, obsDim, actionDim int) *TransitionBatch {
	return &TransitionBatch{
		ObsDim:    obsDim,
		ActionDim: actionDim,
		Obs:       make([]float32, 0, batchSize*obsDim),
		Action:    make([]float32, 0, batchSize*actionDim),
		Reward:    make([]float32, 0, batchSize),
		Done:      make([]float32, 0, batchSize),
		NextObs:   make([]float32, 0, batchSize*obsDim),
	}
}

func normalizeBinary(value uint8) float32 {
	if value > 0 {
		return 1
	}
	return 0
}

func normalizeNonNegative(value, scale float32) float32 {
	if scale <= 0 {
		return maxFloat32(value, 0)
	}
	return clampFloat32(value/scale, 0, 1)
}

func normalizeSymmetric(value, scale float32) float32 {
	if scale <= 0 {
		return clampFloat32(value, -1, 1)
	}
	return clampFloat32(value/scale, -1, 1)
}

func clampFloat32(value, minValue, maxValue float32) float32 {
	return minFloat32(maxFloat32(value, minValue), maxValue)
}

func minFloat32(left, right float32) float32 {
	if left < right {
		return left
	}
	return right
}

func maxFloat32(left, right float32) float32 {
	if left > right {
		return left
	}
	return right
}

func minMaxSlice(values []float32) (float32, float32) {
	if len(values) == 0 {
		return 0, 0
	}
	minValue := float32(math.Inf(1))
	maxValue := float32(math.Inf(-1))
	for _, value := range values {
		minValue = minFloat32(minValue, value)
		maxValue = maxFloat32(maxValue, value)
	}
	return minValue, maxValue
}

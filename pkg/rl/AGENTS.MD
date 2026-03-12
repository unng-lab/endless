# RL package: current state and next plan

## Current state
- `env.go` defines the RL-facing contract: `Environment`, `Policy`, `Observation`, `Action`, `StepResult`.
- `duel_environment.go` adapts the current `unit.Manager` into a deterministic headless `1v1` duel environment.
- Agent actions are still routed only through gameplay API:
  - move via `Manager.IssueMoveOrder`
  - fire via `Manager.IssueFireOrder`
- `policy.go` contains the scripted baseline `LeadAndStrafePolicy`.
- `policy_factory.go` contains named policy construction and deterministic seeding for stochastic policies.
- `evaluation.go` contains duel evaluation without ClickHouse writes.
- `duel.go` writes RL transition rows with:
  - pre-action observation
  - accepted action
  - post-step state
  - reward and done flag
- `observation_builder.go` now enriches observations with:
  - local terrain patch around the shooter
  - local occupancy patch
  - nearest friendly projectile feature
  - nearest hostile projectile feature
- `Observation` now also carries move-order context:
  - has destination
  - destination relative vector
  - distance to destination
  - recent move failure flag
- local occupancy now includes movement blockers / cover tiles.
- `duel_layout.go` contains named duel layouts, including `duel_open` and `duel_with_cover`.
- `unit.Manager` pathfinding now respects blocking units, so cover affects both movement routing and RL observations.
- `clickhouse.go` writes dataset rows to ClickHouse and upgrades older step schema with `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`.
- ClickHouse also exposes a trainer-facing `*_transitions` view with explicit `obs_*` and `next_obs_*` aliases so external trainers do not depend on internal post-state column names.
- `training_contract.go` documents the intended external training shape and fixes the logical contract around `obs_*`, `action_*`, `reward`, `done`, and `next_obs_*`.
- `transitions_export.go` streams the trainer-facing `*_transitions` view and exports it as `jsonl` or `json` without manual SQL assembly.
- `transition_tensor.go` freezes the first Go-side tensor contract: normalization spec, `TrainingTransitionRecord` vectorization, fixed-size batch packing and inspection summaries.
- The first tensor contract currently uses:
  - scalar numeric features normalized into bounded `float32`
  - binary flags encoded as explicit `0.0/1.0`
  - local terrain patch expanded as one-hot over `[-1, 0, 1, 2, 3, 4]`
  - local occupancy patch expanded as one-hot over `[-1, 0, 1, 2, 3, 4, 5]`
  - action type expanded as one-hot over `none`, `move`, `fire`
  - reward kept as raw `float32`
  - done encoded as `0.0/1.0`
- The default tensor spec currently assumes duel-scale constants:
  - patch radius `2` -> patch area `25`
  - observation tensor dim `355`
  - action tensor dim `8`
  - position / relative-position scale `1024`
  - distance scale `1024`
  - health scale `3`
  - projectile-count scale `8`
  - cooldown scale `10`
- `TransitionBatchBuilder` packs normalized transitions into fixed-size contiguous slices:
  - `Obs`
  - `Action`
  - `Reward`
  - `Done`
  - `NextObs`
- `unit.Manager` exposes `Close()` and `ProjectileSnapshots()` for safe episode lifecycle and observation building.
- `cmd/endless-rl-train` exposes `-scenario`, `-policy`, and `-mode` so collection/evaluation can run against cover scenarios from CLI.
- `cmd/endless-rl-train` now also exposes `inspect-batches` mode to validate tensor dimensions and value ranges against live ClickHouse transitions.
- `cmd/endless-rl-train` now also exposes `export` mode with scenario/outcome/episode filters for external dataset consumers.
- `clickhouse_test.go`, `transitions_export_test.go` and `transition_tensor_test.go` verify that the trainer-facing transitions contract remains readable and tensorizable.

## Stable contracts
- `Observation.Snapshot` remains the canonical compact duel state.
- `Observation` may grow with extra derived spatial features, but should not require direct access to mutable unit internals.
- `StepRecord` in ClickHouse is the primary offline RL training row.
- External trainers should prefer the `*_transitions` view over the raw `*_steps` table.
- `TrainingTransitionRecord` is the stable logical read contract for trainer-side consumers.
- `TransitionNormalizationSpec` is now the stable Go-side tensorization contract for the current duel setup.
- Observation tensor layout is fixed as:
  - scalar block first
  - then one-hot terrain patch cells in raster order
  - then one-hot occupancy patch cells in raster order
- Action tensor layout is fixed as:
  - action-type one-hot block first
  - then `action_accepted`
  - then `action_move_target_x`
  - then `action_move_target_y`
  - then `action_dir_x`
  - then `action_dir_y`
- `EventRecord` remains a sparse audit/event log, not the main observation payload.
- `RunDuelCollectionWithPolicy` and `RunDuelEvaluation` must stay deterministic for a fixed seed.
- `inspect-batches` is the canonical smoke-check path for validating that live ClickHouse transitions still match the frozen tensor contract.

## Next plan
1. Add the first Go-side training loop / learner stub on top of the frozen tensor contract.
2. Extend evaluation output for per-policy comparison across fixed seed suites so training regressions can be measured from CLI.
3. Decide whether sequence-oriented exports are needed beyond the current single-row transition view.
4. Revisit normalization scales if world size, health budget or cooldown budget change materially beyond the current duel defaults.
5. Keep trainer/inference outside the Go runtime until the environment and dataset schema stabilize.

## Current limits
- The environment is still `1v1 duel`, not a general battle scene.
- The target actor is still scripted and follows waypoint patrol logic.
- Trainer/inference integration outside Go is not implemented yet.
- Full multi-unit world snapshot is not written to ClickHouse; the dataset is still a duel-focused enriched transition row.
- The first tensor contract is frozen for the current duel setup, but scale constants still assume the present duel world ranges and may need revision if scenario size changes materially.
- The current tensor contract is optimized for single-step transition learning; no sequence/window contract is fixed yet.

# Пакет RL: текущее состояние и следующий план

## Текущее состояние
- `env.go` определяет RL-контракт: `Environment`, `Policy`, `Observation`, `Action`, `StepResult`.
- `duel_environment.go` адаптирует текущий `unit.Manager` в детерминированное headless-окружение дуэли `1v1`.
- Действия агента по-прежнему проходят только через gameplay API:
  - перемещение через `Manager.IssueMoveOrder`
  - выстрел через `Manager.IssueFireOrder`
- `policy.go` содержит scripted baseline `LeadAndStrafePolicy`.
- `policy_factory.go` содержит создание именованных политик и детерминированную инициализацию seed для стохастических политик.
- `evaluation.go` содержит оценку дуэлей без записи в ClickHouse.
- `evaluation_suite.go` добавляет сравнение нескольких политик на фиксированном наборе seed, чтобы CLI-regression checks сравнивали политики на одном и том же наборе дуэльных эпизодов.
- `duel.go` пишет RL transition-строки с:
  - наблюдением до действия
  - принятым действием
  - состоянием после шага
  - наградой и флагом завершения
- `observation_builder.go` теперь обогащает наблюдения:
  - локальным terrain patch вокруг стрелка
  - локальным occupancy patch
  - признаком ближайшего дружественного снаряда
  - признаком ближайшего вражеского снаряда
- `Observation` теперь также несёт контекст move-order:
  - есть ли destination
  - относительный вектор до destination
  - расстояние до destination
  - флаг недавней неудачи перемещения
- Локальный occupancy patch теперь включает movement blockers / cover tiles.
- `duel_layout.go` содержит именованные layouts для дуэлей, включая `duel_open` и `duel_with_cover`.
- Pathfinding в `unit.Manager` теперь учитывает блокирующие юниты, поэтому cover влияет и на маршрутизацию перемещения, и на RL-наблюдения.
- `clickhouse.go` пишет строки датасета в ClickHouse и обновляет старую схему step-таблицы через `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`.
- `clickhouse.go` теперь также разносит диапазоны `episode_id` по сессиям recorder'а, чтобы повторные `collect`-запуски могли дописывать данные в один `table_prefix` без повторного использования локальных `1..N` идентификаторов эпизодов.
- ClickHouse также предоставляет trainer-facing view `*_transitions` с явными alias'ами `obs_*` и `next_obs_*`, чтобы внешние trainer'ы не зависели от внутренних имён post-state колонок.
- `training_contract.go` документирует ожидаемую внешнюю форму training-данных и фиксирует логический контракт вокруг `obs_*`, `action_*`, `reward`, `done` и `next_obs_*`.
- `transitions_export.go` потоково читает trainer-facing view `*_transitions` и экспортирует его как `jsonl` или `json` без ручной сборки SQL.
- `sequence_export.go` группирует trainer-facing transition-строки в sequence export'ы по полным эпизодам или фиксированным окнам для sequence-oriented offline trainer'ов.
- `transition_tensor.go` фиксирует первый Go-side tensor contract: normalization spec, векторизацию `TrainingTransitionRecord`, packing фиксированных batch'ей и inspection summaries.
- `training_stub.go` добавляет первый Go-side learner stub: загрузку vectorized dataset из ClickHouse и batch-trained линейный SARSA-style critic с per-epoch CLI metrics.
- Первый tensor contract сейчас использует:
  - scalar numeric features, нормализованные в ограниченный `float32`
  - бинарные флаги, закодированные как явные `0.0/1.0`
  - local terrain patch, развёрнутый в one-hot по `[-1, 0, 1, 2, 3, 4]`
  - local occupancy patch, развёрнутый в one-hot по `[-1, 0, 1, 2, 3, 4, 5]`
  - тип действия, развёрнутый в one-hot по `none`, `move`, `fire`
  - награду как raw `float32`
  - `done`, закодированный как `0.0/1.0`
- Tensor spec по умолчанию сейчас предполагает duel-scale константы:
  - patch radius `2` -> площадь patch'а `25`
  - размер observation tensor `355`
  - размер action tensor `8`
  - scale для position / relative-position `1024`
  - scale для distance `1024`
  - scale для health `3`
  - scale для projectile-count `8`
  - scale для cooldown `10`
- `TransitionBatchBuilder` упаковывает нормализованные transition'ы в contiguous slices фиксированного размера:
  - `Obs`
  - `Action`
  - `Reward`
  - `Done`
  - `NextObs`
- `unit.Manager` предоставляет `Close()` и `ProjectileSnapshots()` для безопасного жизненного цикла эпизода и построения наблюдений.
- `cmd/endless-rl-train` предоставляет `-scenario`, `-policy` и `-mode`, чтобы сбор и оценка могли запускаться из CLI против сценариев с cover.
- `cmd/endless-rl-train` теперь также предоставляет режим `inspect-batches` для проверки tensor dimensions и value ranges на живых transition'ах из ClickHouse.
- `cmd/endless-rl-train` теперь также предоставляет режим `export` с фильтрами по scenario/outcome/episode для внешних потребителей датасета.
- `cmd/endless-rl-train` теперь также предоставляет режим `export-sequences`, чтобы внешние trainer'ы могли экспортировать сгруппированные sequence'ы по эпизодам/окнам из того же отфильтрованного transition view.
- `cmd/endless-rl-train` теперь также предоставляет режим `train-stub` для запуска первой offline learner smoke-check проверки прямо на trainer-facing transition'ах.
- `cmd/endless-rl-train` теперь также предоставляет режим `compare` с `-policy-suite` и выбором baseline для fixed-seed оценки нескольких политик из CLI.
- `clickhouse_test.go`, `transitions_export_test.go`, `sequence_export_test.go`, `transition_tensor_test.go`, `training_stub_test.go` и `evaluation_suite_test.go` проверяют, что trainer-facing contract остаётся читаемым, tensorizable, обучаемым текущим stub'ом, экспортируемым как grouped sequences и сравнимым между детерминированными policy suite'ами.

## Стабильные контракты
- `Observation.Snapshot` остаётся каноническим компактным состоянием дуэли.
- `Observation` может расти за счёт дополнительных derived spatial features, но не должен требовать прямого доступа к mutable internals юнитов.
- `StepRecord` в ClickHouse является основной offline RL training-строкой.
- Внешним trainer'ам следует предпочитать view `*_transitions`, а не сырую таблицу `*_steps`.
- `TrainingTransitionRecord` является стабильным логическим read-контрактом для trainer-side потребителей.
- `TrainingTransitionSequence` является опциональным grouped export-контрактом, построенным из того же row-level view; он не заменяет каноническую single-row transition schema.
- `TransitionNormalizationSpec` теперь является стабильным Go-side tensorization contract для текущей дуэльной конфигурации.
- `LinearQStub` является только диагностическим training baseline; он валидирует offline tensor pipeline, но не задаёт gameplay/runtime inference contract.
- Layout observation tensor зафиксирован так:
  - сначала scalar block
  - затем one-hot ячейки terrain patch в raster order
  - затем one-hot ячейки occupancy patch в raster order
- Layout action tensor зафиксирован так:
  - сначала one-hot block типа действия
  - затем `action_accepted`
  - затем `action_move_target_x`
  - затем `action_move_target_y`
  - затем `action_dir_x`
  - затем `action_dir_y`
- `EventRecord` остаётся разреженным audit/event log, а не основным observation payload.
- `RunDuelCollectionWithPolicy` и `RunDuelEvaluation` должны оставаться детерминированными при фиксированном seed.
- Сравнение политик на fixed-seed suite должно повторно использовать один и тот же сгенерированный набор episode seed'ов для всех оцениваемых политик.
- `inspect-batches` является каноническим smoke-check путём для проверки того, что живые transition'ы в ClickHouse всё ещё соответствуют зафиксированному tensor contract.

## Следующий план
1. Пересмотреть normalization scale'ы, если размер мира, запас здоровья или бюджет cooldown существенно выйдут за пределы текущих duel defaults.
2. Держать trainer/inference вне Go runtime до тех пор, пока не стабилизируются environment и schema датасета.

## Текущие ограничения
- Окружение по-прежнему является дуэлью `1v1`, а не общей боевой сценой.
- Целевой актор по-прежнему scripted и следует логике waypoint patrol.
- Production trainer/inference integration вне Go пока не реализован; текущий Go-side learner является только smoke-check baseline.
- Полный multi-unit snapshot мира в ClickHouse не пишется; датасет по-прежнему представляет собой enriched transition row, ориентированный на дуэль.
- Первый tensor contract зафиксирован для текущей дуэльной конфигурации, но scale-константы всё ещё предполагают текущие диапазоны мира и могут потребовать пересмотра при заметном изменении размера сценариев.
- Канонический tensor contract по-прежнему оптимизирован под single-step transition learning; sequence export'ы сейчас сохраняют raw row payload вместо того, чтобы задавать второй tensor/window contract.
- `train-stub` сейчас загружает выбранный transition slice целиком в память, чтобы повторные эпохи оставались детерминированными и простыми для инспекции.
- `compare` сейчас логирует только компактные aggregate metrics и delta'ы; он пока не выводит per-episode trace'ы или оценки статистической значимости.

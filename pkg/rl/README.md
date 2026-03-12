# RL Runbook

Этот документ описывает практическую последовательность запуска RL-контура из CLI:

1. как собрать датасет дуэлей в ClickHouse
2. как проверить, что собранные transition-строки соответствуют текущему tensor contract
3. как запустить текущее встроенное offline-обучение `train-stub`
4. как выгрузить датасет для внешнего trainer'а

Документ предполагает, что команды запускаются из корня репозитория `h:\projects\unng\endless` в `PowerShell`.

## Что уже можно делать сейчас

Текущий pipeline уже позволяет:

1. генерировать дуэльные эпизоды и писать `steps`, `events`, `episodes` в ClickHouse
2. читать trainer-facing view `*_transitions`
3. векторизовать transition'ы по текущему `TransitionNormalizationSpec`
4. запускать встроенный Go-side baseline `train-stub`
5. экспортировать row-level или grouped sequence датасет для внешнего обучения

Важно: `train-stub` является диагностическим baseline'ом, а не production trainer'ом. Его задача сейчас в том, чтобы проверить, что сбор данных, tensorization и базовый offline learning-loop работают согласованно.

## Перед запуском

### 1. Убедиться, что доступен ClickHouse

RL CLI читает настройки подключения из переменных окружения:

- `ENDLESS_CLICKHOUSE_ADDR`
- `ENDLESS_CLICKHOUSE_DATABASE`
- `ENDLESS_CLICKHOUSE_USERNAME`
- `ENDLESS_CLICKHOUSE_PASSWORD`
- `ENDLESS_CLICKHOUSE_TABLE_PREFIX`
- `ENDLESS_CLICKHOUSE_BATCH_SIZE`
- `ENDLESS_CLICKHOUSE_DIAL_TIMEOUT`

Если переменные не задать, код подставит built-in defaults. Для воспроизводимых экспериментов лучше задавать их явно, особенно `ENDLESS_CLICKHOUSE_TABLE_PREFIX`, чтобы разные прогоны не смешивали датасеты в одних и тех же таблицах.

Пример подготовки окружения в `PowerShell`:

```powershell
$env:ENDLESS_CLICKHOUSE_ADDR = "127.0.0.1:8123"
$env:ENDLESS_CLICKHOUSE_DATABASE = "default"
$env:ENDLESS_CLICKHOUSE_USERNAME = "default"
$env:ENDLESS_CLICKHOUSE_PASSWORD = "your_password"
$env:ENDLESS_CLICKHOUSE_TABLE_PREFIX = "endless_rl_exp01"
$env:ENDLESS_CLICKHOUSE_BATCH_SIZE = "512"
$env:ENDLESS_CLICKHOUSE_DIAL_TIMEOUT = "5s"
```

Что делает каждый параметр:

1. `ENDLESS_CLICKHOUSE_ADDR` задаёт HTTP-адрес ClickHouse. Можно передать несколько адресов через запятую.
2. `ENDLESS_CLICKHOUSE_DATABASE` задаёт базу данных, в которой будут создаваться таблицы и view RL-контура.
3. `ENDLESS_CLICKHOUSE_USERNAME` и `ENDLESS_CLICKHOUSE_PASSWORD` задают учётные данные.
4. `ENDLESS_CLICKHOUSE_TABLE_PREFIX` задаёт префикс таблиц. Именно он определяет, куда будут писаться и откуда будут читаться RL-данные текущего эксперимента.
5. `ENDLESS_CLICKHOUSE_BATCH_SIZE` задаёт размер буфера batched inserts.
6. `ENDLESS_CLICKHOUSE_DIAL_TIMEOUT` задаёт timeout подключения.

### 2. Скомпилировать или запускать через `go run`

Во всех примерах ниже используется `go run ./cmd/endless-rl-train`. Это удобно для итеративной работы. Если нужен отдельный бинарь, его можно собрать так:

```powershell
go build -o .\bin\endless-rl-train.exe .\cmd\endless-rl-train
```

Тогда вместо `go run ./cmd/endless-rl-train` можно использовать `.\bin\endless-rl-train.exe`.

## Базовая последовательность: сбор данных и встроенное обучение

Это основной сценарий, если нужно быстро проверить весь pipeline внутри текущего Go runtime.

### Шаг 1. Собрать первый датасет дуэлей

Пример команды:

```powershell
go run ./cmd/endless-rl-train `
  -mode collect `
  -scenario duel_with_cover `
  -policy lead_strafe `
  -episodes 500 `
  -max-ticks 600 `
  -seed 1001 `
  -world-columns 64 `
  -world-rows 64 `
  -tile-size 16
```

Что делает эта команда:

1. Запускает headless-генерацию `500` дуэльных эпизодов.
2. В роли policy стрелка использует `lead_strafe`.
3. Для каждой дуэли пишет transition-строки, event log и агрегированный episode summary в ClickHouse.
4. При повторных `collect`-запусках в тот же `table_prefix` не переиспользует локальные `episode_id`, а продолжает общий диапазон идентификаторов.

Что означает каждый ключевой флаг:

1. `-mode collect` включает режим записи датасета в ClickHouse.
2. `-scenario duel_with_cover` выбирает layout дуэли. Для первых запусков полезно сравнивать `duel_open` и `duel_with_cover`.
3. `-policy lead_strafe` задаёт scripted policy стрелка. Сейчас доступны как минимум `lead_strafe` и `random`.
4. `-episodes` задаёт объём собираемого датасета.
5. `-max-ticks` ограничивает длину одного эпизода, чтобы симуляция не зависала на слишком долгих дуэлях.
6. `-seed` делает генерацию воспроизводимой.
7. `-world-columns`, `-world-rows`, `-tile-size` задают геометрию мира.

Какой запуск лучше сделать первым:

1. сначала маленький smoke-test на `50..100` эпизодов
2. затем основной сбор на `500+` эпизодов
3. затем, если нужен более разнообразный датасет, повторить `collect` с другим `-scenario`, `-policy` или `-seed`

### Шаг 2. Проверить, что transition'ы нормально тензоризуются

Сразу после `collect` стоит прогнать `inspect-batches`.

Пример команды:

```powershell
go run ./cmd/endless-rl-train `
  -mode inspect-batches `
  -export-scenario duel_with_cover `
  -batch-size 64
```

Что делает эта команда:

1. Читает trainer-facing view `*_transitions` из ClickHouse.
2. Прогоняет каждую запись через текущий `TransitionNormalizationSpec`.
3. Собирает inspection summary по размерностям и диапазонам значений.
4. Падает сразу, если обнаружит schema drift, неправильную длину patch'а или неизвестные категориальные значения.

На что смотреть в логе:

1. `rows` показывает, что view действительно содержит данные.
2. `obs_dim` и `action_dim` должны быть стабильными между запусками на одном и том же contract.
3. `obs_range`, `action_range`, `next_obs_range` должны выглядеть разумно для нормализованных данных.
4. `action_accepted` и `done` помогают быстро заметить вырожденный датасет, где почти нет принятых действий или почти все эпизоды заканчиваются слишком рано.

Если `inspect-batches` уже на этом шаге падает, запускать обучение рано: сначала нужно исправить проблему со схемой или наблюдениями.

### Шаг 3. Запустить встроенное smoke-check обучение

Пример команды:

```powershell
go run ./cmd/endless-rl-train `
  -mode train-stub `
  -export-scenario duel_with_cover `
  -batch-size 64 `
  -train-epochs 10 `
  -train-learning-rate 0.05 `
  -train-discount 0.99
```

Что делает эта команда:

1. Читает transition'ы из ClickHouse.
2. Векторизует их в фиксированный observation/action tensor contract.
3. Строит простой in-memory dataset.
4. Запускает линейный SARSA-style critic поверх `(obs, action) -> value`.
5. После каждой эпохи печатает агрегированные метрики обучения.

Как читать лог обучения:

1. `[rl-train-stub-epoch]` показывает `loss`, `td_mae`, `avg_prediction`, `avg_target` после каждой эпохи.
2. `[rl-train-stub]` в конце печатает общую форму датасета:
   - `rows` сколько transition'ов попало в обучение
   - `linked_next_actions` сколько строк удалось связать со следующим действием в том же эпизоде
   - `terminal` сколько transition'ов терминальные
   - `unlinked` сколько строк не терминальные, но без следующего contiguous action
   - `obs_dim`, `action_dim`, `input_dim` размеры tensor contract'а
   - `initial_loss`, `final_loss`, `final_td_mae` итоговую динамику

Как интерпретировать результат:

1. если обучение запускается и `final_loss` не уходит в `NaN` или явный разнос, значит базовый pipeline жив
2. если `unlinked` слишком велик, датасет плохо подходит даже для текущего SARSA-style stub
3. если `rows` очень мало, сначала стоит собрать больше эпизодов
4. если диапазоны prediction/target выглядят аномально, нужно перепроверить данные через `inspect-batches`

## Рекомендуемая минимальная последовательность команд

Это короткий сценарий, который обычно стоит выполнять в таком порядке:

```powershell
go run ./cmd/endless-rl-train -mode collect -scenario duel_with_cover -policy lead_strafe -episodes 200 -max-ticks 600 -seed 1001
go run ./cmd/endless-rl-train -mode inspect-batches -export-scenario duel_with_cover -batch-size 64
go run ./cmd/endless-rl-train -mode train-stub -export-scenario duel_with_cover -batch-size 64 -train-epochs 10 -train-learning-rate 0.05 -train-discount 0.99
```

Логика этой последовательности такая:

1. сначала генерируется датасет
2. затем проверяется, что датасет совместим с tensor contract'ом
3. только после этого стартует встроенное обучение

## Готовый сценарий: самый первый запуск

Если нужно просто убедиться, что всё вообще работает от конца до конца, без долгого ожидания и без большого расхода ресурсов, начинай с этого набора команд:

```powershell
$env:ENDLESS_CLICKHOUSE_ADDR = "127.0.0.1:8123"
$env:ENDLESS_CLICKHOUSE_DATABASE = "default"
$env:ENDLESS_CLICKHOUSE_USERNAME = "default"
$env:ENDLESS_CLICKHOUSE_PASSWORD = "your_password"
$env:ENDLESS_CLICKHOUSE_TABLE_PREFIX = "endless_rl_smoke"

go run ./cmd/endless-rl-train -mode collect -scenario duel_open -policy lead_strafe -episodes 50 -max-ticks 400 -seed 101
go run ./cmd/endless-rl-train -mode inspect-batches -export-scenario duel_open -batch-size 32
go run ./cmd/endless-rl-train -mode train-stub -export-scenario duel_open -batch-size 32 -train-epochs 3 -train-learning-rate 0.05 -train-discount 0.99
```

Почему именно такие параметры:

1. `duel_open` проще для первого smoke-test, потому что в нём меньше пространственных факторов и проще понять, что происходит.
2. `50` эпизодов обычно достаточно, чтобы проверить создание таблиц, запись данных, чтение view и запуск обучения.
3. `max-ticks 400` ускоряет первый прогон и не даёт слишком долго висеть на неудачных дуэлях.
4. `batch-size 32` уменьшает размер одного обучающего шага и подходит для короткого первичного теста.
5. `train-epochs 3` достаточно, чтобы увидеть, что модель реально проходит несколько эпох и логирует метрики.

Что считать успешным результатом первого запуска:

1. `collect` завершается без ошибок подключения и без ошибок схемы ClickHouse.
2. `inspect-batches` печатает ненулевой `rows` и не падает на vectorization.
3. `train-stub` проходит все эпохи и печатает финальную строку `[rl-train-stub]`.

Если этот сценарий проходит, можно переходить к более длинным запускам и отдельным экспериментам по сценариям.

## Готовый сценарий: большой прогон для накопления данных

Когда smoke-test уже прошёл и нужен более содержательный датасет, можно запускать такой сценарий:

```powershell
$env:ENDLESS_CLICKHOUSE_TABLE_PREFIX = "endless_rl_cover_exp01"

go run ./cmd/endless-rl-train -mode collect -scenario duel_with_cover -policy lead_strafe -episodes 3000 -max-ticks 800 -seed 2001 -world-columns 64 -world-rows 64 -tile-size 16
go run ./cmd/endless-rl-train -mode collect -scenario duel_with_cover -policy random -episodes 3000 -max-ticks 800 -seed 2002 -world-columns 64 -world-rows 64 -tile-size 16
go run ./cmd/endless-rl-train -mode inspect-batches -export-scenario duel_with_cover -batch-size 128
go run ./cmd/endless-rl-train -mode train-stub -export-scenario duel_with_cover -batch-size 128 -train-epochs 10 -train-learning-rate 0.03 -train-discount 0.99
```

Что даёт такой сценарий:

1. собирает более крупный датасет на сценарии с cover, где поведение и маршрутизация интереснее, чем в `duel_open`
2. смешивает данные как минимум от двух policy, чтобы датасет не был полностью однородным
3. даёт больше transition'ов для проверки того, как ведёт себя текущий tensor contract на реальном объёме
4. позволяет проверить, выдерживает ли текущий in-memory `train-stub` уже не маленький, а рабочий объём

Как запускать большой прогон безопасно:

1. используй новый `ENDLESS_CLICKHOUSE_TABLE_PREFIX`, чтобы не смешивать большой датасет со smoke-тестами
2. сначала сделай один запуск на `500..1000` эпизодов и оцени время выполнения
3. только после этого увеличивай `episodes` до нескольких тысяч
4. если `train-stub` начинает упираться в память, прекращай наращивать объём и используй `export` или `export-sequences` для внешнего trainer'а

## Сценарий для внешнего trainer'а

Если обучение будет происходить не в Go, а во внешнем пайплайне, обычно нужен такой порядок: `collect -> inspect-batches -> export` или `collect -> inspect-batches -> export-sequences`.

### Шаг 1. Собрать данные

Используется тот же `collect`, что и в предыдущем разделе.

### Шаг 2. Проверить tensor contract

Используется тот же `inspect-batches`, что и в предыдущем разделе.

### Шаг 3. Выгрузить row-level transition dataset

Пример команды:

```powershell
go run ./cmd/endless-rl-train `
  -mode export `
  -export-scenario duel_with_cover `
  -export-format jsonl `
  -export-output .\transitions.jsonl
```

Что делает эта команда:

1. Читает trainer-facing view `*_transitions`.
2. Применяет фильтры `scenario`, `outcome`, `episode_id`.
3. Потоково пишет данные в `jsonl` или `json`.

Когда использовать именно этот режим:

1. когда внешний trainer учится на одиночных transition-строках
2. когда нужен канонический single-row contract
3. когда trainer сам умеет строить batch'и и next-step linkage

### Шаг 4. Выгрузить sequence dataset

Пример команды:

```powershell
go run ./cmd/endless-rl-train `
  -mode export-sequences `
  -export-scenario duel_with_cover `
  -export-format jsonl `
  -export-output .\sequences.jsonl `
  -sequence-limit-episodes 200 `
  -sequence-max-steps 128
```

Что делает эта команда:

1. Берёт те же trainer-facing transition'ы.
2. Группирует их по эпизодам или окнам.
3. Пишет сгруппированный экспорт для sequence-oriented trainer'ов.

Когда использовать этот режим:

1. когда внешний trainer ожидает уже сгруппированные эпизоды или окна
2. когда важно сохранить порядок transition'ов внутри sequence
3. когда single-step export недостаточен для выбранной модели

Важно: текущий канонический training contract в проекте остаётся row-level через `*_transitions`. `export-sequences` полезен как отдельный grouped export, но не заменяет основной single-row контракт.

## Отбор подмножеств датасета

Почти все trainer-facing режимы поддерживают фильтрацию:

- `-export-scenario`
- `-export-outcome`
- `-export-episode-id-min`
- `-export-episode-id-max`
- `-export-limit`

Это полезно в трёх типовых случаях:

1. нужно обучаться только на одном сценарии, например `duel_with_cover`
2. нужно разобрать только выигранные или проигранные эпизоды
3. нужно быстро проверить небольшой срез датасета перед большим запуском

Пример обучения stub'а только на ограниченном диапазоне эпизодов:

```powershell
go run ./cmd/endless-rl-train `
  -mode train-stub `
  -export-episode-id-min 1000 `
  -export-episode-id-max 1500 `
  -batch-size 64 `
  -train-epochs 5
```

## Отдельно про `compare`

Режим `compare` сам по себе не обучает модель, но полезен рядом с циклом сбора данных и обучения.

Пример:

```powershell
go run ./cmd/endless-rl-train `
  -mode compare `
  -scenario duel_with_cover `
  -episodes 200 `
  -seed 1001 `
  -policy-suite lead_strafe,random `
  -compare-baseline-policy lead_strafe
```

Зачем это нужно:

1. быстро проверить, что baseline-политики ведут себя ожидаемо на фиксированном наборе seed
2. собрать ориентир, с чем потом сравнивать внешний trainer
3. убедиться, что изменения в окружении не сломали простые scripted baselines

## Практические рекомендации

1. Первый запуск делай на маленьком объёме данных, чтобы быстро проверить ClickHouse schema, сборщик и tensorization.
2. Для каждого отдельного эксперимента используй свой `ENDLESS_CLICKHOUSE_TABLE_PREFIX`.
3. Не начинай обучение, пока `inspect-batches` не проходит без ошибок.
4. Если нужен production trainer, считай `train-stub` не конечной целью, а только проверкой корректности данных и контракта.
5. Если датасет становится слишком большим, текущий `train-stub` может упереться в память, потому что грузит выбранный transition slice целиком.

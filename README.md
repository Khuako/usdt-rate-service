# USDT Rate Service

gRPC-сервис для расчёта курса USDT/USD по стакану Kraken. Каждый успешный вызов `GetRates` получает свежий стакан, рассчитывает ask и bid, сохраняет результат в PostgreSQL и возвращает его клиенту.

Используется `GET /0/public/Depth?pair=USDTUSD&count=100`. Цены обрабатываются через `shopspring/decimal` и передаются в API строками. Объёмы заявок в расчётах не участвуют.

## Запуск через Docker Compose

Нужны Docker с Compose v2 и Make. Команды выполняются из корня репозитория. Для проверки API дополнительно нужен `grpcurl`.

1. Создайте `.env` в корне проекта:

   ```dotenv
   POSTGRES_PASSWORD=replace_with_your_local_password
   ```

   Замените значение своим паролем. Текущий Compose подставляет его непосредственно в URL подключения: используйте для локального запуска буквы, цифры, `_` и `-`. Пароли с зарезервированными символами URL требуют отдельного кодирования для строки подключения. `.env` исключён из Git и контекста Docker-сборки.

2. Запустите сервисы:

   ```bash
   make up
   docker compose ps -a
   ```

   Compose запускает PostgreSQL, ждёт его готовности, применяет миграции и запускает приложение. Jaeger запускается отдельным сервисом. Ожидаемые состояния: PostgreSQL — `healthy`, `migrate` — `Exited (0)`, `server` и `jaeger` — `Up`.

3. Проверьте API по примерам ниже.

Доступные адреса:

| Сервис | Адрес на хосте |
| --- | --- |
| gRPC | `localhost:50051` без TLS |
| Jaeger UI | http://localhost:16686 |

PostgreSQL и OTLP-порт Jaeger не опубликованы на хосте. Внутри Compose приложение использует `postgres:5432` и `jaeger:4317`.

Просмотр логов и остановка:

```bash
make logs
make down
```

`make down` сохраняет volume с данными PostgreSQL. Пароль БД задаётся при первой инициализации volume: изменение `.env` не меняет пароль уже существующего пользователя. Трассы Jaeger хранятся в памяти и не сохраняются при пересоздании его контейнера.

## API и расчёты

Контракт находится в [proto/rates.proto](proto/rates.proto). Метод: `rates.RatesService/GetRates`.

Позиции начинаются с **1**. Ask и bid рассчитываются независимо, в порядке уровней, полученных от Kraken.

| Метод | Параметры | Расчёт |
| --- | --- | --- |
| `TOP_N` | `n >= 1` | Цена на позиции N; `m` не используется |
| `AVG_NM` | `1 <= n <= m` | Среднее арифметическое цен с N по M включительно |

Позиции должны существовать в полученном стакане. Среднее округляется до восьми знаков после запятой; при `n == m` возвращается исходная цена. Нули в конце строкового представления не дополняются.

### TOP_N

```bash
grpcurl -plaintext \
  -import-path proto \
  -proto rates.proto \
  -d '{"method":"TOP_N","n":2}' \
  localhost:50051 rates.RatesService/GetRates
```

### AVG_NM

```bash
grpcurl -plaintext \
  -import-path proto \
  -proto rates.proto \
  -d '{"method":"AVG_NM","n":2,"m":4}' \
  localhost:50051 rates.RatesService/GetRates
```

Если `grpcurl` установлен через Go, но отсутствует в `PATH`, вместо `grpcurl` используйте `"$(go env GOPATH)/bin/grpcurl"`.

Ответ содержит `rate` с полями `id` (UUID), `ask`, `bid` и `receivedAt`. Время фиксируется приложением после получения HTTP-ответа Kraken; это не timestamp отдельного уровня стакана. Ответ возвращается только после успешного сохранения в БД.

Проверить сохранённую строку можно, подставив UUID из ответа:

```bash
docker compose exec postgres psql -U rates -d rates \
  -c "SELECT * FROM rates WHERE id = 'UUID_ИЗ_ОТВЕТА';"
```

Ошибки параметров возвращаются как `InvalidArgument`, отмена контекста — `Canceled`, превышение срока — `DeadlineExceeded`. Остальные ошибки возвращаются как `Internal` без внутренних подробностей; исходная причина записывается в серверный лог.

## Логи, трассировка и healthcheck

Zap пишет JSON-логи запуска, остановки и gRPC-запросов. Лог запроса содержит метод, длительность, код ответа и `trace_id`, если в контексте есть валидный span. Длительность в JSON выражена в секундах.

В Jaeger выберите сервис `usdt-rate-service` или найдите трассу по `trace_id` из лога. Успешный `GetRates` содержит три span: gRPC-запрос, `kraken.GetOrderbook` и `repository.Save`. При ошибке получения стакана span сохранения отсутствует. Отправка трасс выполняется в фоне, поэтому они могут появляться с небольшой задержкой.

Реализован стандартный RPC `grpc.health.v1.Health/Check`. Он принимает имя `rates.RatesService` или пустую строку, проверяет доступность PostgreSQL с таймаутом в одну секунду и возвращает `SERVING` либо `NOT_SERVING`. Неизвестное имя сервиса возвращает `NotFound`. Проверка не обращается к Kraken; `Watch` не реализован. Для вызова через `grpcurl` нужен стандартный `health.proto`: gRPC reflection в приложении не включён.

При SIGINT/SIGTERM сервер отмечает healthcheck как `NOT_SERVING`, ожидает завершения gRPC-запросов до пяти секунд, затем при необходимости останавливает их принудительно. На завершение экспортёра трасс выделено ещё до пяти секунд.

## Локальная разработка

Нужен Go 1.26.2 или новее. Для `make lint` дополнительно нужен `golangci-lint` v2; проект проверялся версией 2.13.2.

| Команда | Действие |
| --- | --- |
| `make build` | Собрать `bin/usdt-rate-server` |
| `make run` | Собрать и запустить сервер локально |
| `make test` | Запустить `go test ./...` |
| `make lint` | Запустить `golangci-lint run ./...` |
| `make docker-build` | Собрать образ `usdt-rate-service:local` |
| `make up` | Собрать и запустить Compose в фоне |
| `make down` | Остановить и удалить контейнеры Compose, сохранив volume БД |
| `make logs` | Следить за логами сервера Compose |

### Настройки приложения

Приоритет: флаг командной строки → переменная окружения → значение по умолчанию.

| Переменная | Флаг | Значение по умолчанию |
| --- | --- | --- |
| `DATABASE_URL` | `-database-url` | Нет, обязательная настройка |
| `GRPC_ADDR` | `-grpc-addr` | `:50051` |
| `KRAKEN_BASE_URL` | `-kraken-base-url` | `https://api.kraken.com` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Нет | Управляется OTLP-экспортёром; задавайте явно |

`POSTGRES_PASSWORD` используется Compose. Само Go-приложение читает готовый `DATABASE_URL` и не загружает `.env` автоматически. Настройки OTEL читает SDK.

Для локального запуска нужна доступная с хоста PostgreSQL с применёнными миграциями из `migrations/`. БД в текущем Compose доступна только контейнерам; `make run` сам не запускает БД и не применяет миграции.

```bash
export DATABASE_URL='postgres://USER:PASSWORD@127.0.0.1:5432/DBNAME?sslmode=disable'
export OTEL_EXPORTER_OTLP_ENDPOINT='http://127.0.0.1:4317'
make run
```

В этом примере PostgreSQL и OTLP-приёмник должны быть отдельно доступны на указанных портах. Перед локальным запуском освободите порт `50051`, если его занимает сервер Compose.

### Тесты

```bash
make test
```

Тесты HTTP-клиента используют локальные тестовые серверы и не обращаются к настоящему Kraken. Тест репозитория требует отдельной PostgreSQL с применёнными миграциями:

```bash
TEST_DATABASE_URL='postgres://USER:PASSWORD@127.0.0.1:5432/TEST_DB?sslmode=disable' \
  go test -count=1 -v ./internal/rates/repository
```

Без `TEST_DATABASE_URL` тест репозитория **пропускается**. Он создаёт записи с UUID и удаляет их после проверки. Обычный `make test` не проверяет доставку трасс в Jaeger; её можно проверить через запрос к запущенному сервису и поиск по `trace_id`.

# v0.1 — Stabilize current Deposit Manager

## Цель

Привести текущий CLI `deposit-manager` в устойчивое состояние перед крупным рефакторингом.

## Что уже есть

* CLI-команды для вкладов.
* Модель `Deposit`.
* Расчет доходности.
* Начисление процентов для `savings`.
* JSON-хранилище.

## TODO

### CLI

* [x] Проверить все текущие команды:

  * [x] `list`
  * [x] `create`
  * [x] `topup`
  * [x] `calculate`
  * [x] `update`
  * [x] `accrue-interest`
  * [x] `find`
  * [x] `help`
* [x] Убрать из help команды, которые реально не реализованы.
* [x] Добавить команду `version`.
* [x] Добавить команду `doctor` для проверки файлов данных и конфига.
* [x] Добавить более понятные ошибки для пользователя.

### Деньги и ставки

* [x] Проверить все места, где суммы переводятся из рублей в копейки.
* [x] Заменить `int` на `int64` для денежных значений.
* [x] Подготовить миграцию ставок с `float64` на decimal/basis-points helpers.
* [x] Добавить helper:

```go
func RubToKopecks(input string) (int64, error)
func KopecksToRubString(amount int64) string
func PercentToBps(input string) (int64, error)
func BpsToPercentString(bps int64) string
```

### Тесты

* [x] Добавить unit-тесты на парсинг суммы.
* [x] Добавить unit-тесты на парсинг ставки.
* [x] Добавить unit-тесты на расчет ежедневного дохода.
* [x] Добавить table-driven tests для разных сценариев:

  * [x] обычная ставка
  * [x] промо-ставка
  * [x] окончание промо-периода
  * [x] нулевая сумма
  * [x] отрицательная сумма
  * [x] високосный год

### Данные

* [x] Добавить backup перед записью JSON.
* [x] Добавить проверку целостности JSON-файлов.
* [x] Добавить команду `export`.
* [x] Добавить команду `backup`.

## Acceptance Criteria

* [x] Проверенные CLI-команды работают без panic (`version`, `help`, `doctor`).
* [x] Все денежные значения внутри бизнес-логики используют `int64`.
* [x] Покрыты тестами базовые расчеты процентов.
* [x] Есть резервная копия данных перед изменением файлов.
* [x] CLI можно использовать как раньше.

---

# v0.2 — Core MVP: Accounts, Transactions, Balances, Interest Rules

## Цель

Сделать минимальное надежное ядро продукта: счет + операция + правило процентов + расчет баланса из операций.

Это главный MVP до PostgreSQL, API, WebUI, budgeting и LLM.

## Главная идея

`Deposit` должен стать частным случаем `Account`, а баланс должен объясняться списком операций.

## Новые сущности

### Account

* [x] Добавить модель `Account`.

```go
type Account struct {
    ID        string
    Name      string
    Bank      string
    Type      AccountType
    Currency  string
    IsActive  bool
    OpenedAt  time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

Типы счетов:

* [x] `cash`
* [x] `card`
* [x] `savings`
* [x] `term_deposit`
* [x] `broker`
* [x] `other`

### Transaction

* [x] Добавить модель `Transaction`.

```go
type Transaction struct {
    ID               string
    AccountID        string
    RelatedAccountID *string
    Type             TransactionType
    AmountMinor      int64
    CategoryID       *string
    Description      string
    OccurredAt       time.Time
    CreatedAt        time.Time
}
```

Типы операций:

* [x] `initial_balance`
* [x] `income`
* [x] `expense`
* [x] `transfer_in`
* [x] `transfer_out`
* [x] `interest_income`
* [x] `adjustment`

### InterestRule

* [x] Добавить модель `InterestRule`.

```go
type InterestRule struct {
    ID                      string
    AccountID               string
    AnnualRateBps           int64
    PromoRateBps            *int64
    PromoEndDate            *time.Time
    AccrualFrequency        AccrualFrequency
    CapitalizationFrequency CapitalizationFrequency
    DayCountConvention      DayCountConvention
    IsActive                bool
    StartDate               time.Time
    EndDate                 *time.Time
}
```

Поддерживаемые значения:

* [x] `accrual_frequency = daily | monthly | end_of_term`
* [x] `capitalization_frequency = daily | monthly | end_of_term | none`
* [x] `day_count_convention = actual_365 | actual_366 | actual_actual`

### Category

* [x] Добавить модель `Category`.

Категории по умолчанию:

* [x] Зарплата
* [x] Проценты по вкладам
* [x] Еда
* [x] Транспорт
* [x] Подписки
* [x] Жилье
* [x] Здоровье
* [x] Обучение
* [x] Инвестиции
* [x] Финансовая подушка
* [x] Развлечения
* [x] Прочее

## Сервисы

* [x] `AccountService`
* [x] `TransactionService`
* [x] `TransferService`
* [x] `InterestRuleService`
* [x] `BalanceService`

## Правила

* [x] Баланс счета считается из операций.
* [x] Перевод создает две операции:

  * [x] `transfer_out` на исходном счете
  * [x] `transfer_in` на целевом счете
* [x] Начисление процентов создает операцию `interest_income`.
* [x] Ручная правка баланса создает `adjustment`.

## Acceptance Criteria

* [x] Можно создать `Account` без WebUI.
* [x] Можно создать `Transaction` без WebUI.
* [x] Можно посчитать баланс счета через список транзакций.
* [x] Можно создать накопительный счет с правилом процентов.
* [x] Можно начислить проценты как отдельную транзакцию.
* [x] Повторное начисление за тот же день не создает дубль на уровне сервиса.
* [x] Старый `Deposit` еще не удален, но новая модель уже работает отдельно.
* [x] Написаны unit-тесты на баланс.
* [x] Написаны unit-тесты на проценты через `InterestRule`.

---

# v0.3 — PostgreSQL Storage

## Цель

Перейти с JSON-файлов на PostgreSQL как основное хранилище.

## Зависимости

* [x] Добавить `pgx`.
* [x] Добавить инструмент миграций:

  * [x] `goose`
  * или [ ] `golang-migrate`
* [x] Добавить Docker Compose для локального PostgreSQL.

## Таблицы

* [x] `accounts`
* [x] `transactions`
* [x] `categories`
* [x] `interest_rules`
* [x] `interest_accruals`
* [x] `balance_snapshots`
* [x] `settings`

## Миграции

* [x] `000002_create_accounts.sql`
* [x] `000003_create_categories.sql`
* [x] `000004_create_transactions.sql`
* [x] `000005_create_interest_rules.sql`
* [x] `000006_create_interest_accruals.sql`
* [x] `000007_create_balance_snapshots.sql`
* [x] `000008_create_settings.sql`

## Индексы

* [x] `transactions(account_id, occurred_at)`
* [x] `transactions(type)`
* [x] `interest_rules(account_id, is_active)`
* [x] `interest_accruals(account_id, accrual_date)`
* [x] unique index для защиты от повторного начисления:

```sql
UNIQUE(account_id, accrual_date, rule_id)
```

## Репозитории

* [x] `AccountRepository`
* [x] `TransactionRepository`
* [x] `CategoryRepository`
* [x] `InterestRuleRepository`
* [x] `InterestAccrualRepository`

## Миграция старых JSON-данных

* [x] Написать команду:

```bash
capitalflow migrate-json
```

Она должна:

* [x] прочитать текущие вклады из JSON
* [x] создать `accounts`
* [x] создать `interest_rules`
* [x] создать `initial_balance` транзакции
* [x] сохранить старые ID как `legacy_id`
* [x] сформировать отчет миграции

## Acceptance Criteria

* [x] PostgreSQL поднимается через Docker Compose.
* [x] Миграции применяются одной командой.
* [x] Старые JSON-вклады переносятся без потери сумм.
* [x] Балансы после миграции совпадают со старыми значениями.
* [x] JSON-хранилище больше не является основным источником данных.

---

# v0.4 — Thin HTTP API

## Цель

Добавить минимальный backend API для будущего WebUI.

API на этом этапе не должен закрывать все будущие сценарии. Он должен показать core-flow: счета, операции, переводы, баланс, ручное начисление процентов.

## Стек

* [x] Go
* [x] `chi`
* [x] `slog`
* [x] `pgx`
* [x] PostgreSQL

## Структура

```text
cmd/server/main.go
internal/http/handlers
internal/http/middleware
internal/http/dto
internal/service
internal/repository
internal/domain
```

## Endpoints

### Health

* [x] `GET /health`
* [x] `GET /ready`

### Accounts

* [x] `GET /api/accounts`
* [x] `POST /api/accounts`
* [x] `GET /api/accounts/{id}`
* [x] `PATCH /api/accounts/{id}`
* [x] `POST /api/accounts/{id}/archive`

### Transactions

* [x] `GET /api/transactions`
* [x] `POST /api/transactions`
* [x] `GET /api/transactions/{id}`
* [x] `DELETE /api/transactions/{id}`
* [x] Добавить pagination для `GET /api/transactions` (`limit`, `cursor` или `page`).
* [x] Добавить server-side filtering для `GET /api/transactions`:

  * [x] счет
  * [x] категория
  * [x] тип операции
  * [x] период дат
  * [x] поиск по описанию

### Transfers

* [x] `POST /api/transfers`

### Interest Rules

* [x] `GET /api/accounts/{id}/interest-rules`
* [x] `POST /api/accounts/{id}/interest-rules`
* [x] `PATCH /api/interest-rules/{id}`
* [x] `POST /api/accounts/{id}/accrue-interest`
* [x] `POST /api/accounts/{id}/recalculate-interest`

### Dashboard

* [x] `GET /api/dashboard/summary`
* [x] `GET /api/dashboard/net-worth`
* [x] `GET /api/dashboard/cashflow`
* [x] `GET /api/dashboard/interest-income`

### API Contract

* [x] Добавить OpenAPI spec для всех `/api/*` endpoints.
* [x] Описать DTO, ошибки, auth, pagination и filtering.
* [x] Добавить проверку OpenAPI spec в CI после стабилизации контракта.

Не делать до стабильного core:

* [ ] advanced reports
* [ ] smart budget recommendations
* [ ] LLM insights
* [ ] full import/export UI

## Ошибки API

Единый формат ошибки:

```json
{
  "error": {
    "code": "validation_error",
    "message": "Invalid amount",
    "details": {}
  }
}
```

## Acceptance Criteria

* [x] API запускается локально.
* [x] Можно создать счет через API.
* [x] Можно создать транзакцию через API.
* [x] Можно получить баланс через API.
* [x] Можно начислить проценты через API.
* [x] Ошибки возвращаются в едином формате.
* [x] API покрывает только core-flow и не раздувается до analytics suite.

---

# v0.5 — WebUI MVP

## Цель

Сделать первый рабочий WebUI поверх API.

Первый экран должен быть рабочим инструментом, а не landing page. WebUI MVP показывает только то, что доказывает core value: счета, балансы, операции, проценты и последние изменения.

## Рекомендуемый стек

Вариант для полноценного pet-project:

* [x] React
* [x] Vite
* [x] TypeScript
* [x] TanStack Query
* [x] shadcn-style local components
* [x] Recharts

Упрощенный вариант:

* [ ] Go templates
* [ ] HTMX
* [ ] Alpine.js

Рекомендуемый выбор: `React + Vite + TypeScript`, потому что это лучше покажет full-stack развитие проекта.

## Страницы

### Product UX

* [x] Добавить переключение светлой/темной темы.
* [x] Сделать современное premium-оформление для dashboard, таблиц, форм и empty states.
* [x] Сохранить выбранную тему в `localStorage`.
* [x] Проверить контраст, hover/focus states и mobile layout.

### Frontend Architecture

* [x] Разбить `web/src/App.tsx` на feature-модули:

  * [x] `features/dashboard`
  * [x] `features/accounts`
  * [x] `features/transactions`
  * [x] `shared/ui`
  * [x] `shared/api`
* [x] Оставить `App.tsx` только для layout, routing/view state и composition.
* [x] Не менять поведение при рефакторинге без отдельной задачи.

### Dashboard

* [x] Общий капитал.
* [x] Список счетов с балансами.
* [x] Доходы за месяц.
* [x] Расходы за месяц.
* [x] Проценты за месяц.
* [x] Последние операции.
* [x] Быстрые действия:

  * [x] добавить доход
  * [x] добавить расход
  * [x] сделать перевод
  * [x] создать счет

### Accounts

* [x] Таблица счетов.
* [x] Фильтр по типу счета.
* [x] Баланс.
* [x] Банк.
* [x] Ставка, если есть.
* [x] Статус счета.

### Account Details

* [x] Карточка счета.
* [x] Текущий баланс.
* [x] История операций.
* [x] График баланса.
* [x] Правила процентов.
* [x] Кнопка ручного начисления процентов.

### Transactions

* [x] Таблица операций.
* [x] Фильтр по дате.
* [x] Фильтр по счету.
* [x] Фильтр по категории.
* [x] Фильтр по типу операции.

### Create Transaction

* [x] Доход.
* [x] Расход.
* [x] Перевод.
* [x] Корректировка.

### Create Account

* [x] Название.
* [x] Тип.
* [x] Банк.
* [x] Валюта.
* [x] Начальный баланс.
* [x] Ставка.
* [x] Капитализация.
* [x] Промо-ставка.
* [x] Дата окончания промо.

## Acceptance Criteria

* [x] WebUI открывается локально.
* [x] Можно создать счет через форму.
* [x] Можно добавить доход/расход.
* [x] Можно сделать перевод между счетами.
* [x] Можно увидеть балансы.
* [x] Можно увидеть последние транзакции.
* [x] Можно увидеть, из каких операций получился баланс.
* [x] Можно увидеть начисленные проценты по счету.
* [x] Нет smart budget, goals и LLM в первом WebUI MVP.
* [x] WebUI проходит CI job: `npm ci`, `npm run lint`, `npm run build`.

---

# v0.5.1 — Auth & Secure Local User

## Цель

Сделать вход пользователя в сам CapitalFlow, чтобы дальнейшая работа с данными шла через личную сессию, а не через общий Bearer token.

## Security Requirements

* [x] Регистрация пользователя при первом заходе в сервис.
* [x] После первого пользователя закрыть публичную регистрацию или требовать admin invite/setup token.
* [x] Хешировать пароль через `argon2id`.
* [x] Не хранить plaintext password, reset tokens или JWT secrets в репозитории.
* [x] Использовать access JWT с коротким TTL.
* [x] Использовать refresh token с rotation и server-side revocation.
* [x] Хранить refresh token безопасно: httpOnly cookie или hashed token в БД.
* [x] Добавить logout с отзывом refresh token.
* [x] Добавить защиту от brute force:

  * [x] rate limit на login/register
  * [x] одинаковые сообщения для неверного email/password
  * [x] audit log для auth-событий
* [x] Продумать CSRF модель, если refresh хранится в cookie.
* [x] Не отдавать чувствительные auth-ошибки в UI.

## Backend

* [x] Таблица пользователей.
* [x] Таблица refresh sessions/tokens.
* [x] `POST /auth/setup` для первого пользователя.
* [x] `POST /auth/login`.
* [x] `POST /auth/refresh`.
* [x] `POST /auth/logout`.
* [x] Middleware auth через JWT claims.
* [x] Основная валюта пользователя хранится в профиле.
* [x] Привязать пользовательские данные к owner/user id до multi-user сценариев.

## Frontend

* [x] Первый экран setup/register, если пользователей нет.
* [x] Login screen.
* [x] Session bootstrap при открытии приложения.
* [x] Авто-refresh access token.
* [x] Выбор основной валюты при setup/register.
* [x] Настройка основной валюты в Settings.
* [x] Ясные, но безопасные сообщения об ошибках входа.

## Acceptance Criteria

* [x] Новый пользователь может настроить сервис при первом запуске.
* [x] После setup dashboard доступен только после login.
* [x] Пароли хранятся только как Argon2id hash.
* [x] JWT нельзя использовать после logout/refresh rotation revoke.
* [x] Auth покрыт unit и handler tests.

---

---

# v0.5.2 — Auth Security Hardening

## Цель

Довести auth-систему до production-grade security baseline:
защитить refresh flow, сессии, password policy, audit trail и observability.

## Security Hardening

* [x] Reuse Detection для refresh token
* [x] Политика сложности пароля (`zxcvbn`)
* [x] Account lockout с нарастающей задержкой
* [x] Смена пароля + выход со всех устройств
* [x] Управление сессиями (список, отзыв)
* [x] Подготовка email-поля и верификации (схема)
* [x] Audit log таблица и запись всех событий
* [x] Secure cookie:
  * [x] `Secure`
  * [x] `HttpOnly`
  * [x] `SameSite`
  * [x] `Path`
* [x] Middleware JWT -> `userID` в context
* [x] Unit + handler + security tests (включая reuse)
* [ ] Observability:
  * [x] метрики для auth
  * [x] алерты для auth incidents
* [x] Документация:
  * [x] Security Model
  * [x] Runbook
  * [x] ADR

## Acceptance Criteria

* [x] Reused refresh token немедленно инвалидирует session family
* [x] Password policy блокирует слабые и компрометированные пароли
* [x] Suspicious login attempts приводят к progressive lockout
* [x] Пользователь может завершить все активные сессии
* [x] Все auth-события попадают в audit log
* [x] Auth security покрыт тестами и метриками
* [x] Есть документация для эксплуатации и incident response

---

# v0.5.3 — Passkey Login / WebAuthn

## Цель

Добавить вход по passkey как более безопасный и удобный способ авторизации поверх уже существующей auth-системы.

На этом этапе passkey не должен ломать текущий password login. Сначала passkey добавляется как дополнительный способ входа для существующего пользователя, а password login остается fallback-механизмом до появления полноценного recovery-flow.

## Security Requirements

* [ ] Использовать WebAuthn / passkeys через `PublicKeyCredential`.
* [ ] Не хранить приватные ключи пользователя на сервере.
* [ ] Хранить только публичный ключ credential, credential ID, user ID, sign counter и технические metadata.
* [ ] Привязать passkey к конкретному `rpID` и разрешенным origins.
* [ ] Генерировать challenge только на backend.
* [ ] Challenge должен быть одноразовым и иметь короткий TTL.
* [ ] Нельзя повторно использовать старый challenge.
* [ ] Нельзя зарегистрировать passkey без активной authenticated session.
* [ ] Для добавления первого passkey к существующему аккаунту требовать повторное подтверждение пароля или свежую сессию.
* [ ] Поддержать несколько passkeys на одного пользователя.
* [ ] Добавить удаление passkey из Settings.
* [ ] Добавить audit log для passkey-событий:

  * [ ] registration started
  * [ ] registration completed
  * [ ] registration failed
  * [ ] login completed
  * [ ] login failed
  * [ ] credential removed
* [ ] Добавить rate limit на passkey registration/login endpoints.
* [ ] В production требовать HTTPS; `localhost` разрешить только для dev.
* [ ] Не раскрывать в UI чувствительные причины ошибки WebAuthn.

## Backend

* [ ] Добавить WebAuthn config:

```go
type WebAuthnConfig struct {
    RPID           string
    RPName         string
    AllowedOrigins []string
}
```

* [ ] Добавить таблицу `passkey_credentials`.

Пример полей:

```sql
id
user_id
credential_id
public_key
sign_count
aaguid
name
transports
backup_eligible
backup_state
created_at
last_used_at
revoked_at
```

* [ ] Добавить таблицу или storage для одноразовых WebAuthn challenges.
* [ ] Добавить repository для passkey credentials.
* [ ] Добавить service для registration flow.
* [ ] Добавить service для authentication flow.
* [ ] Интегрировать успешный passkey login в текущий access/refresh token flow.
* [ ] После passkey login создавать обычную refresh session.
* [ ] При logout/revoke sessions поведение должно остаться единым для password и passkey login.

## API Endpoints

### Passkey Registration

* [ ] `POST /auth/passkeys/register/options`
* [ ] `POST /auth/passkeys/register/verify`

### Passkey Login

* [ ] `POST /auth/passkeys/login/options`
* [ ] `POST /auth/passkeys/login/verify`

### Passkey Management

* [ ] `GET /auth/passkeys`
* [ ] `PATCH /auth/passkeys/{id}` для переименования passkey.
* [ ] `DELETE /auth/passkeys/{id}` для удаления passkey.

## Frontend

* [ ] Добавить кнопку `Sign in with passkey` на login screen.
* [ ] Добавить блок `Settings -> Security -> Passkeys`.
* [ ] Добавить кнопку `Add passkey`.
* [ ] Показать список passkeys пользователя.
* [ ] Добавить rename passkey.
* [ ] Добавить delete passkey.
* [ ] Добавить fallback-сообщение, если браузер не поддерживает passkeys.
* [ ] Добавить понятные, но безопасные ошибки:

  * [ ] passkey cancelled
  * [ ] browser not supported
  * [ ] credential not found
  * [ ] login failed

## Tests

* [ ] Unit tests для challenge lifecycle.
* [ ] Unit tests для credential storage.
* [ ] Handler tests для registration options.
* [ ] Handler tests для registration verify.
* [ ] Handler tests для login options.
* [ ] Handler tests для login verify.
* [ ] Security tests:

  * [ ] replayed challenge rejected
  * [ ] expired challenge rejected
  * [ ] wrong origin rejected
  * [ ] wrong rpID rejected
  * [ ] revoked credential rejected
  * [ ] credential from another user rejected

## Acceptance Criteria

* [ ] Пользователь может добавить passkey в Settings.
* [ ] Пользователь может войти через passkey без ввода пароля.
* [ ] Password login остается рабочим fallback-способом входа.
* [ ] Один пользователь может иметь несколько passkeys.
* [ ] Пользователь может удалить passkey.
* [ ] Успешный passkey login создает обычную refresh session.
* [ ] Passkey-события попадают в audit log.
* [ ] Повторный, просроченный или чужой challenge отклоняется.
* [ ] Passkey flow покрыт unit, handler и security tests.

---

# v0.5.4 — E2E Testing Baseline

## Цель

Добавить end-to-end тестирование, которое проверяет реальные пользовательские сценарии через браузер: от открытия WebUI до изменения данных через backend и PostgreSQL.

E2E не заменяет unit, handler и integration tests. Его задача — проверять, что основные product flows работают вместе: frontend, API, auth, database и routing.

## Стек

* [ ] Playwright.
* [ ] TypeScript.
* [ ] Chromium как обязательный browser target.
* [ ] Firefox/WebKit как optional browser targets после стабилизации.
* [ ] Docker Compose для тестовой PostgreSQL.
* [ ] Отдельная test database.

## Test Environment

* [ ] Добавить `docker-compose.e2e.yml`.
* [ ] Поднимать PostgreSQL для E2E отдельно от dev DB.
* [ ] Применять миграции перед запуском E2E.
* [ ] Очищать test DB перед каждым test suite или worker.
* [ ] Добавить seed для базовых категорий.
* [ ] Запускать backend в `e2e`/`test` mode.
* [ ] Запускать WebUI с `VITE_API_URL`, указывающим на test backend.
* [ ] Не использовать production secrets.
* [ ] Добавить стабильные test users через setup helper.

## Scripts

```json
{
  "test:e2e": "playwright test",
  "test:e2e:ui": "playwright test --ui",
  "test:e2e:headed": "playwright test --headed",
  "test:e2e:report": "playwright show-report"
}
```

## CI

* [ ] Добавить отдельный CI job `e2e`.
* [ ] Запускать backend tests, frontend tests и E2E отдельными checks.
* [ ] Сохранять Playwright report как CI artifact.
* [ ] Сохранять trace/screenshot/video только при падении теста.
* [ ] E2E job должен запускаться после успешного backend/frontend build.
* [ ] Добавить timeout для E2E job.
* [ ] Не блокировать локальную разработку слишком медленными test suites.

## Test Scenarios

### Auth

* [ ] Первый setup пользователя.
* [ ] Login по email/password.
* [ ] Logout.
* [ ] Session bootstrap после reload страницы.
* [ ] Access token refresh flow.
* [ ] Redirect на login screen без активной сессии.
* [ ] Неверный пароль показывает безопасную ошибку.

### Passkey

* [ ] Добавление passkey из Settings.
* [ ] Login через passkey.
* [ ] Удаление passkey.
* [ ] Fallback на password login.
* [ ] Smoke-test passkey flow через virtual authenticator.

### Accounts

* [ ] Создание счета.
* [ ] Просмотр списка счетов.
* [ ] Открытие account details.
* [ ] Архивация счета.
* [ ] Проверка empty state без счетов.

### Transactions

* [ ] Создание income transaction.
* [ ] Создание expense transaction.
* [ ] Создание adjustment transaction.
* [ ] Удаление transaction.
* [ ] Фильтр по счету.
* [ ] Фильтр по категории.
* [ ] Фильтр по дате.
* [ ] Поиск по описанию.

### Transfers

* [ ] Перевод между двумя счетами.
* [ ] Проверка списания с исходного счета.
* [ ] Проверка зачисления на целевой счет.
* [ ] Проверка истории операций после transfer.
* [ ] Ошибка при переводе на тот же счет.

### Interest / Deposits

* [ ] Создание накопительного счета со ставкой.
* [ ] Ручное начисление процентов.
* [ ] Повторное начисление за тот же день не создает дубль.
* [ ] Forecast отображается в UI.

### Dashboard

* [ ] Net worth обновляется после создания счета.
* [ ] Доходы/расходы за месяц обновляются после операций.
* [ ] Последние операции отображаются после создания transaction.
* [ ] Быстрые действия открывают нужные формы.

### UI Stability

* [ ] Переключение light/dark theme.
* [ ] Theme сохраняется после reload.
* [ ] Основные страницы не имеют critical console errors.
* [ ] Mobile viewport smoke test для dashboard и transactions.

## Test Data Rules

* [ ] Каждый тест создает свои данные или использует изолированный seed.
* [ ] Тесты не зависят от порядка выполнения.
* [ ] Тесты можно запускать параллельно после стабилизации isolation.
* [ ] Деньги в тестах проверяются через точные значения minor units.
* [ ] Даты фиксируются через controlled clock/test helpers там, где это возможно.

## Acceptance Criteria

* [ ] `npm run test:e2e` запускает E2E локально.
* [ ] E2E поднимает или использует отдельную test database.
* [ ] CI имеет отдельный `e2e` check.
* [ ] Покрыты P0 flows: setup, login, account, transaction, transfer, dashboard.
* [ ] Passkey flow покрыт smoke E2E через virtual authenticator.
* [ ] При падении тестов сохраняется Playwright report.
* [ ] E2E тесты не используют production secrets и production database.
* [ ] Добавлена документация `docs/testing/e2e.md`.

---

# v0.6 — Deposit & Capitalization Engine

## Цель

Сделать сильное ядро расчета процентов и капитализации.

## Сценарии

### Накопительный счет

* [x] Ставка годовая.
* [x] Расчет каждый день.
* [x] Начисление каждый день.
* [x] Капитализация каждый день.

### Срочный вклад

* [x] Ставка годовая.
* [x] Срок вклада.
* [ ] Пополнение до определенной даты.
* [ ] Начисление процентов:

  * [x] ежедневно
  * [x] ежемесячно
  * [x] в конце срока
* [x] Капитализация:

  * [x] ежедневно
  * [x] ежемесячно
  * [x] в конце срока
  * [x] без капитализации

### Промо-ставка

* [x] Промо-ставка до даты.
* [x] Базовая ставка после промо.
* [x] Корректное разбиение периода расчета на части.

## Идемпотентность

* [x] Нельзя дважды начислить проценты за один день по одному правилу.
* [ ] Повторный запуск job должен быть безопасным.
* [x] Должна быть таблица `interest_accruals`.

## Фоновые задачи

* [ ] `daily_interest_accrual_job`
* [ ] `monthly_interest_accrual_job`
* [ ] `deposit_maturity_check_job`

## Команды

* [x] `capitalflow accrue --date YYYY-MM-DD`
* [x] `capitalflow accrue --account <id>`
* [x] `capitalflow forecast --account <id> --days 365`
* [x] `capitalflow recalculate --account <id> --from YYYY-MM-DD`

## Acceptance Criteria

* [x] Яндекс-like накопительный счет с 12% и daily capitalization считается корректно.
* [x] Альфа-like накопительный счет с 10% считается корректно.
* [x] Повторное начисление за тот же день не создает дубль.
* [x] Все начисления видны в истории операций.
* [x] Можно построить прогноз на 30/90/365 дней.

---

# v0.7 — Manual Diversification / Allocation Calculator

## Цель

Добавить ручной калькулятор распределения дохода: пользователь вводит сумму дохода, а система показывает, сколько куда нужно отправить по заданным процентам.

Название в коде лучше сделать не `diversion`, а `allocation` или `income_distribution`.

## Пример сценария

Пользователь вводит:

```text
Income: 100000 RUB
```

Система показывает:

```text
50% обязательные расходы: 50000 RUB
20% финансовая подушка: 20000 RUB
15% инвестиции: 15000 RUB
10% обучение: 10000 RUB
5% развлечения: 5000 RUB
```

## Сущности

### AllocationPreset

```go
type AllocationPreset struct {
    ID          string
    Name        string
    Description string
    IsDefault   bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### AllocationRule

```go
type AllocationRule struct {
    ID              string
    PresetID        string
    TargetCategoryID string
    PercentBps      int64
    Priority         int
}
```

### AllocationSimulation

```go
type AllocationSimulation struct {
    IncomeAmountMinor int64
    PresetID          string
    Results           []AllocationResult
}
```

## WebUI

Страница: `Allocation Calculator`

* [ ] Поле ввода дохода.
* [ ] Выбор пресета.
* [ ] Таблица распределения.
* [ ] Кнопка “создать план операций”.
* [ ] Кнопка “применить как транзакции”.
* [ ] Настройка процентов.

## Пресеты

* [ ] `Balanced`
* [ ] `Aggressive Saving`
* [ ] `Debt Payoff`
* [ ] `Low Income Survival`
* [ ] `Custom`

## Валидация

* [ ] Сумма процентов должна быть 100%.
* [ ] Доход должен быть положительным.
* [ ] Все категории должны существовать.

## Acceptance Criteria

* [ ] Пользователь вводит доход.
* [ ] Система показывает распределение по категориям.
* [ ] Пользователь может настроить проценты.
* [ ] Пользователь может сохранить пресет.
* [ ] Пользователь может создать план операций на основе результата.

---

# v0.8 — Smart Budget Calculator

## Цель

Добавить умную версию калькулятора, которая анализирует реальные расходы пользователя и предлагает более реалистичное распределение дохода.

## Идея

Ручной калькулятор работает по фиксированным процентам.

Умный калькулятор должен учитывать:

* сколько пользователь реально тратит в месяц
* какие категории обязательные
* какие категории можно сократить
* сколько уже лежит в финансовой подушке
* есть ли цель накопления
* какой доход был за последние месяцы
* какие расходы регулярные

## Входные данные

* [ ] Доход пользователя за месяц.
* [ ] Средние расходы по категориям за 1/3/6 месяцев.
* [ ] Текущие балансы счетов.
* [ ] Цель по финансовой подушке.
* [ ] Цель инвестирования.
* [ ] Минимальные обязательные расходы.

## Расчетные показатели

* [ ] Средний месячный расход.
* [ ] Средний обязательный расход.
* [ ] Средний необязательный расход.
* [ ] Savings rate.
* [ ] Emergency fund coverage.
* [ ] Процент дохода, уходящий на каждую категорию.
* [ ] Отклонение от желаемого бюджета.

## Логика рекомендаций

Система должна уметь:

* [ ] определить базовый обязательный минимум
* [ ] рассчитать, сколько можно безопасно отложить
* [ ] предложить сумму в финансовую подушку
* [ ] предложить сумму в инвестиции
* [ ] предупредить о слишком больших тратах в категории
* [ ] предложить лимит на категорию в следующем месяце

## Пример вывода

```text
Доход: 100000 RUB
Средние обязательные расходы: 62000 RUB
Средние необязательные расходы: 18000 RUB
Свободный остаток: 20000 RUB

Рекомендация:
- 62000 RUB оставить на обязательные расходы
- 12000 RUB отправить в финансовую подушку
- 5000 RUB отправить в инвестиции
- 3000 RUB оставить на развлечения

Комментарий:
Категория “Еда вне дома” выше среднего на 23%.
Рекомендуемый лимит на следующий месяц: 7000 RUB.
```

## WebUI

Страница: `Smart Budget`

* [ ] Выбор периода анализа.
* [ ] Ввод ожидаемого дохода.
* [ ] Автоматический расчет средних расходов.
* [ ] Карточки рекомендаций.
* [ ] График расходов по категориям.
* [ ] Сравнение текущего бюджета с рекомендуемым.
* [ ] Кнопка “создать бюджет на месяц”.

## Acceptance Criteria

* [ ] Система считает средние расходы по категориям.
* [ ] Система предлагает распределение дохода на основе истории.
* [ ] Пользователь видит, какие категории тянут бюджет вниз.
* [ ] Пользователь может создать месячный бюджет из рекомендации.

---

# v0.9 — Budgeting & Goals

## Цель

Добавить полноценное планирование бюджета и финансовые цели.

## Budget

* [ ] Создание бюджета на месяц.
* [ ] Лимиты по категориям.
* [ ] Отображение прогресса.
* [ ] Перенос остатка лимита на следующий месяц.
* [ ] Предупреждение при превышении лимита.

## Goals

Цели:

* [ ] Финансовая подушка.
* [ ] Инвестиции.
* [ ] Крупная покупка.
* [ ] Отпуск.
* [ ] Обучение.

Поля цели:

* [ ] название
* [ ] целевая сумма
* [ ] текущая сумма
* [ ] дедлайн
* [ ] связанный счет
* [ ] ежемесячный план пополнения

## WebUI

* [ ] Страница бюджетов.
* [ ] Страница целей.
* [ ] Виджет прогресса целей на dashboard.
* [ ] Рекомендации по ежемесячному пополнению цели.

## Acceptance Criteria

* [ ] Можно создать бюджет на месяц.
* [ ] Можно задать лимиты по категориям.
* [ ] Можно создать финансовую цель.
* [ ] Dashboard показывает прогресс целей.

---

# v0.10 — Analytics, Reports, Forecasts

## Цель

Сделать систему полезной для анализа, а не только для ввода данных.

## Отчеты

* [ ] Доходы и расходы по месяцам.
* [ ] Расходы по категориям.
* [ ] Рост капитала.
* [ ] Доход от процентов.
* [ ] Прогноз финансовой подушки.
* [ ] Прогноз вкладов.
* [ ] Cashflow.
* [ ] Savings rate.

## Графики

* [ ] Net worth over time.
* [ ] Category spending pie/bar chart.
* [ ] Income vs expenses.
* [ ] Interest income over time.
* [ ] Goal progress.

## Экспорт

* [ ] CSV export.
* [ ] JSON backup.
* [ ] Markdown monthly report.

## Acceptance Criteria

* [ ] Пользователь видит динамику капитала.
* [ ] Пользователь видит, какие категории самые дорогие.
* [ ] Пользователь может экспортировать отчет.
* [ ] Пользователь может увидеть прогноз на 3/6/12 месяцев.

---

# v0.11 — Import, Backup, Sync Helpers

## Цель

Упростить наполнение системы данными и защитить данные от потери.

## Import

* [ ] Импорт CSV.
* [ ] Импорт ручных таблиц.
* [ ] Настраиваемый mapping колонок:

  * [ ] дата
  * [ ] сумма
  * [ ] описание
  * [ ] категория
  * [ ] счет
* [ ] Preview перед импортом.
* [ ] Дедупликация операций.

## Backup

* [ ] Автоматический ежедневный backup БД.
* [ ] Ручной backup из WebUI.
* [ ] Restore из backup.
* [ ] Backup перед опасными операциями: import, bulk delete, restore, migrations.
* [ ] Настроить retention policy:

  * [ ] ежедневные backup за 7 дней
  * [ ] еженедельные backup за 4 недели
  * [ ] ручные backup без автоудаления
* [ ] Шифровать backup или хранить в защищенной директории.
* [ ] Проверять restore на тестовой БД.
* [ ] Документировать команды backup/restore.

## NixOS integration

* [ ] devShell.
* [ ] systemd service.
* [ ] local reverse proxy option.
* [ ] backup timer.

## Docker

* [ ] Добавить Dockerfile для backend.
* [ ] Добавить Dockerfile для web.
* [ ] Добавить production docker-compose для backend + web + PostgreSQL.
* [ ] Добавить healthcheck для backend container.
* [ ] Не запекать secrets в image.

## Acceptance Criteria

* [ ] Можно импортировать CSV.
* [ ] Можно сделать backup через UI.
* [ ] Можно восстановить данные из backup.
* [ ] Приложение можно запустить как сервис на NixOS.

---

# v0.12 — LLM Assistant Foundation

## Цель

Подготовить безопасную архитектуру для LLM-рекомендаций, но не делать LLM главным источником финансовых решений.

LLM должна объяснять, анализировать и предлагать идеи, но не должна самостоятельно менять данные без подтверждения пользователя.

## Возможные задачи LLM

* [ ] Объяснить, почему бюджет просел.
* [ ] Найти категории, где расходы выросли.
* [ ] Сформировать месячный финансовый отчет обычным языком.
* [ ] Дать рекомендации по снижению расходов.
* [ ] Объяснить прогноз по вкладам.
* [ ] Предложить распределение дохода.
* [ ] Найти аномалии в тратах.
* [ ] Ответить на вопрос пользователя по его данным.

## Ограничения

* [ ] LLM не должна иметь прямой доступ к сырой БД.
* [ ] LLM получает только подготовленный безопасный summary.
* [ ] Пользователь должен явно включить LLM-интеграцию.
* [ ] Cloud-модели не должны получать чувствительные данные без предупреждения.
* [ ] Все советы должны сопровождаться дисклеймером: это не финансовая рекомендация.
* [ ] Любое действие по изменению данных требует подтверждения пользователя.

## LLM Provider Interface

```go
type LLMProvider interface {
    Generate(ctx context.Context, req LLMRequest) (*LLMResponse, error)
}
```

Провайдеры:

* [ ] Ollama local.
* [ ] Ollama cloud.
* [ ] OpenAI-compatible API.
* [ ] Mock provider for tests.

## Data Context Builder

* [ ] `MonthlyFinancialSummaryBuilder`
* [ ] `CategorySpendingSummaryBuilder`
* [ ] `GoalProgressSummaryBuilder`
* [ ] `DepositForecastSummaryBuilder`

LLM должна получать примерно такой контекст:

```json
{
  "period": "2026-05",
  "income_total": 100000,
  "expense_total": 72000,
  "savings_rate": 28,
  "top_categories": [
    {"name": "Food", "amount": 23000, "change_percent": 18},
    {"name": "Subscriptions", "amount": 4500, "change_percent": 0}
  ],
  "goals": [
    {"name": "Emergency Fund", "progress_percent": 22}
  ]
}
```

## WebUI

Страница: `AI Insights`

* [ ] Кнопка “Analyze my month”.
* [ ] Кнопка “Explain my spending”.
* [ ] Кнопка “Suggest next month budget”.
* [ ] История LLM-ответов.
* [ ] Настройка провайдера.
* [ ] Настройка уровня приватности.

## Acceptance Criteria

* [ ] Можно подключить mock LLM provider.
* [ ] Можно получить summary без отправки сырых транзакций.
* [ ] Можно сгенерировать месячный текстовый отчет.
* [ ] LLM не может менять данные без подтверждения.

---

# v0.13 — WebUI Design Realisation

## Цель

Перенести конкретный roadmap из `DESIGN.md` в рабочий TODO для реализации Nordic WebUI, не меняя уже завершенную историю `v0.5`.

## PR Roadmap

### PR 1 — Web App Shell

Branch:

```text
feature/web-app-shell
```

Scope:

* [ ] AppShell.
* [ ] Sidebar.
* [ ] Topbar.
* [ ] Page container.
* [ ] Card component.
* [ ] Button component.
* [ ] CSS variables / design tokens.
* [ ] Light Nordic theme.
* [ ] Basic responsive behavior.

### PR 2 — Dashboard Foundation

Scope:

* [ ] Net worth summary.
* [ ] Metric cards.
* [ ] Account cards.
* [ ] Quick actions.
* [ ] Recent transactions block.
* [ ] Static chart placeholders.
* [ ] Loading/empty/error states.

### PR 3 — Transactions UX

Scope:

* [ ] Transaction list/table.
* [ ] Search input.
* [ ] Filter chips.
* [ ] Status chips.
* [ ] Mobile transaction layout.
* [ ] Empty filtered state.

### PR 4 — Transfers UX

Scope:

* [ ] Transfer form.
* [ ] Account selectors.
* [ ] Validation.
* [ ] Review step.
* [ ] Submit state.
* [ ] Success/error feedback.

### PR 5 — Savings / Deposits UX

Scope:

* [ ] Savings overview.
* [ ] APY/rate display.
* [ ] Earned interest.
* [ ] Projected value chart.
* [ ] Goal progress.
* [ ] Deposit/withdraw actions.

## Acceptance Criteria

* [ ] TODO reflects `DESIGN.md` implementation priority.
* [ ] Completed `v0.5` checklist stays unchanged.
* [ ] New tasks are unchecked and ready for feature realisation.

---

# v1.0 — Personal CapitalFlow Core Release

## Цель

Собрать стабильную локальную версию приложения, которой можно пользоваться каждый день.

## В v1.0 core должно быть

* [ ] WebUI.
* [ ] PostgreSQL.
* [ ] Счета.
* [ ] Транзакции.
* [ ] Переводы.
* [ ] Категории.
* [ ] Вклады.
* [ ] Накопительные счета.
* [ ] Капитализация.
* [ ] Отчеты.
* [ ] Backup/restore.
* [ ] Passkey login как optional secure login method.
* [ ] E2E тесты для critical user flows.
* [ ] NixOS-friendly запуск.

## После v1.0 / v1.x

* [ ] Бюджеты.
* [ ] Цели.
* [ ] Ручной калькулятор распределения дохода.
* [ ] Умный калькулятор бюджета.
* [ ] LLM insights.
* [ ] Telegram bot.
* [ ] Investments.
* [ ] Multi-currency.

## Что можно показать работодателю

* [ ] Чистая архитектура Go-проекта.
* [ ] Работа с PostgreSQL.
* [ ] Финансовая доменная модель.
* [ ] Идемпотентные background jobs.
* [ ] Безопасная работа с деньгами.
* [ ] Web API.
* [ ] WebUI.
* [ ] Миграции.
* [ ] Тестирование.
* [ ] E2E testing через браузер.
* [ ] Passkey/WebAuthn security flow.
* [ ] Docker/NixOS окружение.

## CI/CD до v1.0

* [x] Добавить frontend CI job:

  * [x] `npm ci`
  * [x] `npm run lint`
  * [x] `npm run build`
* [x] Backend CI и frontend CI должны быть отдельными checks.
* [x] Добавить OpenAPI validation check, когда spec станет обязательной.
* [ ] Добавить Playwright E2E CI job.
* [ ] Сохранять Playwright report как CI artifact.

---

# Future Ideas After v1.0

## Telegram Bot

* [ ] Дневной финансовый дайджест.
* [ ] Уведомление о начислении процентов.
* [ ] Быстрое добавление расхода.
* [ ] Уведомление о превышении бюджета.

## Investments

* [ ] Учет брокерского счета.
* [ ] Акции.
* [ ] ETF/фонды.
* [ ] Дивиденды.
* [ ] Доходность портфеля.

## Multi-currency

* [ ] RUB.
* [ ] USD.
* [ ] EUR.
* [ ] USDT.
* [x] Fiat FX endpoint for ISO currency rates.
* [x] Cross-currency transfer conversion.
* [x] Dashboard base-currency tabs and converted totals.
* [ ] Persist historical FX rates for repeatable reports.
* [ ] Курсы валют.
* [ ] Курсы криптовалют.
* [ ] Курсы металлов.
* [ ] Переоценка капитала.

## HomeLab Integration

* [ ] Запуск в Docker.
* [ ] Nginx reverse proxy.
* [ ] Auth proxy.
* [ ] Grafana metrics.
* [ ] Prometheus endpoint.

# Notes

* `Actual Budget` можно использовать как источник идей по UX и budget flow, но не нужно копировать его архитектуру один в один.
* Главная уникальность проекта: учет вкладов, накопительных счетов, капитализации, объяснимых балансов и приватной аналитики.
* Сначала проект должен стать надежным инструментом для личного учета, а уже потом красивым приложением.
* Не позиционировать проект как startup до проверки ICP на 5-10 реальных пользователях.
* Allocation, smart budget и LLM остаются важными, но идут после надежного core.

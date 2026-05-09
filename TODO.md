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
* [ ] Добавить pagination для `GET /api/transactions` (`limit`, `cursor` или `page`).
* [ ] Добавить server-side filtering для `GET /api/transactions`:

  * [ ] счет
  * [ ] категория
  * [ ] тип операции
  * [ ] период дат
  * [ ] поиск по описанию

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

* [ ] Добавить OpenAPI spec для всех `/api/*` endpoints.
* [ ] Описать DTO, ошибки, auth, pagination и filtering.
* [ ] Добавить проверку OpenAPI spec в CI после стабилизации контракта.

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

* [ ] Добавить переключение светлой/темной темы.
* [ ] Сделать современное premium-оформление для dashboard, таблиц, форм и empty states.
* [ ] Сохранить выбранную тему в `localStorage`.
* [ ] Проверить контраст, hover/focus states и mobile layout.

### Frontend Architecture

* [ ] Разбить `web/src/App.tsx` на feature-модули:

  * [ ] `features/dashboard`
  * [ ] `features/accounts`
  * [ ] `features/transactions`
  * [ ] `shared/ui`
  * [ ] `shared/api`
* [ ] Оставить `App.tsx` только для layout, routing/view state и composition.
* [ ] Не менять поведение при рефакторинге без отдельной задачи.

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
* [ ] Промо-ставка.
* [ ] Дата окончания промо.

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
* [ ] WebUI проходит CI job: `npm ci`, `npm run lint`, `npm run build`.

---

# v0.5.1 — Auth & Secure Local User

## Цель

Сделать вход пользователя в сам CapitalFlow, чтобы дальнейшая работа с данными шла через личную сессию, а не через общий Bearer token.

## Security Requirements

* [ ] Регистрация пользователя при первом заходе в сервис.
* [ ] После первого пользователя закрыть публичную регистрацию или требовать admin invite/setup token.
* [ ] Хешировать пароль через `argon2id`.
* [ ] Не хранить plaintext password, reset tokens или JWT secrets в репозитории.
* [ ] Использовать access JWT с коротким TTL.
* [ ] Использовать refresh token с rotation и server-side revocation.
* [ ] Хранить refresh token безопасно: httpOnly cookie или hashed token в БД.
* [ ] Добавить logout с отзывом refresh token.
* [ ] Добавить защиту от brute force:

  * [ ] rate limit на login/register
  * [ ] одинаковые сообщения для неверного email/password
  * [ ] audit log для auth-событий
* [ ] Продумать CSRF модель, если refresh хранится в cookie.
* [ ] Не отдавать чувствительные auth-ошибки в UI.

## Backend

* [ ] Таблица пользователей.
* [ ] Таблица refresh sessions/tokens.
* [ ] `POST /auth/setup` для первого пользователя.
* [ ] `POST /auth/login`.
* [ ] `POST /auth/refresh`.
* [ ] `POST /auth/logout`.
* [ ] Middleware auth через JWT claims.
* [ ] Привязать пользовательские данные к owner/user id до multi-user сценариев.

## Frontend

* [ ] Первый экран setup/register, если пользователей нет.
* [ ] Login screen.
* [ ] Session bootstrap при открытии приложения.
* [ ] Авто-refresh access token.
* [ ] Ясные, но безопасные сообщения об ошибках входа.

## Acceptance Criteria

* [ ] Новый пользователь может настроить сервис при первом запуске.
* [ ] После setup dashboard доступен только после login.
* [ ] Пароли хранятся только как Argon2id hash.
* [ ] JWT нельзя использовать после logout/refresh rotation revoke.
* [ ] Auth покрыт unit и handler tests.

---

# v0.6 — Deposit & Capitalization Engine

## Цель

Сделать сильное ядро расчета процентов и капитализации.

## Сценарии

### Накопительный счет

* [ ] Ставка годовая.
* [ ] Расчет каждый день.
* [ ] Начисление каждый день.
* [ ] Капитализация каждый день.

### Срочный вклад

* [ ] Ставка годовая.
* [ ] Срок вклада.
* [ ] Пополнение до определенной даты.
* [ ] Начисление процентов:

  * [ ] ежедневно
  * [ ] ежемесячно
  * [ ] в конце срока
* [ ] Капитализация:

  * [ ] ежедневно
  * [ ] ежемесячно
  * [ ] в конце срока
  * [ ] без капитализации

### Промо-ставка

* [ ] Промо-ставка до даты.
* [ ] Базовая ставка после промо.
* [ ] Корректное разбиение периода расчета на части.

## Идемпотентность

* [ ] Нельзя дважды начислить проценты за один день по одному правилу.
* [ ] Повторный запуск job должен быть безопасным.
* [ ] Должна быть таблица `interest_accruals`.

## Фоновые задачи

* [ ] `daily_interest_accrual_job`
* [ ] `monthly_interest_accrual_job`
* [ ] `deposit_maturity_check_job`

## Команды

* [ ] `capitalflow accrue --date YYYY-MM-DD`
* [ ] `capitalflow accrue --account <id>`
* [ ] `capitalflow forecast --account <id> --days 365`
* [ ] `capitalflow recalculate --account <id> --from YYYY-MM-DD`

## Acceptance Criteria

* [ ] Яндекс-like накопительный счет с 12% и daily capitalization считается корректно.
* [ ] Альфа-like накопительный счет с 10% считается корректно.
* [ ] Повторное начисление за тот же день не создает дубль.
* [ ] Все начисления видны в истории операций.
* [ ] Можно построить прогноз на 30/90/365 дней.

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
* [ ] Docker/NixOS окружение.

## CI/CD до v1.0

* [ ] Добавить frontend CI job:

  * [ ] `npm ci`
  * [ ] `npm run lint`
  * [ ] `npm run build`
* [ ] Backend CI и frontend CI должны быть отдельными checks.
* [ ] Добавить OpenAPI validation check, когда spec станет обязательной.

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
* [ ] Курсы валют.
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

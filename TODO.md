# v0.1 — Stabilize current Deposit Manager

## Цель

Привести текущий CLI `deposit-manager` в устойчивое состояние перед крупным рефакторингом.

## Что уже есть

* CLI-команды для вкладов.
* Модель `Deposit`.
* Расчет доходности.
* Начисление процентов для `savings`.
* JSON-хранилище.
* Ledger-записи.

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
* [ ] `InterestRuleService`
* [x] `BalanceService`

## Правила

* [x] Баланс счета считается из операций.
* [x] Перевод создает две операции:

  * [x] `transfer_out` на исходном счете
  * [x] `transfer_in` на целевом счете
* [ ] Начисление процентов создает операцию `interest_income`.
* [ ] Ручная правка баланса создает `adjustment`.

## Acceptance Criteria

* [x] Можно создать `Account` без WebUI.
* [x] Можно создать `Transaction` без WebUI.
* [x] Можно посчитать баланс счета через список транзакций.
* [ ] Можно создать накопительный счет с правилом процентов.
* [ ] Можно начислить проценты как отдельную транзакцию.
* [ ] Повторное начисление за тот же день не создает дубль на уровне сервиса.
* [ ] Старый `Deposit` еще не удален, но новая модель уже работает отдельно.
* [x] Написаны unit-тесты на баланс.
* [ ] Написаны unit-тесты на проценты через `InterestRule`.

---

# v0.3 — PostgreSQL Storage

## Цель

Перейти с JSON-файлов на PostgreSQL как основное хранилище.

## Зависимости

* [ ] Добавить `pgx`.
* [ ] Добавить инструмент миграций:

  * [ ] `goose`
  * или [ ] `golang-migrate`
* [ ] Добавить Docker Compose для локального PostgreSQL.

## Таблицы

* [ ] `accounts`
* [ ] `transactions`
* [ ] `categories`
* [ ] `interest_rules`
* [ ] `interest_accruals`
* [ ] `balance_snapshots`
* [ ] `settings`

## Миграции

* [ ] `000001_create_accounts.sql`
* [ ] `000002_create_categories.sql`
* [ ] `000003_create_transactions.sql`
* [ ] `000004_create_interest_rules.sql`
* [ ] `000005_create_interest_accruals.sql`
* [ ] `000006_create_balance_snapshots.sql`

## Индексы

* [ ] `transactions(account_id, occurred_at)`
* [ ] `transactions(type)`
* [ ] `interest_rules(account_id, is_active)`
* [ ] `interest_accruals(account_id, accrual_date)`
* [ ] unique index для защиты от повторного начисления:

```sql
UNIQUE(account_id, accrual_date, rule_id)
```

## Репозитории

* [ ] `AccountRepository`
* [ ] `TransactionRepository`
* [ ] `CategoryRepository`
* [ ] `InterestRuleRepository`
* [ ] `InterestAccrualRepository`

## Миграция старых JSON-данных

* [ ] Написать команду:

```bash
finance-manager migrate-json
```

Она должна:

* [ ] прочитать текущие вклады из JSON
* [ ] создать `accounts`
* [ ] создать `interest_rules`
* [ ] создать `initial_balance` транзакции
* [ ] сохранить старые ID как `legacy_id`
* [ ] сформировать отчет миграции

## Acceptance Criteria

* [ ] PostgreSQL поднимается через Docker Compose.
* [ ] Миграции применяются одной командой.
* [ ] Старые JSON-вклады переносятся без потери сумм.
* [ ] Балансы после миграции совпадают со старыми значениями.
* [ ] JSON-хранилище больше не является основным источником данных.

---

# v0.4 — Thin HTTP API

## Цель

Добавить минимальный backend API для будущего WebUI.

API на этом этапе не должен закрывать все будущие сценарии. Он должен показать core-flow: счета, операции, переводы, баланс, ручное начисление процентов.

## Стек

* [ ] Go
* [ ] `chi`
* [ ] `slog`
* [ ] `pgx`
* [ ] PostgreSQL

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

* [ ] `GET /health`
* [ ] `GET /ready`

### Accounts

* [ ] `GET /api/accounts`
* [ ] `POST /api/accounts`
* [ ] `GET /api/accounts/{id}`
* [ ] `PATCH /api/accounts/{id}`
* [ ] `POST /api/accounts/{id}/archive`

### Transactions

* [ ] `GET /api/transactions`
* [ ] `POST /api/transactions`
* [ ] `GET /api/transactions/{id}`
* [ ] `DELETE /api/transactions/{id}`

### Transfers

* [ ] `POST /api/transfers`

### Interest Rules

* [ ] `GET /api/accounts/{id}/interest-rules`
* [ ] `POST /api/accounts/{id}/interest-rules`
* [ ] `PATCH /api/interest-rules/{id}`
* [ ] `POST /api/accounts/{id}/accrue-interest`
* [ ] `POST /api/accounts/{id}/recalculate-interest`

### Dashboard

* [ ] `GET /api/dashboard/summary`
* [ ] `GET /api/dashboard/net-worth`
* [ ] `GET /api/dashboard/cashflow`
* [ ] `GET /api/dashboard/interest-income`

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

* [ ] API запускается локально.
* [ ] Можно создать счет через API.
* [ ] Можно создать транзакцию через API.
* [ ] Можно получить баланс через API.
* [ ] Можно начислить проценты через API.
* [ ] Ошибки возвращаются в едином формате.
* [ ] API покрывает только core-flow и не раздувается до analytics suite.

---

# v0.5 — WebUI MVP

## Цель

Сделать первый рабочий WebUI поверх API.

Первый экран должен быть рабочим инструментом, а не landing page. WebUI MVP показывает только то, что доказывает core value: счета, балансы, операции, проценты и последние изменения.

## Рекомендуемый стек

Вариант для полноценного pet-project:

* [ ] React
* [ ] Vite
* [ ] TypeScript
* [ ] TanStack Query
* [ ] shadcn/ui
* [ ] Recharts

Упрощенный вариант:

* [ ] Go templates
* [ ] HTMX
* [ ] Alpine.js

Рекомендуемый выбор: `React + Vite + TypeScript`, потому что это лучше покажет full-stack развитие проекта.

## Страницы

### Dashboard

* [ ] Общий капитал.
* [ ] Список счетов с балансами.
* [ ] Доходы за месяц.
* [ ] Расходы за месяц.
* [ ] Проценты за месяц.
* [ ] Последние операции.
* [ ] Быстрые действия:

  * [ ] добавить доход
  * [ ] добавить расход
  * [ ] сделать перевод
  * [ ] создать счет

### Accounts

* [ ] Таблица счетов.
* [ ] Фильтр по типу счета.
* [ ] Баланс.
* [ ] Банк.
* [ ] Ставка, если есть.
* [ ] Статус счета.

### Account Details

* [ ] Карточка счета.
* [ ] Текущий баланс.
* [ ] История операций.
* [ ] График баланса.
* [ ] Правила процентов.
* [ ] Кнопка ручного начисления процентов.

### Transactions

* [ ] Таблица операций.
* [ ] Фильтр по дате.
* [ ] Фильтр по счету.
* [ ] Фильтр по категории.
* [ ] Фильтр по типу операции.

### Create Transaction

* [ ] Доход.
* [ ] Расход.
* [ ] Перевод.
* [ ] Корректировка.

### Create Account

* [ ] Название.
* [ ] Тип.
* [ ] Банк.
* [ ] Валюта.
* [ ] Начальный баланс.
* [ ] Ставка.
* [ ] Капитализация.
* [ ] Промо-ставка.
* [ ] Дата окончания промо.

## Acceptance Criteria

* [ ] WebUI открывается локально.
* [ ] Можно создать счет через форму.
* [ ] Можно добавить доход/расход.
* [ ] Можно сделать перевод между счетами.
* [ ] Можно увидеть балансы.
* [ ] Можно увидеть последние транзакции.
* [ ] Можно увидеть, из каких операций получился баланс.
* [ ] Можно увидеть начисленные проценты по счету.
* [ ] Нет smart budget, goals и LLM в первом WebUI MVP.

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

* [ ] `finance-manager accrue --date YYYY-MM-DD`
* [ ] `finance-manager accrue --account <id>`
* [ ] `finance-manager forecast --account <id> --days 365`
* [ ] `finance-manager recalculate --account <id> --from YYYY-MM-DD`

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

## NixOS integration

* [ ] devShell.
* [ ] systemd service.
* [ ] local reverse proxy option.
* [ ] backup timer.

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

# v1.0 — Personal Finance Tracker Core Release

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

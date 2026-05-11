# Finance Tracker Design Guide — Nordic UI

Based on Lazyweb references from Wise, Mercury, Capital One, Chime, Copilot, Credit Karma, Robinhood, E*TRADE, Cash App, Empower, and modern finance dashboard examples.

This version adds implementation-level rules: states, accessibility, money formatting, loading/error behavior, table behavior, responsive breakpoints, and a Nordic color system inspired by cold mountain/night palettes: slate, blue-gray, muted teal, frost, and snow.

## Product Direction

Finance Tracker should feel calm, exact, and fast. The UI should look like a daily money tool, not a marketing site.

Use a Nordic-inspired interface: quiet surfaces, blue-gray structure, restrained teal/blue accents, high readability, and low visual noise. The UI should feel trustworthy, precise, and slightly cold, but not gloomy.

The product should help users answer:

* How much money do I have?
* What changed recently?
* What needs action?
* Where did money go?
* What can I move, save, or improve?

## Visual Direction

The product should use a calm Nordic financial style:

* Light mode as the default production theme.
* Dark mode as a first-class future theme, not an afterthought.
* Soft blue-gray page background.
* White/frosted surfaces for cards and tables.
* Deep slate text for numbers and labels.
* Muted teal as the primary action color.
* Cool blue for informational states and charts.
* Controlled red/amber/green for finance semantics.
* Minimal gradients; use them only for non-critical decorative headers if needed.
* No glassmorphism for core banking surfaces.

The reference image suggests the emotional direction: dark mountain, cold air, muted stars, icy gray-blue sky, and strong contrast between surface and background. Translate that into UI as:

* `frost` backgrounds
* `fjord` blue-gray borders
* `pine` or `aurora` teal actions
* `night` dark text
* `snow` cards

## Common Patterns

* Show balance first. Use one strong total, then account cards.
* Put primary money actions near balances: transfer, add transaction, add account.
* Use compact account cards with balance, type, currency, and last activity.
* Keep transaction history dense with search, filters, date groups, merchant/category, account, amount, and status.
* Use transfer forms as a review flow: from, to, amount, date, note, review.
* Show savings/deposit screens around APY, projected interest, goal progress, and deposit/withdraw actions.
* Use charts to support decisions, not decorate. Pair every chart with a plain metric.
* Empty states should explain what is missing and offer one action.
* Mobile layout should stack: summary, actions, cards, list. Keep bottom actions reachable.

## What To Avoid

* Do not use glassmorphism for core banking surfaces.
* Do not hide actions inside decorative cards.
* Do not use huge hero text inside app screens.
* Do not make every card the same visual weight.
* Do not use donut charts when a bar, line, or progress list is clearer.
* Do not mix too many bright finance colors.
* Do not make transaction rows too tall on desktop.
* Do not use vague empty states like "Nothing here" without a next step.
* Do not show charts without labels, time range, and units.
* Do not rely on color alone for income, expense, warning, or success.
* Do not use heavy neon effects. Nordic style should be restrained, not cyberpunk.
* Do not make dark surfaces pure black. Use deep blue/slate instead.
* Do not use low-contrast gray text on gray-blue backgrounds.

## App Structure

Desktop shell:

```text
Top bar: app name, date range, search, user/settings
Left nav: Dashboard, Accounts, Transactions, Transfers, Savings, Analytics, Settings
Main: max 1200px content width
Right rail: optional insights, upcoming, or savings goals
```

Mobile shell:

```text
Header: page title, search/settings icon
Content: stacked sections
Bottom nav: Dashboard, Accounts, Transactions, Transfer, Settings
Primary action: sticky button only on forms or create flows
```

Dashboard:

```text
Net worth / total balance
Quick actions
Account overview cards
Cash flow chart
Recent transactions
Savings/deposit preview
```

Accounts:

```text
Account cards or table
Filters by type/currency/status
Account detail with balance trend, transactions, edit action
```

Transactions:

```text
Search and filters
Date-grouped transaction list/table
Inline category/status chips
Empty state with "Add transaction"
```

Transfer:

```text
From account
To account
Amount
Date
Note
Review summary
Submit
```

Savings / Deposits:

```text
Balance
APY / rate
Interest earned
Goal progress
Projected value chart
Deposit / withdraw actions
Linked transactions
```

Analytics:

```text
Period selector
Income vs expense summary
Cash flow line or bar chart
Category progress list
Account balance trend
```

## Layout Rules

* Use `24px` page padding on desktop, `16px` on mobile.
* Use a 12-column grid on desktop.
* Use two main columns for dashboard desktop: `2fr 1fr`.
* Keep content width between `1040px` and `1200px`.
* Use `16px` gaps inside sections and `24px` gaps between sections.
* Stack all dashboard sections below `768px`.
* Keep forms to `480px` to `640px` width.
* Keep tables full width and horizontally scroll only as a last resort.
* Keep sidebar width between `220px` and `260px`.
* Keep right rail width between `300px` and `360px`.
* Avoid center-aligned finance data. Money, dates, and labels should scan predictably.

## Responsive Breakpoints

Use these breakpoints as implementation defaults:

```css
:root {
  --breakpoint-sm: 480px;
  --breakpoint-md: 768px;
  --breakpoint-lg: 1024px;
  --breakpoint-xl: 1200px;
}
```

Behavior:

* Below `480px`: compact mobile, single column, bottom navigation.
* `480px` to `767px`: mobile/tablet stacked layout.
* `768px` to `1023px`: tablet layout, sidebar may collapse to icons.
* `1024px` and above: desktop layout with sidebar.
* `1200px` and above: max-width content container centered inside the main area.

## Component Rules

### Account Cards

* Show account name, type, currency, balance, last update.
* Use a small icon or color strip for account type.
* Add one secondary metric only, such as monthly change or APY.
* Keep actions as icons or a compact menu.
* Use stronger visual weight for primary/current account.
* Avoid gradients behind balance values.

### Transaction Rows

* Desktop: merchant/category, account, date, status, amount.
* Mobile: merchant/category top, date/account secondary, amount right.
* Use green for income and dark text/red accent for expense.
* Support pending, completed, failed, reversed, cancelled, and transfer states.
* Add signs and labels, not only color: `+`, `−`, `Income`, `Expense`, `Transfer`.
* Keep desktop rows compact: `44px` to `56px` height.

### Transfer Form

* Use selectable account rows, not raw dropdowns when space allows.
* Show available balance for source account.
* Disable submit until valid.
* Always include a review step for real money movement.
* Show validation near fields, not only at the top.
* Show final summary before submit: from, to, amount, date, fee if any, note.

### Empty States

* Use a clear title, one sentence, and one action.
* Match the page context: transactions -> add/import, savings -> create goal, analytics -> add data.
* Keep illustrations optional and small.
* Avoid vague text like `Nothing here`.

Example:

```text
No transactions yet
Add your first transaction or import a CSV file to start tracking cash flow.
[Add transaction] [Import CSV]
```

### Charts

* Prefer line charts for balances over time.
* Prefer bar charts for income vs expense.
* Prefer progress lists for category budgets.
* Avoid 3D, heavy gradients, and unlabeled slices.
* Always show period, unit, and summary metric.
* Every chart should answer one question.
* Pair every chart with a plain text metric.

Example:

```text
Cash flow
Last 30 days
Income: ₽120,000
Expense: ₽84,300
Net: +₽35,700
```

## State Rules

### Hover

* Cards: subtle border or shadow change only.
* Buttons: slight background darkening.
* Table rows: muted background highlight.
* Do not use scale animations for financial rows or cards.

### Focus

All interactive elements must have a visible keyboard focus state.

```css
:focus-visible {
  outline: 2px solid var(--color-focus-ring);
  outline-offset: 2px;
}
```

### Disabled

Disabled controls should look inactive but readable:

* Use muted background.
* Use muted text.
* Keep cursor default/not-allowed depending on component.
* Explain why submit is disabled if validation is not obvious.

### Error

* Show error next to the field when possible.
* Use red accent plus text label.
* Do not rely only on border color.
* Keep server/API errors in a visible alert region above the form.

### Success

* Use restrained green.
* Confirm what changed.
* Avoid large success animations for routine finance actions.

## Loading, Empty, and Error States

### Loading

Use skeletons for cards, tables, and charts:

* Dashboard metrics: skeleton cards.
* Tables: 5-8 skeleton rows.
* Charts: muted chart placeholder with title retained.
* Forms: keep structure visible if possible.

Avoid full-page spinners unless the whole app is booting.

### Empty

Each empty state needs:

1. Clear title.
2. One sentence explaining what is missing.
3. One primary action.
4. Optional secondary action.

### Error

Errors should be specific:

* Bad: `Something went wrong`.
* Better: `Could not load transactions. Check connection or try again.`

Every error state should provide at least one recovery action:

* Retry.
* Back to dashboard.
* Clear filters.
* Contact/support placeholder if needed later.

## Money Formatting Rules

Use a single formatting helper for all money values.

Rules:

* Always show currency.
* Use locale-aware grouping.
* Use signs for deltas.
* Do not rely only on color for positive/negative values.
* Align money values to the right in tables.
* Use tabular numbers if the font supports it.
* Do not mix `RUB`, `₽`, and `руб.` randomly.

Recommended default for Russian rubles:

```text
₽ 128,450.20
+₽ 12,300.00
−₽ 4,990.00
```

Alternative compact display:

```text
128,450 ₽
+12,300 ₽
−4,990 ₽
```

Choose one style and keep it everywhere.

Recommended CSS:

```css
.money {
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.01em;
}
```

## Table Rules

Transaction tables should support:

* Search.
* Date range filter.
* Account filter.
* Category filter.
* Status filter.
* Sorting by date and amount.
* Pagination or infinite loading.
* Empty result state after filters.

Desktop table:

```text
Date | Merchant / Note | Category | Account | Status | Amount
```

Rules:

* Date, status, and amount should be easy to scan.
* Amount column should be right-aligned.
* Header may be sticky if the list is long.
* Do not force desktop table layout on mobile.

Mobile transaction list:

```text
Merchant / Note              −₽ 1,200.00
Category · Account · Date    Completed
```

## Accessibility Rules

* Minimum text contrast should pass WCAG AA.
* Do not use color alone for meaning.
* All controls must be keyboard accessible.
* Icon-only buttons need `aria-label`.
* Form fields need visible labels.
* Error messages should be connected to fields.
* Modals should trap focus and close with Escape.
* Tables should use real table markup for tabular data.
* Loading states should not cause large layout shift.

## Nordic Design Tokens

### Light Theme

```css
:root {
  /* Base */
  --color-bg: #eef2f6;
  --color-bg-strong: #e4ebf2;
  --color-surface: #ffffff;
  --color-surface-muted: #f6f8fb;
  --color-surface-raised: #ffffff;

  /* Borders */
  --color-border: #d3dce6;
  --color-border-strong: #aebdca;

  /* Text */
  --color-text: #111827;
  --color-text-muted: #526170;
  --color-text-subtle: #8291a3;
  --color-text-inverse: #f8fafc;

  /* Nordic accents */
  --color-primary: #0f766e;
  --color-primary-hover: #115e59;
  --color-primary-soft: #d8f1ee;

  --color-secondary: #334e68;
  --color-secondary-hover: #243b53;
  --color-secondary-soft: #e6edf5;

  --color-info: #2f6f9f;
  --color-info-soft: #dcebf7;

  /* Finance semantics */
  --color-income: #2f855a;
  --color-income-soft: #dff3e8;
  --color-expense: #b42318;
  --color-expense-soft: #fee4e2;
  --color-warning: #b7791f;
  --color-warning-soft: #fef3c7;

  /* Status */
  --color-success: #2f855a;
  --color-success-soft: #dff3e8;
  --color-danger: #b42318;
  --color-danger-soft: #fee4e2;
  --color-focus-ring: #5aa9a6;

  /* Charts */
  --color-chart-1: #0f766e;
  --color-chart-2: #2f6f9f;
  --color-chart-3: #5b6f8f;
  --color-chart-4: #7c6f9f;
  --color-chart-5: #8a9a5b;
  --color-chart-grid: #d9e2ec;
}
```

### Dark Theme

```css
[data-theme="dark"] {
  /* Base */
  --color-bg: #0b1220;
  --color-bg-strong: #070d18;
  --color-surface: #111827;
  --color-surface-muted: #172033;
  --color-surface-raised: #1b2638;

  /* Borders */
  --color-border: #263445;
  --color-border-strong: #3b4b5f;

  /* Text */
  --color-text: #eef2f6;
  --color-text-muted: #b8c4d2;
  --color-text-subtle: #8091a5;
  --color-text-inverse: #0b1220;

  /* Nordic accents */
  --color-primary: #5aa9a6;
  --color-primary-hover: #7bc4c0;
  --color-primary-soft: #123b3d;

  --color-secondary: #9fb3c8;
  --color-secondary-hover: #c8d6e5;
  --color-secondary-soft: #1e2c3d;

  --color-info: #7aa7d9;
  --color-info-soft: #132f4c;

  /* Finance semantics */
  --color-income: #68d391;
  --color-income-soft: #123524;
  --color-expense: #f97066;
  --color-expense-soft: #3a1717;
  --color-warning: #f6c76f;
  --color-warning-soft: #3a2a0c;

  /* Status */
  --color-success: #68d391;
  --color-success-soft: #123524;
  --color-danger: #f97066;
  --color-danger-soft: #3a1717;
  --color-focus-ring: #7bc4c0;

  /* Charts */
  --color-chart-1: #5aa9a6;
  --color-chart-2: #7aa7d9;
  --color-chart-3: #9fb3c8;
  --color-chart-4: #b8a7d9;
  --color-chart-5: #d3c07a;
  --color-chart-grid: #263445;
}
```

### Optional Hero/Overview Gradient

Use only for a top balance summary or dashboard background accent, never behind dense transaction data.

```css
.nordic-overview-gradient {
  background:
    radial-gradient(circle at 75% 15%, rgb(122 167 217 / 0.22), transparent 32%),
    linear-gradient(135deg, #0b1220 0%, #172033 48%, #334e68 100%);
  color: #eef2f6;
}
```

## Typography

```css
:root {
  --font-sans: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  --font-mono: "SFMono-Regular", Consolas, "Liberation Mono", monospace;

  --text-xs: 12px;
  --text-sm: 14px;
  --text-md: 16px;
  --text-lg: 18px;
  --text-xl: 22px;
  --text-2xl: 28px;

  --leading-tight: 1.2;
  --leading-base: 1.5;
}
```

Use:

* Page title: `28px / 1.2 / 700`
* Section title: `18px / 1.2 / 650`
* Card label: `12px / 1.5 / 600`
* Body: `14px-16px / 1.5 / 400`
* Money value: `22px-28px / 1.2 / 700`
* Table text: `14px / 1.5 / 400`

Recommended global number style:

```css
body {
  font-family: var(--font-sans);
}

.money,
.metric-value,
.table-amount {
  font-variant-numeric: tabular-nums;
}
```

## Spacing

```css
:root {
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;
}
```

Rules:

* Use `8px` rhythm for most layout decisions.
* Use `16px` inside cards.
* Use `24px` between dashboard sections.
* Avoid random values like `13px`, `19px`, `27px` unless solving a specific alignment issue.

## Radius and Shadow

```css
:root {
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 8px;
  --radius-xl: 12px;

  --shadow-card: 0 1px 2px rgb(16 24 40 / 0.06);
  --shadow-card-hover: 0 4px 12px rgb(16 24 40 / 0.08);
  --shadow-popover: 0 12px 28px rgb(16 24 40 / 0.14);
}
```

Use `8px` as the max normal card radius. Use `12px` only for modals, mobile sheets, or a primary dashboard summary card.

## Buttons

Primary:

```css
.button-primary {
  background: var(--color-primary);
  color: var(--color-text-inverse);
  border: 1px solid var(--color-primary);
}

.button-primary:hover {
  background: var(--color-primary-hover);
}
```

Secondary:

```css
.button-secondary {
  background: var(--color-surface);
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.button-secondary:hover {
  background: var(--color-surface-muted);
}
```

Danger:

```css
.button-danger {
  background: var(--color-danger);
  color: white;
  border: 1px solid var(--color-danger);
}
```

Rules:

* One primary button per section.
* Use secondary buttons for alternatives.
* Use danger buttons only for destructive actions.
* Do not use outline-only buttons for the main money movement action.

## Forms

Field structure:

```text
Label
Input/select
Hint or error
```

Rules:

* Labels are always visible.
* Placeholder text is not a replacement for labels.
* Validation runs on blur and submit.
* Money fields should show currency clearly.
* Account selectors should show account name and available balance.
* Transfer forms should have a review step.

## Good Decisions

* Good: balance summary + action strip + recent activity.
* Good: transfer form with visible from/to balances and review.
* Good: transaction filters as chips above the list.
* Good: category analytics as progress rows with exact amounts.
* Good: savings screen that shows APY, earned interest, and projected growth.
* Good: empty transaction state with "Add transaction" and "Import CSV".
* Good: Nordic palette with muted teal/blue accents and strong number readability.
* Good: separate mobile transaction list instead of squeezing desktop table into small screens.

## Bad Decisions

* Bad: dashboard starts with a large welcome banner.
* Bad: five unrelated charts above account balances.
* Bad: transfer action hidden in an overflow menu.
* Bad: red/green-only indicators without signs or labels.
* Bad: mobile table with tiny columns.
* Bad: empty analytics page that gives no reason or next step.
* Bad: cards with gradients that reduce number readability.
* Bad: dark theme with pure black surfaces and weak gray text.
* Bad: finance actions that move money without review.

## Reference Notes

* Wise: strong desktop account overview with balances, currency accounts, recent transactions, and primary actions.
* Mercury: strong mobile account balance, spending trend, money movement, transaction filters, and data views.
* Capital One and E*TRADE: clear transfer form flow with from/to accounts, amount, schedule, memo, and review.
* Chime and Cash App: simple mobile money hubs with account balance and quick actions.
* Copilot: strong category analytics with progress bars and budget comparison.
* Credit Karma and Robinhood: savings/APY screens that pair rate, balance, offer details, and primary action.
* Empower and Yahoo Finance: dashboard and transaction search patterns for desktop.

## Implementation Priority

### PR 1 — Web App Shell

Branch:

```text
feature/web-app-shell
```

Scope:

* AppShell
* Sidebar
* Topbar
* Page container
* Card component
* Button component
* CSS variables / design tokens
* Light Nordic theme
* Basic responsive behavior

Do not change backend API.

### PR 2 — Dashboard Foundation

Scope:

* Net worth summary
* Metric cards
* Account cards
* Quick actions
* Recent transactions block
* Static chart placeholders
* Loading/empty/error states

### PR 3 — Transactions UX

Scope:

* Transaction list/table
* Search input
* Filter chips
* Status chips
* Mobile transaction layout
* Empty filtered state

### PR 4 — Transfers UX

Scope:

* Transfer form
* Account selectors
* Validation
* Review step
* Submit state
* Success/error feedback

### PR 5 — Savings / Deposits UX

Scope:

* Savings overview
* APY/rate display
* Earned interest
* Projected value chart
* Goal progress
* Deposit/withdraw actions

## First Coding Prompt

Use this as the first implementation prompt for Codex:

```text
Implement the first UI foundation PR for finance-tracker.

Branch: feature/web-app-shell

Goal:
Create a modern Nordic-style finance app shell using the project design guide.

Scope:
1. Add CSS design tokens for the Nordic light theme.
2. Add AppShell with sidebar, topbar, and main content area.
3. Add reusable Card and Button components.
4. Add a Dashboard page layout with placeholder sections.
5. Add responsive behavior for desktop/tablet/mobile.
6. Add visible focus states and basic accessibility labels.

Constraints:
- Do not change backend API.
- Do not rewrite unrelated frontend code.
- Do not add heavy UI libraries unless already used.
- Keep TypeScript strict.
- Keep components small.
- Do not put all UI into App.tsx.
- Use semantic HTML where possible.
```

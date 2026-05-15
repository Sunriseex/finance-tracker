# TODO Fixes

## v0.6 Remaining Work

- [ ] Add production background job wrappers for the PostgreSQL interest engine:
  - `daily_interest_accrual_job`
  - `monthly_interest_accrual_job`
  - `deposit_maturity_check_job`
- [ ] Define runtime scheduling and locking for interest jobs in the VM/NixOS deployment.
- [ ] Add top-up cutoff support to deposit rules if this must be enforced by the engine.
- [ ] Re-run race tests in the VM. Windows currently fails before tests with `runtime/cgo ... cgo.exe exit status 2`.

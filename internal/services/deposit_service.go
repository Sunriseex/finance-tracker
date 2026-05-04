package services

import (
	"log/slog"
	"time"

	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/storage"
	"github.com/sunriseex/finance-manager/pkg/calculator"
	"github.com/sunriseex/finance-manager/pkg/dates"
	"github.com/sunriseex/finance-manager/pkg/errors"
	"github.com/sunriseex/finance-manager/pkg/validation"
)

type DepositService struct {
	validator *validation.DepositValidator
}

func NewDepositService() *DepositService {
	return &DepositService{
		validator: validation.NewDepositValidator(),
	}
}

type CreateDepositRequest struct {
	Name         string
	Bank         string
	Type         string
	Amount       int64
	InterestRate float64
	TermMonths   int
	PromoRate    *float64
	PromoEndDate string
}

type CreateDepositResponse struct {
	Deposit   *models.Deposit
	DepositID string
	Success   bool
	Message   string
}

func (s *DepositService) Create(req *CreateDepositRequest) (*CreateDepositResponse, error) {

	slog.Debug("Создание вклада",
		"name", req.Name,
		"bank", req.Bank,
		"amount", req.Amount)

	if err := s.validator.ValidateCreateRequest(
		req.Name, req.Bank, req.Type, req.Amount, req.InterestRate,
		req.TermMonths, req.PromoRate, req.PromoEndDate,
	); err != nil {

		slog.Error("Ошибка валидации вклада",
			"name", req.Name,
			"bank", req.Bank,
			"error", err)

		return nil, errors.NewValidationError(
			"некорректные параметры вклада",
			map[string]interface{}{
				"name":          req.Name,
				"bank":          req.Bank,
				"type":          req.Type,
				"amount":        req.Amount,
				"interest_rate": req.InterestRate,
				"term_months":   req.TermMonths,
			},
		)
	}

	deposit := &models.Deposit{
		Name:           req.Name,
		Bank:           req.Bank,
		Type:           req.Type,
		Amount:         req.Amount,
		InitialAmount:  req.Amount,
		InterestRate:   req.InterestRate,
		PromoRate:      req.PromoRate,
		PromoEndDate:   req.PromoEndDate,
		StartDate:      time.Now().Format("2006-01-02"),
		AutoRenewal:    true,
		Capitalization: s.getCapitalizationType(req.Bank),
	}

	if req.Type == "term" {
		deposit.TermMonths = req.TermMonths
		endDate, err := dates.CalculateMaturityDate(deposit.StartDate, req.TermMonths)
		if err != nil {

			slog.Error("Ошибка расчета даты окончания вклада",
				"start_date", deposit.StartDate,
				"term_months", req.TermMonths,
				"error", err)

			return nil, errors.NewCalculationError(
				"ошибка расчета даты окончания вклада",
				err,
			)
		}
		deposit.EndDate = endDate
		deposit.TopUpEndDate = dates.CalculateTopUpEndDate(deposit.StartDate)
	}

	if err := s.validator.Validate(deposit); err != nil {

		slog.Error("Ошибка валидации вклада",
			"deposit_name", deposit.Name,
			"validation_error", err)

		return nil, errors.NewValidationError(
			"ошибка валидации данных вклада",
			map[string]interface{}{
				"deposit_name":     deposit.Name,
				"validation_error": err.Error(),
			},
		)
	}

	if err := storage.CreateDeposit(deposit, config.AppConfig.DepositsDataPath); err != nil {

		slog.Error("Ошибка сохранения вклада",
			"deposit_name", deposit.Name,
			"error", err)

		return nil, errors.NewStorageError("создание вклада", err)
	}
	slog.Info("Вклад успешно создан",
		"id", deposit.ID,
		"name", deposit.Name,
		"bank", deposit.Bank,
		"amount", deposit.Amount)

	return &CreateDepositResponse{
		Deposit:   deposit,
		DepositID: deposit.ID,
		Success:   true,
		Message:   "Вклад успешно создан",
	}, nil
}

type TopUpRequest struct {
	DepositID   string
	Amount      int64
	Description string
}

type TopUpResponse struct {
	Success        bool
	NewAmount      int64
	PreviousAmount int64
	Message        string
}

func (s *DepositService) TopUp(req *TopUpRequest) (*TopUpResponse, error) {

	slog.Debug("Пополнение вклада",
		"deposit_id", req.DepositID,
		"amount", req.Amount)

	if req.Amount <= 0 {

		slog.Warn("Попытка пополнения неположительной суммой",
			"deposit_id", req.DepositID,
			"amount", req.Amount)

		return nil, errors.NewValidationError(
			"сумма пополнения должна быть положительной",
			map[string]interface{}{
				"amount":     req.Amount,
				"deposit_id": req.DepositID,
			},
		)
	}

	if req.Amount > 10000000 {

		slog.Warn("Попытка пополнения слишком большой суммой",
			"deposit_id", req.DepositID,
			"amount", req.Amount)

		return nil, errors.NewValidationError(
			"сумма пополнения слишком большая",
			map[string]interface{}{
				"deposit_id":  req.DepositID,
				"max_allowed": 10000000,
				"amount":      req.Amount,
			},
		)
	}

	deposit, err := storage.GetDepositByID(req.DepositID, config.AppConfig.DepositsDataPath)
	if err != nil {

		slog.Error("Ошибка получения вклада для пополнения",
			"deposit_id", req.DepositID,
			"error", err)

		return nil, errors.WrapError(
			errors.ErrStorage,
			"ошибка получения данных вклада",
			err,
		)
	}

	previousAmount := deposit.Amount

	if err := storage.UpdateDepositAmount(req.DepositID, req.Amount, config.AppConfig.DepositsDataPath); err != nil {

		slog.Error("Ошибка пополнения вклада",
			"deposit_id", req.DepositID,
			"error", err)

		return nil, errors.WrapError(
			errors.ErrStorage,
			"ошибка пополнения вклада",
			err,
		)
	}

	slog.Info("Вклад успешно пополнен",
		"deposit_id", req.DepositID,
		"previous_amount", previousAmount,
		"new_amount", previousAmount+req.Amount,
		"topup_amount", req.Amount)

	return &TopUpResponse{
		Success:        true,
		NewAmount:      previousAmount + req.Amount,
		PreviousAmount: previousAmount,
		Message:        "Вклад успешно пополнен",
	}, nil
}

type CalculateIncomeRequest struct {
	DepositID string
	Days      int
}

type CalculateIncomeResponse struct {
	Success        bool
	DepositName    string
	Amount         float64
	InterestRate   float64
	Capitalization string
	PeriodDays     int
	ExpectedIncome float64
	TotalAmount    float64
}

func (s *DepositService) CalculateIncome(req *CalculateIncomeRequest) (*CalculateIncomeResponse, error) {

	slog.Debug("Рассчет дохода по вкладу", "deposit_id", req.DepositID, "days", req.Days)

	if req.Days <= 0 {

		slog.Warn("Некорректный период расчета",
			"deposit_id", req.DepositID,
			"days", req.Days)

		return nil, errors.NewValidationError(
			"период расчета должен быть положительным",
			map[string]interface{}{
				"days": req.Days,
			},
		)
	}

	deposit, err := storage.GetDepositByID(req.DepositID, config.AppConfig.DepositsDataPath)
	if err != nil {

		slog.Error("Ошибка получения вклада для расчета дохода",
			"deposit_id", req.DepositID,
			"error", err)

		return nil, errors.WrapError(
			errors.ErrStorage,
			"ошибка получения данных вклада для расчета",
			err,
		)
	}

	active, daysUntilPromoEnd := calculator.CheckPromoStatus(*deposit)
	if active {
		slog.Debug("Учтена промо-ставка при расчете",
			"promo_rate", *deposit.PromoRate,
			"promo_end_date", deposit.PromoEndDate,
			"days_until_promo_end", daysUntilPromoEnd,
			"base_rate", deposit.InterestRate)
	}

	income := calculator.CalculateIncome(*deposit, req.Days)
	incomeFloat, _ := income.Float64()
	amountRubles := float64(deposit.Amount) / 100.0

	slog.Debug("Расчет дохода завершен",
		"deposit_id", req.DepositID,
		"income", incomeFloat,
		"amount", amountRubles,
		"total_days", req.Days)

	return &CalculateIncomeResponse{
		Success:        true,
		DepositName:    deposit.Name,
		Amount:         amountRubles,
		InterestRate:   deposit.InterestRate,
		Capitalization: deposit.Capitalization,
		PeriodDays:     req.Days,
		ExpectedIncome: incomeFloat,
		TotalAmount:    amountRubles + incomeFloat,
	}, nil
}

type UpdateDepositRequest struct {
	DepositID string
}

type UpdateDepositResponse struct {
	Success      bool
	DepositName  string
	StartDate    string
	EndDate      string
	TopUpEndDate string
	Message      string
}

func (s *DepositService) Update(req *UpdateDepositRequest) (*UpdateDepositResponse, error) {
	slog.Info("Обновление вклада", "deposit_id", req.DepositID)

	deposit, err := storage.GetDepositByID(req.DepositID, config.AppConfig.DepositsDataPath)
	if err != nil {

		slog.Error("Вклад не найден для обновления",
			"deposit_id", req.DepositID,
			"error", err)

		return nil, errors.NewNotFoundError("вклад", req.DepositID)
	}

	if deposit.Type != "term" {

		slog.Warn("Попытка обновления не срочного вклада",
			"deposit_id", req.DepositID,
			"deposit_type", deposit.Type)

		return nil, errors.NewBusinessLogicError(
			"только срочные вклады могут быть обновлены (пролонгированы)",
			map[string]interface{}{
				"deposit_id":   req.DepositID,
				"deposit_type": deposit.Type,
			},
		)
	}

	if !dates.CanBeProlonged(deposit.EndDate) {

		slog.Warn("Вклад не может быть пролонгирован",
			"deposit_id", req.DepositID,
			"end_date", deposit.EndDate)

		return nil, errors.NewBusinessLogicError(
			"вклад не может быть пролонгирован в данный момент",
			map[string]interface{}{
				"deposit_id": req.DepositID,
				"end_date":   deposit.EndDate,
			},
		)
	}

	today := time.Now().Format("2006-01-02")
	deposit.StartDate = today

	endDate, err := dates.CalculateMaturityDate(today, deposit.TermMonths)
	if err != nil {

		slog.Error("Ошибка расчета даты окончания вклада",
			"deposit_id", req.DepositID,
			"error", err)

		return nil, errors.NewCalculationError(
			"ошибка расчета даты окончания при обновлении вклада",
			err,
		)
	}
	deposit.EndDate = endDate
	deposit.TopUpEndDate = dates.CalculateTopUpEndDate(today)

	if err := s.validator.Validate(deposit); err != nil {

		slog.Error("Ошибка валидации данных после обновления",
			"deposit_name", deposit.Name,
			"validate_error", err)

		return nil, errors.NewValidationError(
			"ошибка валидации данных после обновления",
			map[string]interface{}{
				"deposit_name":     deposit.Name,
				"validation_error": err.Error(),
			},
		)
	}

	if err := storage.UpdateDeposit(deposit, config.AppConfig.DepositsDataPath); err != nil {

		slog.Error("Ошибка обновления вклада",
			"deposit_id", req.DepositID,
			"error", err)

		return nil, errors.NewStorageError("обновление вклада", err)
	}

	slog.Info("Вклад успешно обновлен",
		"deposit_id", req.DepositID,
		"deposit_name", deposit.Name,
		"new_start_date", deposit.StartDate,
		"new_end_date", deposit.EndDate)

	return &UpdateDepositResponse{
		Success:      true,
		DepositName:  deposit.Name,
		StartDate:    deposit.StartDate,
		EndDate:      deposit.EndDate,
		TopUpEndDate: deposit.TopUpEndDate,
		Message:      "Вклад успешно обновлен",
	}, nil
}

type ListDepositsResponse struct {
	Success     bool
	Deposits    []models.Deposit
	TotalCount  int
	TotalAmount int64
	Message     string
}

func (s *DepositService) List() (*ListDepositsResponse, error) {

	slog.Debug("Загрузка списка вкладов")

	data, err := storage.LoadDeposits(config.AppConfig.DepositsDataPath)
	if err != nil {
		slog.Error("Ошибка загрузки списка вкладов", "error", err)
		return nil, errors.NewStorageError("загрузка списка вкладов", err)
	}

	var totalAmount int64
	for i := range data.Deposits {
		totalAmount += data.Deposits[i].Amount
	}

	slog.Info("Список вкладов загружен",
		"count", len(data.Deposits),
		"total_amount", totalAmount)

	return &ListDepositsResponse{
		Success:     true,
		Deposits:    data.Deposits,
		TotalCount:  len(data.Deposits),
		TotalAmount: totalAmount,
		Message:     "Список вкладов успешно загружен",
	}, nil
}

type GetDepositRequest struct {
	DepositID string
}

type GetDepositResponse struct {
	Success bool
	Deposit *models.Deposit
	Message string
}

func (s *DepositService) Get(req *GetDepositRequest) (*GetDepositResponse, error) {

	slog.Debug("Получение вклада по ID", "deposit_id", req.DepositID)

	deposit, err := storage.GetDepositByID(req.DepositID, config.AppConfig.DepositsDataPath)
	if err != nil {
		slog.Error("Вклад не найден", "deposit_id", req.DepositID, "error", err)
		return nil, errors.NewNotFoundError("вклад", req.DepositID)
	}

	slog.Info("Вклад найден",
		"deposit_id", req.DepositID,
		"deposit_name", deposit.Name)

	return &GetDepositResponse{
		Success: true,
		Deposit: deposit,
		Message: "Вклад найден",
	}, nil
}

type FindDepositRequest struct {
	Name string
	Bank string
}

type FindDepositResponse struct {
	Success bool
	Deposit *models.Deposit
	Found   bool
	Message string
}

func (s *DepositService) Find(req *FindDepositRequest) (*FindDepositResponse, error) {

	slog.Debug("Поиск вклада по имени и банку",
		"name", req.Name,
		"bank", req.Bank)

	deposit, err := storage.FindDepositByNameAndBank(req.Name, req.Bank, config.AppConfig.DepositsDataPath)
	if err != nil {

		slog.Error("Ошибка поиска вклада",
			"name", req.Name,
			"bank", req.Bank,
			"error", err)

		return nil, errors.NewStorageError("поиск вклада", err)
	}

	if deposit == nil {
		slog.Debug("Вклад не найден", "name", req.Name, "bank", req.Bank)
		return &FindDepositResponse{
			Success: true,
			Deposit: nil,
			Found:   false,
			Message: "Вклад не найден",
		}, nil
	}

	slog.Debug("Вклад найден",
		"name", req.Name,
		"bank", req.Bank,
		"deposit_id", deposit.ID)

	return &FindDepositResponse{
		Success: true,
		Deposit: deposit,
		Found:   true,
		Message: "Вклад найден",
	}, nil
}

func (s *DepositService) getCapitalizationType(bank string) string {
	if bank == "Яндекс Банк" || bank == "Yandex" {
		return "daily"
	}
	return "daily"
}

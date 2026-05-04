package services

import (
	"fmt"
	"log/slog"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/storage"
	"github.com/sunriseex/finance-manager/pkg/calculator"
	"github.com/sunriseex/finance-manager/pkg/errors"
)

type InterestService struct{}

func NewInterestService() *InterestService {
	return &InterestService{}
}

type AccrueInterestRequest struct {
}

type AccrualResult struct {
	DepositID   string
	DepositName string
	Income      decimal.Decimal
	Success     bool
	Error       error
}

type AccrueInterestResponse struct {
	Success      bool
	TotalAccrued decimal.Decimal
	SuccessCount int
	ErrorCount   int
	Results      []AccrualResult
	Message      string
}

func (s *InterestService) AccrueInterest(_ *AccrueInterestRequest) (*AccrueInterestResponse, error) {

	slog.Info("Начало начисления процентов по вкладам")

	data, err := storage.LoadDeposits(config.AppConfig.DepositsDataPath)
	if err != nil {

		slog.Error("Ошибка загрузки вкладов для зачисления процентов", "error", err)

		return nil, errors.NewStorageError("загрузка вкладов для начисления процентов", err)
	}

	slog.Debug("Загружено вкладов для зачисления", "count", len(data.Deposits))

	response := &AccrueInterestResponse{
		Results:      make([]AccrualResult, 0),
		TotalAccrued: decimal.Zero,
	}

	for i := range data.Deposits {
		deposit := &data.Deposits[i]
		result := s.processDepositAccrual(deposit)
		response.Results = append(response.Results, result)

		if result.Success {
			response.TotalAccrued = response.TotalAccrued.Add(result.Income)
			response.SuccessCount++
			slog.Debug("Проценты начислены успешно",
				"deposit", deposit.Name,
				"income", result.Income)
		} else {
			response.ErrorCount++
			slog.Warn("Ошибка начисления процентов",
				"deposit", deposit.Name,
				"error", result.Error)
		}
	}

	response.Success = response.ErrorCount == 0

	totalAccruedFloat, _ := response.TotalAccrued.Round(2).Float64()
	if response.SuccessCount > 0 {
		response.Message = fmt.Sprintf("Начислено процентов: %.2f руб. по %d вкладам", totalAccruedFloat, response.SuccessCount)

		slog.Info("Начисление процентов завершено успешно",
			"total_accrued", totalAccruedFloat,
			"success_count", response.SuccessCount)

	} else {
		response.Message = "Не найдено вкладов для начисления процентов"
		slog.Info("Не найдено вкладов для зачисления процентов")
	}

	return response, nil
}

func (s *InterestService) processDepositAccrual(deposit *models.Deposit) AccrualResult {
	income, description := s.calculateDailyInterest(deposit)
	if income.IsZero() || description == "" {
		return AccrualResult{
			DepositID:   deposit.ID,
			DepositName: deposit.Name,
			Income:      decimal.Zero,
			Success:     true,
			Error:       nil,
		}
	}

	amountKopecks := income.Mul(decimal.NewFromInt(100)).IntPart()

	if amountKopecks <= 0 {
		return AccrualResult{
			DepositID:   deposit.ID,
			DepositName: deposit.Name,
			Income:      income,
			Success:     true,
			Error:       nil,
		}
	}

	if err := storage.UpdateDepositAmount(deposit.ID, amountKopecks, config.AppConfig.DepositsDataPath); err != nil {
		return AccrualResult{
			DepositID:   deposit.ID,
			DepositName: deposit.Name,
			Income:      income,
			Success:     false,
			Error: errors.WrapError(
				errors.ErrStorage,
				fmt.Sprintf("ошибка обновления суммы вклада '%s'", deposit.Name),
				err,
			),
		}
	}

	return AccrualResult{
		DepositID:   deposit.ID,
		DepositName: deposit.Name,
		Income:      income,
		Success:     true,
		Error:       nil,
	}
}

func (s *InterestService) calculateDailyInterest(deposit *models.Deposit) (income decimal.Decimal, description string) {
	switch deposit.Type {
	case "savings":
		income := calculator.CalculateIncome(*deposit, 1)
		return income, "Ежедневная выплата процентов"
	case "term":
		return decimal.Zero, ""
	default:
		return decimal.Zero, ""
	}
}

type CalculateProjectedIncomeRequest struct {
	DepositID string
	Days      int
}

type CalculateProjectedIncomeResponse struct {
	Success         bool
	DepositName     string
	Amount          float64
	InterestRate    float64
	PeriodDays      int
	ProjectedIncome float64
	TotalAmount     float64
	Message         string
}

func (s *InterestService) CalculateProjectedIncome(req *CalculateProjectedIncomeRequest) (*CalculateProjectedIncomeResponse, error) {
	if req.Days <= 0 {
		return nil, errors.NewValidationError(
			"период расчета должен быть положительным",
			map[string]interface{}{
				"days": req.Days,
			},
		)
	}

	deposit, err := storage.GetDepositByID(req.DepositID, config.AppConfig.DepositsDataPath)
	if err != nil {
		return nil, errors.NewNotFoundError("вклад", req.DepositID)
	}

	income := calculator.CalculateIncome(*deposit, req.Days)
	incomeFloat, _ := income.Float64()
	amountRubles := float64(deposit.Amount) / 100.0

	return &CalculateProjectedIncomeResponse{
		Success:         true,
		DepositName:     deposit.Name,
		Amount:          amountRubles,
		InterestRate:    deposit.InterestRate,
		PeriodDays:      req.Days,
		ProjectedIncome: incomeFloat,
		TotalAmount:     amountRubles + incomeFloat,
		Message:         "Расчет projected income выполнен успешно",
	}, nil
}

package service

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

const defaultBalanceRechargeMultiplier = 1.0

type BalanceRechargeTier struct {
	Amount float64 `json:"amount"`
	Credit float64 `json:"credit"`
}

func normalizeBalanceRechargeMultiplier(multiplier float64) float64 {
	if math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier <= 0 {
		return defaultBalanceRechargeMultiplier
	}
	return multiplier
}

func parseBalanceRechargeTiers(raw string) []BalanceRechargeTier {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var tiers []BalanceRechargeTier
	if err := json.Unmarshal([]byte(raw), &tiers); err != nil {
		return nil
	}
	return normalizeBalanceRechargeTiers(tiers)
}

func normalizeBalanceRechargeTiers(tiers []BalanceRechargeTier) []BalanceRechargeTier {
	if len(tiers) == 0 {
		return nil
	}
	out := make([]BalanceRechargeTier, 0, len(tiers))
	for _, tier := range tiers {
		amount := roundPaymentAmount(tier.Amount)
		credit := roundPaymentAmount(tier.Credit)
		if amount <= 0 || credit <= 0 {
			continue
		}
		out = append(out, BalanceRechargeTier{Amount: amount, Credit: credit})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Amount < out[j].Amount
	})
	return out
}

func calculateCreditedBalance(paymentAmount, multiplier float64, tiers []BalanceRechargeTier) float64 {
	amount := roundPaymentAmount(paymentAmount)
	for _, tier := range normalizeBalanceRechargeTiers(tiers) {
		if tier.Amount == amount {
			return tier.Credit
		}
	}
	return decimal.NewFromFloat(amount).
		Mul(decimal.NewFromFloat(normalizeBalanceRechargeMultiplier(multiplier))).
		Round(2).
		InexactFloat64()
}

func roundPaymentAmount(amount float64) float64 {
	return decimal.NewFromFloat(amount).Round(2).InexactFloat64()
}

func calculateGatewayRefundAmount(orderAmount, payAmount, refundAmount float64, currency string) float64 {
	if orderAmount <= 0 || payAmount <= 0 || refundAmount <= 0 {
		return 0
	}
	fractionDigits := int32(payment.CurrencyMaxFractionDigits(currency))
	if math.Abs(refundAmount-orderAmount) <= paymentAmountToleranceForCurrency(currency) {
		return decimal.NewFromFloat(payAmount).Round(fractionDigits).InexactFloat64()
	}
	return decimal.NewFromFloat(payAmount).
		Mul(decimal.NewFromFloat(refundAmount)).
		Div(decimal.NewFromFloat(orderAmount)).
		Round(fractionDigits).
		InexactFloat64()
}

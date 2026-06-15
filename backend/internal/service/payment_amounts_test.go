package service

import "testing"

func TestCalculateCreditedBalanceUsesConfiguredRechargeTier(t *testing.T) {
	t.Parallel()

	tiers := []BalanceRechargeTier{
		{Amount: 10, Credit: 20},
		{Amount: 30, Credit: 70},
		{Amount: 300, Credit: 850},
	}

	got := calculateCreditedBalance(30, 2.5, tiers)
	if got != 70 {
		t.Fatalf("calculateCreditedBalance tier match = %v, want 70", got)
	}
}

func TestCalculateCreditedBalanceFallsBackToMultiplierForCustomAmount(t *testing.T) {
	t.Parallel()

	tiers := []BalanceRechargeTier{
		{Amount: 10, Credit: 20},
		{Amount: 30, Credit: 70},
	}

	got := calculateCreditedBalance(25, 2.5, tiers)
	if got != 62.5 {
		t.Fatalf("calculateCreditedBalance custom amount = %v, want 62.5", got)
	}
}

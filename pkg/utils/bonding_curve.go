package utils

import (
	// "fmt"
	"math"
)

// TradeResult represents the result of a trade operation
type TradeResult struct {
	GetToken            string  // "token" or "sol"
	GetAmount          float64
	CostAmount         float64
	PriceBeforeSwap    float64
	VSolAfterSwap      float64
	VTokenAfterSwap    float64
	PriceAfterSwap     float64
	ChangeRatioAfterSwap float64
}

// InputType and OutputType constants
const (
	InputVSol   = "vSol"
	InputVToken = "vToken"
	OutputVSol  = "vSol"
	OutputVToken = "vToken"
)

// EstimateBuyCostWithIncrease estimates the SOL cost needed to increase price by a given percentage
func EstimateBuyCostWithIncrease(increase, vSOL, vToken float64, feeRate float64) (*TradeResult, error) {
	priceBeforeSwap := vSOL / vToken
	targetPrice := priceBeforeSwap * (1 + increase)

	// Based on constant product formula: k = vSOL * vToken
	k := vSOL * vToken
	newVToken := math.Sqrt(k / targetPrice)
	newVSOL := k / newVToken

	solNeeded := newVSOL - vSOL
	// tokensOut := vToken - newVToken

	// Add fees
	totalCost := solNeeded / (1 - feeRate)

	// Calculate actual tokens received (verify with actual trade logic)
	solAfterFee := totalCost * (1 - feeRate)
	actualNewVSOL := vSOL + solAfterFee
	actualNewVToken := k / actualNewVSOL
	actualTokensOut := vToken - actualNewVToken

	// Post-trade reserves
	vSolAfterSwap := actualNewVSOL
	vTokenAfterSwap := actualNewVToken
	priceAfterSwap := vSolAfterSwap / vTokenAfterSwap
	changeRatioAfterSwap := (priceAfterSwap - priceBeforeSwap) / priceBeforeSwap

	return &TradeResult{
		GetToken:            "token",
		GetAmount:          actualTokensOut,
		CostAmount:         totalCost,
		PriceBeforeSwap:    priceBeforeSwap,
		VSolAfterSwap:      vSolAfterSwap,
		VTokenAfterSwap:    vTokenAfterSwap,
		PriceAfterSwap:     priceAfterSwap,
		ChangeRatioAfterSwap: changeRatioAfterSwap,
	}, nil
}

// EstimateSellReturnWithDecrease estimates the token amount needed to decrease price by a given percentage
func EstimateSellReturnWithDecrease(decrease, vSOL, vToken float64, feeRate float64) (*TradeResult, error) {
	priceBeforeSwap := vSOL / vToken
	targetPrice := priceBeforeSwap * (1 - decrease)

	k := vSOL * vToken
	newVToken := math.Sqrt(k / targetPrice)
	newVSOL := k / newVToken

	solToRemove := vSOL - newVSOL
	targetNetSolOut := solToRemove / (1 - feeRate)

	solBeforeFee := targetNetSolOut / (1 - feeRate)
	tokensToSell := k/(vSOL-solBeforeFee) - vToken

	// Calculate actual SOL amount (verify with actual trade logic)
	actualNewVToken := vToken + tokensToSell
	actualNewVSOL := k / actualNewVToken
	actualSolOut := vSOL - actualNewVSOL
	actualNetSolOut := actualSolOut * (1 - feeRate)

	// Post-trade reserves
	vSolAfterSwap := actualNewVSOL
	vTokenAfterSwap := actualNewVToken
	priceAfterSwap := vSolAfterSwap / vTokenAfterSwap
	changeRatioAfterSwap := (priceAfterSwap - priceBeforeSwap) / priceBeforeSwap

	return &TradeResult{
		GetToken:            "sol",
		GetAmount:          actualNetSolOut,
		CostAmount:         tokensToSell,
		PriceBeforeSwap:    priceBeforeSwap,
		VSolAfterSwap:      vSolAfterSwap,
		VTokenAfterSwap:    vTokenAfterSwap,
		PriceAfterSwap:     priceAfterSwap,
		ChangeRatioAfterSwap: changeRatioAfterSwap,
	}, nil
}

// SimulateBondingCurveAmountOut calculates the output amount given an input amount
func SimulateBondingCurveAmountOut(amountIn float64, inputType string, vSOL, vToken float64, feeRate float64) (*TradeResult, error) {
	priceBeforeSwap := vSOL / vToken

	if inputType == InputVSol {
		// Buy tokens with SOL
		solAfterFee := amountIn * (1 - feeRate)

		k := vSOL * vToken
		newVSOL := vSOL + solAfterFee
		newVToken := k / newVSOL
		tokensOut := vToken - newVToken

		vSolAfterSwap := newVSOL
		vTokenAfterSwap := newVToken
		priceAfterSwap := vSolAfterSwap / vTokenAfterSwap
		changeRatioAfterSwap := (priceAfterSwap - priceBeforeSwap) / priceBeforeSwap

		return &TradeResult{
			GetToken:            "token",
			GetAmount:          tokensOut,
			CostAmount:         amountIn,
			PriceBeforeSwap:    priceBeforeSwap,
			VSolAfterSwap:      vSolAfterSwap,
			VTokenAfterSwap:    vTokenAfterSwap,
			PriceAfterSwap:     priceAfterSwap,
			ChangeRatioAfterSwap: changeRatioAfterSwap,
		}, nil
	} else {
		// Sell tokens for SOL
		// if amountIn >= vToken {
		// 	return nil, fmt.Errorf("input amount exceeds virtual token reserves")
		// }

		k := vSOL * vToken
		newVToken := vToken + amountIn
		newVSOL := k / newVToken
		solOut := vSOL - newVSOL

		netSolOut := solOut * (1 - feeRate)

		vSolAfterSwap := newVSOL
		vTokenAfterSwap := newVToken
		priceAfterSwap := vSolAfterSwap / vTokenAfterSwap
		changeRatioAfterSwap := (priceAfterSwap - priceBeforeSwap) / priceBeforeSwap

		return &TradeResult{
			GetToken:            "sol",
			GetAmount:          netSolOut,
			CostAmount:         amountIn,
			PriceBeforeSwap:    priceBeforeSwap,
			VSolAfterSwap:      vSolAfterSwap,
			VTokenAfterSwap:    vTokenAfterSwap,
			PriceAfterSwap:     priceAfterSwap,
			ChangeRatioAfterSwap: changeRatioAfterSwap,
		}, nil
	}
}

// SimulateBondingCurveAmountIn calculates the input amount needed to get a desired output amount
func SimulateBondingCurveAmountIn(amountOut float64, outputType string, vSOL, vToken float64, feeRate float64) (*TradeResult, error) {
	priceBeforeSwap := vSOL / vToken

	if outputType == OutputVToken {
		// Buy tokens with SOL
		// if amountOut >= vToken {
		// 	return nil, fmt.Errorf("output amount exceeds virtual token reserves")
		// }

		k := vSOL * vToken
		newVToken := vToken - amountOut
		newVSOL := k / newVToken
		solNeeded := newVSOL - vSOL

		solInput := solNeeded / (1 - feeRate)

		vSolAfterSwap := newVSOL
		vTokenAfterSwap := newVToken
		priceAfterSwap := vSolAfterSwap / vTokenAfterSwap
		changeRatioAfterSwap := (priceAfterSwap - priceBeforeSwap) / priceBeforeSwap

		return &TradeResult{
			GetToken:            "token",
			GetAmount:          amountOut,
			CostAmount:         solInput,
			PriceBeforeSwap:    priceBeforeSwap,
			VSolAfterSwap:      vSolAfterSwap,
			VTokenAfterSwap:    vTokenAfterSwap,
			PriceAfterSwap:     priceAfterSwap,
			ChangeRatioAfterSwap: changeRatioAfterSwap,
		}, nil
	} else {
		// Sell tokens for SOL
		// if amountOut >= vSOL {
		// 	return nil, fmt.Errorf("output amount exceeds virtual SOL reserves")
		// }

		solBeforeFee := amountOut / (1 - feeRate)

		k := vSOL * vToken
		newVSOL := vSOL - solBeforeFee
		newVToken := k / newVSOL
		tokenIn := newVToken - vToken

		vSolAfterSwap := newVSOL
		vTokenAfterSwap := newVToken
		priceAfterSwap := vSolAfterSwap / vTokenAfterSwap
		changeRatioAfterSwap := (priceAfterSwap - priceBeforeSwap) / priceBeforeSwap

		return &TradeResult{
			GetToken:            "sol",
			GetAmount:          amountOut,
			CostAmount:         tokenIn,
			PriceBeforeSwap:    priceBeforeSwap,
			VSolAfterSwap:      vSolAfterSwap,
			VTokenAfterSwap:    vTokenAfterSwap,
			PriceAfterSwap:     priceAfterSwap,
			ChangeRatioAfterSwap: changeRatioAfterSwap,
		}, nil
	}
}

// GetVirtualReserves calculates virtual SOL and token reserves based on token amount
// vToken = 1073000000 - (1000000000 - tokenAmount)
// vSol = 32190000000 / vToken
func GetVirtualReserves(tokenAmount float64) (vSol float64, vToken float64) {
	vToken = 1073000000.0 - (1000000000.0 - tokenAmount)
	vSol = 32190000000.0 / vToken
	return vSol, vToken
} 
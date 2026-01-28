package utils

// SimulateConstantProductAmountOut: 已知某一侧的输入，求另一侧的输出
// inputType: "x" 表示用 x 换 y，"y" 表示用 y 换 x
func SimulateConstantProductAmountOut(amountIn float64, inputType string, x, y, fee float64) float64 {
	switch inputType {
	case "x":
		// 用 x 换 y
		dxWithFee := amountIn * (1 - fee)
		k := x * y
		newX := x + dxWithFee
		newY := k / newX
		dy := y - newY
		return dy
	case "y":
		// 用 y 换 x
		dyWithFee := amountIn * (1 - fee)
		k := x * y
		newY := y + dyWithFee
		newX := k / newY
		dx := x - newX
		return dx
	default:
		return 0
	}
}

// SimulateConstantProductAmountIn: 已知想要获得某一侧的输出，求需要输入多少另一侧
// outputType: "y" 表示想获得 y，"x" 表示想获得 x
func SimulateConstantProductAmountIn(amountOut float64, outputType string, x, y, fee float64) float64 {
	switch outputType {
	case "y":
		// 想获得 y，需要输入 x
		k := x * y
		newY := y - amountOut
		newX := k / newY
		dxWithFee := newX - x
		dx := dxWithFee / (1 - fee)
		return dx
	case "x":
		// 想获得 x，需要输入 y
		k := x * y
		newX := x - amountOut
		newY := k / newX
		dyWithFee := newY - y
		dy := dyWithFee / (1 - fee)
		return dy
	default:
		return 0
	} 
}

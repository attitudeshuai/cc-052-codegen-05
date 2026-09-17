package service

import "math"

// gramsPerKg 计算统一换算成克（整数），避免浮点误差。
// 产量与箱规均为 DECIMAL(?,2)，0.01kg = 10g，换算后必为整数。
const gramsPerKg = 1000

func kgToGrams(kg float64) int64 {
	return int64(math.Round(kg * gramsPerKg))
}

// PackCalc 按「一箱装 capacityKg 公斤」计算 yieldKg 能装的整箱数与装不满一箱的余量。
func PackCalc(yieldKg, capacityKg float64) (boxes int64, leftoverKg float64) {
	yg := kgToGrams(yieldKg)
	cg := kgToGrams(capacityKg)
	if yg <= 0 || cg <= 0 {
		return 0, 0
	}
	return yg / cg, float64(yg%cg) / gramsPerKg
}

// BalanceDiffKg 返回 总产量 - 各级别合计（kg）。
// >0 表示还有产量没分级，<0 表示分级合计超出总产量。
func BalanceDiffKg(totalKg float64, gradeYields []float64) float64 {
	diff := kgToGrams(totalKg)
	for _, y := range gradeYields {
		diff -= kgToGrams(y)
	}
	return float64(diff) / gramsPerKg
}

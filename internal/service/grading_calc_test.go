package service

import "testing"

func TestPackCalc(t *testing.T) {
	cases := []struct {
		name       string
		yieldKg    float64
		capacityKg float64
		wantBoxes  int64
		wantLeft   float64
	}{
		{"整除无余量", 100, 10, 10, 0},
		{"有余量", 107.5, 10, 10, 7.5},
		{"不足一箱", 7.5, 10, 0, 7.5},
		{"小数容量", 33, 2.5, 13, 0.5},
		{"两位小数产量", 0.07, 0.05, 1, 0.02},
		{"零产量", 0, 10, 0, 0},
		{"非法容量", 100, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			boxes, left := PackCalc(c.yieldKg, c.capacityKg)
			if boxes != c.wantBoxes {
				t.Errorf("boxes = %d, want %d", boxes, c.wantBoxes)
			}
			if left != c.wantLeft {
				t.Errorf("leftover = %v, want %v", left, c.wantLeft)
			}
			// 箱数*容量+余量 必须等于原产量
			if c.capacityKg > 0 && c.yieldKg > 0 {
				back := float64(boxes)*c.capacityKg + left
				if kgToGrams(back) != kgToGrams(c.yieldKg) {
					t.Errorf("boxes*capacity+leftover = %v, want %v", back, c.yieldKg)
				}
			}
		})
	}
}

func TestBalanceDiffKg(t *testing.T) {
	cases := []struct {
		name   string
		total  float64
		yields []float64
		want   float64
	}{
		{"正好分完", 1000, []float64{500, 300, 200}, 0},
		{"有产量未分级", 1000, []float64{500, 300}, 200},
		{"超出总产量", 1000, []float64{600, 500}, -100},
		{"小数平衡", 107.55, []float64{50.25, 57.3}, 0},
		{"小数差额", 100.01, []float64{50}, 50.01},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := BalanceDiffKg(c.total, c.yields); got != c.want {
				t.Errorf("BalanceDiffKg(%v, %v) = %v, want %v", c.total, c.yields, got, c.want)
			}
		})
	}
}

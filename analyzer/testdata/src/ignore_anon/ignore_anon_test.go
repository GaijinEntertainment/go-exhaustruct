package ignore_anon

import "testing"

type TestPosition struct {
	X int
	Y int
}

func (me TestPosition) Add(other TestPosition) TestPosition {
	return TestPosition{X: me.X + other.X, Y: me.Y + other.Y}
}

func TestPosition_Addition(t *testing.T) {
	for _, testCase := range []struct {
		a               TestPosition
		b               TestPosition
		expectPositiveX bool
		expectPositiveY bool
	}{
		{a: TestPosition{X: 1, Y: 1}, b: TestPosition{X: 1, Y: 1}, expectPositiveX: true, expectPositiveY: true},
		{a: TestPosition{X: 1, Y: 1}, b: TestPosition{X: -1, Y: -1}, expectPositiveX: false, expectPositiveY: false},
		{a: TestPosition{X: 1, Y: 1}, b: TestPosition{X: -1, Y: 1}, expectPositiveY: true},
		{a: TestPosition{X: 1, Y: 0}, b: TestPosition{X: 1}, expectPositiveX: true}, // want "ignore_anon.TestPosition is missing field Y"
	} {
		t.Run("Addition", func(t *testing.T) {
			sum := testCase.a.Add(testCase.b)
			if testCase.expectPositiveX && sum.X <= 0 {
				t.Errorf("expected positive X, got %d", sum.X)
			}
			if testCase.expectPositiveY && sum.Y <= 0 {
				t.Errorf("expected positive Y, got %d", sum.Y)
			}
		})
	}
}

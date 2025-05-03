package bullion_main_server_interfaces

import (
	"testing"
)

func TestCalculate(t *testing.T) {
	snapshot := &CshPremiumBuySellSnapshot{
		Premium: 10,
		Tax:     5,
	}

	t.Run("Positive values calculation", func(t *testing.T) {
		symbol := 100.0
		expected := 115.5
		result := Calculate(symbol, snapshot)
		if result != expected {
			t.Errorf("Expected %f, but got %f", expected, result)
		}
	})

	t.Run("Negative values calculation", func(t *testing.T) {
		symbol := -50.0
		expected := -42.0
		result := Calculate(symbol, snapshot)
		if result != expected {
			t.Errorf("Expected %f, but got %f", expected, result)
		}
	})

	t.Run("Zero values calculation", func(t *testing.T) {
		symbol := 0.0
		expected := 10.5
		result := Calculate(symbol, snapshot)
		if result != expected {
			t.Errorf("Expected %f, but got %f", expected, result)
		}
	})
}

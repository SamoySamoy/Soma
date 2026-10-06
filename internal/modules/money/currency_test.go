package money

import "testing"

func TestConvert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		amount int64
		from   string
		to     string
		rate   int64
		want   int64
	}{
		{"same currency is unchanged", 12345, "VND", "VND", 0, 12345},
		// 10.00 USD at 25,000 VND per USD = 250,000 VND.
		{"USD to VND", 1000, "USD", "VND", 25_000 * rateScale, 250_000},
		// 250,000 VND at 1/25,000 USD per VND = 10.00 USD = 1000 cents.
		{"VND to USD", 250_000, "VND", "USD", rateScale / 25_000, 1000},
		// At 50 VND per USD cent, 1 cent is exactly 0.5 VND: a tie rounds to the
		// even 0; 3 cents is 1.5 VND, which rounds to the even 2.
		{"tie rounds to even down", 1, "USD", "VND", 50 * rateScale, 0},
		{"tie rounds to even up", 3, "USD", "VND", 50 * rateScale, 2},
		{"negative ties round symmetrically", -1, "USD", "VND", 50 * rateScale, 0},
		// Three-digit KWD into two-digit USD: 1.000 KWD = 3.25 USD at 3.25 USD per KWD.
		{"KWD to USD", 1000, "KWD", "USD", 325 * rateScale / 100, 325},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Convert(tt.amount, tt.from, tt.to, tt.rate)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Convert(%d %s->%s) = %d, want %d", tt.amount, tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestConvertRejectsUnknownCurrency(t *testing.T) {
	t.Parallel()
	if _, err := Convert(1, "XXX", "VND", rateScale); err == nil {
		t.Error("Convert with an unknown currency succeeded")
	}
}

func TestCurrenciesAreSortedAndKnown(t *testing.T) {
	t.Parallel()
	list := Currencies()
	for i := 1; i < len(list); i++ {
		if list[i-1].Code >= list[i].Code {
			t.Fatalf("currencies not sorted at %s", list[i].Code)
		}
	}
	if !Supported("VND") || Supported("ZZZ") {
		t.Error("Supported() gave the wrong answer")
	}
}

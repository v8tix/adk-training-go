package calculator

import (
	"encoding/json"
	"testing"
)

// These are pure, fast, no-LLM unit tests — the base of the test pyramid for
// this module. agent_test.go covers the LLM actually choosing and invoking
// these tools; these tests only cover the tools' own arithmetic and error
// handling.

// TestCalcResult_ZeroResultIsMarshaled proves a real bug class the other
// tests can't catch: they assert on CalcResult's Go struct field directly,
// never on what actually gets marshaled and handed to the model. A zero
// Result is a legitimate computed value (0+0, 5-5, 7*0, 0/5 all produce it),
// not an absent one — CalcResult.Result must not have an "omitempty" tag, or
// a "success" result with a real zero silently loses its number.
func TestCalcResult_ZeroResultIsMarshaled(t *testing.T) {
	result := CalcResult{Status: "success", Result: 0}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal(%+v) error = %v", result, err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", raw, err)
	}
	if _, ok := decoded["result"]; !ok {
		t.Errorf("marshaled JSON %s has no \"result\" key for a zero result — the model would see a success response with no number", raw)
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		args AddArgs
		want float64
	}{
		{name: "positive numbers", args: AddArgs{A: 42, B: 118}, want: 160},
		{name: "negative numbers", args: AddArgs{A: -5, B: -3}, want: -8},
		{name: "zero", args: AddArgs{A: 0, B: 0}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := add(nil, tt.args)
			if err != nil {
				t.Fatalf("add() error = %v, want nil", err)
			}
			if got.Status != "success" {
				t.Fatalf("add() Status = %q, want %q", got.Status, "success")
			}
			if got.Result != tt.want {
				t.Errorf("add() Result = %v, want %v", got.Result, tt.want)
			}
		})
	}
}

func TestSubtract(t *testing.T) {
	tests := []struct {
		name string
		args SubtractArgs
		want float64
	}{
		{name: "positive result", args: SubtractArgs{A: 10, B: 4}, want: 6},
		{name: "negative result", args: SubtractArgs{A: 4, B: 10}, want: -6},
		{name: "zero", args: SubtractArgs{A: 5, B: 5}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := subtract(nil, tt.args)
			if err != nil {
				t.Fatalf("subtract() error = %v, want nil", err)
			}
			if got.Status != "success" {
				t.Fatalf("subtract() Status = %q, want %q", got.Status, "success")
			}
			if got.Result != tt.want {
				t.Errorf("subtract() Result = %v, want %v", got.Result, tt.want)
			}
		})
	}
}

func TestMultiply(t *testing.T) {
	tests := []struct {
		name string
		args MultiplyArgs
		want float64
	}{
		{name: "positive numbers", args: MultiplyArgs{A: 15, B: 3}, want: 45},
		{name: "multiply by zero", args: MultiplyArgs{A: 7, B: 0}, want: 0},
		{name: "negative numbers", args: MultiplyArgs{A: -4, B: 3}, want: -12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := multiply(nil, tt.args)
			if err != nil {
				t.Fatalf("multiply() error = %v, want nil", err)
			}
			if got.Status != "success" {
				t.Fatalf("multiply() Status = %q, want %q", got.Status, "success")
			}
			if got.Result != tt.want {
				t.Errorf("multiply() Result = %v, want %v", got.Result, tt.want)
			}
		})
	}
}

func TestDivide(t *testing.T) {
	tests := []struct {
		name      string
		args      DivideArgs
		wantValue float64
		wantErr   bool
	}{
		{name: "exact division", args: DivideArgs{A: 10, B: 2}, wantValue: 5},
		{name: "non-exact division", args: DivideArgs{A: 7, B: 2}, wantValue: 3.5},
		{name: "division by zero returns a structured error, not a Go error", args: DivideArgs{A: 10, B: 0}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := divide(nil, tt.args)
			if err != nil {
				t.Fatalf("divide() error = %v, want nil (errors must be structured results, not Go errors)", err)
			}
			if tt.wantErr {
				if got.Status != "error" {
					t.Fatalf("divide() Status = %q, want %q", got.Status, "error")
				}
				if got.Error == "" {
					t.Error("divide() Error message is empty, want a description of the failure")
				}
				return
			}
			if got.Status != "success" {
				t.Fatalf("divide() Status = %q, want %q", got.Status, "success")
			}
			if got.Result != tt.wantValue {
				t.Errorf("divide() Result = %v, want %v", got.Result, tt.wantValue)
			}
		})
	}
}

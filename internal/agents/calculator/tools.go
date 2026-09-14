package calculator

import "google.golang.org/adk/v2/agent"

// AddArgs, SubtractArgs, MultiplyArgs, and DivideArgs are kept as four
// distinct types rather than one shared two-number type — functiontool.New
// is generic per tool, and four separate, obviously-named types read
// clearer at each functiontool.New call site than one generic type whose
// name doesn't say which operation it's for.

// AddArgs holds the two numbers to add.
type AddArgs struct {
	A int `json:"a" jsonschema:"the first number"`
	B int `json:"b" jsonschema:"the second number"`
}

// SubtractArgs holds the two numbers to subtract.
type SubtractArgs struct {
	A int `json:"a" jsonschema:"the first number"`
	B int `json:"b" jsonschema:"the second number to subtract"`
}

// MultiplyArgs holds the two numbers to multiply.
type MultiplyArgs struct {
	A int `json:"a" jsonschema:"the first number"`
	B int `json:"b" jsonschema:"the second number"`
}

// DivideArgs holds the numerator and denominator to divide.
type DivideArgs struct {
	A int `json:"a" jsonschema:"the numerator"`
	B int `json:"b" jsonschema:"the denominator"`
}

// CalcResult is the uniform result shape every calculator tool returns,
// matching Python's lab's own dict shape ({"status": ..., "result": ...} or
// an error dict). All four tools share this one type, unlike their args
// types, because they all produce the exact same kind of result.
type CalcResult struct {
	Status string `json:"status"`
	// Result deliberately has no "omitempty" — unlike Error, 0 is a
	// legitimate computed value (0+0, 5-5, 7*0, 0/5 all produce it), and
	// "omitempty" on a float64 zero-value would silently drop it from the
	// JSON handed to the model, leaving a "success" result with no number
	// for the LLM to work with. Confirmed live: json.Marshal on a
	// zero-Result CalcResult with omitempty produced {"status":"success"}
	// with no "result" key at all.
	Result float64 `json:"result"`
	Error  string  `json:"error,omitempty"`
}

func add(_ agent.Context, args AddArgs) (CalcResult, error) {
	return CalcResult{Status: "success", Result: float64(args.A + args.B)}, nil
}

func subtract(_ agent.Context, args SubtractArgs) (CalcResult, error) {
	return CalcResult{Status: "success", Result: float64(args.A - args.B)}, nil
}

func multiply(_ agent.Context, args MultiplyArgs) (CalcResult, error) {
	return CalcResult{Status: "success", Result: float64(args.A * args.B)}, nil
}

// divide returns a structured error result for division by zero — not a Go
// error — so the LLM receives a reasoning-able result instead of the tool
// call itself failing at the framework level, matching Python's lab
// instruction to "return an error dictionary."
func divide(_ agent.Context, args DivideArgs) (CalcResult, error) {
	if args.B == 0 {
		return CalcResult{Status: "error", Error: "division by zero"}, nil
	}
	return CalcResult{Status: "success", Result: float64(args.A) / float64(args.B)}, nil
}

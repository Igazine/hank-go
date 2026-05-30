package ext

import (
	"math"
	"github.com/Igazine/hank-go"
)

const safeIntMax = 9007199254740991.0

func checkSafeInt(n float64, taskName string) (int64, *hank.Value) {
	if math.Abs(n) > safeIntMax || math.IsInf(n, 0) || math.IsNaN(n) {
		return 0, &hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.BitwiseOutOfBounds, Args: []hank.Value{{Type: hank.TypeNumber, Number: n}, {Type: hank.TypeString, String: taskName}}}}
	}
	return int64(n), nil
}

func fromSafeInt(n int64, taskName string) hank.Value {
	f := float64(n)
	if math.Abs(f) > safeIntMax {
		return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.BitwiseOutOfBounds, Args: []hank.Value{{Type: hank.TypeNumber, Number: f}, {Type: hank.TypeString, String: taskName}}}}
	}
	return hank.Value{Type: hank.TypeNumber, Number: f}
}

type PlatformExtension struct{}

func (e *PlatformExtension) Name() string {
	return "PlatformExtension"
}

func (e *PlatformExtension) GetModules() map[string]map[string]hank.NativeFunc {
	mods := make(map[string]map[string]hank.NativeFunc)

	mods["bin"] = map[string]hank.NativeFunc{
		"and": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			var a, b float64
			if len(args) < 2 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeNumber {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "Number"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "bin.and"}}}}
			}
			if args[1].Type != hank.TypeNumber {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "Number"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "bin.and"}}}}
			}
			a = args[0].Number
			b = args[1].Number

			ia, err := checkSafeInt(a, "bin.and")
			if err != nil { return *err }
			ib, err := checkSafeInt(b, "bin.and")
			if err != nil { return *err }
			return fromSafeInt(ia & ib, "bin.and")
		},
		"or": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			var a, b float64
			if len(args) < 2 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeNumber {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "Number"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "bin.or"}}}}
			}
			if args[1].Type != hank.TypeNumber {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "Number"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "bin.or"}}}}
			}
			a = args[0].Number
			b = args[1].Number

			ia, err := checkSafeInt(a, "bin.or")
			if err != nil { return *err }
			ib, err := checkSafeInt(b, "bin.or")
			if err != nil { return *err }
			return fromSafeInt(ia | ib, "bin.or")
		},
		"xor": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			var a, b float64
			if len(args) < 2 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeNumber {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "Number"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "bin.xor"}}}}
			}
			if args[1].Type != hank.TypeNumber {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "Number"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "bin.xor"}}}}
			}
			a = args[0].Number
			b = args[1].Number

			ia, err := checkSafeInt(a, "bin.xor")
			if err != nil { return *err }
			ib, err := checkSafeInt(b, "bin.xor")
			if err != nil { return *err }
			return fromSafeInt(ia ^ ib, "bin.xor")
		},
		"not": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 || args[0].Type != hank.TypeNumber {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "Number"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "bin.not"}}}}
			}
			a := args[0].Number
			ia, err := checkSafeInt(a, "bin.not")
			if err != nil { return *err }
			return fromSafeInt(^ia, "bin.not")
		},
		"shiftL": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) < 2 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeNumber || args[1].Type != hank.TypeNumber {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "Number"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "bin.shiftL"}}}}
			}
			a := args[0].Number
			b := args[1].Number

			ia, err := checkSafeInt(a, "bin.shiftL")
			if err != nil { return *err }
			return fromSafeInt(ia << uint(b), "bin.shiftL")
		},
		"shiftR": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) < 2 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeNumber || args[1].Type != hank.TypeNumber {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "Number"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "bin.shiftR"}}}}
			}
			a := args[0].Number
			b := args[1].Number

			ia, err := checkSafeInt(a, "bin.shiftR")
			if err != nil { return *err }
			return fromSafeInt(ia >> uint(b), "bin.shiftR")
		},
	}

	return mods
}

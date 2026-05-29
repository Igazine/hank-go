package ext

import (
	"math"
	"github.com/Igazine/hank-go"
)

const safeIntMax = 9007199254740991.0

func checkSafeInt(n float64) int64 {
	if math.Abs(n) > safeIntMax || math.IsInf(n, 0) || math.IsNaN(n) {
		panic(hank.CreateHankError(hank.BitwiseOutOfBounds, []interface{}{n}, "", 0, ""))
	}
	return int64(n)
}

func fromSafeInt(n int64) float64 {
	f := float64(n)
	if math.Abs(f) > safeIntMax {
		panic(hank.CreateHankError(hank.BitwiseOutOfBounds, []interface{}{f}, "", 0, ""))
	}
	return f
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
			if len(args) > 0 && args[0].Type == hank.TypeNumber {
				a = args[0].Number
			}
			if len(args) > 1 && args[1].Type == hank.TypeNumber {
				b = args[1].Number
			}
			return hank.Value{Type: hank.TypeNumber, Number: fromSafeInt(checkSafeInt(a) & checkSafeInt(b))}
		},
		"or": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			var a, b float64
			if len(args) > 0 && args[0].Type == hank.TypeNumber {
				a = args[0].Number
			}
			if len(args) > 1 && args[1].Type == hank.TypeNumber {
				b = args[1].Number
			}
			return hank.Value{Type: hank.TypeNumber, Number: fromSafeInt(checkSafeInt(a) | checkSafeInt(b))}
		},
		"xor": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			var a, b float64
			if len(args) > 0 && args[0].Type == hank.TypeNumber {
				a = args[0].Number
			}
			if len(args) > 1 && args[1].Type == hank.TypeNumber {
				b = args[1].Number
			}
			return hank.Value{Type: hank.TypeNumber, Number: fromSafeInt(checkSafeInt(a) ^ checkSafeInt(b))}
		},
		"not": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			var a float64
			if len(args) > 0 && args[0].Type == hank.TypeNumber {
				a = args[0].Number
			}
			return hank.Value{Type: hank.TypeNumber, Number: fromSafeInt(^checkSafeInt(a))}
		},
		"shiftL": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			var a, b float64
			if len(args) > 0 && args[0].Type == hank.TypeNumber {
				a = args[0].Number
			}
			if len(args) > 1 && args[1].Type == hank.TypeNumber {
				b = args[1].Number
			}
			return hank.Value{Type: hank.TypeNumber, Number: fromSafeInt(checkSafeInt(a) << uint(b))}
		},
		"shiftR": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			var a, b float64
			if len(args) > 0 && args[0].Type == hank.TypeNumber {
				a = args[0].Number
			}
			if len(args) > 1 && args[1].Type == hank.TypeNumber {
				b = args[1].Number
			}
			return hank.Value{Type: hank.TypeNumber, Number: fromSafeInt(checkSafeInt(a) >> uint(b))}
		},
	}

	return mods
}

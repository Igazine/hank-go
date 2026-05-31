package hank

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type StdLib struct {
	EnvState map[string]Value
}

func NewStdLib() *StdLib {
	return &StdLib{EnvState: make(map[string]Value)}
}

func (s *StdLib) Name() string {
	return "StdLib"
}

func (s *StdLib) GetTasks() map[string]NativeFunc {
	tasks := make(map[string]NativeFunc)

	// log
	tasks["log_print"] = func(args []Value, ctx ExecutionContext) Value {
		var strs []string
		for _, a := range args {
			strs = append(strs, ValueToString(a))
		}
		fmt.Println(strings.Join(strs, " "))
		return Value{Type: TypeVoid}
	}
	tasks["log_error"] = func(args []Value, ctx ExecutionContext) Value {
		var strs []string
		for _, a := range args {
			strs = append(strs, ValueToString(a))
		}
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(strs, " "))
		return Value{Type: TypeVoid}
	}
	tasks["log_warn"] = func(args []Value, ctx ExecutionContext) Value {
		var strs []string
		for _, a := range args {
			strs = append(strs, ValueToString(a))
		}
		fmt.Printf("WARNING: %s\n", strings.Join(strs, " "))
		return Value{Type: TypeVoid}
	}

	// runtime
	tasks["runtime_halt"] = func(args []Value, ctx ExecutionContext) Value {
		code := 0
		if len(args) > 0 && args[0].Type == TypeNumber {
			code = int(args[0].Number)
		}
		os.Exit(code)
		return Value{Type: TypeVoid}
	}
	tasks["runtime_elapsedTime"] = func(args []Value, ctx ExecutionContext) Value {
		return Value{Type: TypeNumber, Number: float64(time.Now().UnixNano() / 1e6)}
	}
	tasks["runtime_signal"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) > 0 {
			fmt.Printf("[SIGNAL] %v\n", ValueToString(args[0]))
		}
		return Value{Type: TypeVoid}
	}

	// loop
	tasks["loop_while"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		cond := args[0]
		body := args[1]
		var last Value = Value{Type: TypeVoid}
		for {
			condVal := ctx.Call(cond, []Value{})
			if ctx.IsError(condVal) {
				return condVal
			}
			if condVal.Type == TypeVoid {
				break
			}
			res := ctx.Call(body, []Value{})
			if res.Type == TypeOpaque && res.Opaque.Label == "__ControlFlow" && fmt.Sprintf("%v", res.Opaque.Data) == "Break" {
				break
			}
			if ctx.IsError(res) {
				return res
			}
			last = res
		}
		return last
	}
	tasks["loop_break"] = func(args []Value, ctx ExecutionContext) Value {
		return Value{Type: TypeOpaque, Opaque: &OpaqueValue{Label: "__ControlFlow", Data: "Break"}}
	}

	// env
	tasks["env_get"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		key := ValueToString(args[0])
		if val, ok := s.EnvState[key]; ok {
			return val
		}
		return Value{Type: TypeVoid}
	}
	tasks["env_set"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		key := ValueToString(args[0])
		s.EnvState[key] = args[1]
		return Value{Type: TypeVoid}
	}
	tasks["env_keys"] = func(args []Value, ctx ExecutionContext) Value {
		keys := make([]Value, 0, len(s.EnvState))
		for k := range s.EnvState {
			keys = append(keys, Value{Type: TypeString, String: k})
		}
		return Value{Type: TypeArray, Array: &keys}
	}

	// str
	tasks["str_length"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeString {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "String"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "str_length"}}}}
		}
		return Value{Type: TypeNumber, Number: float64(len(args[0].String))}
	}
	tasks["str_format"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		template := ValueToString(args[0])
		res := template
		for i := 1; i < len(args); i++ {
			placeholder := fmt.Sprintf("%%%d", i)
			res = strings.ReplaceAll(res, placeholder, ValueToString(args[i]))
		}
		return Value{Type: TypeString, String: res}
	}
	tasks["str_concat"] = func(args []Value, ctx ExecutionContext) Value {
		var res strings.Builder
		for _, a := range args {
			res.WriteString(ValueToString(a))
		}
		return Value{Type: TypeString, String: res.String()}
	}
	tasks["str_trim"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeString {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "String"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "str_trim"}}}}
		}
		return Value{Type: TypeString, String: strings.TrimSpace(args[0].String)}
	}

	// num
	tasks["num_parse"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		s := ValueToString(args[0])
		base := 0
		if len(args) > 1 && args[1].Type == TypeNumber {
			base = int(args[1].Number)
		}
		val, err := strconv.ParseInt(s, base, 64)
		if err != nil {
			if base == 0 || base == 10 {
				fval, err := strconv.ParseFloat(s, 64)
				if err == nil {
					return Value{Type: TypeNumber, Number: fval}
				}
			}
			return Value{Type: TypeVoid}
		}
		return Value{Type: TypeNumber, Number: float64(val)}
	}
	tasks["num_format"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 || args[0].Type != TypeNumber {
			return Value{Type: TypeVoid}
		}
		n := int64(args[0].Number)
		base := 10
		if len(args) > 1 && args[1].Type == TypeNumber {
			base = int(args[1].Number)
		}
		if base < 2 || base > 36 {
			return Value{Type: TypeVoid}
		}
		return Value{Type: TypeString, String: strconv.FormatInt(n, base)}
	}

	// regex
	tasks["regex_parse"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		pattern := ValueToString(args[0])
		flags := ""
		if len(args) > 1 {
			flags = ValueToString(args[1])
		}
		goFlags := ""
		if strings.Contains(flags, "i") {
			goFlags += "i"
		}
		if strings.Contains(flags, "m") {
			goFlags += "m"
		}
		finalPattern := pattern
		if goFlags != "" {
			finalPattern = "(?" + goFlags + ")" + pattern
		}
		re, err := regexp.Compile(finalPattern)
		if err != nil {
			return Value{Type: TypeVoid}
		}
		return Value{Type: TypeOpaque, Opaque: &OpaqueValue{Label: "RegExp", Data: re}}
	}
	tasks["regex_match"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		s := ValueToString(args[0])
		patternVal := args[1]
		if patternVal.Type == TypeOpaque && patternVal.Opaque.Label == "RegExp" {
			re := patternVal.Opaque.Data.(*regexp.Regexp)
			if re.MatchString(s) {
				return Value{Type: TypeNumber, Number: 1}
			}
		} else {
			if strings.Contains(s, ValueToString(patternVal)) {
				return Value{Type: TypeNumber, Number: 1}
			}
		}
		return Value{Type: TypeVoid}
	}
	tasks["regex_replace"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 3 {
			return Value{Type: TypeVoid}
		}
		s := ValueToString(args[0])
		repl := ValueToString(args[2])
		if args[1].Type == TypeOpaque && args[1].Opaque.Label == "RegExp" {
			re := args[1].Opaque.Data.(*regexp.Regexp)
			return Value{Type: TypeString, String: re.ReplaceAllString(s, repl)}
		} else {
			return Value{Type: TypeString, String: strings.ReplaceAll(s, ValueToString(args[1]), repl)}
		}
	}

	// math
	tasks["math_add"] = func(args []Value, ctx ExecutionContext) Value {
		res := 0.0
		for _, a := range args {
			if a.Type != TypeNumber {
				return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(a.Type)}, {Type: TypeString, String: "math_add"}}}}
			}
			res += a.Number
		}
		return Value{Type: TypeNumber, Number: res}
	}
	tasks["math_sub"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeNumber {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "math_sub"}}}}
		}
		if args[1].Type != TypeNumber {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(args[1].Type)}, {Type: TypeString, String: "math_sub"}}}}
		}
		return Value{Type: TypeNumber, Number: args[0].Number - args[1].Number}
	}
	tasks["math_mul"] = func(args []Value, ctx ExecutionContext) Value {
		res := 1.0
		if len(args) == 0 {
			return Value{Type: TypeNumber, Number: 0}
		}
		for _, a := range args {
			if a.Type != TypeNumber {
				return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(a.Type)}, {Type: TypeString, String: "math_mul"}}}}
			}
			res *= a.Number
		}
		return Value{Type: TypeNumber, Number: res}
	}
	tasks["math_div"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeNumber {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "math_div"}}}}
		}
		if args[1].Type != TypeNumber {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(args[1].Type)}, {Type: TypeString, String: "math_div"}}}}
		}
		if args[1].Number == 0 {
			return Value{Type: TypeVoid}
		}
		return Value{Type: TypeNumber, Number: args[0].Number / args[1].Number}
	}
	tasks["math_gt"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeNumber {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "math_gt"}}}}
		}
		if args[1].Type != TypeNumber {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(args[1].Type)}, {Type: TypeString, String: "math_gt"}}}}
		}
		if args[0].Number > args[1].Number {
			return Value{Type: TypeNumber, Number: 1}
		}
		return Value{Type: TypeVoid}
	}
	tasks["math_lt"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeNumber {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "math_lt"}}}}
		}
		if args[1].Type != TypeNumber {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(args[1].Type)}, {Type: TypeString, String: "math_lt"}}}}
		}
		if args[0].Number < args[1].Number {
			return Value{Type: TypeNumber, Number: 1}
		}
		return Value{Type: TypeVoid}
	}
	tasks["math_eq"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		if hankEquals(args[0], args[1]) {
			return Value{Type: TypeNumber, Number: 1}
		}
		return Value{Type: TypeVoid}
	}

	// logic
	tasks["logic_and"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		var last Value = Value{Type: TypeVoid}
		for _, a := range args {
			if a.Type == TypeVoid {
				return Value{Type: TypeVoid}
			}
			last = a
		}
		return last
	}
	tasks["logic_or"] = func(args []Value, ctx ExecutionContext) Value {
		for _, a := range args {
			if a.Type != TypeVoid {
				return a
			}
		}
		return Value{Type: TypeVoid}
	}
	tasks["logic_eq"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		if hankEquals(args[0], args[1]) {
			return Value{Type: TypeNumber, Number: 1}
		}
		return Value{Type: TypeVoid}
	}

	// arr
	tasks["arr_length"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeArray {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Array"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "arr_length"}}}}
		}
		return Value{Type: TypeNumber, Number: float64(len(*args[0].Array))}
	}
	tasks["arr_get"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeArray {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Array"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "arr_get"}}}}
		}
		if args[1].Type != TypeNumber {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Number"}, {Type: TypeString, String: typeToString(args[1].Type)}, {Type: TypeString, String: "arr_get"}}}}
		}
		idx := int(args[1].Number)
		if idx < 0 || idx >= len(*args[0].Array) {
			return Value{Type: TypeVoid}
		}
		return (*args[0].Array)[idx]
	}
	tasks["arr_push"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeArray {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Array"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "arr_push"}}}}
		}
		*args[0].Array = append(*args[0].Array, args[1])
		return Value{Type: TypeVoid}
	}
	tasks["arr_pop"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeArray {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Array"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "arr_pop"}}}}
		}
		if len(*args[0].Array) == 0 {
			return Value{Type: TypeVoid}
		}
		idx := len(*args[0].Array) - 1
		val := (*args[0].Array)[idx]
		*args[0].Array = (*args[0].Array)[:idx]
		return val
	}
	tasks["arr_each"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeArray {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Array"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "arr_each"}}}}
		}
		items := make([]Value, len(*args[0].Array))
		copy(items, *args[0].Array)
		for idx, item := range items {
			res := ctx.Call(args[1], []Value{item, {Type: TypeNumber, Number: float64(idx)}})
			if res.Type == TypeOpaque && res.Opaque.Label == "__ControlFlow" && fmt.Sprintf("%v", res.Opaque.Data) == "Break" {
				break
			}
			if ctx.IsError(res) {
				return res
			}
		}
		return Value{Type: TypeVoid}
	}

	// map
	tasks["map_get"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 2 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeMap {
			return Value{Type: TypeVoid}
		}
		key := ValueToString(args[1])
		if val, ok := args[0].Map[key]; ok {
			return val
		}
		return Value{Type: TypeVoid}
	}
	tasks["map_set"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) < 3 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeMap {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Map"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "map_set"}}}}
		}
		key := ValueToString(args[1])
		args[0].Map[key] = args[2]
		return Value{Type: TypeVoid}
	}
	tasks["map_keys"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 || args[0].Type != TypeMap {
			return Value{Type: TypeVoid}
		}
		var keys []Value
		for k := range args[0].Map {
			keys = append(keys, Value{Type: TypeString, String: k})
		}
		return Value{Type: TypeArray, Array: &keys}
	}

	// json
	tasks["json_parse"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		s := ValueToString(args[0])
		var data interface{}
		if err := json.Unmarshal([]byte(s), &data); err != nil {
			return Value{Type: TypeVoid}
		}
		return mapAnyToHank(data)
	}
	tasks["json_stringify"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		any, ok := mapHankToAny(args[0])
		if !ok {
			return Value{Type: TypeVoid}
		}
		b, err := json.Marshal(any)
		if err != nil {
			return Value{Type: TypeVoid}
		}
		return Value{Type: TypeString, String: string(b)}
	}

	// err
	tasks["err_code"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeError {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Error"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "err_code"}}}}
		}
		return Value{Type: TypeNumber, Number: float64(args[0].Error.Code)}
	}
	tasks["err_message"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeError {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Error"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "err_message"}}}}
		}
		err := args[0].Error
		loc := ctx.GetLocalization()
		tmpl, ok := loc[int(err.Code)]
		if !ok {
			tmpl = "Unknown Error"
		}
		for i, arg := range err.Args {
			tmpl = strings.ReplaceAll(tmpl, fmt.Sprintf("{%d}", i), ValueToString(arg))
		}
		return Value{Type: TypeString, String: tmpl}
	}
	tasks["err_args"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) == 0 {
			return Value{Type: TypeVoid}
		}
		if args[0].Type != TypeError {
			return Value{Type: TypeError, Error: &ErrorValue{Code: TypeMismatch, Args: []Value{{Type: TypeString, String: "Error"}, {Type: TypeString, String: typeToString(args[0].Type)}, {Type: TypeString, String: "err_args"}}}}
		}
		return Value{Type: TypeArray, Array: &args[0].Error.Args}
	}
	tasks["err_isError"] = func(args []Value, ctx ExecutionContext) Value {
		if len(args) > 0 && args[0].Type == TypeError {
			return Value{Type: TypeNumber, Number: 1}
		}
		return Value{Type: TypeVoid}
	}

	return tasks
}

func hankEquals(a, b Value) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case TypeVoid:
		return true
	case TypeNumber:
		return a.Number == b.Number
	case TypeString:
		return a.String == b.String
	case TypeArray:
		if len(*a.Array) != len(*b.Array) {
			return false
		}
		for i := range *a.Array {
			if !hankEquals((*a.Array)[i], (*b.Array)[i]) {
				return false
			}
		}
		return true
	case TypeMap:
		if len(a.Map) != len(b.Map) {
			return false
		}
		for k, v1 := range a.Map {
			v2, ok := b.Map[k]
			if !ok || !hankEquals(v1, v2) {
				return false
			}
		}
		return true
	case TypeOpaque:
		return a.Opaque.Label == b.Opaque.Label && a.Opaque.Data == b.Opaque.Data
	case TypeError:
		if a.Error.Code != b.Error.Code || len(a.Error.Args) != len(b.Error.Args) {
			return false
		}
		for i := range a.Error.Args {
			if !hankEquals(a.Error.Args[i], b.Error.Args[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func mapAnyToHank(v interface{}) Value {
	if v == nil {
		return Value{Type: TypeVoid}
	}
	if s, ok := v.(IHALSerializable); ok {
		return Value{Type: TypeString, String: s.SerializeHAL()}
	}
	switch val := v.(type) {
	case string:
		return Value{Type: TypeString, String: val}
	case float64:
		return Value{Type: TypeNumber, Number: val}
	case int:
		return Value{Type: TypeNumber, Number: float64(val)}
	case bool:
		if val {
			return Value{Type: TypeNumber, Number: 1}
		}
		return Value{Type: TypeVoid}
	case []interface{}:
		arr := make([]Value, 0, len(val))
		for _, item := range val {
			arr = append(arr, mapAnyToHank(item))
		}
		return Value{Type: TypeArray, Array: &arr}
	case map[string]interface{}:
		m := make(map[string]Value)
		for k, v := range val {
			m[k] = mapAnyToHank(v)
		}
		return Value{Type: TypeMap, Map: m}
	default:
		return Value{Type: TypeVoid}
	}
}

func mapHankToAny(v Value) (interface{}, bool) {
	switch v.Type {
	case TypeString:
		return v.String, true
	case TypeNumber:
		return v.Number, true
	case TypeArray:
		var arr []interface{}
		for _, item := range *v.Array {
			any, ok := mapHankToAny(item)
			if !ok {
				return nil, false
			}
			arr = append(arr, any)
		}
		return arr, true
	case TypeMap:
		m := make(map[string]interface{})
		for k, val := range v.Map {
			any, ok := mapHankToAny(val)
			if !ok {
				return nil, false
			}
			m[k] = any
		}
		return m, true
	case TypeOpaque:
		return nil, false // Non-serializable
	default:
		return nil, true
	}
}

func typeToString(t ValueType) string {
	switch t {
	case TypeVoid:
		return "Void"
	case TypeNumber:
		return "Number"
	case TypeString:
		return "String"
	case TypeArray:
		return "Array"
	case TypeMap:
		return "Map"
	case TypeOpaque:
		return "Opaque"
	case TypeTask:
		return "Task"
	case TypeError:
		return "Error"
	default:
		return "Unknown"
	}
}

func ValueToString(v Value) string {
	switch v.Type {
	case TypeString:
		return v.String
	case TypeNumber:
		s := fmt.Sprintf("%g", v.Number)
		if strings.HasSuffix(s, ".0") {
			s = s[:len(s)-2]
		}
		return s
	case TypeArray:
		return "[Array]"
	case TypeMap:
		return "[Map]"
	case TypeOpaque:
		return fmt.Sprintf("[Opaque:%s]", v.Opaque.Label)
	case TypeTask:
		return "[Task]"
	case TypeError:
		return fmt.Sprintf("[Error:%d]", v.Error.Code)
	case TypeVoid:
		return "Void"
	default:
		return "Void"
	}
}

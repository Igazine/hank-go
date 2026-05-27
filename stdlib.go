package hal

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func halEquals(a, b Value) bool {
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
		if len(a.Array) != len(b.Array) {
			return false
		}
		for i := range a.Array {
			if !halEquals(a.Array[i], b.Array[i]) {
				return false
			}
		}
		return true
	case TypeObject:
		if len(a.Object) != len(b.Object) {
			return false
		}
		for k, v1 := range a.Object {
			v2, ok := b.Object[k]
			if !ok || !halEquals(v1, v2) {
				return false
			}
		}
		return true
	case TypeOpaque:
		return a.Opaque.Label == b.Opaque.Label && a.Opaque.Data == b.Opaque.Data
	default:
		return false
	}
}

func GetStdlibModules() map[string]map[string]NativeFunc {
	mods := make(map[string]map[string]NativeFunc)

	mods["log"] = map[string]NativeFunc{
		"print": func(args []Value, ctx ExecutionContext) Value {
			var strs []string
			for _, a := range args {
				strs = append(strs, ValueToString(a))
			}
			fmt.Println(strings.Join(strs, " "))
			return Value{Type: TypeVoid}
		},
		"error": func(args []Value, ctx ExecutionContext) Value {
			var strs []string
			for _, a := range args {
				strs = append(strs, ValueToString(a))
			}
			fmt.Fprintf(os.Stderr, "%s\n", strings.Join(strs, " "))
			return Value{Type: TypeVoid}
		},
		"warn": func(args []Value, ctx ExecutionContext) Value {
			var strs []string
			for _, a := range args {
				strs = append(strs, ValueToString(a))
			}
			fmt.Printf("WARNING: %s\n", strings.Join(strs, " "))
			return Value{Type: TypeVoid}
		},
	}

	mods["runtime"] = map[string]NativeFunc{
		"halt": func(args []Value, ctx ExecutionContext) Value {
			code := 0
			if len(args) > 0 && args[0].Type == TypeNumber {
				code = int(args[0].Number)
			}
			fmt.Printf("Exiting with code %d\n", code)
			return Value{Type: TypeVoid}
		},
		"elapsedTime": func(args []Value, ctx ExecutionContext) Value {
			return Value{Type: TypeNumber, Number: 0}
		},
	}

	mods["env"] = map[string]NativeFunc{
		"get": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 {
				return Value{Type: TypeVoid}
			}
			return Value{Type: TypeVoid}
		},
		"set": func(args []Value, ctx ExecutionContext) Value {
			return Value{Type: TypeVoid}
		},
		"keys": func(args []Value, ctx ExecutionContext) Value {
			return Value{Type: TypeArray, Array: []Value{}}
		},
	}

	mods["str"] = map[string]NativeFunc{
		"length": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 {
				return Value{Type: TypeVoid}
			}
			return Value{Type: TypeNumber, Number: float64(len(ValueToString(args[0])))}
		},
		"format": func(args []Value, ctx ExecutionContext) Value {
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
		},
		"concat": func(args []Value, ctx ExecutionContext) Value {
			var res strings.Builder
			for _, a := range args {
				res.WriteString(ValueToString(a))
			}
			return Value{Type: TypeString, String: res.String()}
		},
		"trim": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 {
				return Value{Type: TypeVoid}
			}
			return Value{Type: TypeString, String: strings.TrimSpace(ValueToString(args[0]))}
		},
	}

	mods["regex"] = map[string]NativeFunc{
		"parse": func(args []Value, ctx ExecutionContext) Value {
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
			return Value{
				Type: TypeOpaque,
				Opaque: &OpaqueValue{
					Label: "RegExp",
					Data:  re,
				},
			}
		},
		"match": func(args []Value, ctx ExecutionContext) Value {
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
		},
	}

	mods["math"] = map[string]NativeFunc{
		"add": func(args []Value, ctx ExecutionContext) Value {
			res := 0.0
			for _, a := range args {
				if a.Type == TypeNumber {
					res += a.Number
				}
			}
			return Value{Type: TypeNumber, Number: res}
		},
		"sub": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 {
				return Value{Type: TypeVoid}
			}
			return Value{Type: TypeNumber, Number: args[0].Number - args[1].Number}
		},
		"mul": func(args []Value, ctx ExecutionContext) Value {
			res := 1.0
			if len(args) == 0 {
				return Value{Type: TypeNumber, Number: 0}
			}
			for _, a := range args {
				if a.Type == TypeNumber {
					res *= a.Number
				}
			}
			return Value{Type: TypeNumber, Number: res}
		},
		"div": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 || args[1].Number == 0 {
				return Value{Type: TypeVoid}
			}
			return Value{Type: TypeNumber, Number: args[0].Number / args[1].Number}
		},
		"gt": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 {
				return Value{Type: TypeVoid}
			}
			if args[0].Number > args[1].Number {
				return Value{Type: TypeNumber, Number: 1}
			}
			return Value{Type: TypeVoid}
		},
		"lt": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 {
				return Value{Type: TypeVoid}
			}
			if args[0].Number < args[1].Number {
				return Value{Type: TypeNumber, Number: 1}
			}
			return Value{Type: TypeVoid}
		},
		"eq": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 {
				return Value{Type: TypeVoid}
			}
			if halEquals(args[0], args[1]) {
				return Value{Type: TypeNumber, Number: 1}
			}
			return Value{Type: TypeVoid}
		},
	}

	mods["logic"] = map[string]NativeFunc{
		"and": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 {
				return Value{Type: TypeVoid}
			}
			var last Value
			for _, a := range args {
				if a.Type == TypeVoid {
					return Value{Type: TypeVoid}
				}
				last = a
			}
			return last
		},
		"or": func(args []Value, ctx ExecutionContext) Value {
			for _, a := range args {
				if a.Type != TypeVoid {
					return a
				}
			}
			return Value{Type: TypeVoid}
		},
		"eq": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 {
				return Value{Type: TypeVoid}
			}
			if halEquals(args[0], args[1]) {
				return Value{Type: TypeNumber, Number: 1}
			}
			return Value{Type: TypeVoid}
		},
	}

	mods["arr"] = map[string]NativeFunc{
		"length": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 || args[0].Type != TypeArray {
				return Value{Type: TypeVoid}
			}
			return Value{Type: TypeNumber, Number: float64(len(args[0].Array))}
		},
		"get": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 || args[0].Type != TypeArray || args[1].Type != TypeNumber {
				return Value{Type: TypeVoid}
			}
			idx := int(args[1].Number)
			if idx < 0 || idx >= len(args[0].Array) {
				return Value{Type: TypeVoid}
			}
			return args[0].Array[idx]
		},
		"push": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 || args[0].Type != TypeArray {
				return Value{Type: TypeVoid}
			}
			args[0].Array = append(args[0].Array, args[1])
			return Value{Type: TypeVoid}
		},
		"pop": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 || args[0].Type != TypeArray || len(args[0].Array) == 0 {
				return Value{Type: TypeVoid}
			}
			idx := len(args[0].Array) - 1
			val := args[0].Array[idx]
			args[0].Array = args[0].Array[:idx]
			return val
		},
		"each": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 || args[0].Type != TypeArray || args[1].Type != TypeTask {
				return Value{Type: TypeVoid}
			}
			items := make([]Value, len(args[0].Array))
			copy(items, args[0].Array)
			for idx, item := range items {
				ctx.Call(args[1], []Value{item, {Type: TypeNumber, Number: float64(idx)}})
			}
			return Value{Type: TypeVoid}
		},
	}

	mods["obj"] = map[string]NativeFunc{
		"get": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 || args[0].Type != TypeObject {
				return Value{Type: TypeVoid}
			}
			key := ValueToString(args[1])
			if val, ok := args[0].Object[key]; ok {
				return val
			}
			return Value{Type: TypeVoid}
		},
		"keys": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 || args[0].Type != TypeObject {
				return Value{Type: TypeVoid}
			}
			var keys []Value
			for k := range args[0].Object {
				keys = append(keys, Value{Type: TypeString, String: k})
			}
			return Value{Type: TypeArray, Array: keys}
		},
		"values": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 || args[0].Type != TypeObject {
				return Value{Type: TypeVoid}
			}
			var vals []Value
			for _, v := range args[0].Object {
				vals = append(vals, v)
			}
			return Value{Type: TypeArray, Array: vals}
		},
	}

	mods["json"] = map[string]NativeFunc{
		"parse": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 {
				return Value{Type: TypeVoid}
			}
			s := ValueToString(args[0])
			var data interface{}
			if err := json.Unmarshal([]byte(s), &data); err != nil {
				return Value{Type: TypeVoid}
			}
			return mapAnyToHal(data)
		},
		"stringify": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 {
				return Value{Type: TypeVoid}
			}
			// Check for Opaque values
			any, ok := mapHalToAny(args[0])
			if !ok {
				return Value{Type: TypeVoid}
			}
			b, err := json.Marshal(any)
			if err != nil {
				return Value{Type: TypeVoid}
			}
			return Value{Type: TypeString, String: string(b)}
		},
	}

	return mods
}

func mapAnyToHal(v interface{}) Value {
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
		var arr []Value
		for _, item := range val {
			arr = append(arr, mapAnyToHal(item))
		}
		return Value{Type: TypeArray, Array: arr}
	case map[string]interface{}:
		obj := make(map[string]Value)
		for k, v := range val {
			obj[k] = mapAnyToHal(v)
		}
		return Value{Type: TypeObject, Object: obj}
	default:
		panic(fmt.Sprintf("HAL Boundary Error: Complex host object [%T] must implement SerializeHAL() to bridge into HAL.", v))
	}
}

func mapHalToAny(v Value) (interface{}, bool) {
	switch v.Type {
	case TypeString:
		return v.String, true
	case TypeNumber:
		return v.Number, true
	case TypeArray:
		var arr []interface{}
		for _, item := range v.Array {
			any, ok := mapHalToAny(item)
			if !ok { return nil, false }
			arr = append(arr, any)
		}
		return arr, true
	case TypeObject:
		obj := make(map[string]interface{})
		for k, val := range v.Object {
			any, ok := mapHalToAny(val)
			if !ok { return nil, false }
			obj[k] = any
		}
		return obj, true
	case TypeOpaque:
		return nil, false // Non-serializable
	default:
		return nil, true
	}
}

func ValueToString(v Value) string {
	switch v.Type {
	case TypeString:
		return v.String
	case TypeNumber:
		return fmt.Sprintf("%g", v.Number)
	case TypeArray:
		return "[Array]"
	case TypeObject:
		return "{Object}"
	case TypeOpaque:
		return fmt.Sprintf("[Opaque:%s]", v.Opaque.Label)
	case TypeTask:
		return "[Task]"
	case TypeVoid:
		return "Void"
	default:
		return "Void"
	}
}

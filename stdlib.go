package hal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

func (r *Runner) RegisterStd() {
	r.RegisterModule("log", map[string]NativeFunc{
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
			for _, a := range args { strs = append(strs, ValueToString(a)) }
			fmt.Fprintf(os.Stderr, "%s\n", strings.Join(strs, " "))
			return Value{Type: TypeVoid}
		},
		"warn": func(args []Value, ctx ExecutionContext) Value {
			var strs []string
			for _, a := range args { strs = append(strs, ValueToString(a)) }
			fmt.Printf("WARNING: %s\n", strings.Join(strs, " "))
			return Value{Type: TypeVoid}
		},
	})

	r.RegisterModule("runtime", map[string]NativeFunc{
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
	})

	r.RegisterModule("env", map[string]NativeFunc{
		"get": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			return Value{Type: TypeVoid}
		},
		"set": func(args []Value, ctx ExecutionContext) Value {
			return Value{Type: TypeVoid}
		},
		"keys": func(args []Value, ctx ExecutionContext) Value {
			return Value{Type: TypeArray, Array: []Value{}}
		},
	})

	r.RegisterModule("str", map[string]NativeFunc{
		"length": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			return Value{Type: TypeNumber, Number: float64(len(ValueToString(args[0])))}
		},
		"format": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
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
			for _, a := range args { res.WriteString(ValueToString(a)) }
			return Value{Type: TypeString, String: res.String()}
		},
		"trim": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			return Value{Type: TypeString, String: strings.TrimSpace(ValueToString(args[0]))}
		},
	})

	r.RegisterModule("regex", map[string]NativeFunc{
		"parse": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			pattern := ValueToString(args[0])
			flags := ""
			if len(args) > 1 { flags = ValueToString(args[1]) }
			goFlags := ""
			if strings.Contains(flags, "i") { goFlags += "i" }
			if strings.Contains(flags, "m") { goFlags += "m" }
			finalPattern := pattern
			if goFlags != "" { finalPattern = "(?" + goFlags + ")" + pattern }
			re, err := regexp.Compile(finalPattern)
			if err != nil { return Value{Type: TypeVoid} }
			return Value{
				Type: TypeRegex,
				Regex: &RegexValue{Pattern: pattern, Flags: flags, Regexp: re},
			}
		},
		"match": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 { return Value{Type: TypeVoid} }
			s := ValueToString(args[0])
			patternVal := args[1]
			if patternVal.Type == TypeRegex {
				if patternVal.Regex.Regexp != nil && patternVal.Regex.Regexp.MatchString(s) {
					return Value{Type: TypeNumber, Number: 1}
				}
			} else {
				if strings.Contains(s, ValueToString(patternVal)) {
					return Value{Type: TypeNumber, Number: 1}
				}
			}
			return Value{Type: TypeVoid}
		},
	})

	r.RegisterModule("math", map[string]NativeFunc{
		"add": func(args []Value, ctx ExecutionContext) Value {
			res := 0.0
			for _, a := range args {
				if a.Type == TypeNumber { res += a.Number }
			}
			return Value{Type: TypeNumber, Number: res}
		},
		"sub": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 { return Value{Type: TypeVoid} }
			return Value{Type: TypeNumber, Number: args[0].Number - args[1].Number}
		},
		"mul": func(args []Value, ctx ExecutionContext) Value {
			res := 1.0
			if len(args) == 0 { return Value{Type: TypeNumber, Number: 0} }
			for _, a := range args {
				if a.Type == TypeNumber { res *= a.Number }
			}
			return Value{Type: TypeNumber, Number: res}
		},
		"div": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 || args[1].Number == 0 { return Value{Type: TypeVoid} }
			return Value{Type: TypeNumber, Number: args[0].Number / args[1].Number}
		},
		"gt": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 { return Value{Type: TypeVoid} }
			if args[0].Number > args[1].Number { return Value{Type: TypeNumber, Number: 1} }
			return Value{Type: TypeVoid}
		},
		"lt": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 { return Value{Type: TypeVoid} }
			if args[0].Number < args[1].Number { return Value{Type: TypeNumber, Number: 1} }
			return Value{Type: TypeVoid}
		},
		"eq": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 { return Value{Type: TypeVoid} }
			if ValueToString(args[0]) == ValueToString(args[1]) { return Value{Type: TypeNumber, Number: 1} }
			return Value{Type: TypeVoid}
		},
	})

	r.RegisterModule("logic", map[string]NativeFunc{
		"and": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			var last Value
			for _, a := range args {
				if a.Type == TypeVoid { return Value{Type: TypeVoid} }
				last = a
			}
			return last
		},
		"or": func(args []Value, ctx ExecutionContext) Value {
			for _, a := range args {
				if a.Type != TypeVoid { return a }
			}
			return Value{Type: TypeVoid}
		},
	})
	
	r.RegisterModule("arr", map[string]NativeFunc{
		"length": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 || args[0].Type != TypeArray { return Value{Type: TypeVoid} }
			return Value{Type: TypeNumber, Number: float64(len(args[0].Array))}
		},
		"get": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 || args[0].Type != TypeArray || args[1].Type != TypeNumber { return Value{Type: TypeVoid} }
			idx := int(args[1].Number)
			if idx < 0 || idx >= len(args[0].Array) { return Value{Type: TypeVoid} }
			return args[0].Array[idx]
		},
		"push": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 || args[0].Type != TypeArray { return Value{Type: TypeVoid} }
			args[0].Array = append(args[0].Array, args[1])
			return Value{Type: TypeVoid}
		},
		"pop": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 || args[0].Type != TypeArray || len(args[0].Array) == 0 { return Value{Type: TypeVoid} }
			idx := len(args[0].Array) - 1
			val := args[0].Array[idx]
			args[0].Array = args[0].Array[:idx]
			return val
		},
		"each": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 || args[0].Type != TypeArray || args[1].Type != TypeTask { return Value{Type: TypeVoid} }
			items := make([]Value, len(args[0].Array))
			copy(items, args[0].Array)
			for idx, item := range items {
				ctx.Call(args[1], []Value{item, {Type: TypeNumber, Number: float64(idx)}})
			}
			return Value{Type: TypeVoid}
		},
	})

	r.RegisterModule("obj", map[string]NativeFunc{
		"get": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 || args[0].Type != TypeObject { return Value{Type: TypeVoid} }
			key := ValueToString(args[1])
			if val, ok := args[0].Object[key]; ok { return val }
			return Value{Type: TypeVoid}
		},
		"keys": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 || args[0].Type != TypeObject { return Value{Type: TypeVoid} }
			var keys []Value
			for k := range args[0].Object { keys = append(keys, Value{Type: TypeString, String: k}) }
			return Value{Type: TypeArray, Array: keys}
		},
		"values": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 || args[0].Type != TypeObject { return Value{Type: TypeVoid} }
			var vals []Value
			for _, v := range args[0].Object { vals = append(vals, v) }
			return Value{Type: TypeArray, Array: vals}
		},
	})

	r.RegisterModule("json", map[string]NativeFunc{
		"parse": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			s := ValueToString(args[0])
			var data interface{}
			if err := json.Unmarshal([]byte(s), &data); err != nil { return Value{Type: TypeVoid} }
			return mapAnyToHal(data)
		},
		"stringify": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			any := mapHalToAny(args[0])
			b, err := json.Marshal(any)
			if err != nil { return Value{Type: TypeVoid} }
			return Value{Type: TypeString, String: string(b)}
		},
	})

	r.RegisterModule("os", map[string]NativeFunc{
		"type": func(args []Value, ctx ExecutionContext) Value {
			return Value{Type: TypeString, String: runtime.GOOS}
		},
		"name": func(args []Value, ctx ExecutionContext) Value {
			return Value{Type: TypeString, String: runtime.GOOS}
		},
		"arch": func(args []Value, ctx ExecutionContext) Value {
			return Value{Type: TypeString, String: runtime.GOARCH}
		},
		"memory": func(args []Value, ctx ExecutionContext) Value {
			obj := make(map[string]Value)
			obj["total"] = Value{Type: TypeNumber, Number: 1024}
			obj["free"] = Value{Type: TypeNumber, Number: 512}
			obj["used"] = Value{Type: TypeNumber, Number: 512}
			return Value{Type: TypeObject, Object: obj}
		},
		"cpu": func(args []Value, ctx ExecutionContext) Value {
			return Value{Type: TypeNumber, Number: 0}
		},
	})

	r.RegisterModule("host", map[string]NativeFunc{
		"cwd": func(args []Value, ctx ExecutionContext) Value {
			cwd, _ := os.Getwd()
			return Value{Type: TypeString, String: cwd}
		},
		"pid": func(args []Value, ctx ExecutionContext) Value {
			return Value{Type: TypeNumber, Number: float64(os.Getpid())}
		},
		"isRoot": func(args []Value, ctx ExecutionContext) Value {
			if os.Getuid() == 0 { return Value{Type: TypeNumber, Number: 1} }
			return Value{Type: TypeVoid}
		},
		"signal": func(args []Value, ctx ExecutionContext) Value {
			if len(args) > 0 { fmt.Printf("[SIGNAL] %v\n", ValueToString(args[0])) }
			return Value{Type: TypeVoid}
		},
	})

	r.RegisterModule("fs", map[string]NativeFunc{
		"exists": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			if _, err := os.Stat(ValueToString(args[0])); err == nil { return Value{Type: TypeNumber, Number: 1} }
			return Value{Type: TypeVoid}
		},
		"read": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			b, err := os.ReadFile(ValueToString(args[0]))
			if err != nil { return Value{Type: TypeVoid} }
			return Value{Type: TypeString, String: string(b)}
		},
		"write": func(args []Value, ctx ExecutionContext) Value {
			if len(args) < 2 { return Value{Type: TypeVoid} }
			if err := os.WriteFile(ValueToString(args[0]), []byte(ValueToString(args[1])), 0644); err == nil { return Value{Type: TypeNumber, Number: 1} }
			return Value{Type: TypeVoid}
		},
		"deleteFile": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			if err := os.Remove(ValueToString(args[0])); err == nil { return Value{Type: TypeNumber, Number: 1} }
			return Value{Type: TypeVoid}
		},
		"stat": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			info, err := os.Stat(ValueToString(args[0]))
			if err != nil { return Value{Type: TypeVoid} }
			obj := make(map[string]Value)
			obj["size"] = Value{Type: TypeNumber, Number: float64(info.Size())}
			obj["mtime"] = Value{Type: TypeNumber, Number: float64(info.ModTime().UnixMilli())}
			obj["isDir"] = Value{Type: TypeVoid}
			if info.IsDir() { obj["isDir"] = Value{Type: TypeNumber, Number: 1} }
			return Value{Type: TypeObject, Object: obj}
		},
	})

	r.RegisterModule("proc", map[string]NativeFunc{
		"run": func(args []Value, ctx ExecutionContext) Value {
			if len(args) == 0 { return Value{Type: TypeVoid} }
			cmdName := ValueToString(args[0])
			var cmdArgs []string
			if len(args) > 1 && args[1].Type == TypeArray {
				for _, a := range args[1].Array { cmdArgs = append(cmdArgs, ValueToString(a)) }
			}
			cmd := exec.Command(cmdName, cmdArgs...)
			out, err := cmd.CombinedOutput()
			code := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok { code = exitErr.ExitCode() } else { code = 1 }
			}
			res := make(map[string]Value)
			res["code"] = Value{Type: TypeNumber, Number: float64(code)}
			res["stdout"] = Value{Type: TypeString, String: string(out)}
			res["stderr"] = Value{Type: TypeString, String: ""}
			return Value{Type: TypeObject, Object: res}
		},
	})
}

func mapAnyToHal(v interface{}) Value {
	if v == nil { return Value{Type: TypeVoid} }
	if s, ok := v.(IHALSerializable); ok {
		return Value{Type: TypeString, String: s.SerializeHAL()}
	}
	switch val := v.(type) {
	case string: return Value{Type: TypeString, String: val}
	case float64: return Value{Type: TypeNumber, Number: val}
	case int: return Value{Type: TypeNumber, Number: float64(val)}
	case bool: 
		if val { return Value{Type: TypeNumber, Number: 1} }
		return Value{Type: TypeVoid}
	case []interface{}:
		var arr []Value
		for _, item := range val { arr = append(arr, mapAnyToHal(item)) }
		return Value{Type: TypeArray, Array: arr}
	case map[string]interface{}:
		obj := make(map[string]Value)
		for k, v := range val { obj[k] = mapAnyToHal(v) }
		return Value{Type: TypeObject, Object: obj}
	default: panic(fmt.Sprintf("HAL Boundary Error: Complex host object [%T] must implement SerializeHAL() to bridge into HAL.", v))
	}
}

func mapHalToAny(v Value) interface{} {
	switch v.Type {
	case TypeString: return v.String
	case TypeNumber: return v.Number
	case TypeArray:
		var arr []interface{}
		for _, item := range v.Array { arr = append(arr, mapHalToAny(item)) }
		return arr
	case TypeObject:
		obj := make(map[string]interface{})
		for k, val := range v.Object { obj[k] = mapHalToAny(val) }
		return obj
	default: return nil
	}
}

func ValueToString(v Value) string {
	switch v.Type {
	case TypeString: return v.String
	case TypeNumber: return fmt.Sprintf("%g", v.Number)
	case TypeArray: return "[Array]"
	case TypeObject: return "{Object}"
	case TypeRegex: return "[Regex]"
	case TypeTask: return "[Task]"
	case TypeVoid: return "null"
	default: return "null"
	}
}

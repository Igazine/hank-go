package ext

import (
        "os"
        "os/exec"
        "runtime"
        "github.com/Igazine/hank-go"
)

type SysExtension struct{}

func (e *SysExtension) Name() string {
	return "SysExtension"
}

func (e *SysExtension) GetModules() map[string]map[string]hank.NativeFunc {
	mods := make(map[string]map[string]hank.NativeFunc)

	mods["host"] = map[string]hank.NativeFunc{
		"cwd": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			cwd, _ := os.Getwd()
			return hank.Value{Type: hank.TypeString, String: cwd}
		},
		"isRoot": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if os.Getuid() == 0 {
				return hank.Value{Type: hank.TypeNumber, Number: 1}
			}
			return hank.Value{Type: hank.TypeVoid}
		},
		"pid": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			return hank.Value{Type: hank.TypeNumber, Number: float64(os.Getpid())}
		},
	}

	mods["os"] = map[string]hank.NativeFunc{
		"type": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			return hank.Value{Type: hank.TypeString, String: runtime.GOOS}
		},
		"name": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			return hank.Value{Type: hank.TypeString, String: runtime.GOOS}
		},
		"arch": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			return hank.Value{Type: hank.TypeString, String: runtime.GOARCH}
		},
		"memory": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			fields := make(map[string]hank.Value)
			fields["total"] = hank.Value{Type: hank.TypeNumber, Number: float64(m.Sys)}
			fields["free"] = hank.Value{Type: hank.TypeNumber, Number: float64(m.HeapIdle)}
			fields["used"] = hank.Value{Type: hank.TypeNumber, Number: float64(m.Alloc)}
			return hank.Value{Type: hank.TypeMap, Map: fields}
		},
		"cpu": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			return hank.Value{Type: hank.TypeNumber, Number: 0}
		},
	}

	mods["fs"] = map[string]hank.NativeFunc{
		"exists": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeString {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "String"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "fs.exists"}}}}
			}
			path := args[0].String
			if _, err := os.Stat(path); err == nil {
				return hank.Value{Type: hank.TypeNumber, Number: 1}
			}
			return hank.Value{Type: hank.TypeVoid}
		},
		"read": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeString {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "String"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "fs.read"}}}}
			}
			path := args[0].String
			content, err := os.ReadFile(path)
			if err != nil {
				return hank.Value{Type: hank.TypeVoid}
			}
			return hank.Value{Type: hank.TypeString, String: string(content)}
		},
		"write": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) < 2 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeString {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "String"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "fs.write"}}}}
			}
			if args[1].Type != hank.TypeString {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "String"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "fs.write"}}}}
			}
			path := args[0].String
			content := args[1].String
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return hank.Value{Type: hank.TypeVoid}
			}
			return hank.Value{Type: hank.TypeNumber, Number: 1}
		},
		"deleteFile": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeString {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "String"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "fs.deleteFile"}}}}
			}
			path := args[0].String
			if err := os.Remove(path); err != nil {
				return hank.Value{Type: hank.TypeVoid}
			}
			return hank.Value{Type: hank.TypeNumber, Number: 1}
		},
		"stat": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeString {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "String"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "fs.stat"}}}}
			}
			path := args[0].String
			info, err := os.Stat(path)
			if err != nil {
				return hank.Value{Type: hank.TypeVoid}
			}
			fields := make(map[string]hank.Value)
			fields["size"] = hank.Value{Type: hank.TypeNumber, Number: float64(info.Size())}
			fields["mtime"] = hank.Value{Type: hank.TypeNumber, Number: float64(info.ModTime().UnixMilli())}
			if info.IsDir() {
				fields["isDir"] = hank.Value{Type: hank.TypeNumber, Number: 1}
			} else {
				fields["isDir"] = hank.Value{Type: hank.TypeVoid}
			}
			return hank.Value{Type: hank.TypeMap, Map: fields}
		},
	}

	mods["proc"] = map[string]hank.NativeFunc{
		"run": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 { return hank.Value{Type: hank.TypeVoid} }
			if args[0].Type != hank.TypeString {
				return hank.Value{Type: hank.TypeError, Error: &hank.ErrorValue{Code: hank.TypeMismatch, Args: []hank.Value{{Type: hank.TypeString, String: "String"}, {Type: hank.TypeString, String: "Any"}, {Type: hank.TypeString, String: "proc.run"}}}}
			}
			cmdName := args[0].String
			var cmdArgs []string
			if len(args) > 1 && args[1].Type == hank.TypeArray {
				for _, a := range *args[1].Array {
					cmdArgs = append(cmdArgs, hank.ValueToString(a))
				}
			}
			cmd := exec.Command(cmdName, cmdArgs...)
			out, err := cmd.CombinedOutput()
			code := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					code = exitError.ExitCode()
				} else {
					code = 1
				}
			}
			fields := make(map[string]hank.Value)
			fields["code"] = hank.Value{Type: hank.TypeNumber, Number: float64(code)}
			fields["stdout"] = hank.Value{Type: hank.TypeString, String: string(out)}
			fields["stderr"] = hank.Value{Type: hank.TypeString, String: ""}
			return hank.Value{Type: hank.TypeMap, Map: fields}
		},
	}

	return mods
}

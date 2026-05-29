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
			obj := make(map[string]hank.Value)
			obj["total"] = hank.Value{Type: hank.TypeNumber, Number: float64(m.Sys)}
			obj["free"] = hank.Value{Type: hank.TypeNumber, Number: float64(m.HeapIdle)}
			obj["used"] = hank.Value{Type: hank.TypeNumber, Number: float64(m.Alloc)}
			return hank.Value{Type: hank.TypeObject, Object: obj}
		},
		"cpu": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			return hank.Value{Type: hank.TypeNumber, Number: 0}
		},
	}

	mods["fs"] = map[string]hank.NativeFunc{
		"exists": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			path := hank.ValueToString(args[0])
			if _, err := os.Stat(path); err == nil {
				return hank.Value{Type: hank.TypeNumber, Number: 1}
			}
			return hank.Value{Type: hank.TypeVoid}
		},
		"read": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			path := hank.ValueToString(args[0])
			content, err := os.ReadFile(path)
			if err != nil {
				return hank.Value{Type: hank.TypeVoid}
			}
			return hank.Value{Type: hank.TypeString, String: string(content)}
		},
		"write": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			path := hank.ValueToString(args[0])
			content := hank.ValueToString(args[1])
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return hank.Value{Type: hank.TypeVoid}
			}
			return hank.Value{Type: hank.TypeNumber, Number: 1}
		},
		"deleteFile": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			path := hank.ValueToString(args[0])
			if err := os.Remove(path); err != nil {
				return hank.Value{Type: hank.TypeVoid}
			}
			return hank.Value{Type: hank.TypeNumber, Number: 1}
		},
		"stat": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			path := hank.ValueToString(args[0])
			info, err := os.Stat(path)
			if err != nil {
				return hank.Value{Type: hank.TypeVoid}
			}
			obj := make(map[string]hank.Value)
			obj["size"] = hank.Value{Type: hank.TypeNumber, Number: float64(info.Size())}
			obj["mtime"] = hank.Value{Type: hank.TypeNumber, Number: float64(info.ModTime().UnixMilli())}
			if info.IsDir() {
				obj["isDir"] = hank.Value{Type: hank.TypeNumber, Number: 1}
			} else {
				obj["isDir"] = hank.Value{Type: hank.TypeVoid}
			}
			return hank.Value{Type: hank.TypeObject, Object: obj}
		},
	}

	mods["proc"] = map[string]hank.NativeFunc{
		"run": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			cmdName := hank.ValueToString(args[0])
			var cmdArgs []string
			if len(args) > 1 && args[1].Type == hank.TypeArray {
				for _, a := range args[1].Array {
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
			obj := make(map[string]hank.Value)
			obj["code"] = hank.Value{Type: hank.TypeNumber, Number: float64(code)}
			obj["stdout"] = hank.Value{Type: hank.TypeString, String: string(out)}
			obj["stderr"] = hank.Value{Type: hank.TypeString, String: ""}
			return hank.Value{Type: hank.TypeObject, Object: obj}
		},
	}

	return mods
}

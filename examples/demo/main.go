package main

import (
	"fmt"
	"github.com/Igazine/hank-go"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	cwd, _ := os.Getwd()
	// Submodule is at vendor/hank relative to the hank-go project root
	root := filepath.Join(cwd, "vendor/hank")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		root = filepath.Join(cwd, "../../vendor/hank")
	}
	
	if len(os.Args) < 2 {
		runConformance(root)
		return
	}

	r := createRunner()

	// Map CLI strings to Hank Host Arguments
	var hankArgs []hank.Value
	if len(os.Args) > 2 {
		for _, arg := range os.Args[2:] {
			hankArgs = append(hankArgs, hank.Value{Type: hank.TypeString, String: arg})
		}
	}

	absPath, _ := filepath.Abs(os.Args[1])
	res := NewFileResource(absPath)

	result, err := r.Run(res, hankArgs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if result.Type == hank.TypeNumber {
		os.Exit(int(result.Number))
	}
	os.Exit(0)
}

func createRunner() *hank.Runner {
	// 1. Instantiate the core Runner
	r := hank.NewRunner()

	// 2. Register the Standard Library manually (Optional, per-task registration)
	std := hank.GetStdlibModules()
	for modName, tasks := range std {
		r.RegisterModule(modName, tasks)
	}

	// 3. Register Custom SYSLIB
	registerSyslib(r)

	return r
}

func registerSyslib(r *hank.Runner) {
	r.RegisterModule("os", map[string]hank.NativeFunc{
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
			obj := make(map[string]hank.Value)
			obj["total"] = hank.Value{Type: hank.TypeNumber, Number: 1024}
			obj["free"] = hank.Value{Type: hank.TypeNumber, Number: 512}
			obj["used"] = hank.Value{Type: hank.TypeNumber, Number: 512}
			return hank.Value{Type: hank.TypeObject, Object: obj}
		},
		"cpu": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			return hank.Value{Type: hank.TypeNumber, Number: 0}
		},
	})

	r.RegisterModule("host", map[string]hank.NativeFunc{
		"cwd": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			cwd, _ := os.Getwd()
			return hank.Value{Type: hank.TypeString, String: cwd}
		},
		"pid": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			return hank.Value{Type: hank.TypeNumber, Number: float64(os.Getpid())}
		},
		"isRoot": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if os.Getuid() == 0 {
				return hank.Value{Type: hank.TypeNumber, Number: 1}
			}
			return hank.Value{Type: hank.TypeVoid}
		},
	})

	r.RegisterModule("fs", map[string]hank.NativeFunc{
		"exists": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 {
				return hank.Value{Type: hank.TypeVoid}
			}
			if _, err := os.Stat(hank.ValueToString(args[0])); err == nil {
				return hank.Value{Type: hank.TypeNumber, Number: 1}
			}
			return hank.Value{Type: hank.TypeVoid}
		},
		"read": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 {
				return hank.Value{Type: hank.TypeVoid}
			}
			b, err := os.ReadFile(hank.ValueToString(args[0]))
			if err != nil {
				return hank.Value{Type: hank.TypeVoid}
			}
			return hank.Value{Type: hank.TypeString, String: string(b)}
		},
		"write": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) < 2 {
				return hank.Value{Type: hank.TypeVoid}
			}
			if err := os.WriteFile(hank.ValueToString(args[0]), []byte(hank.ValueToString(args[1])), 0644); err == nil {
				return hank.Value{Type: hank.TypeNumber, Number: 1}
			}
			return hank.Value{Type: hank.TypeVoid}
		},
		"deleteFile": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 {
				return hank.Value{Type: hank.TypeVoid}
			}
			if err := os.Remove(hank.ValueToString(args[0])); err == nil {
				return hank.Value{Type: hank.TypeNumber, Number: 1}
			}
			return hank.Value{Type: hank.TypeVoid}
		},
		"stat": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 {
				return hank.Value{Type: hank.TypeVoid}
			}
			info, err := os.Stat(hank.ValueToString(args[0]))
			if err != nil {
				return hank.Value{Type: hank.TypeVoid}
			}
			obj := make(map[string]hank.Value)
			obj["size"] = hank.Value{Type: hank.TypeNumber, Number: float64(info.Size())}
			obj["mtime"] = hank.Value{Type: hank.TypeNumber, Number: float64(info.ModTime().UnixMilli())}
			obj["isDir"] = hank.Value{Type: hank.TypeVoid}
			if info.IsDir() {
				obj["isDir"] = hank.Value{Type: hank.TypeNumber, Number: 1}
			}
			return hank.Value{Type: hank.TypeObject, Object: obj}
		},
	})

	r.RegisterModule("proc", map[string]hank.NativeFunc{
		"run": func(args []hank.Value, ctx hank.ExecutionContext) hank.Value {
			if len(args) == 0 {
				return hank.Value{Type: hank.TypeVoid}
			}
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
				if exitErr, ok := err.(*exec.ExitError); ok {
					code = exitErr.ExitCode()
				} else {
					code = 1
				}
			}
			res := make(map[string]hank.Value)
			res["code"] = hank.Value{Type: hank.TypeNumber, Number: float64(code)}
			res["stdout"] = hank.Value{Type: hank.TypeString, String: string(out)}
			res["stderr"] = hank.Value{Type: hank.TypeString, String: ""}
			return hank.Value{Type: hank.TypeObject, Object: res}
		},
	})
}

func runConformance(root string) {
	conformanceTests := []string{
		"test/conformance/01_literals.hank",
		"test/conformance/02_gates.hank",
		"test/conformance/03_scoping.hank",
		"test/conformance/04_hoisting.hank",
		"test/conformance/05_params.hank",
		"test/conformance/06_macros.hank",
		"test/conformance/07_returns.hank",
		"test/conformance/08_host_args.hank",
		"test/conformance/09_deep_nesting.hank",
		"test/conformance/10_edge_cases.hank",
		"test/conformance/11_regex_parse.hank",
		"test/conformance/12_data_advanced.hank",
		"test/conformance/13_logic_module.hank",
		"test/conformance/14_syslib_hank.hank",
		"test/conformance/15_logic_eq.hank",
		"test/conformance/16_chained_assign.hank",
	}

	for _, t := range conformanceTests {
		fmt.Printf("--- Running: %s ---\n", t)
		r := createRunner()
		path, _ := filepath.Abs(filepath.Join(root, t))
		res := NewFileResource(path)
		args := []hank.Value{}
		if strings.HasSuffix(t, "08_host_args.hank") {
			args = append(args, hank.Value{Type: hank.TypeString, String: "Tamas"})
		}
		_, err := r.Run(res, args)
		if err != nil {
			fmt.Printf("Test Failed: %v\n", err)
		}
		fmt.Printf("--------------------\n\n")
	}
}

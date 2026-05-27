package main

import (
	"fmt"
	"github.com/Igazine/hal-go"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	cwd, _ := os.Getwd()
	// Submodule is at vendor/hal relative to the runner project root
	root := filepath.Join(cwd, "../../vendor/hal")
	
	if len(os.Args) < 2 {
		runConformance(root)
		return
	}

	r := hal.NewRunner()
	hal.RegisterStdlib(r)
	registerSyslib(r)

	// Map CLI strings to HAL Host Arguments
	var halArgs []hal.Value
	if len(os.Args) > 2 {
		for _, arg := range os.Args[2:] {
			halArgs = append(halArgs, hal.Value{Type: hal.TypeString, String: arg})
		}
	}

	result, err := r.Run(os.Args[1], halArgs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if result.Type == hal.TypeNumber {
		os.Exit(int(result.Number))
	}
	os.Exit(0)
}

func registerSyslib(r *hal.Runner) {
	r.RegisterModule("os", map[string]hal.NativeFunc{
		"type": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			return hal.Value{Type: hal.TypeString, String: runtime.GOOS}
		},
		"name": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			return hal.Value{Type: hal.TypeString, String: runtime.GOOS}
		},
		"arch": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			return hal.Value{Type: hal.TypeString, String: runtime.GOARCH}
		},
		"memory": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			obj := make(map[string]hal.Value)
			obj["total"] = hal.Value{Type: hal.TypeNumber, Number: 1024}
			obj["free"] = hal.Value{Type: hal.TypeNumber, Number: 512}
			obj["used"] = hal.Value{Type: hal.TypeNumber, Number: 512}
			return hal.Value{Type: hal.TypeObject, Object: obj}
		},
		"cpu": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			return hal.Value{Type: hal.TypeNumber, Number: 0}
		},
	})

	r.RegisterModule("host", map[string]hal.NativeFunc{
		"cwd": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			cwd, _ := os.Getwd()
			return hal.Value{Type: hal.TypeString, String: cwd}
		},
		"pid": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			return hal.Value{Type: hal.TypeNumber, Number: float64(os.Getpid())}
		},
		"isRoot": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			if os.Getuid() == 0 {
				return hal.Value{Type: hal.TypeNumber, Number: 1}
			}
			return hal.Value{Type: hal.TypeVoid}
		},
		"signal": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			if len(args) > 0 {
				fmt.Printf("[SIGNAL] %v\n", hal.ValueToString(args[0]))
			}
			return hal.Value{Type: hal.TypeVoid}
		},
	})

	r.RegisterModule("fs", map[string]hal.NativeFunc{
		"exists": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			if len(args) == 0 {
				return hal.Value{Type: hal.TypeVoid}
			}
			if _, err := os.Stat(hal.ValueToString(args[0])); err == nil {
				return hal.Value{Type: hal.TypeNumber, Number: 1}
			}
			return hal.Value{Type: hal.TypeVoid}
		},
		"read": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			if len(args) == 0 {
				return hal.Value{Type: hal.TypeVoid}
			}
			b, err := os.ReadFile(hal.ValueToString(args[0]))
			if err != nil {
				return hal.Value{Type: hal.TypeVoid}
			}
			return hal.Value{Type: hal.TypeString, String: string(b)}
		},
		"write": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			if len(args) < 2 {
				return hal.Value{Type: hal.TypeVoid}
			}
			if err := os.WriteFile(hal.ValueToString(args[0]), []byte(hal.ValueToString(args[1])), 0644); err == nil {
				return hal.Value{Type: hal.TypeNumber, Number: 1}
			}
			return hal.Value{Type: hal.TypeVoid}
		},
		"deleteFile": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			if len(args) == 0 {
				return hal.Value{Type: hal.TypeVoid}
			}
			if err := os.Remove(hal.ValueToString(args[0])); err == nil {
				return hal.Value{Type: hal.TypeNumber, Number: 1}
			}
			return hal.Value{Type: hal.TypeVoid}
		},
		"stat": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			if len(args) == 0 {
				return hal.Value{Type: hal.TypeVoid}
			}
			info, err := os.Stat(hal.ValueToString(args[0]))
			if err != nil {
				return hal.Value{Type: hal.TypeVoid}
			}
			obj := make(map[string]hal.Value)
			obj["size"] = hal.Value{Type: hal.TypeNumber, Number: float64(info.Size())}
			obj["mtime"] = hal.Value{Type: hal.TypeNumber, Number: float64(info.ModTime().UnixMilli())}
			obj["isDir"] = hal.Value{Type: hal.TypeVoid}
			if info.IsDir() {
				obj["isDir"] = hal.Value{Type: hal.TypeNumber, Number: 1}
			}
			return hal.Value{Type: hal.TypeObject, Object: obj}
		},
	})

	r.RegisterModule("proc", map[string]hal.NativeFunc{
		"run": func(args []hal.Value, ctx hal.ExecutionContext) hal.Value {
			if len(args) == 0 {
				return hal.Value{Type: hal.TypeVoid}
			}
			cmdName := hal.ValueToString(args[0])
			var cmdArgs []string
			if len(args) > 1 && args[1].Type == hal.TypeArray {
				for _, a := range args[1].Array {
					cmdArgs = append(cmdArgs, hal.ValueToString(a))
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
			res := make(map[string]hal.Value)
			res["code"] = hal.Value{Type: hal.TypeNumber, Number: float64(code)}
			res["stdout"] = hal.Value{Type: hal.TypeString, String: string(out)}
			res["stderr"] = hal.Value{Type: hal.TypeString, String: ""}
			return hal.Value{Type: hal.TypeObject, Object: res}
		},
	})
}

func runConformance(root string) {
	conformanceTests := []string{
		"test/conformance/01_literals.hal",
		"test/conformance/02_gates.hal",
		"test/conformance/03_scoping.hal",
		"test/conformance/04_hoisting.hal",
		"test/conformance/05_params.hal",
		"test/conformance/06_macros.hal",
		"test/conformance/07_returns.hal",
		"test/conformance/08_host_args.hal",
		"test/conformance/09_deep_nesting.hal",
		"test/conformance/10_edge_cases.hal",
		"test/conformance/11_regex_parse.hal",
		"test/conformance/12_data_advanced.hal",
		"test/conformance/13_logic_module.hal",
		"test/conformance/14_syslib_hank.hal",
	}

	for _, t := range conformanceTests {
		fmt.Printf("--- Running: %s ---\n", t)
		r := hal.NewRunner()
		hal.RegisterStdlib(r)
		registerSyslib(r)
		path := filepath.Join(root, t)
		args := []hal.Value{}
		if strings.HasSuffix(t, "08_host_args.hal") {
			args = append(args, hal.Value{Type: hal.TypeString, String: "Tamas"})
		}
		_, err := r.Run(path, args)
		if err != nil {
			fmt.Printf("Test Failed: %v\n", err)
		}
		fmt.Printf("--------------------\n\n")
	}
}

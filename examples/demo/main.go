package main

import (
	"fmt"
	"github.com/Igazine/hank-go"
	"github.com/Igazine/hank-go/ext"
	"os"
	"path/filepath"
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

	if result.Type == hank.TypeError {
		fmt.Printf("Runtime Error %d: %s\n", result.Error.Code, result.Error)
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

	// 2. Register Localization
	r.RegisterLocalization(map[int]string{
		4001: "Target is not a function: {0}",
		4007: "Type Mismatch: Expected {0}, got {1} in {2}",
		4005: "Value exceeds safe integer bounds: {0} in {1}",
	})

	// 3. Register Extensions (Batteries included, but disconnected)
	r.RegisterExtension(hank.NewStdLib())
	r.RegisterExtension(&ext.PlatformExtension{})
	r.RegisterExtension(&ext.SysExtension{})

	return r
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
		// "test/conformance/14_syslib_hank.hank", // MOVED to extensions
		"test/conformance/15_logic_eq.hank",
		"test/conformance/16_chained_assign.hank",
		"test/conformance/17_num_module.hank",
		"test/conformance/18_runtime_module.hank",
		"test/conformance/19_error_handling.hank",
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
		hres, err := r.Run(res, args)
		if err != nil {
			fmt.Printf("Test Failed: %v\n", err)
		} else if hres.Type == hank.TypeError {
			fmt.Printf("Test Runtime Error %d: %s\n", hres.Error.Code, hres.Error)
		}
		fmt.Printf("--------------------\n\n")
	}

	// Run Extension Tests
	extTests := []string{
		"test/extensions/sys.hank",
		"test/extensions/platform_bin.hank",
	}

	for _, t := range extTests {
		fmt.Printf("--- Running Extension Test: %s ---\n", t)
		r := createRunner()
		path, _ := filepath.Abs(filepath.Join(root, t))
		res := NewFileResource(path)
		eres, err := r.Run(res, []hank.Value{})
		if err != nil {
			fmt.Printf("Extension Test Failed: %v\n", err)
		} else if eres.Type == hank.TypeError {
			fmt.Printf("Extension Runtime Error %d: %s\n", eres.Error.Code, eres.Error)
		}
		fmt.Printf("--------------------\n\n")
	}
}

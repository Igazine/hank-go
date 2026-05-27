package main

import (
	"fmt"
	"github.com/Igazine/hal-go"
	"os"
	"path/filepath"
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
	r.RegisterStd()

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
		r.RegisterStd()
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

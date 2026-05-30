package hank

import (
	"fmt"
	"strings"
)

var HankErrorMessages = map[HankError]string{
	UnexpectedCharacter:           "Unexpected character: %v",
	UnclosedStringLiteral:         "Unclosed string literal",
	EmptyScript:                   "Syntax Error: Script is empty.",
	ExpectedMainTask:              "Syntax Error: Expected main task definition (a closure or a block).",
	UnexpectedCodeOutsideMainTask: "Syntax Error: Unexpected code outside of main task. A Hank script must contain exactly one Task definition.",
	InvalidAssignmentTarget:       "Invalid assignment target",
	UnexpectedToken:               "Unexpected token: %v (%v)",
	MacroRequiresString:           "Syntax Error: The '@' macro strictly requires a string literal path (e.g., @ \"utils\"). Identifier shorthand is not allowed.",
	ExpectedIdentifier:            "Expected identifier, found %v",
	CircularDependency:           "Circular Dependency: %v",
	ResourceContentNotLoaded:      "Resource content not loaded: %v",
	ScriptMustBeTask:              "Hank Error: Script must evaluate to a Task definition.",
	MacroResourceNotFound:         "Macro resource not found: @%v",
	TargetNotFunction:             "Target is not a function: %v",
	TooManyArguments:              "Too many arguments",
	MissingRequiredParameter:      "Missing required parameter: %v",
	Halt:                          "HANK_HALT:%v",
	BitwiseOutOfBounds:            "Value exceeds safe integer bounds for bitwise operation: %v",
	GenericRuntimeError:           "%v",
	TypeMismatch:                  "Type Mismatch: Expected %v, got %v in %v",
}

func CreateHankError(code HankError, args []interface{}, filename string, line int, lineText string) *HankErrorValue {
	tmpl, ok := HankErrorMessages[code]
	if !ok {
		tmpl = "Unknown Error"
	}

	// Handle %v vs {i} mapping if needed, but for internal Go errors we use %v
	msg := tmpl
	for i, arg := range args {
		placeholder := fmt.Sprintf("{%d}", i)
		if strings.Contains(msg, placeholder) {
			msg = strings.ReplaceAll(msg, placeholder, fmt.Sprintf("%v", arg))
		}
	}
	// Fallback to fmt.Sprintf if placeholders weren't replaced
	if strings.Contains(msg, "%") {
		msg = fmt.Sprintf(msg, args...)
	}

	if filename != "" {
		msg = fmt.Sprintf("ERROR: %s in %s at\n\t%d:\t%s", msg, filename, line, lineText)
	}

	return &HankErrorValue{
		Code:    code,
		Message: msg,
	}
}

package hank

import (
	"fmt"
)

type EvalResultType int

const (
	ResultValue EvalResultType = iota
	ResultReturn
	ResultBreak
	ResultError
)

type EvalResult struct {
	Type  EvalResultType
	Value Value
}

type Interpreter struct {
	globalScope  Scope
	coreScope    Scope
	localization map[int]string
	depth        int
}

const MaxDepth = 500

func NewInterpreter(parentScope Scope, coreScope Scope, localization map[int]string) *Interpreter {
	if coreScope == nil {
		coreScope = NewScope(nil)
	}
	if parentScope == nil {
		parentScope = NewScope(coreScope)
	}
	if localization == nil {
		localization = make(map[int]string)
	}
	return &Interpreter{
		globalScope:  parentScope,
		coreScope:    coreScope,
		localization: localization,
	}
}

func (i *Interpreter) Run(expr Expr) (Value, error) {
	res := i.evalInScope(expr, i.globalScope)
	switch res.Type {
	case ResultValue, ResultReturn:
		return res.Value, nil
	case ResultBreak:
		return Value{Type: TypeVoid}, nil
	case ResultError:
		return res.Value, nil
	}
	return Value{Type: TypeVoid}, nil
}

func (i *Interpreter) Eval(expr Expr, scope Scope) Value {
	res := i.evalInScope(expr, scope)
	switch res.Type {
	case ResultValue, ResultReturn:
		return res.Value
	case ResultBreak:
		return Value{Type: TypeOpaque, Opaque: &OpaqueValue{Label: "__ControlFlow", Data: "Break"}}
	case ResultError:
		return res.Value
	}
	return Value{Type: TypeVoid}
}

func (i *Interpreter) IsError(val Value) bool {
	return val.Type == TypeError
}

func (i *Interpreter) GetLocalization() map[int]string {
	return i.localization
}

func (i *Interpreter) evalInScope(expr Expr, scope Scope) EvalResult {
	if expr == nil {
		return EvalResult{Type: ResultValue, Value: Value{Type: TypeVoid}}
	}

	switch e := expr.(type) {
	case *BlockExpr:
		// --- TASK HOISTING PASS ---
		for _, stmt := range e.Statements {
			if assign, ok := stmt.(*AssignExpr); ok {
				if _, isFunc := assign.Value.(*FuncDefExpr); isFunc {
					res := i.evalInScope(assign.Value, scope)
					if res.Type == ResultValue {
						scope.Set(assign.Name, res.Value)
					}
				}
				if nested, ok := assign.Value.(*AssignExpr); ok {
					if _, isFunc := nested.Value.(*FuncDefExpr); isFunc {
						res := i.evalInScope(nested.Value, scope)
						if res.Type == ResultValue {
							scope.Set(nested.Name, res.Value)
						}
					}
				}
			}
		}

		var last Value = Value{Type: TypeVoid}
		for _, stmt := range e.Statements {
			if assign, ok := stmt.(*AssignExpr); ok {
				if _, isFunc := assign.Value.(*FuncDefExpr); isFunc {
					continue
				}
				if nested, ok := assign.Value.(*AssignExpr); ok {
					if _, isFunc := nested.Value.(*FuncDefExpr); isFunc {
						continue
					}
				}
			}
			res := i.evalInScope(stmt, scope)
			if res.Type != ResultValue {
				return res
			}
			last = res.Value
		}
		return EvalResult{Type: ResultValue, Value: last}

	case *AssignExpr:
		res := i.evalInScope(e.Value, scope)
		if res.Type == ResultValue {
			scope.Set(e.Name, res.Value)
		}
		return res

	case *LiteralExpr:
		return EvalResult{Type: ResultValue, Value: e.Value}

	case *ErrorExpr:
		var args []Value
		for _, argExpr := range e.Args {
			res := i.evalInScope(argExpr, scope)
			if res.Type != ResultValue {
				return res
			}
			args = append(args, res.Value)
		}
		return EvalResult{Type: ResultValue, Value: Value{Type: TypeError, Error: &ErrorValue{Code: e.Code, Args: args}}}

	case *IdentExpr:
		if e.IsCore {
			return EvalResult{Type: ResultValue, Value: i.coreScope.Get(e.Name)}
		}
		if val := scope.Get(e.Name); val.Type != TypeVoid {
			return EvalResult{Type: ResultValue, Value: val}
		}
		return EvalResult{Type: ResultValue, Value: i.coreScope.Get(e.Name)}

	case *FieldExpr:
		res := i.evalInScope(e.Collection, scope)
		if res.Type != ResultValue {
			return res
		}
		coll := res.Value
		if coll.Type == TypeMap {
			if val, ok := coll.Map[e.FieldName]; ok {
				return EvalResult{Type: ResultValue, Value: val}
			}
		} else if coll.Type == TypeArray && e.FieldName == "length" {
			return EvalResult{Type: ResultValue, Value: Value{Type: TypeNumber, Number: float64(len(*coll.Array))}}
		} else if coll.Type == TypeString && e.FieldName == "length" {
			return EvalResult{Type: ResultValue, Value: Value{Type: TypeNumber, Number: float64(len(coll.String))}}
		}
		return EvalResult{Type: ResultValue, Value: Value{Type: TypeVoid}}

	case *FuncDefExpr:
		return EvalResult{Type: ResultValue, Value: Value{
			Type: TypeTask,
			Task: &TaskValue{
				IsNative: false,
				Params:   e.Params,
				Body:     e.Body,
				Closure:  scope,
			},
		}}

	case *FuncCallExpr:
		if i.depth > MaxDepth {
			return EvalResult{Type: ResultError, Value: Value{Type: TypeError, Error: &ErrorValue{Code: GenericRuntimeError, Args: []Value{{Type: TypeString, String: "Stack overflow"}}}}}
		}
		res := i.evalInScope(e.Target, scope)
		if res.Type != ResultValue {
			return res
		}
		target := res.Value

		var args []Value
		for _, argExpr := range e.Args {
			argRes := i.evalInScope(argExpr, scope)
			if argRes.Type != ResultValue {
				return argRes
			}
			args = append(args, argRes.Value)
		}
		return i.callInternal(target, args, scope)

	case *UnOpExpr:
		res := i.evalInScope(e.Target, scope)
		if res.Type != ResultValue {
			return res
		}
		val := res.Value

		switch e.Op {
		case "!":
			if i.isTruthy(val) {
				return EvalResult{Type: ResultValue, Value: Value{Type: TypeVoid}}
			}
			return EvalResult{Type: ResultValue, Value: Value{Type: TypeNumber, Number: 1}}
		case "^":
			return EvalResult{Type: ResultReturn, Value: val}
		case "?":
			return res
		}

	case *FlowControlExpr:
		condRes := i.evalInScope(e.Condition, scope)
		var branchRes EvalResult

		if condRes.Type == ResultValue {
			if i.isTruthy(condRes.Value) {
				branchRes = i.evalInScope(e.SuccessBlock, scope)
			} else if e.FallbackBlock != nil {
				branchRes = i.evalInScope(e.FallbackBlock, scope)
			} else {
				branchRes = EvalResult{Type: ResultValue, Value: Value{Type: TypeVoid}}
			}
		} else {
			branchRes = condRes
		}

		if branchRes.Type == ResultError && e.RescueBlock != nil {
			rescueScope := NewScope(scope)
			if e.CatchVar != "" {
				rescueScope.Set(e.CatchVar, branchRes.Value)
			}
			return i.evalInScope(e.RescueBlock, rescueScope)
		}
		return branchRes

	case *MapExpr:
		fields := make(map[string]Value)
		for k, vExpr := range e.Fields {
			res := i.evalInScope(vExpr, scope)
			if res.Type != ResultValue {
				return res
			}
			fields[k] = res.Value
		}
		return EvalResult{Type: ResultValue, Value: Value{Type: TypeMap, Map: fields}}

	case *ArrayExpr:
		items := new([]Value)
		for _, itemExpr := range e.Items {
			res := i.evalInScope(itemExpr, scope)
			if res.Type != ResultValue {
				return res
			}
			*items = append(*items, res.Value)
		}
		return EvalResult{Type: ResultValue, Value: Value{Type: TypeArray, Array: items}}
	}

	return EvalResult{Type: ResultValue, Value: Value{Type: TypeVoid}}
}

func (i *Interpreter) callInternal(task Value, args []Value, scope Scope) EvalResult {
	if task.Type != TypeTask {
		return EvalResult{Type: ResultError, Value: Value{Type: TypeError, Error: &ErrorValue{Code: TargetNotFunction, Args: []Value{{Type: TypeString, String: ValueToString(task)}}}}}
	}

	tv := task.Task
	if tv.IsNative {
		ctx := &executionContextImpl{
			interp: i,
			scope:  scope,
		}
		res := tv.Native(args, ctx)
		if res.Type == TypeOpaque && res.Opaque.Label == "__ControlFlow" && fmt.Sprintf("%v", res.Opaque.Data) == "Break" {
			return EvalResult{Type: ResultBreak}
		}
		if res.Type == TypeError {
			return EvalResult{Type: ResultError, Value: res}
		}
		return EvalResult{Type: ResultValue, Value: res}
	}

	i.depth++
	defer func() { i.depth-- }()

	if len(args) > len(tv.Params) {
		return EvalResult{Type: ResultError, Value: Value{Type: TypeError, Error: &ErrorValue{Code: TooManyArguments}}}
	}

	taskScope := NewScope(tv.Closure)
	for idx, p := range tv.Params {
		var val Value = Value{Type: TypeVoid}
		if idx < len(args) {
			val = args[idx]
		} else if p.DefaultValue != nil {
			res := i.evalInScope(p.DefaultValue.(Expr), taskScope)
			if res.Type != ResultValue {
				return res
			}
			val = res.Value
		} else if !p.IsOptional {
			return EvalResult{Type: ResultError, Value: Value{Type: TypeError, Error: &ErrorValue{Code: MissingRequiredParameter, Args: []Value{{Type: TypeString, String: p.Name}}}}}
		}
		taskScope.Set(p.Name, val)
	}

	bodyExpr := tv.Body.(Expr)
	res := i.evalInScope(bodyExpr, taskScope)
	if res.Type == ResultValue || res.Type == ResultReturn {
		if res.Value.Type == TypeError {
			return EvalResult{Type: ResultError, Value: res.Value}
		}
		return EvalResult{Type: ResultValue, Value: res.Value}
	}
	return res
}

func (i *Interpreter) Call(task Value, args []Value, scope Scope) Value {
	finalArgs := args
	if task.Type == TypeTask && !task.Task.IsNative {
		if len(args) > len(task.Task.Params) {
			finalArgs = args[:len(task.Task.Params)]
		}
	}
	res := i.callInternal(task, finalArgs, scope)
	switch res.Type {
	case ResultValue, ResultReturn:
		return res.Value
	case ResultBreak:
		return Value{Type: TypeOpaque, Opaque: &OpaqueValue{Label: "__ControlFlow", Data: "Break"}}
	case ResultError:
		return res.Value
	}
	return Value{Type: TypeVoid}
}

func (i *Interpreter) isTruthy(v Value) bool {
	return v.Type != TypeVoid
}

type executionContextImpl struct {
	interp *Interpreter
	scope  Scope
}

func (ctx *executionContextImpl) Parse(source string) (any, error) {
	l := NewLexer(source)
	p := NewParser(l.Tokenize(), "dynamic", nil)
	return p.Parse()
}

func (ctx *executionContextImpl) Eval(node any) Value {
	return ctx.interp.Eval(node.(Expr), ctx.scope)
}

func (ctx *executionContextImpl) Call(task Value, args []Value) Value {
	return ctx.interp.Call(task, args, ctx.scope)
}

func (ctx *executionContextImpl) IsError(val Value) bool {
	return ctx.interp.IsError(val)
}

func (ctx *executionContextImpl) GetLocalization() map[int]string {
	return ctx.interp.GetLocalization()
}

func (ctx *executionContextImpl) Scope() Scope {
	return ctx.scope
}

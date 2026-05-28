package hank

import (
	"fmt"
)

type ReturnSignal struct {
	Value Value
}

func (r *ReturnSignal) Error() string {
	return "Return signal"
}

type Interpreter struct {
	globalScope Scope
	coreScope   Scope
	depth       int
}

const MaxDepth = 500

func NewInterpreter(parentScope Scope, coreScope Scope) *Interpreter {
	if coreScope == nil {
		coreScope = NewScope(nil)
	}
	if parentScope == nil {
		parentScope = NewScope(coreScope)
	}
	return &Interpreter{
		globalScope: parentScope,
		coreScope:   coreScope,
	}
}

func (i *Interpreter) Run(expr Expr) (Value, error) {
	// Root level hoisting is now handled inside Eval for BlockExpr
	return i.Eval(expr, i.globalScope), nil
}

func (i *Interpreter) Eval(expr Expr, scope Scope) Value {
	if expr == nil {
		return Value{Type: TypeVoid}
	}

	switch e := expr.(type) {
	case *BlockExpr:
		// --- TASK HOISTING PASS ---
		for _, stmt := range e.Statements {
			if assign, ok := stmt.(*AssignExpr); ok {
				// Direct func def
				if _, isFunc := assign.Value.(*FuncDefExpr); isFunc {
					val := i.Eval(assign.Value, scope)
					scope.Set(assign.Name, val)
				}
				// Nested assignments (usually from macros)
				if nested, ok := assign.Value.(*AssignExpr); ok {
					if _, isFunc := nested.Value.(*FuncDefExpr); isFunc {
						val := i.Eval(nested.Value, scope)
						scope.Set(nested.Name, val)
					}
				}
			}
		}

		var last Value = Value{Type: TypeVoid}
		for _, stmt := range e.Statements {
			// Skip already hoisted tasks in eval pass
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
			last = i.Eval(stmt, scope)
		}
		return last

	case *AssignExpr:
		val := i.Eval(e.Value, scope)
		scope.Set(e.Name, val)
		return val

	case *LiteralExpr:
		return e.Value

	case *IdentExpr:
		if e.IsCore {
			return i.coreScope.Get(e.Name)
		}
		if val := scope.Get(e.Name); val.Type != TypeVoid {
			return val
		}
		// Fallback to core if not shadowed
		return i.coreScope.Get(e.Name)

	case *FieldExpr:
		obj := i.Eval(e.Object, scope)
		if obj.Type == TypeObject {
			if val, ok := obj.Object[e.FieldName]; ok {
				return val
			}
		}
		return Value{Type: TypeVoid}

	case *FuncDefExpr:
		return Value{
			Type: TypeTask,
			Task: &TaskValue{
				IsNative: false,
				Params:   e.Params,
				Body:     e.Body,
				Closure:  scope,
			},
		}

	case *FuncCallExpr:
		if i.depth > MaxDepth {
			panic("Stack overflow")
		}
		target := i.Eval(e.Target, scope)
		var args []Value
		for _, argExpr := range e.Args {
			args = append(args, i.Eval(argExpr, scope))
		}
		return i.Call(target, args, scope)

	case *UnOpExpr:
		switch e.Op {
		case "!":
			val := i.Eval(e.Target, scope)
			if i.isTruthy(val) {
				return Value{Type: TypeVoid}
			}
			return Value{Type: TypeNumber, Number: 1}
		case "^":
			var val Value = Value{Type: TypeVoid}
			if e.Target != nil {
				val = i.Eval(e.Target, scope)
			}
			panic(&ReturnSignal{Value: val})
		case "?":
			return i.Eval(e.Target, scope)
		}

	case *FlowControlExpr:
		return i.evalFlowControl(e, scope)

	case *ObjectExpr:
		fields := make(map[string]Value)
		for k, vExpr := range e.Fields {
			fields[k] = i.Eval(vExpr, scope)
		}
		return Value{Type: TypeObject, Object: fields}

	case *ArrayExpr:
		var items []Value
		for _, itemExpr := range e.Items {
			items = append(items, i.Eval(itemExpr, scope))
		}
		return Value{Type: TypeArray, Array: items}
	}

	return Value{Type: TypeVoid}
}

func (i *Interpreter) evalFlowControl(e *FlowControlExpr, scope Scope) (result Value) {
	defer func() {
		if r := recover(); r != nil {
			if sig, ok := r.(*ReturnSignal); ok {
				panic(sig)
			}
			if e.RescueBlock != nil {
				errStr := fmt.Sprintf("%v", r)
				rescueScope := NewScope(scope)
				rescueScope.Set(e.CatchVar, Value{Type: TypeString, String: errStr})
				result = i.Eval(e.RescueBlock, rescueScope)
			} else {
				panic(r)
			}
		}
	}()

	cond := i.Eval(e.Condition, scope)
	if i.isTruthy(cond) {
		return i.Eval(e.SuccessBlock, scope)
	} else if e.FallbackBlock != nil {
		return i.Eval(e.FallbackBlock, scope)
	}
	return Value{Type: TypeVoid}
}

func (i *Interpreter) Call(task Value, args []Value, scope Scope) Value {
	if task.Type != TypeTask {
		panic(fmt.Sprintf("Target is not a function: %v", task.Type))
	}

	tv := task.Task
	if tv.IsNative {
		ctx := &executionContextImpl{
			interp: i,
			scope:  scope,
		}
		return tv.Native(args, ctx)
	}

	i.depth++
	defer func() { i.depth-- }()

	// Use captured closure as parent
	taskScope := NewScope(tv.Closure)
	i.mapArgsToParams(tv.Params, args, taskScope)

	bodyExpr := tv.Body.(Expr)
	return i.evalTaskBody(bodyExpr, taskScope)
}

func (i *Interpreter) evalTaskBody(body Expr, scope Scope) (val Value) {
	defer func() {
		if r := recover(); r != nil {
			if sig, ok := r.(*ReturnSignal); ok {
				val = sig.Value
			} else {
				panic(r)
			}
		}
	}()
	return i.Eval(body, scope)
}

func (i *Interpreter) mapArgsToParams(params []Param, args []Value, scope Scope) {
	if len(args) > len(params) {
		panic("Too many arguments")
	}

	for idx, p := range params {
		var val Value = Value{Type: TypeVoid}
		if idx < len(args) {
			val = args[idx]
		} else if p.DefaultValue != nil {
			val = i.Eval(p.DefaultValue.(Expr), scope)
		} else if !p.IsOptional {
			panic(fmt.Sprintf("Missing required argument: %s", p.Name))
		}
		scope.Set(p.Name, val)
	}
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
	finalArgs := args
	if task.Type == TypeTask && !task.Task.IsNative {
		if len(args) > len(task.Task.Params) {
			finalArgs = args[:len(task.Task.Params)]
		}
	}
	return ctx.interp.Call(task, finalArgs, ctx.scope)
}

func (ctx *executionContextImpl) Scope() Scope {
	return ctx.scope
}

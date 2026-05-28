package hank

import (
	"fmt"
)

/**
 * A Hank Host Runner.
 * Handles resource orchestration, macro resolution, and AST caching.
 * Platform-agnostic: uses the Resource model for all content retrieval.
 */
type Runner struct {
	resourceCache map[string]Resource
	coreScope     Scope
}

func NewRunner() *Runner {
	return &Runner{
		resourceCache: make(map[string]Resource),
		coreScope:     NewScope(nil),
	}
}

func (r *Runner) RegisterModule(name string, tasks map[string]NativeFunc) {
	moduleObj := make(map[string]Value)
	for tName, fn := range tasks {
		moduleObj[tName] = Value{
			Type: TypeTask,
			Task: &TaskValue{
				IsNative: true,
				Name:     fmt.Sprintf("%s.%s", name, tName),
				Native:   fn,
			},
		}
	}
	r.coreScope.Set(name, Value{Type: TypeObject, Object: moduleObj})
}

/**
 * Pre-loads and caches a resource for execution.
 */
func (r *Runner) Load(resource Resource, stack []string) (Expr, error) {
	if stack == nil {
		stack = []string{}
	}

	// Check cache
	if cached, ok := r.resourceCache[resource.ID()]; ok && cached.AST() != nil {
		return cached.AST(), nil
	}

	// Circular Dependency Check
	for _, s := range stack {
		if s == resource.ID() {
			return nil, fmt.Errorf("Circular Dependency: %s", resource.ID())
		}
	}

	// Reconcile with cache
	cached, ok := r.resourceCache[resource.ID()]
	if !ok {
		r.resourceCache[resource.ID()] = resource
		cached = resource
	}

	err := cached.Load()
	if err != nil {
		return nil, err
	}

	newStack := append(stack, cached.ID())

	lexer := NewLexer(cached.Content())
	parser := NewParser(lexer.Tokenize(), cached.ID(), func(macroPath string) (Expr, error) {
		mRes, err := cached.Resolve(macroPath)
		if err != nil {
			return nil, err
		}
		return r.Load(mRes, newStack)
	})

	ast, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	cached.SetAST(ast)
	return ast, nil
}

/**
 * Removes a resource and its AST from the cache.
 */
func (r *Runner) Unload(resource Resource) {
	delete(r.resourceCache, resource.ID())
}

/**
 * Executes a Hank Resource.
 */
func (r *Runner) Run(resource Resource, args []Value) (Value, error) {
	ast, err := r.Load(resource, nil)
	if err != nil {
		return Value{Type: TypeVoid}, err
	}

	interp := NewInterpreter(nil, r.coreScope)
	scriptTask := interp.Eval(ast, interp.globalScope)

	if scriptTask.Type != TypeTask {
		return Value{Type: TypeVoid}, fmt.Errorf("Script did not evaluate to a Task")
	}

	result := interp.Call(scriptTask, args, interp.globalScope)
	return result, nil
}

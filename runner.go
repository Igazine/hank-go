package hal

import (
	"fmt"
	"os"
	"path/filepath"
)

type Runner struct {
	pathCache map[string]string
	astCache  map[string]Expr
	macroMap  map[string]string
	coreScope Scope
}

func NewRunner() *Runner {
	return &Runner{
		pathCache: make(map[string]string),
		astCache:  make(map[string]Expr),
		macroMap:  make(map[string]string),
		coreScope: NewScope(nil),
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

func (r *Runner) Load(scriptPath string) (string, error) {
	absPath, err := filepath.Abs(scriptPath)
	if err != nil { return "", err }

	if _, ok := r.astCache[absPath]; ok { return absPath, nil }

	// Pre-process (macro includes)
	err = r.preprocess(absPath, []string{})
	if err != nil { return "", err }

	content := r.pathCache[absPath]
	lexer := NewLexer(content)
	parser := NewParser(lexer.Tokenize(), absPath, r.macroMap)
	
	ast, err := parser.Parse()
	if err != nil { return "", err }

	r.astCache[absPath] = ast
	return absPath, nil
}

func (r *Runner) Unload(scriptPath string) {
	absPath, err := filepath.Abs(scriptPath)
	if err != nil { return }
	delete(r.astCache, absPath)
	delete(r.pathCache, absPath)
}

func (r *Runner) Run(scriptPath string, args []Value) (Value, error) {
	absPath, err := r.Load(scriptPath)
	if err != nil { return Value{Type: TypeVoid}, err }

	ast := r.astCache[absPath]
	interp := NewInterpreter(nil, r.coreScope)
	
	// Evaluating the script AST yields the script's main TaskValue
	scriptTask := interp.Eval(ast, interp.globalScope)
	
	if scriptTask.Type != TypeTask {
		return Value{Type: TypeVoid}, fmt.Errorf("Script did not evaluate to a Task")
	}

	// Now invoke the script Task with Host arguments
	result := interp.Call(scriptTask, args, interp.globalScope)
	return result, nil
}

func (r *Runner) preprocess(path string, stack []string) error {
	for _, s := range stack {
		if s == path { return fmt.Errorf("Circular Dependency: %s", path) }
	}
	if _, ok := r.pathCache[path]; ok { return nil }

	content, err := os.ReadFile(path)
	if err != nil { return err }
	r.pathCache[path] = string(content)

	newStack := append(stack, path)
	macros := r.scanMacros(string(content))
	dir := filepath.Dir(path)

	for _, m := range macros {
		mPath := r.resolvePath(m, dir)
		mAbs, err := filepath.Abs(mPath)
		if err != nil { return err }
		err = r.preprocess(mAbs, newStack)
		if err != nil { return err }
		r.macroMap[m] = r.pathCache[mAbs]
	}
	return nil
}

func (r *Runner) scanMacros(content string) []string {
	l := NewLexer(content)
	tokens := l.Tokenize()
	var macros []string
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Type == TokenAt {
			next := tokens[i+1]
			if next.Type == TokenString || next.Type == TokenIdentifier {
				macros = append(macros, next.Literal)
			}
		}
	}
	return macros
}

func (r *Runner) resolvePath(m string, dir string) string {
	if filepath.IsAbs(m) { return m }
	joined := filepath.Join(dir, m)
	if filepath.Ext(joined) == "" {
		if _, err := os.Stat(joined + ".hal"); err == nil {
			return joined + ".hal"
		}
	}
	return joined
}

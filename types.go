package hank

type ValueType int

const (
	TypeVoid ValueType = iota
	TypeNumber
	TypeString
	TypeArray
	TypeObject
	TypeOpaque
	TypeTask
)

type Value struct {
	Type   ValueType
	Number float64
	String string
	Array  []Value
	Object map[string]Value
	Opaque *OpaqueValue
	Task   *TaskValue
}

type OpaqueValue struct {
	Label string
	Data  interface{}
}

type TaskValue struct {
	IsNative bool
	Name     string
	Params   []Param
	Body     any // AST Expr
	Native   NativeFunc
	Closure  Scope
}

type Param struct {
	Name         string
	IsOptional   bool
	DefaultValue any // AST Expr
}

type NativeFunc func(args []Value, ctx ExecutionContext) Value

type ExecutionContext interface {
	Parse(source string) (any, error)
	Eval(node any) Value
	Call(task Value, args []Value) Value
	Scope() Scope
}

type Scope interface {
	Get(name string) Value
	Set(name string, val Value)
	Exists(name string) bool
}

type IHALSerializable interface {
	SerializeHAL() string
}

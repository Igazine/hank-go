package hank

type ValueType int

const (
	TypeVoid ValueType = iota
	TypeNumber
	TypeString
	TypeArray
	TypeMap
	TypeOpaque
	TypeTask
	TypeError
)

type Value struct {
	Type   ValueType
	Number float64
	String string
	Array  *[]Value
	Map    map[string]Value
	Opaque *OpaqueValue
	Task   *TaskValue
	Error  *ErrorValue
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
	IsError(val Value) bool
	GetLocalization() map[int]string
	Scope() Scope
}

type Scope interface {
	Get(name string) Value
	Set(name string, val Value)
	Exists(name string) bool
}

type Resource interface {
	ID() string
	Content() string
	AST() Expr
	SetAST(Expr)
	Load() error
	Resolve(id string) (Resource, error)
}

type IHALSerializable interface {
	SerializeHAL() string
}

type HankExtension interface {
	Name() string
	GetModules() map[string]map[string]NativeFunc
}

type HankError int

const (
	// Lexical Errors (10xx)
	UnexpectedCharacter   HankError = 1001
	UnclosedStringLiteral HankError = 1002

	// Syntax Errors (20xx)
	EmptyScript                   HankError = 2001
	ExpectedMainTask              HankError = 2002
	UnexpectedCodeOutsideMainTask HankError = 2003
	InvalidAssignmentTarget       HankError = 2004
	UnexpectedToken               HankError = 2005
	MacroRequiresString           HankError = 2006
	ExpectedIdentifier             HankError = 2007

	// Resolution & Runner Errors (30xx)
	CircularDependency       HankError = 3001
	ResourceContentNotLoaded HankError = 3002
	ScriptMustBeTask         HankError = 3003
	MacroResourceNotFound   HankError = 3004

	// Runtime Errors (40xx)
	TargetNotFunction        HankError = 4001
	TooManyArguments         HankError = 4002
	MissingRequiredParameter HankError = 4003
	Halt                     HankError = 4004
	BitwiseOutOfBounds       HankError = 4005
	GenericRuntimeError      HankError = 4006
	TypeMismatch             HankError = 4007
)

type ErrorValue struct {
	Code HankError
	Args []Value
}

type HankErrorValue struct {
	Code    HankError
	Message string
}

func (e *HankErrorValue) Error() string {
	return e.Message
}

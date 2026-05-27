package hank

type Expr interface {
	isExpr()
	GetTokenData() Token
}

type BlockExpr struct {
	Statements []Expr
	Token      Token
}

type AssignExpr struct {
	Name  string
	Value Expr
	Token Token
}

type LiteralExpr struct {
	Value Value
	Token Token
}

type IdentExpr struct {
	Name   string
	IsCore bool
	Token  Token
}

type FieldExpr struct {
	Object    Expr
	FieldName string
	Token     Token
}

type FuncDefExpr struct {
	Params []Param
	Body   Expr
	Token  Token
}

type FuncCallExpr struct {
	Target Expr
	Args   []Expr
	Token  Token
}

type UnOpExpr struct {
	Op     string
	Target Expr
	Token  Token
}

type ObjectExpr struct {
	Fields map[string]Expr
	Token  Token
}

type ArrayExpr struct {
	Items []Expr
	Token Token
}

type FlowControlExpr struct {
	Condition     Expr
	SuccessBlock  Expr
	FallbackBlock Expr
	RescueBlock   Expr
	CatchVar      string
	Token         Token
}

// Marker methods
func (e *BlockExpr) isExpr()       {}
func (e *AssignExpr) isExpr()      {}
func (e *LiteralExpr) isExpr()     {}
func (e *IdentExpr) isExpr()       {}
func (e *FieldExpr) isExpr()       {}
func (e *FuncDefExpr) isExpr()     {}
func (e *FuncCallExpr) isExpr()    {}
func (e *UnOpExpr) isExpr()        {}
func (e *ObjectExpr) isExpr()      {}
func (e *ArrayExpr) isExpr()       {}
func (e *FlowControlExpr) isExpr() {}

func (e *BlockExpr) GetTokenData() Token       { return e.Token }
func (e *AssignExpr) GetTokenData() Token      { return e.Token }
func (e *LiteralExpr) GetTokenData() Token     { return e.Token }
func (e *IdentExpr) GetTokenData() Token       { return e.Token }
func (e *FieldExpr) GetTokenData() Token       { return e.Token }
func (e *FuncDefExpr) GetTokenData() Token     { return e.Token }
func (e *FuncCallExpr) GetTokenData() Token    { return e.Token }
func (e *UnOpExpr) GetTokenData() Token        { return e.Token }
func (e *ObjectExpr) GetTokenData() Token      { return e.Token }
func (e *ArrayExpr) GetTokenData() Token       { return e.Token }
func (e *FlowControlExpr) GetTokenData() Token { return e.Token }

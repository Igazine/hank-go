package hank

import (
	"path/filepath"
	"strconv"
	"strings"
)

type Parser struct {
	tokens        []Token
	pos           int
	filename      string
	macroResolver func(string) (Expr, error)
}

func NewParser(tokens []Token, filename string, macroResolver func(string) (Expr, error)) *Parser {
	return &Parser{
		tokens:        tokens,
		filename:      filename,
		macroResolver: macroResolver,
	}
}

func (p *Parser) Parse() (Expr, error) {
	tdRoot := p.peekTd()
	var exprs []Expr

	// 1. Consume Macro Includes
	for !p.isEof() {
		p.skipNewlines()
		if p.peek().Type != TokenAt {
			break
		}
		inc, err := p.parseInclude()
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, inc)
	}

	p.skipNewlines()
	if p.isEof() {
		return nil, p.error(EmptyScript)
	}

	// 2. Parse exactly ONE TaskDef (FuncDef or Block)
	var task Expr
	var err error
	if p.peek().Type == TokenLParen && p.isFuncDefStart() {
		task, err = p.parseFuncDef()
	} else if p.peek().Type == TokenLBrace {
		task, err = p.parseBlock()
	} else {
		return nil, p.error(ExpectedMainTask)
	}
	if err != nil {
		return nil, err
	}

	exprs = append(exprs, task)

	// 3. Assert EOF
	p.skipNewlines()
	if !p.isEof() {
		return nil, p.error(UnexpectedCodeOutsideMainTask)
	}

	if len(exprs) == 1 {
		return exprs[0], nil
	}
	return &BlockExpr{Statements: exprs, Token: tdRoot}, nil
}

func (p *Parser) parseStatement() (Expr, error) {
	p.skipNewlines()
	td := p.peekTd()

	switch td.Type {
	case TokenQuestion:
		return p.parseFlowControl()
	case TokenCaret:
		return p.parseReturn()
	case TokenAt:
		return p.parseInclude()
	default:
		return p.parseExpression()
	}
}

func (p *Parser) parseFlowControl() (Expr, error) {
	td := p.consume(TokenQuestion)
	p.consume(TokenLParen)
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	p.consume(TokenRParen)

	success, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	var fallback, rescue Expr
	var catchVar string

	savedPos := p.pos
	p.skipNewlines()
	if !p.isEof() && p.peek().Type == TokenColon {
		p.consume(TokenColon)
		fallback, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
		savedPos = p.pos
		p.skipNewlines()
	} else {
		p.pos = savedPos
	}

	if !p.isEof() && p.peek().Type == TokenRescue {
		p.consume(TokenRescue)
		if p.peek().Type == TokenLParen {
			p.consume(TokenLParen)
			catchVar = p.consumeIdentifier()
			p.consume(TokenRParen)
		}
		rescue, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
	} else {
		p.pos = savedPos
	}

	return &FlowControlExpr{
		Condition:     cond,
		SuccessBlock:  success,
		FallbackBlock: fallback,
		RescueBlock:   rescue,
		CatchVar:      catchVar,
		Token:         td,
	}, nil
}

func (p *Parser) parseExpression() (Expr, error) {
	return p.parseAssignment()
}

func (p *Parser) parseAssignment() (Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	if !p.isEof() && p.peek().Type == TokenAssign {
		switch e := expr.(type) {
		case *IdentExpr:
			if !e.IsCore {
				p.consume(TokenAssign)
				val, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				return &AssignExpr{Name: e.Name, Value: val, Token: e.Token}, nil
			}
		}
		return nil, p.error(InvalidAssignmentTarget)
	}

	return expr, nil
}

func (p *Parser) parsePrimary() (Expr, error) {
	td := p.peekTd()
	var expr Expr
	var err error

	switch td.Type {
	case TokenLParen:
		if p.isFuncDefStart() {
			expr, err = p.parseFuncDef()
		} else {
			p.pos++
			expr, err = p.parseExpression()
			if err != nil {
				return nil, err
			}
			p.consume(TokenRParen)
		}
	case TokenLBrace:
		expr, err = p.parseBlock()
	case TokenNot:
		p.pos++
		target, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		expr = &UnOpExpr{Op: "!", Target: target, Token: td}
	case TokenLBracket:
		expr, err = p.parseCollectionLiteral()
	case TokenHash:
		p.pos++
		name := p.consumeIdentifier()
		expr = &IdentExpr{Name: name, IsCore: true, Token: td}
	case TokenIdentifier:
		name := p.consumeIdentifier()
		expr = &IdentExpr{Name: name, IsCore: false, Token: td}
	case TokenString:
		p.pos++
		expr = &LiteralExpr{Value: Value{Type: TypeString, String: td.Literal}, Token: td}
	case TokenNumber:
		p.pos++
		val, _ := strconv.ParseFloat(td.Literal, 64)
		expr = &LiteralExpr{Value: Value{Type: TypeNumber, Number: val}, Token: td}
	case TokenCaret:
		expr, err = p.parseReturn()
	case TokenAt:
		expr, err = p.parseInclude()
	default:
		return nil, p.error(UnexpectedToken, td.Type, td.Literal)
	}

	if err != nil {
		return nil, err
	}
	return p.finishPrimary(expr)
}

func (p *Parser) finishPrimary(expr Expr) (Expr, error) {
	for {
		if p.isEof() {
			break
		}
		td := p.peekTd()
		if td.Type == TokenDot {
			p.consume(TokenDot)
			expr = &FieldExpr{Collection: expr, FieldName: p.consumeIdentifier(), Token: td}
		} else if td.Type == TokenLParen {
			args, err := p.parseArgList()
			if err != nil {
				return nil, err
			}
			expr = &FuncCallExpr{Target: expr, Args: args, Token: td}
		} else {
			break
		}
	}
	return expr, nil
}

func (p *Parser) isFuncDefStart() bool {
	savedPos := p.pos
	defer func() { p.pos = savedPos }()

	p.pos++ // skip (
	depth := 1
	for p.pos < len(p.tokens) && depth > 0 {
		if p.tokens[p.pos].Type == TokenLParen {
			depth++
		}
		if p.tokens[p.pos].Type == TokenRParen {
			depth--
		}
		p.pos++
	}
	if p.isEof() {
		return false
	}
	p.skipNewlines()
	return !p.isEof() && p.peek().Type == TokenLBrace
}

func (p *Parser) parseFuncDef() (Expr, error) {
	td := p.peekTd()
	p.consume(TokenLParen)
	var params []Param
	if p.peek().Type != TokenRParen {
		param, err := p.parseParam()
		if err != nil {
			return nil, err
		}
		params = append(params, param)
		for p.peek().Type == TokenComma {
			p.consume(TokenComma)
			param, err = p.parseParam()
			if err != nil {
				return nil, err
			}
			params = append(params, param)
		}
	}
	p.consume(TokenRParen)
	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}
	return &FuncDefExpr{Params: params, Body: body, Token: td}, nil
}

func (p *Parser) parseParam() (Param, error) {
	isOptional := false
	if p.peek().Type == TokenQuestion {
		p.consume(TokenQuestion)
		isOptional = true
	}
	name := p.consumeIdentifier()
	var defVal Expr
	if !p.isEof() && p.peek().Type == TokenAssign {
		p.consume(TokenAssign)
		var err error
		defVal, err = p.parseExpression()
		if err != nil {
			return Param{}, err
		}
		isOptional = true
	}
	return Param{Name: name, IsOptional: isOptional, DefaultValue: defVal}, nil
}

func (p *Parser) parseBlock() (Expr, error) {
	td := p.consume(TokenLBrace)
	var exprs []Expr
	for !p.isEof() && p.peek().Type != TokenRBrace {
		p.skipNewlines()
		if p.peek().Type == TokenRBrace {
			break
		}
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, stmt)
	}
	p.consume(TokenRBrace)
	return &BlockExpr{Statements: exprs, Token: td}, nil
}

func (p *Parser) parseCollectionLiteral() (Expr, error) {
	td := p.consume(TokenLBracket)
	p.skipNewlines()

	// 1. Handle [:]
	if p.peek().Type == TokenColon {
		p.consume(TokenColon)
		p.consume(TokenRBracket)
		return &MapExpr{Fields: make(map[string]Expr), Token: td}, nil
	}

	// 2. Handle []
	if p.peek().Type == TokenRBracket {
		p.consume(TokenRBracket)
		return &ArrayExpr{Items: []Expr{}, Token: td}, nil
	}

	// 3. Parse first element
	first, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	p.skipNewlines()

	if p.peek().Type == TokenColon {
		// This is a Map
		p.consume(TokenColon)
		val, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		fields := make(map[string]Expr)
		fields[p.getStaticKey(first)] = val

		for {
			p.skipNewlines()
			if !p.isEof() && p.peek().Type == TokenComma {
				p.consume(TokenComma)
				p.skipNewlines()
				if p.peek().Type == TokenRBracket {
					break
				}
				keyExpr, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				p.consume(TokenColon)
				valExpr, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				fields[p.getStaticKey(keyExpr)] = valExpr
			} else {
				break
			}
		}
		p.consume(TokenRBracket)
		return &MapExpr{Fields: fields, Token: td}, nil
	} else {
		// This is an Array
		items := []Expr{first}
		for {
			p.skipNewlines()
			if !p.isEof() && p.peek().Type == TokenComma {
				p.consume(TokenComma)
				p.skipNewlines()
				if p.peek().Type == TokenRBracket {
					break
				}
				item, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				items = append(items, item)
			} else {
				break
			}
		}
		p.consume(TokenRBracket)
		return &ArrayExpr{Items: items, Token: td}, nil
	}
}

func (p *Parser) getStaticKey(e Expr) string {
	switch expr := e.(type) {
	case *LiteralExpr:
		if expr.Value.Type == TypeString {
			return expr.Value.String
		}
	case *IdentExpr:
		if !expr.IsCore {
			return expr.Name
		}
	}
	panic(p.error(ExpectedIdentifier, p.peek().Type))
}

func (p *Parser) parseArgList() ([]Expr, error) {
	p.consume(TokenLParen)
	var args []Expr
	p.skipNewlines()
	if p.peek().Type != TokenRParen {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		for {
			p.skipNewlines()
			if !p.isEof() && p.peek().Type == TokenComma {
				p.consume(TokenComma)
				p.skipNewlines()
				arg, err = p.parseExpression()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
			} else {
				break
			}
		}
	}
	p.skipNewlines()
	p.consume(TokenRParen)
	return args, nil
}

func (p *Parser) parseReturn() (Expr, error) {
	td := p.consume(TokenCaret)
	var val Expr
	if !p.isEof() {
		next := p.peek().Type
		if next != TokenNewline && next != TokenRBrace && next != TokenRBracket && next != TokenComma && next != TokenRParen {
			var err error
			val, err = p.parseExpression()
			if err != nil {
				return nil, err
			}
		}
	}
	return &UnOpExpr{Op: "^", Target: val, Token: td}, nil
}
func (p *Parser) parseInclude() (Expr, error) {
	td := p.consume(TokenAt)
	var rawPath string
	if p.peek().Type == TokenString {
		t := p.consume(TokenString)
		rawPath = t.Literal
	} else {
		return nil, p.error(MacroRequiresString)
	}

	taskAst, err := p.macroResolver(rawPath)
	if err != nil {
		return nil, err
	}

	base := filepath.Base(rawPath)
	taskName := strings.TrimSuffix(base, filepath.Ext(base))

	return &AssignExpr{Name: taskName, Value: taskAst, Token: td}, nil
}

func (p *Parser) consumeIdentifier() string {
	td := p.peekTd()
	if td.Type != TokenIdentifier {
		panic(p.error(ExpectedIdentifier, td.Type))
	}
	p.pos++
	return td.Literal
}

func (p *Parser) consume(t TokenType) Token {
	td := p.peekTd()
	if td.Type != t {
		panic(p.error(UnexpectedToken, t, td.Type))
	}
	p.pos++
	return td
}

func (p *Parser) peek() Token {
	if p.isEof() {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos]
}

func (p *Parser) peekTd() Token {
	if p.isEof() {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos]
}

func (p *Parser) skipNewlines() {
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenNewline {
		p.pos++
	}
}

func (p *Parser) isEof() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == TokenEOF
}

func (p *Parser) error(code HankError, args ...interface{}) error {
	td := p.peekTd()
	return CreateHankError(code, args, p.filename, td.Line, td.LineText)
}

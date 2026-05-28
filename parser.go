package hank

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type Parser struct {
	tokens   []Token
	pos      int
	filename string
	macroMap map[string]string
}

func NewParser(tokens []Token, filename string, macroMap map[string]string) *Parser {
	if macroMap == nil {
		macroMap = make(map[string]string)
	}
	return &Parser{
		tokens:   tokens,
		filename: filename,
		macroMap: macroMap,
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
		return nil, p.error("Syntax Error: Script is empty.")
	}

	// 2. Parse exactly ONE TaskDef (FuncDef or Block)
	var task Expr
	var err error
	if p.peek().Type == TokenLParen && p.isFuncDefStart() {
		task, err = p.parseFuncDef()
	} else if p.peek().Type == TokenLBrace {
		body, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		task = &FuncDefExpr{Params: []Param{}, Body: body, Token: p.peekTd()}
	} else {
		return nil, p.error("Syntax Error: Expected main task definition (a closure or a block).")
	}
	if err != nil {
		return nil, err
	}

	exprs = append(exprs, task)

	// 3. Assert EOF
	p.skipNewlines()
	if !p.isEof() {
		return nil, p.error("Syntax Error: Unexpected code outside of main task. A Hank script must contain exactly one Task definition.")
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
	}

	if !p.isEof() && p.peek().Type == TokenRescue {
		p.consume(TokenRescue)
		p.consume(TokenLParen)
		catchVar = p.consumeIdentifier()
		p.consume(TokenRParen)
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
	return p.parsePrimary()
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
		expr, err = p.parseObjectLiteral()
	case TokenNot:
		p.pos++
		target, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		expr = &UnOpExpr{Op: "!", Target: target, Token: td}
	case TokenQuestion:
		p.pos++
		target, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		expr = &UnOpExpr{Op: "?", Target: target, Token: td}
	case TokenLBracket:
		expr, err = p.parseArrayLiteral()
	case TokenHash:
		p.pos++
		name := p.consumeIdentifier()
		expr = &IdentExpr{Name: name, IsCore: true, Token: td}
	case TokenIdentifier:
		id := p.consumeIdentifier()
		if !p.isEof() && p.peek().Type == TokenAssign {
			p.consume(TokenAssign)
			val, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			return &AssignExpr{Name: id, Value: val, Token: td}, nil
		}
		expr = &IdentExpr{Name: id, IsCore: false, Token: td}
	case TokenString:
		p.pos++
		expr = &LiteralExpr{Value: Value{Type: TypeString, String: td.Literal}, Token: td}
	case TokenNumber:
		p.pos++
		val, _ := strconv.ParseFloat(td.Literal, 64)
		expr = &LiteralExpr{Value: Value{Type: TypeNumber, Number: val}, Token: td}
	default:
		return nil, p.error(fmt.Sprintf("Unexpected token: %v", td.Type))
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
			expr = &FieldExpr{Object: expr, FieldName: p.consumeIdentifier(), Token: td}
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

func (p *Parser) parseObjectLiteral() (Expr, error) {
	td := p.consume(TokenLBrace)
	fields := make(map[string]Expr)
	for !p.isEof() && p.peek().Type != TokenRBrace {
		p.skipNewlines()
		if p.peek().Type == TokenRBrace {
			break
		}
		key := p.consumeIdentifier()
		p.consume(TokenColon)
		val, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		fields[key] = val
		if !p.isEof() && p.peek().Type == TokenComma {
			p.consume(TokenComma)
		}
	}
	p.consume(TokenRBrace)
	return &ObjectExpr{Fields: fields, Token: td}, nil
}

func (p *Parser) parseArrayLiteral() (Expr, error) {
	td := p.consume(TokenLBracket)
	var items []Expr
	for !p.isEof() && p.peek().Type != TokenRBracket {
		p.skipNewlines()
		if p.peek().Type == TokenRBracket {
			break
		}
		item, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		if !p.isEof() && p.peek().Type == TokenComma {
			p.consume(TokenComma)
		}
	}
	p.consume(TokenRBracket)
	return &ArrayExpr{Items: items, Token: td}, nil
}

func (p *Parser) parseReturn() (Expr, error) {
	td := p.consume(TokenCaret)
	var val Expr
	if !p.isEof() && p.peek().Type != TokenNewline && p.peek().Type != TokenRBrace && p.peek().Type != TokenRBracket && p.peek().Type != TokenComma {
		var err error
		val, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}
	return &UnOpExpr{Op: "^", Target: val, Token: td}, nil
}

func (p *Parser) parseInclude() (Expr, error) {
	td := p.consume(TokenAt)
	var rawPath string
	if p.peek().Type == TokenString {
		rawPath = p.consume(TokenString).Literal
	} else {
		rawPath = p.consumeIdentifier()
	}

	content, ok := p.macroMap[rawPath]
	if !ok {
		return nil, p.error(fmt.Sprintf("Macro resource not found: @%s", rawPath))
	}

	base := filepath.Base(rawPath)
	taskName := strings.TrimSuffix(base, ".hank")

	subLexer := NewLexer(content)
	subTokens := subLexer.Tokenize()
	subParser := NewParser(subTokens, rawPath, p.macroMap)

	taskAst, err := subParser.Parse()
	if err != nil {
		return nil, err
	}

	return &AssignExpr{Name: taskName, Value: taskAst, Token: td}, nil
}

func (p *Parser) consumeIdentifier() string {
	td := p.peekTd()
	if td.Type != TokenIdentifier {
		panic(p.error(fmt.Sprintf("Expected identifier, found %v", td.Type)))
	}
	p.pos++
	return td.Literal
}

func (p *Parser) consume(t TokenType) Token {
	td := p.peekTd()
	if td.Type != t {
		panic(p.error(fmt.Sprintf("Expected %v, found %v", t, td.Type)))
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

func (p *Parser) error(msg string) error {
	td := p.peekTd()
	return fmt.Errorf("ERROR: %s in %s at\n\t%d:\t%s", msg, p.filename, td.Line, td.LineText)
}

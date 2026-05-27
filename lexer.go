package hank

import (
	"fmt"
	"unicode"
)

type TokenType int

const (
	TokenIdentifier TokenType = iota
	TokenNumber
	TokenString
	
	TokenAssign    // =
	TokenQuestion  // ?
	TokenColon     // :
	TokenRescue    // ~
	TokenAt        // @
	TokenHash      // #
	TokenNot       // !
	TokenCaret     // ^
	TokenDot       // .
	TokenComma     // ,
	
	TokenLParen    // (
	TokenRParen    // )
	TokenLBrace    // {
	TokenRBrace    // }
	TokenLBracket  // [
	TokenRBracket  // ]
	
	TokenNewline
	TokenEOF
	TokenError
)

type Token struct {
	Type     TokenType
	Literal  string
	Line     int
	LineText string
}

type Lexer struct {
	input       []rune
	pos         int
	line        int
	lineStart   int
	tokens      []Token
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: []rune(input),
		line:  1,
	}
}

func (l *Lexer) Tokenize() []Token {
	for l.pos < len(l.input) {
		char := l.input[l.pos]

		if unicode.IsSpace(char) {
			if char == '\n' {
				l.addToken(TokenNewline, "\n")
				l.line++
				l.pos++
				l.lineStart = l.pos
			} else {
				l.pos++
			}
			continue
		}

		if char == '/' && l.peek() == '/' {
			l.skipComment()
			continue
		}

		if char == '-' && unicode.IsDigit(l.peek()) {
			l.readNumber()
			continue
		}

		if unicode.IsDigit(char) {
			l.readNumber()
			continue
		}

		if unicode.IsLetter(char) || char == '_' {
			l.readIdentifier()
			continue
		}

		if char == '"' || char == '\'' {
			l.readString(char)
			continue
		}

		switch char {
		case '=': l.addToken(TokenAssign, "=")
		case '?': l.addToken(TokenQuestion, "?")
		case ':': l.addToken(TokenColon, ":")
		case '~': l.addToken(TokenRescue, "~")
		case '@': l.addToken(TokenAt, "@")
		case '#': l.addToken(TokenHash, "#")
		case '!': l.addToken(TokenNot, "!")
		case '^': l.addToken(TokenCaret, "^")
		case '.': l.addToken(TokenDot, ".")
		case ',': l.addToken(TokenComma, ",")
		case '(': l.addToken(TokenLParen, "(")
		case ')': l.addToken(TokenRParen, ")")
		case '{': l.addToken(TokenLBrace, "{")
		case '}': l.addToken(TokenRBrace, "}")
		case '[': l.addToken(TokenLBracket, "[")
		case ']': l.addToken(TokenRBracket, "]")
		default:
			l.addToken(TokenError, fmt.Sprintf("Unexpected character: %c", char))
		}
		l.pos++
	}
	l.addToken(TokenEOF, "")
	return l.tokens
}

func (l *Lexer) addToken(t TokenType, lit string) {
	l.tokens = append(l.tokens, Token{
		Type:     t,
		Literal:  lit,
		Line:     l.line,
		LineText: l.getCurrentLineText(),
	})
}

func (l *Lexer) peek() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return l.input[l.pos+1]
}

func (l *Lexer) skipComment() {
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.pos++
	}
}

func (l *Lexer) readNumber() {
	start := l.pos
	if l.input[l.pos] == '-' {
		l.pos++
	}
	for l.pos < len(l.input) && (unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '.') {
		l.pos++
	}
	l.addToken(TokenNumber, string(l.input[start:l.pos]))
}

func (l *Lexer) readIdentifier() {
	start := l.pos
	l.pos++
	for l.pos < len(l.input) && (unicode.IsLetter(l.input[l.pos]) || unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '_') {
		l.pos++
	}
	l.addToken(TokenIdentifier, string(l.input[start:l.pos]))
}

func (l *Lexer) readString(quote rune) {
	l.pos++ // skip quote
	var val string
	for l.pos < len(l.input) && l.input[l.pos] != quote {
		if l.input[l.pos] == '\\' {
			l.pos++
			if l.pos >= len(l.input) { break }
			switch l.input[l.pos] {
			case 'n': val += "\n"
			case 't': val += "\t"
			default: val += string(l.input[l.pos])
			}
		} else {
			val += string(l.input[l.pos])
		}
		l.pos++
	}
	if l.pos >= len(l.input) {
		l.addToken(TokenError, "Unclosed string literal")
		return
	}
	l.pos++ // skip quote
	l.addToken(TokenString, val)
}

func (l *Lexer) getCurrentLineText() string {
	end := l.pos
	for end < len(l.input) && l.input[end] != '\n' {
		end++
	}
	return string(l.input[l.lineStart:end])
}

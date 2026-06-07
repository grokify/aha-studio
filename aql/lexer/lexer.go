package lexer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/grokify/aha-studio/aql/ast"
)

// Lexer tokenizes an AQL query string.
type Lexer struct {
	input  string
	pos    int // current position in input (points to current char)
	start  int // start position of current token
	line   int // current line number (1-based)
	column int // current column number (1-based)
	tokens []Token
	err    error
}

// New creates a new lexer for the given input.
func New(input string) *Lexer {
	return &Lexer{
		input:  input,
		line:   1,
		column: 1,
	}
}

// Tokenize returns all tokens from the input.
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		tok := l.NextToken()
		l.tokens = append(l.tokens, tok)
		if tok.Type == TokenEOF || tok.Type == TokenError {
			break
		}
	}
	if l.err != nil {
		return nil, l.err
	}
	return l.tokens, nil
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Pos: l.position()}
	}

	l.start = l.pos
	ch := l.input[l.pos]

	// Single character tokens
	switch ch {
	case '(':
		return l.singleChar(TokenLParen)
	case ')':
		return l.singleChar(TokenRParen)
	case ',':
		return l.singleChar(TokenComma)
	case '.':
		return l.singleChar(TokenDot)
	case '*':
		return l.singleChar(TokenStar)
	case '+':
		return l.singleChar(TokenPlus)
	case '-':
		// Could be minus or negative number
		if l.pos+1 < len(l.input) && isDigit(l.input[l.pos+1]) {
			return l.scanNumber()
		}
		return l.singleChar(TokenMinus)
	case '=':
		return l.singleChar(TokenEQ)
	case '<':
		if l.peek() == '=' {
			return l.doubleChar(TokenLE)
		}
		if l.peek() == '>' {
			return l.doubleChar(TokenNE)
		}
		return l.singleChar(TokenLT)
	case '>':
		if l.peek() == '=' {
			return l.doubleChar(TokenGE)
		}
		return l.singleChar(TokenGT)
	case '!':
		if l.peek() == '=' {
			return l.doubleChar(TokenNE)
		}
		return l.error("unexpected character '!'")
	case '"', '\'':
		return l.scanString(ch)
	}

	// Numbers
	if isDigit(ch) {
		return l.scanNumber()
	}

	// Identifiers and keywords
	if isLetter(ch) || ch == '_' {
		return l.scanIdentifier()
	}

	return l.error(fmt.Sprintf("unexpected character '%c'", ch))
}

// singleChar returns a token for a single character.
func (l *Lexer) singleChar(typ TokenType) Token {
	tok := Token{
		Type:    typ,
		Literal: string(l.input[l.pos]),
		Pos:     l.position(),
	}
	l.advance()
	return tok
}

// doubleChar returns a token for two characters.
func (l *Lexer) doubleChar(typ TokenType) Token {
	tok := Token{
		Type:    typ,
		Literal: l.input[l.pos : l.pos+2],
		Pos:     l.position(),
	}
	l.advance()
	l.advance()
	return tok
}

// scanString scans a quoted string.
func (l *Lexer) scanString(quote byte) Token {
	pos := l.position()
	l.advance() // skip opening quote

	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == quote {
			l.advance() // skip closing quote
			return Token{
				Type:    TokenString,
				Literal: sb.String(),
				Pos:     pos,
			}
		}
		if ch == '\\' && l.pos+1 < len(l.input) {
			// Handle escape sequences
			l.advance()
			switch l.input[l.pos] {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '\'':
				sb.WriteByte('\'')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(l.input[l.pos])
			}
		} else if ch == '\n' {
			return l.error("unterminated string: newline in string")
		} else {
			sb.WriteByte(ch)
		}
		l.advance()
	}

	return l.error("unterminated string")
}

// scanNumber scans a number (integer, float, or duration).
func (l *Lexer) scanNumber() Token {
	pos := l.position()
	start := l.pos

	// Handle negative sign
	if l.pos < len(l.input) && l.input[l.pos] == '-' {
		l.advance()
	}

	// Scan digits
	for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
		l.advance()
	}

	// Check for decimal point
	isFloat := false
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		if l.pos+1 < len(l.input) && isDigit(l.input[l.pos+1]) {
			isFloat = true
			l.advance() // consume '.'
			for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
				l.advance()
			}
		}
	}

	// Check for duration suffix
	if l.pos < len(l.input) && isDurationSuffix(l.input[l.pos]) {
		suffix := l.input[l.pos]
		l.advance()
		// Allow 'ms' for milliseconds
		if suffix == 'm' && l.pos < len(l.input) && l.input[l.pos] == 's' {
			l.advance()
		}
		return Token{
			Type:    TokenDuration,
			Literal: l.input[start:l.pos],
			Pos:     pos,
		}
	}

	typ := TokenInt
	if isFloat {
		typ = TokenFloat
	}
	return Token{
		Type:    typ,
		Literal: l.input[start:l.pos],
		Pos:     pos,
	}
}

// scanIdentifier scans an identifier or keyword.
func (l *Lexer) scanIdentifier() Token {
	pos := l.position()
	start := l.pos

	for l.pos < len(l.input) && (isLetter(l.input[l.pos]) || isDigit(l.input[l.pos]) || l.input[l.pos] == '_') {
		l.advance()
	}

	literal := l.input[start:l.pos]
	typ := LookupIdent(literal)

	return Token{
		Type:    typ,
		Literal: literal,
		Pos:     pos,
	}
}

// skipWhitespace skips whitespace characters.
func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		switch l.input[l.pos] {
		case ' ', '\t', '\r':
			l.advance()
		case '\n':
			l.advance()
			l.line++
			l.column = 1
		default:
			return
		}
	}
}

// advance moves the position forward by one character.
func (l *Lexer) advance() {
	if l.pos < len(l.input) {
		if l.input[l.pos] == '\n' {
			l.line++
			l.column = 1
		} else {
			l.column++
		}
		l.pos++
	}
}

// peek returns the next character without advancing.
func (l *Lexer) peek() byte {
	if l.pos+1 < len(l.input) {
		return l.input[l.pos+1]
	}
	return 0
}

// position returns the current position.
func (l *Lexer) position() ast.Position {
	return ast.Position{
		Offset: l.pos,
		Line:   l.line,
		Column: l.column,
	}
}

// error creates an error token.
func (l *Lexer) error(msg string) Token {
	l.err = fmt.Errorf("line %d, column %d: %s", l.line, l.column, msg)
	return Token{
		Type:    TokenError,
		Literal: msg,
		Pos:     l.position(),
	}
}

// isLetter returns true if c is a letter.
func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isDigit returns true if c is a digit.
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// isDurationSuffix returns true if c is a valid duration suffix.
func isDurationSuffix(c byte) bool {
	return c == 'd' || c == 'h' || c == 'm' || c == 's' || c == 'w'
}

// IsWhitespace returns true if r is a whitespace character.
func IsWhitespace(r rune) bool {
	return unicode.IsSpace(r)
}

// Package lexer provides tokenization for AQL queries.
package lexer

import "github.com/grokify/aha-studio/aql/ast"

// TokenType represents the type of a token.
type TokenType int

// Token types.
const (
	TokenEOF TokenType = iota
	TokenError

	// Literals
	TokenIdent    // identifier (field names, entity names)
	TokenString   // "string" or 'string'
	TokenInt      // integer
	TokenFloat    // float
	TokenDuration // duration literal like 30d, 4h

	// Keywords
	TokenFROM
	TokenWHERE
	TokenSELECT
	TokenORDER
	TokenBY
	TokenLIMIT
	TokenASC
	TokenDESC
	TokenAND
	TokenOR
	TokenNOT
	TokenIN
	TokenIS
	TokenNULL
	TokenTRUE
	TokenFALSE
	TokenCONTAINS
	TokenLIKE

	// Aggregation and Grouping
	TokenGROUP
	TokenHAVING
	TokenAS
	TokenDISTINCT

	// Aggregate functions
	TokenCOUNT
	TokenSUM
	TokenAVG
	TokenMIN
	TokenMAX

	// JOINs
	TokenJOIN
	TokenLEFT
	TokenRIGHT
	TokenON

	// Mutations
	TokenINSERT
	TokenINTO
	TokenVALUES
	TokenUPDATE
	TokenSET
	TokenDELETE

	// Operators
	TokenEQ // =
	TokenNE // != or <>
	TokenLT // <
	TokenLE // <=
	TokenGT // >
	TokenGE // >=

	// Delimiters
	TokenLParen // (
	TokenRParen // )
	TokenComma  // ,
	TokenDot    // .
	TokenStar   // *
	TokenMinus  // -
	TokenPlus   // +
)

// String returns the string representation of the token type.
func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenError:
		return "ERROR"
	case TokenIdent:
		return "IDENT"
	case TokenString:
		return "STRING"
	case TokenInt:
		return "INT"
	case TokenFloat:
		return "FLOAT"
	case TokenDuration:
		return "DURATION"
	case TokenFROM:
		return "FROM"
	case TokenWHERE:
		return "WHERE"
	case TokenSELECT:
		return "SELECT"
	case TokenORDER:
		return "ORDER"
	case TokenBY:
		return "BY"
	case TokenLIMIT:
		return "LIMIT"
	case TokenASC:
		return "ASC"
	case TokenDESC:
		return "DESC"
	case TokenAND:
		return "AND"
	case TokenOR:
		return "OR"
	case TokenNOT:
		return "NOT"
	case TokenIN:
		return "IN"
	case TokenIS:
		return "IS"
	case TokenNULL:
		return "NULL"
	case TokenTRUE:
		return "TRUE"
	case TokenFALSE:
		return "FALSE"
	case TokenCONTAINS:
		return "CONTAINS"
	case TokenLIKE:
		return "LIKE"
	case TokenGROUP:
		return "GROUP"
	case TokenHAVING:
		return "HAVING"
	case TokenAS:
		return "AS"
	case TokenDISTINCT:
		return "DISTINCT"
	case TokenCOUNT:
		return "COUNT"
	case TokenSUM:
		return "SUM"
	case TokenAVG:
		return "AVG"
	case TokenMIN:
		return "MIN"
	case TokenMAX:
		return "MAX"
	case TokenJOIN:
		return "JOIN"
	case TokenLEFT:
		return "LEFT"
	case TokenRIGHT:
		return "RIGHT"
	case TokenON:
		return "ON"
	case TokenINSERT:
		return "INSERT"
	case TokenINTO:
		return "INTO"
	case TokenVALUES:
		return "VALUES"
	case TokenUPDATE:
		return "UPDATE"
	case TokenSET:
		return "SET"
	case TokenDELETE:
		return "DELETE"
	case TokenEQ:
		return "="
	case TokenNE:
		return "!="
	case TokenLT:
		return "<"
	case TokenLE:
		return "<="
	case TokenGT:
		return ">"
	case TokenGE:
		return ">="
	case TokenLParen:
		return "("
	case TokenRParen:
		return ")"
	case TokenComma:
		return ","
	case TokenDot:
		return "."
	case TokenStar:
		return "*"
	case TokenMinus:
		return "-"
	case TokenPlus:
		return "+"
	default:
		return "UNKNOWN"
	}
}

// Token represents a lexical token.
type Token struct {
	Type    TokenType
	Literal string
	Pos     ast.Position
}

// keywords maps keyword strings to token types.
var keywords = map[string]TokenType{
	"FROM":     TokenFROM,
	"WHERE":    TokenWHERE,
	"SELECT":   TokenSELECT,
	"ORDER":    TokenORDER,
	"BY":       TokenBY,
	"LIMIT":    TokenLIMIT,
	"ASC":      TokenASC,
	"DESC":     TokenDESC,
	"AND":      TokenAND,
	"OR":       TokenOR,
	"NOT":      TokenNOT,
	"IN":       TokenIN,
	"IS":       TokenIS,
	"NULL":     TokenNULL,
	"TRUE":     TokenTRUE,
	"FALSE":    TokenFALSE,
	"CONTAINS": TokenCONTAINS,
	"LIKE":     TokenLIKE,
	// Aggregation and Grouping
	"GROUP":    TokenGROUP,
	"HAVING":   TokenHAVING,
	"AS":       TokenAS,
	"DISTINCT": TokenDISTINCT,
	// Aggregate functions
	"COUNT": TokenCOUNT,
	"SUM":   TokenSUM,
	"AVG":   TokenAVG,
	"MIN":   TokenMIN,
	"MAX":   TokenMAX,
	// JOINs
	"JOIN":  TokenJOIN,
	"LEFT":  TokenLEFT,
	"RIGHT": TokenRIGHT,
	"ON":    TokenON,
	// Mutations
	"INSERT": TokenINSERT,
	"INTO":   TokenINTO,
	"VALUES": TokenVALUES,
	"UPDATE": TokenUPDATE,
	"SET":    TokenSET,
	"DELETE": TokenDELETE,
}

// LookupIdent returns the token type for an identifier.
// If the identifier is a keyword, it returns the keyword token type.
func LookupIdent(ident string) TokenType {
	// Convert to uppercase for case-insensitive keyword matching
	upper := toUpper(ident)
	if tok, ok := keywords[upper]; ok {
		return tok
	}
	return TokenIdent
}

// toUpper converts a string to uppercase (simple ASCII version).
func toUpper(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

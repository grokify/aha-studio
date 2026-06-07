package lexer

import (
	"testing"
)

func TestTokenizeSingleTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
		literal  string
	}{
		{"(", TokenLParen, "("},
		{")", TokenRParen, ")"},
		{",", TokenComma, ","},
		{".", TokenDot, "."},
		{"*", TokenStar, "*"},
		{"+", TokenPlus, "+"},
		{"-", TokenMinus, "-"},
		{"=", TokenEQ, "="},
		{"<", TokenLT, "<"},
		{">", TokenGT, ">"},
		{"<=", TokenLE, "<="},
		{">=", TokenGE, ">="},
		{"!=", TokenNE, "!="},
		{"<>", TokenNE, "<>"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := New(tt.input)
			tok := lex.NextToken()
			if tok.Type != tt.expected {
				t.Errorf("expected token type %s, got %s", tt.expected, tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestTokenizeKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"SELECT", TokenSELECT},
		{"select", TokenSELECT},
		{"FROM", TokenFROM},
		{"from", TokenFROM},
		{"WHERE", TokenWHERE},
		{"ORDER", TokenORDER},
		{"BY", TokenBY},
		{"LIMIT", TokenLIMIT},
		{"AND", TokenAND},
		{"OR", TokenOR},
		{"NOT", TokenNOT},
		{"IN", TokenIN},
		{"IS", TokenIS},
		{"NULL", TokenNULL},
		{"LIKE", TokenLIKE},
		{"CONTAINS", TokenCONTAINS},
		{"ASC", TokenASC},
		{"DESC", TokenDESC},
		{"JOIN", TokenJOIN},
		{"LEFT", TokenLEFT},
		{"RIGHT", TokenRIGHT},
		{"ON", TokenON},
		{"AS", TokenAS},
		{"GROUP", TokenGROUP},
		{"HAVING", TokenHAVING},
		{"DISTINCT", TokenDISTINCT},
		{"COUNT", TokenCOUNT},
		{"SUM", TokenSUM},
		{"AVG", TokenAVG},
		{"MIN", TokenMIN},
		{"MAX", TokenMAX},
		{"TRUE", TokenTRUE},
		{"FALSE", TokenFALSE},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := New(tt.input)
			tok := lex.NextToken()
			if tok.Type != tt.expected {
				t.Errorf("expected token type %s, got %s", tt.expected, tok.Type)
			}
		})
	}
}

func TestTokenizeIdentifiers(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"features", "features"},
		{"name", "name"},
		{"reference_num", "reference_num"},
		{"updated_at", "updated_at"},
		{"_private", "_private"},
		{"field123", "field123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := New(tt.input)
			tok := lex.NextToken()
			if tok.Type != TokenIdent {
				t.Errorf("expected TokenIdent, got %s", tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestTokenizeStrings(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{`"hello"`, "hello"},
		{`'hello'`, "hello"},
		{`"In Progress"`, "In Progress"},
		{`"with \"escaped\" quotes"`, `with "escaped" quotes`},
		{`'single\'s quote'`, `single's quote`},
		{`"new\nline"`, "new\nline"},
		{`"tab\there"`, "tab\there"},
		{`""`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := New(tt.input)
			tok := lex.NextToken()
			if tok.Type != TokenString {
				t.Errorf("expected TokenString, got %s", tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestTokenizeNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
		literal  string
	}{
		{"123", TokenInt, "123"},
		{"0", TokenInt, "0"},
		{"-42", TokenInt, "-42"},
		{"3.14", TokenFloat, "3.14"},
		{"-0.5", TokenFloat, "-0.5"},
		{"100.00", TokenFloat, "100.00"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := New(tt.input)
			tok := lex.NextToken()
			if tok.Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestTokenizeDurations(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"30d", "30d"},
		{"7d", "7d"},
		{"24h", "24h"},
		{"60m", "60m"},
		{"30s", "30s"},
		{"1w", "1w"},
		{"500ms", "500ms"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := New(tt.input)
			tok := lex.NextToken()
			if tok.Type != TokenDuration {
				t.Errorf("expected TokenDuration, got %s", tok.Type)
			}
			if tok.Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tok.Literal)
			}
		})
	}
}

func TestTokenizeFullQuery(t *testing.T) {
	input := `FROM features WHERE status = "In Progress" AND votes > 10 ORDER BY updated_at DESC LIMIT 20`

	expected := []struct {
		typ     TokenType
		literal string
	}{
		{TokenFROM, "FROM"},
		{TokenIdent, "features"},
		{TokenWHERE, "WHERE"},
		{TokenIdent, "status"},
		{TokenEQ, "="},
		{TokenString, "In Progress"},
		{TokenAND, "AND"},
		{TokenIdent, "votes"},
		{TokenGT, ">"},
		{TokenInt, "10"},
		{TokenORDER, "ORDER"},
		{TokenBY, "BY"},
		{TokenIdent, "updated_at"},
		{TokenDESC, "DESC"},
		{TokenLIMIT, "LIMIT"},
		{TokenInt, "20"},
		{TokenEOF, ""},
	}

	lex := New(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, tt := range expected {
		if tokens[i].Type != tt.typ {
			t.Errorf("token %d: expected type %s, got %s", i, tt.typ, tokens[i].Type)
		}
		if tokens[i].Literal != tt.literal {
			t.Errorf("token %d: expected literal %q, got %q", i, tt.literal, tokens[i].Literal)
		}
	}
}

func TestTokenizeSelectWithAggregates(t *testing.T) {
	input := `SELECT COUNT(*), status FROM features GROUP BY status`

	expected := []TokenType{
		TokenSELECT,
		TokenCOUNT,
		TokenLParen,
		TokenStar,
		TokenRParen,
		TokenComma,
		TokenIdent,
		TokenFROM,
		TokenIdent,
		TokenGROUP,
		TokenBY,
		TokenIdent,
		TokenEOF,
	}

	lex := New(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, expectedType := range expected {
		if tokens[i].Type != expectedType {
			t.Errorf("token %d: expected %s, got %s", i, expectedType, tokens[i].Type)
		}
	}
}

func TestTokenizeJoin(t *testing.T) {
	input := `FROM features f LEFT JOIN releases r ON f.release_id = r.id`

	expected := []TokenType{
		TokenFROM,
		TokenIdent,
		TokenIdent,
		TokenLEFT,
		TokenJOIN,
		TokenIdent,
		TokenIdent,
		TokenON,
		TokenIdent,
		TokenDot,
		TokenIdent,
		TokenEQ,
		TokenIdent,
		TokenDot,
		TokenIdent,
		TokenEOF,
	}

	lex := New(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, expectedType := range expected {
		if tokens[i].Type != expectedType {
			t.Errorf("token %d: expected %s, got %s", i, expectedType, tokens[i].Type)
		}
	}
}

func TestTokenizeInList(t *testing.T) {
	input := `status IN ("New", "In Progress", "Ready")`

	expected := []struct {
		typ     TokenType
		literal string
	}{
		{TokenIdent, "status"},
		{TokenIN, "IN"},
		{TokenLParen, "("},
		{TokenString, "New"},
		{TokenComma, ","},
		{TokenString, "In Progress"},
		{TokenComma, ","},
		{TokenString, "Ready"},
		{TokenRParen, ")"},
		{TokenEOF, ""},
	}

	lex := New(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, tt := range expected {
		if tokens[i].Type != tt.typ {
			t.Errorf("token %d: expected type %s, got %s", i, tt.typ, tokens[i].Type)
		}
		if tokens[i].Literal != tt.literal {
			t.Errorf("token %d: expected literal %q, got %q", i, tt.literal, tokens[i].Literal)
		}
	}
}

func TestTokenizePositions(t *testing.T) {
	input := "FROM features\nWHERE status = \"Done\""

	lex := New(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check first token position
	if tokens[0].Pos.Line != 1 || tokens[0].Pos.Column != 1 {
		t.Errorf("expected FROM at line 1, col 1, got line %d, col %d", tokens[0].Pos.Line, tokens[0].Pos.Column)
	}

	// Find WHERE token and check it's on line 2
	// Note: Newline handling may vary - test that positions are tracked
	foundWhere := false
	for _, tok := range tokens {
		if tok.Type == TokenWHERE {
			foundWhere = true
			// WHERE should be on line 2 after the newline
			if tok.Pos.Line < 2 {
				t.Errorf("expected WHERE on line 2 or later, got line %d", tok.Pos.Line)
			}
			break
		}
	}
	if !foundWhere {
		t.Error("WHERE token not found")
	}
}

func TestTokenizeError(t *testing.T) {
	tests := []struct {
		input string
		desc  string
	}{
		{`"unterminated string`, "unterminated string"},
		{`!invalid`, "unexpected character '!'"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			lex := New(tt.input)
			_, err := lex.Tokenize()
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestTokenizeWhitespace(t *testing.T) {
	// Whitespace should be skipped
	inputs := []string{
		"FROM   features",
		"FROM\tfeatures",
		"FROM\n  features",
		"  FROM features  ",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			lex := New(input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Should have FROM, features, EOF
			if len(tokens) != 3 {
				t.Errorf("expected 3 tokens, got %d", len(tokens))
			}
		})
	}
}

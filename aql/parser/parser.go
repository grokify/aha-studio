// Package parser provides a recursive descent parser for AQL queries.
package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/aql/lexer"
)

// Parser parses AQL queries into an AST.
type Parser struct {
	tokens []lexer.Token
	pos    int
}

// New creates a new parser for the given input.
func New(input string) *Parser {
	lex := lexer.New(input)
	tokens, _ := lex.Tokenize()
	return &Parser{tokens: tokens}
}

// ParseStatement parses the input and returns a Statement (Query or Mutation).
func (p *Parser) ParseStatement() (ast.Statement, error) {
	switch p.current().Type {
	case lexer.TokenINSERT:
		return p.parseInsertStatement()
	case lexer.TokenUPDATE:
		return p.parseUpdateStatement()
	case lexer.TokenDELETE:
		return p.parseDeleteStatement()
	default:
		// Default to query parsing
		return p.Parse()
	}
}

// Parse parses the input and returns a Query AST.
func (p *Parser) Parse() (*ast.Query, error) {
	query := &ast.Query{}

	// SELECT clause (optional, can appear first or after FROM)
	if p.check(lexer.TokenSELECT) {
		sel, err := p.parseSelectClause()
		if err != nil {
			return nil, err
		}
		query.Select = sel
	}

	// FROM clause (required)
	if !p.check(lexer.TokenFROM) {
		return nil, p.errorf("expected FROM clause")
	}
	from, err := p.parseFromClause()
	if err != nil {
		return nil, err
	}
	query.From = from

	// JOIN clauses (optional, multiple allowed)
	for p.check(lexer.TokenJOIN) || p.check(lexer.TokenLEFT) || p.check(lexer.TokenRIGHT) {
		join, err := p.parseJoinClause()
		if err != nil {
			return nil, err
		}
		query.Joins = append(query.Joins, join)
	}

	// WHERE clause (optional)
	if p.check(lexer.TokenWHERE) {
		where, err := p.parseWhereClause()
		if err != nil {
			return nil, err
		}
		query.Where = where
	}

	// SELECT clause (optional) - AQL supports SELECT after WHERE for flexibility
	if query.Select == nil && p.check(lexer.TokenSELECT) {
		sel, err := p.parseSelectClause()
		if err != nil {
			return nil, err
		}
		query.Select = sel
	}

	// GROUP BY clause (optional)
	if p.check(lexer.TokenGROUP) {
		groupBy, err := p.parseGroupByClause()
		if err != nil {
			return nil, err
		}
		query.GroupBy = groupBy
	}

	// HAVING clause (optional, requires GROUP BY)
	if p.check(lexer.TokenHAVING) {
		having, err := p.parseHavingClause()
		if err != nil {
			return nil, err
		}
		query.Having = having
	}

	// ORDER BY clause (optional)
	if p.check(lexer.TokenORDER) {
		orderBy, err := p.parseOrderByClause()
		if err != nil {
			return nil, err
		}
		query.OrderBy = orderBy
	}

	// LIMIT clause (optional)
	if p.check(lexer.TokenLIMIT) {
		limit, err := p.parseLimitClause()
		if err != nil {
			return nil, err
		}
		query.Limit = limit
	}

	// Ensure we've consumed all tokens
	if !p.isAtEnd() {
		tok := p.current()
		return nil, p.errorf("unexpected token %s at position %d", tok.Type, tok.Pos.Offset)
	}

	return query, nil
}

// parseFromClause parses: FROM <entity> [AS <alias>]
func (p *Parser) parseFromClause() (*ast.FromClause, error) {
	pos := p.current().Pos
	p.advance() // consume FROM

	if !p.check(lexer.TokenIdent) {
		return nil, p.errorf("expected entity name after FROM")
	}

	entityName := strings.ToLower(p.current().Literal)
	p.advance()

	entity := ast.EntityType(entityName)
	if !entity.IsValid() {
		return nil, p.errorf("invalid entity type '%s': expected features, ideas, releases, or initiatives", entityName)
	}

	from := &ast.FromClause{
		Entity: entity,
		Pos:    pos,
	}

	// Optional AS alias
	if p.check(lexer.TokenAS) {
		p.advance() // consume AS
		if !p.check(lexer.TokenIdent) {
			return nil, p.errorf("expected alias name after AS")
		}
		from.Alias = p.current().Literal
		p.advance()
	} else if p.check(lexer.TokenIdent) {
		// Allow alias without AS keyword (e.g., FROM features f)
		from.Alias = p.current().Literal
		p.advance()
	}

	return from, nil
}

// parseJoinClause parses: [LEFT|RIGHT] JOIN <entity> [AS <alias>] ON <condition>
func (p *Parser) parseJoinClause() (*ast.JoinClause, error) {
	pos := p.current().Pos
	joinType := ast.JoinInner

	// Check for LEFT or RIGHT
	if p.check(lexer.TokenLEFT) {
		joinType = ast.JoinLeft
		p.advance()
	} else if p.check(lexer.TokenRIGHT) {
		joinType = ast.JoinRight
		p.advance()
	}

	// Expect JOIN keyword
	if !p.check(lexer.TokenJOIN) {
		return nil, p.errorf("expected JOIN")
	}
	p.advance()

	// Parse entity name
	if !p.check(lexer.TokenIdent) {
		return nil, p.errorf("expected entity name after JOIN")
	}
	entityName := strings.ToLower(p.current().Literal)
	p.advance()

	entity := ast.EntityType(entityName)
	if !entity.IsValid() {
		return nil, p.errorf("invalid entity type '%s' in JOIN", entityName)
	}

	join := &ast.JoinClause{
		Type:   joinType,
		Entity: entity,
		Pos:    pos,
	}

	// Optional AS alias
	if p.check(lexer.TokenAS) {
		p.advance()
		if !p.check(lexer.TokenIdent) {
			return nil, p.errorf("expected alias name after AS")
		}
		join.Alias = p.current().Literal
		p.advance()
	} else if p.check(lexer.TokenIdent) && !p.check(lexer.TokenON) {
		// Allow alias without AS keyword
		join.Alias = p.current().Literal
		p.advance()
	}

	// Expect ON condition
	if !p.check(lexer.TokenON) {
		return nil, p.errorf("expected ON after JOIN entity")
	}
	p.advance()

	// Parse join condition
	condition, err := p.parseOrExpr()
	if err != nil {
		return nil, err
	}
	join.Condition = condition

	return join, nil
}

// parseGroupByClause parses: GROUP BY field1, field2, ...
func (p *Parser) parseGroupByClause() (*ast.GroupByClause, error) {
	pos := p.current().Pos
	p.advance() // consume GROUP

	if !p.check(lexer.TokenBY) {
		return nil, p.errorf("expected BY after GROUP")
	}
	p.advance()

	var fields []ast.FieldRef

	for {
		field, err := p.parseFieldRef()
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)

		if !p.check(lexer.TokenComma) {
			break
		}
		p.advance() // consume comma
	}

	return &ast.GroupByClause{
		Fields: fields,
		Pos:    pos,
	}, nil
}

// parseHavingClause parses: HAVING <expr>
func (p *Parser) parseHavingClause() (*ast.HavingClause, error) {
	pos := p.current().Pos
	p.advance() // consume HAVING

	expr, err := p.parseOrExpr()
	if err != nil {
		return nil, err
	}

	return &ast.HavingClause{
		Expr: expr,
		Pos:  pos,
	}, nil
}

// parseWhereClause parses: WHERE <expr>
func (p *Parser) parseWhereClause() (*ast.WhereClause, error) {
	pos := p.current().Pos
	p.advance() // consume WHERE

	expr, err := p.parseOrExpr()
	if err != nil {
		return nil, err
	}

	return &ast.WhereClause{
		Expr: expr,
		Pos:  pos,
	}, nil
}

// parseOrExpr parses OR expressions (lowest precedence).
func (p *Parser) parseOrExpr() (ast.Expr, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}

	for p.check(lexer.TokenOR) {
		pos := p.current().Pos
		p.advance() // consume OR

		right, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}

		left = ast.NewBinaryExpr(left, ast.OpOr, right, pos)
	}

	return left, nil
}

// parseAndExpr parses AND expressions.
func (p *Parser) parseAndExpr() (ast.Expr, error) {
	left, err := p.parseNotExpr()
	if err != nil {
		return nil, err
	}

	for p.check(lexer.TokenAND) {
		pos := p.current().Pos
		p.advance() // consume AND

		right, err := p.parseNotExpr()
		if err != nil {
			return nil, err
		}

		left = ast.NewBinaryExpr(left, ast.OpAnd, right, pos)
	}

	return left, nil
}

// parseNotExpr parses NOT expressions (highest precedence).
func (p *Parser) parseNotExpr() (ast.Expr, error) {
	if p.check(lexer.TokenNOT) {
		pos := p.current().Pos
		p.advance() // consume NOT

		expr, err := p.parseNotExpr()
		if err != nil {
			return nil, err
		}

		return ast.NewNotExpr(expr, pos), nil
	}

	return p.parsePrimaryExpr()
}

// parsePrimaryExpr parses primary expressions (comparisons, parenthesized).
func (p *Parser) parsePrimaryExpr() (ast.Expr, error) {
	// Parenthesized expression
	if p.check(lexer.TokenLParen) {
		pos := p.current().Pos
		p.advance() // consume (

		expr, err := p.parseOrExpr()
		if err != nil {
			return nil, err
		}

		if !p.check(lexer.TokenRParen) {
			return nil, p.errorf("expected ')' after expression")
		}
		p.advance() // consume )

		return ast.NewParenExpr(expr, pos), nil
	}

	// Comparison expression
	return p.parseCompareExpr()
}

// parseCompareExpr parses comparison expressions.
func (p *Parser) parseCompareExpr() (ast.Expr, error) {
	// Parse field reference
	field, err := p.parseFieldRef()
	if err != nil {
		return nil, err
	}
	pos := field.Pos

	// Parse operator
	op, err := p.parseCompareOp()
	if err != nil {
		return nil, err
	}

	// IS NULL / IS NOT NULL don't have a value
	if op == ast.OpIsNull || op == ast.OpIsNotNull {
		return ast.NewCompareExpr(field, op, nil, pos), nil
	}

	// Parse value
	value, err := p.parseValue(op)
	if err != nil {
		return nil, err
	}

	return ast.NewCompareExpr(field, op, value, pos), nil
}

// parseFieldRef parses a field reference (possibly qualified).
func (p *Parser) parseFieldRef() (ast.FieldRef, error) {
	if !p.check(lexer.TokenIdent) {
		return ast.FieldRef{}, p.errorf("expected field name")
	}

	ref := ast.FieldRef{
		Name: p.current().Literal,
		Pos:  p.current().Pos,
	}
	p.advance()

	// Check for qualified name (e.g., custom.field_name)
	if p.check(lexer.TokenDot) {
		p.advance() // consume .
		if !p.check(lexer.TokenIdent) {
			return ast.FieldRef{}, p.errorf("expected field name after '.'")
		}
		ref.Qualifier = ref.Name
		ref.Name = p.current().Literal
		p.advance()
	}

	return ref, nil
}

// parseCompareOp parses a comparison operator.
func (p *Parser) parseCompareOp() (ast.CompareOp, error) {
	tok := p.current()

	switch tok.Type {
	case lexer.TokenEQ:
		p.advance()
		return ast.OpEQ, nil
	case lexer.TokenNE:
		p.advance()
		return ast.OpNE, nil
	case lexer.TokenLT:
		p.advance()
		return ast.OpLT, nil
	case lexer.TokenLE:
		p.advance()
		return ast.OpLE, nil
	case lexer.TokenGT:
		p.advance()
		return ast.OpGT, nil
	case lexer.TokenGE:
		p.advance()
		return ast.OpGE, nil
	case lexer.TokenIN:
		p.advance()
		return ast.OpIN, nil
	case lexer.TokenCONTAINS:
		p.advance()
		return ast.OpContains, nil
	case lexer.TokenLIKE:
		p.advance()
		return ast.OpLike, nil
	case lexer.TokenIS:
		p.advance()
		// IS NULL or IS NOT NULL
		if p.check(lexer.TokenNOT) {
			p.advance()
			if !p.check(lexer.TokenNULL) {
				return "", p.errorf("expected NULL after IS NOT")
			}
			p.advance()
			return ast.OpIsNotNull, nil
		}
		if !p.check(lexer.TokenNULL) {
			return "", p.errorf("expected NULL after IS")
		}
		p.advance()
		return ast.OpIsNull, nil
	case lexer.TokenNOT:
		p.advance()
		if !p.check(lexer.TokenIN) {
			return "", p.errorf("expected IN after NOT")
		}
		p.advance()
		return ast.OpNotIn, nil
	default:
		return "", p.errorf("expected comparison operator, got %s", tok.Type)
	}
}

// parseValue parses a value based on the operator.
func (p *Parser) parseValue(op ast.CompareOp) (*ast.Value, error) {
	// IN/NOT IN expects a list or subquery
	if op == ast.OpIN || op == ast.OpNotIn {
		// Check for subquery: (SELECT ...)
		if p.check(lexer.TokenLParen) && p.peek().Type == lexer.TokenSELECT {
			return p.parseSubqueryValue()
		}
		return p.parseValueList()
	}

	// Check for subquery for scalar comparison: field > (SELECT AVG(...) ...)
	if p.check(lexer.TokenLParen) && p.peek().Type == lexer.TokenSELECT {
		return p.parseSubqueryValue()
	}

	return p.parseSingleValue()
}

// parseSubqueryValue parses a subquery as a value: (SELECT ...)
func (p *Parser) parseSubqueryValue() (*ast.Value, error) {
	if !p.check(lexer.TokenLParen) {
		return nil, p.errorf("expected '(' for subquery")
	}
	p.advance() // consume (

	// Parse the inner query
	subquery, err := p.parseSubquery()
	if err != nil {
		return nil, err
	}

	if !p.check(lexer.TokenRParen) {
		return nil, p.errorf("expected ')' after subquery")
	}
	p.advance() // consume )

	return &ast.Value{
		Type:     ast.ValueTypeSubquery,
		Raw:      "(SELECT ...)",
		Subquery: subquery,
	}, nil
}

// parseSubquery parses a nested query (inside parentheses).
func (p *Parser) parseSubquery() (*ast.Query, error) {
	query := &ast.Query{}

	// SELECT clause (required for subqueries)
	if !p.check(lexer.TokenSELECT) {
		return nil, p.errorf("expected SELECT in subquery")
	}
	sel, err := p.parseSelectClause()
	if err != nil {
		return nil, err
	}
	query.Select = sel

	// FROM clause (required)
	if !p.check(lexer.TokenFROM) {
		return nil, p.errorf("expected FROM clause in subquery")
	}
	from, err := p.parseFromClause()
	if err != nil {
		return nil, err
	}
	query.From = from

	// WHERE clause (optional)
	if p.check(lexer.TokenWHERE) {
		where, err := p.parseWhereClause()
		if err != nil {
			return nil, err
		}
		query.Where = where
	}

	// GROUP BY clause (optional)
	if p.check(lexer.TokenGROUP) {
		groupBy, err := p.parseGroupByClause()
		if err != nil {
			return nil, err
		}
		query.GroupBy = groupBy
	}

	// HAVING clause (optional)
	if p.check(lexer.TokenHAVING) {
		having, err := p.parseHavingClause()
		if err != nil {
			return nil, err
		}
		query.Having = having
	}

	// ORDER BY is not supported in subqueries (would be ignored anyway for scalar results)
	// LIMIT is also typically not supported in scalar subqueries

	return query, nil
}

// parseValueList parses a list of values for IN clauses: (val1, val2, ...)
func (p *Parser) parseValueList() (*ast.Value, error) {
	if !p.check(lexer.TokenLParen) {
		return nil, p.errorf("expected '(' for value list")
	}
	p.advance()

	var strings []string
	var rawParts []string

	for {
		val, err := p.parseSingleValue()
		if err != nil {
			return nil, err
		}
		strings = append(strings, val.String)
		rawParts = append(rawParts, val.Raw)

		if !p.check(lexer.TokenComma) {
			break
		}
		p.advance() // consume comma
	}

	if !p.check(lexer.TokenRParen) {
		return nil, p.errorf("expected ')' after value list")
	}
	p.advance()

	return &ast.Value{
		Type:    ast.ValueTypeStringList,
		Raw:     "(" + joinStrings(rawParts, ", ") + ")",
		Strings: strings,
	}, nil
}

// parseSingleValue parses a single value.
func (p *Parser) parseSingleValue() (*ast.Value, error) {
	tok := p.current()

	switch tok.Type {
	case lexer.TokenString:
		p.advance()
		return &ast.Value{
			Type:   ast.ValueTypeString,
			Raw:    "\"" + tok.Literal + "\"",
			String: tok.Literal,
		}, nil

	case lexer.TokenInt:
		p.advance()
		n, err := strconv.ParseInt(tok.Literal, 10, 64)
		if err != nil {
			return nil, p.errorf("invalid integer: %s", tok.Literal)
		}
		return &ast.Value{
			Type: ast.ValueTypeInt,
			Raw:  tok.Literal,
			Int:  n,
		}, nil

	case lexer.TokenFloat:
		p.advance()
		f, err := strconv.ParseFloat(tok.Literal, 64)
		if err != nil {
			return nil, p.errorf("invalid float: %s", tok.Literal)
		}
		return &ast.Value{
			Type:  ast.ValueTypeFloat,
			Raw:   tok.Literal,
			Float: f,
		}, nil

	case lexer.TokenDuration:
		p.advance()
		d, err := parseDuration(tok.Literal)
		if err != nil {
			return nil, p.errorf("invalid duration: %s", tok.Literal)
		}
		return &ast.Value{
			Type:     ast.ValueTypeDuration,
			Raw:      tok.Literal,
			Duration: d,
		}, nil

	case lexer.TokenTRUE:
		p.advance()
		return &ast.Value{
			Type: ast.ValueTypeBool,
			Raw:  "true",
			Bool: true,
		}, nil

	case lexer.TokenFALSE:
		p.advance()
		return &ast.Value{
			Type: ast.ValueTypeBool,
			Raw:  "false",
			Bool: false,
		}, nil

	case lexer.TokenNULL:
		p.advance()
		return &ast.Value{
			Type: ast.ValueTypeNull,
			Raw:  "null",
		}, nil

	case lexer.TokenIdent:
		// Could be a function call like now() or duration("30d")
		if p.peek().Type == lexer.TokenLParen {
			return p.parseFunctionCall()
		}
		// Otherwise treat as string literal (for backwards compatibility)
		p.advance()
		return &ast.Value{
			Type:   ast.ValueTypeString,
			Raw:    tok.Literal,
			String: tok.Literal,
		}, nil

	default:
		return nil, p.errorf("expected value, got %s", tok.Type)
	}
}

// parseFunctionCall parses a function call like now() or duration("30d").
func (p *Parser) parseFunctionCall() (*ast.Value, error) {
	funcName := strings.ToLower(p.current().Literal)
	p.advance() // consume function name
	p.advance() // consume (

	switch funcName {
	case "now":
		if !p.check(lexer.TokenRParen) {
			return nil, p.errorf("now() takes no arguments")
		}
		p.advance()
		now := time.Now()
		return &ast.Value{
			Type: ast.ValueTypeTime,
			Raw:  "now()",
			Time: now,
		}, nil

	case "duration":
		if !p.check(lexer.TokenString) && !p.check(lexer.TokenDuration) {
			return nil, p.errorf("duration() expects a string argument")
		}
		durStr := p.current().Literal
		p.advance()

		if !p.check(lexer.TokenRParen) {
			return nil, p.errorf("expected ')' after duration argument")
		}
		p.advance()

		d, err := parseDuration(durStr)
		if err != nil {
			return nil, p.errorf("invalid duration: %s", durStr)
		}
		return &ast.Value{
			Type:     ast.ValueTypeDuration,
			Raw:      fmt.Sprintf("duration(\"%s\")", durStr),
			Duration: d,
		}, nil

	default:
		return nil, p.errorf("unknown function: %s", funcName)
	}
}

// parseSelectClause parses: SELECT [DISTINCT] item1, item2, ...
// where item can be: *, field, field AS alias, aggregate_func(...) [AS alias]
func (p *Parser) parseSelectClause() (*ast.SelectClause, error) {
	pos := p.current().Pos
	p.advance() // consume SELECT

	sel := &ast.SelectClause{Pos: pos}

	// Check for DISTINCT
	if p.check(lexer.TokenDISTINCT) {
		sel.Distinct = true
		p.advance()
	}

	// Handle SELECT * or SELECT DISTINCT *
	if p.check(lexer.TokenStar) {
		p.advance()
		sel.Items = []ast.SelectItem{{Star: true, Pos: pos}}
		return sel, nil
	}

	for {
		item, err := p.parseSelectItem()
		if err != nil {
			return nil, err
		}
		sel.Items = append(sel.Items, item)

		if !p.check(lexer.TokenComma) {
			break
		}
		p.advance() // consume comma
	}

	return sel, nil
}

// parseSelectItem parses a single select item (field, aggregate, or *)
func (p *Parser) parseSelectItem() (ast.SelectItem, error) {
	pos := p.current().Pos
	item := ast.SelectItem{Pos: pos}

	// Check for * (star)
	if p.check(lexer.TokenStar) {
		p.advance()
		item.Star = true
		return item, nil
	}

	// Check for aggregate functions
	if p.isAggregateFunc() {
		agg, err := p.parseAggregateFunc()
		if err != nil {
			return item, err
		}
		item.Aggregate = agg
	} else {
		// Regular field reference
		field, err := p.parseFieldRef()
		if err != nil {
			return item, err
		}
		item.Field = &field
	}

	// Check for AS alias
	if p.check(lexer.TokenAS) {
		p.advance()
		if !p.check(lexer.TokenIdent) {
			return item, p.errorf("expected alias name after AS")
		}
		item.Alias = p.current().Literal
		p.advance()
	}

	return item, nil
}

// isAggregateFunc returns true if the current token is an aggregate function.
func (p *Parser) isAggregateFunc() bool {
	switch p.current().Type {
	case lexer.TokenCOUNT, lexer.TokenSUM, lexer.TokenAVG, lexer.TokenMIN, lexer.TokenMAX:
		return true
	}
	return false
}

// parseAggregateFunc parses an aggregate function call.
func (p *Parser) parseAggregateFunc() (*ast.AggregateFunc, error) {
	pos := p.current().Pos
	agg := &ast.AggregateFunc{Pos: pos}

	// Get aggregate type
	switch p.current().Type {
	case lexer.TokenCOUNT:
		agg.Func = ast.AggCount
	case lexer.TokenSUM:
		agg.Func = ast.AggSum
	case lexer.TokenAVG:
		agg.Func = ast.AggAvg
	case lexer.TokenMIN:
		agg.Func = ast.AggMin
	case lexer.TokenMAX:
		agg.Func = ast.AggMax
	default:
		return nil, p.errorf("expected aggregate function")
	}
	p.advance()

	// Expect (
	if !p.check(lexer.TokenLParen) {
		return nil, p.errorf("expected '(' after %s", agg.Func)
	}
	p.advance()

	// Check for DISTINCT
	if p.check(lexer.TokenDISTINCT) {
		agg.Distinct = true
		p.advance()
	}

	// Check for * (COUNT(*))
	if p.check(lexer.TokenStar) {
		p.advance()
		// Only COUNT can use *
		if agg.Func != ast.AggCount {
			return nil, p.errorf("%s cannot use *, only COUNT(*) is allowed", agg.Func)
		}
		agg.Field = nil // nil field means *
	} else {
		// Parse field reference
		field, err := p.parseFieldRef()
		if err != nil {
			return nil, err
		}
		agg.Field = &field
	}

	// Expect )
	if !p.check(lexer.TokenRParen) {
		return nil, p.errorf("expected ')' after aggregate argument")
	}
	p.advance()

	return agg, nil
}

// parseOrderByClause parses: ORDER BY field [ASC|DESC]
func (p *Parser) parseOrderByClause() (*ast.OrderByClause, error) {
	pos := p.current().Pos
	p.advance() // consume ORDER

	if !p.check(lexer.TokenBY) {
		return nil, p.errorf("expected BY after ORDER")
	}
	p.advance()

	field, err := p.parseFieldRef()
	if err != nil {
		return nil, err
	}

	dir := ast.SortAsc // default
	if p.check(lexer.TokenASC) {
		p.advance()
		dir = ast.SortAsc
	} else if p.check(lexer.TokenDESC) {
		p.advance()
		dir = ast.SortDesc
	}

	return &ast.OrderByClause{
		Field: field,
		Dir:   dir,
		Pos:   pos,
	}, nil
}

// parseLimitClause parses: LIMIT n
func (p *Parser) parseLimitClause() (*int, error) {
	p.advance() // consume LIMIT

	if !p.check(lexer.TokenInt) {
		return nil, p.errorf("expected integer after LIMIT")
	}

	n, err := strconv.Atoi(p.current().Literal)
	if err != nil {
		return nil, p.errorf("invalid LIMIT value: %s", p.current().Literal)
	}
	p.advance()

	return &n, nil
}

// Helper methods

func (p *Parser) current() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek() lexer.Token {
	if p.pos+1 >= len(p.tokens) {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) check(typ lexer.TokenType) bool {
	return p.current().Type == typ
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) isAtEnd() bool {
	return p.current().Type == lexer.TokenEOF
}

func (p *Parser) errorf(format string, args ...any) error {
	tok := p.current()
	return fmt.Errorf("parse error at line %d, column %d: %s",
		tok.Pos.Line, tok.Pos.Column, fmt.Sprintf(format, args...))
}

// parseDuration parses a duration string like "30d", "4h", "1w".
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	// Find where the number ends
	numEnd := 0
	for numEnd < len(s) && (s[numEnd] >= '0' && s[numEnd] <= '9') {
		numEnd++
	}

	if numEnd == 0 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	num, err := strconv.ParseInt(s[:numEnd], 10, 64)
	if err != nil {
		return 0, err
	}

	unit := s[numEnd:]
	switch unit {
	case "ms":
		return time.Duration(num) * time.Millisecond, nil
	case "s":
		return time.Duration(num) * time.Second, nil
	case "m":
		return time.Duration(num) * time.Minute, nil
	case "h":
		return time.Duration(num) * time.Hour, nil
	case "d":
		return time.Duration(num) * 24 * time.Hour, nil
	case "w":
		return time.Duration(num) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}

// joinStrings joins strings with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// parseInsertStatement parses: INSERT INTO entity (col1, col2) VALUES (val1, val2)
func (p *Parser) parseInsertStatement() (*ast.InsertStatement, error) {
	pos := p.current().Pos
	p.advance() // consume INSERT

	if !p.check(lexer.TokenINTO) {
		return nil, p.errorf("expected INTO after INSERT")
	}
	p.advance() // consume INTO

	// Parse entity name
	if !p.check(lexer.TokenIdent) {
		return nil, p.errorf("expected entity name after INTO")
	}
	entityName := strings.ToLower(p.current().Literal)
	p.advance()

	entity := ast.EntityType(entityName)
	if !entity.IsValid() {
		return nil, p.errorf("invalid entity type '%s'", entityName)
	}

	stmt := &ast.InsertStatement{
		Entity: entity,
		Pos:    pos,
	}

	// Parse column list: (col1, col2, ...)
	if !p.check(lexer.TokenLParen) {
		return nil, p.errorf("expected '(' for column list")
	}
	p.advance()

	for {
		if !p.check(lexer.TokenIdent) {
			return nil, p.errorf("expected column name")
		}
		stmt.Columns = append(stmt.Columns, p.current().Literal)
		p.advance()

		if !p.check(lexer.TokenComma) {
			break
		}
		p.advance() // consume comma
	}

	if !p.check(lexer.TokenRParen) {
		return nil, p.errorf("expected ')' after column list")
	}
	p.advance()

	// Parse VALUES
	if !p.check(lexer.TokenVALUES) {
		return nil, p.errorf("expected VALUES")
	}
	p.advance()

	// Parse value list: (val1, val2, ...)
	if !p.check(lexer.TokenLParen) {
		return nil, p.errorf("expected '(' for value list")
	}
	p.advance()

	for {
		val, err := p.parseSingleValue()
		if err != nil {
			return nil, err
		}
		stmt.Values = append(stmt.Values, *val)

		if !p.check(lexer.TokenComma) {
			break
		}
		p.advance() // consume comma
	}

	if !p.check(lexer.TokenRParen) {
		return nil, p.errorf("expected ')' after value list")
	}
	p.advance()

	// Validate column count matches value count
	if len(stmt.Columns) != len(stmt.Values) {
		return nil, p.errorf("column count (%d) doesn't match value count (%d)",
			len(stmt.Columns), len(stmt.Values))
	}

	return stmt, nil
}

// parseUpdateStatement parses: UPDATE entity SET col1 = val1, col2 = val2 [WHERE ...]
func (p *Parser) parseUpdateStatement() (*ast.UpdateStatement, error) {
	pos := p.current().Pos
	p.advance() // consume UPDATE

	// Parse entity name
	if !p.check(lexer.TokenIdent) {
		return nil, p.errorf("expected entity name after UPDATE")
	}
	entityName := strings.ToLower(p.current().Literal)
	p.advance()

	entity := ast.EntityType(entityName)
	if !entity.IsValid() {
		return nil, p.errorf("invalid entity type '%s'", entityName)
	}

	stmt := &ast.UpdateStatement{
		Entity: entity,
		Pos:    pos,
	}

	// Parse SET
	if !p.check(lexer.TokenSET) {
		return nil, p.errorf("expected SET after entity name")
	}
	p.advance()

	// Parse assignments: col1 = val1, col2 = val2, ...
	for {
		if !p.check(lexer.TokenIdent) {
			return nil, p.errorf("expected column name in SET clause")
		}
		fieldName := p.current().Literal
		fieldPos := p.current().Pos
		p.advance()

		if !p.check(lexer.TokenEQ) {
			return nil, p.errorf("expected '=' after column name")
		}
		p.advance()

		val, err := p.parseSingleValue()
		if err != nil {
			return nil, err
		}

		stmt.Assignments = append(stmt.Assignments, ast.Assignment{
			Field: fieldName,
			Value: *val,
			Pos:   fieldPos,
		})

		if !p.check(lexer.TokenComma) {
			break
		}
		p.advance() // consume comma
	}

	// Optional WHERE clause
	if p.check(lexer.TokenWHERE) {
		where, err := p.parseWhereClause()
		if err != nil {
			return nil, err
		}
		stmt.Where = where
	}

	return stmt, nil
}

// parseDeleteStatement parses: DELETE FROM entity [WHERE ...]
func (p *Parser) parseDeleteStatement() (*ast.DeleteStatement, error) {
	pos := p.current().Pos
	p.advance() // consume DELETE

	if !p.check(lexer.TokenFROM) {
		return nil, p.errorf("expected FROM after DELETE")
	}
	p.advance() // consume FROM

	// Parse entity name
	if !p.check(lexer.TokenIdent) {
		return nil, p.errorf("expected entity name after FROM")
	}
	entityName := strings.ToLower(p.current().Literal)
	p.advance()

	entity := ast.EntityType(entityName)
	if !entity.IsValid() {
		return nil, p.errorf("invalid entity type '%s'", entityName)
	}

	stmt := &ast.DeleteStatement{
		Entity: entity,
		Pos:    pos,
	}

	// WHERE clause is required for DELETE (safety)
	if !p.check(lexer.TokenWHERE) {
		return nil, p.errorf("WHERE clause is required for DELETE (for safety)")
	}
	where, err := p.parseWhereClause()
	if err != nil {
		return nil, err
	}
	stmt.Where = where

	return stmt, nil
}

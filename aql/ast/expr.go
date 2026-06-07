package ast

import "fmt"

// Expr is the interface for all expression nodes.
type Expr interface {
	exprNode()
	Pos() Position
	String() string
}

// BinaryExpr represents a binary logical expression (AND/OR).
type BinaryExpr struct {
	Left  Expr
	Op    LogicalOp
	Right Expr
	pos   Position
}

func (e *BinaryExpr) exprNode()     {}
func (e *BinaryExpr) Pos() Position { return e.pos }
func (e *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", e.Left.String(), e.Op, e.Right.String())
}

// NewBinaryExpr creates a new binary expression.
func NewBinaryExpr(left Expr, op LogicalOp, right Expr, pos Position) *BinaryExpr {
	return &BinaryExpr{Left: left, Op: op, Right: right, pos: pos}
}

// NotExpr represents a NOT expression.
type NotExpr struct {
	Expr Expr
	pos  Position
}

func (e *NotExpr) exprNode()     {}
func (e *NotExpr) Pos() Position { return e.pos }
func (e *NotExpr) String() string {
	return fmt.Sprintf("NOT %s", e.Expr.String())
}

// NewNotExpr creates a new NOT expression.
func NewNotExpr(expr Expr, pos Position) *NotExpr {
	return &NotExpr{Expr: expr, pos: pos}
}

// CompareExpr represents a comparison expression.
type CompareExpr struct {
	Field FieldRef
	Op    CompareOp
	Value *Value // nil for IS NULL/IS NOT NULL
	pos   Position
}

func (e *CompareExpr) exprNode()     {}
func (e *CompareExpr) Pos() Position { return e.pos }
func (e *CompareExpr) String() string {
	if e.Value == nil {
		return fmt.Sprintf("%s %s", e.Field.String(), e.Op)
	}
	return fmt.Sprintf("%s %s %s", e.Field.String(), e.Op, e.Value.Raw)
}

// NewCompareExpr creates a new comparison expression.
func NewCompareExpr(field FieldRef, op CompareOp, value *Value, pos Position) *CompareExpr {
	return &CompareExpr{Field: field, Op: op, Value: value, pos: pos}
}

// ParenExpr represents a parenthesized expression.
type ParenExpr struct {
	Expr Expr
	pos  Position
}

func (e *ParenExpr) exprNode()     {}
func (e *ParenExpr) Pos() Position { return e.pos }
func (e *ParenExpr) String() string {
	return fmt.Sprintf("(%s)", e.Expr.String())
}

// NewParenExpr creates a new parenthesized expression.
func NewParenExpr(expr Expr, pos Position) *ParenExpr {
	return &ParenExpr{Expr: expr, pos: pos}
}

// FunctionExpr represents a function call in an expression.
type FunctionExpr struct {
	Name string
	Args []Expr
	pos  Position
}

func (e *FunctionExpr) exprNode()     {}
func (e *FunctionExpr) Pos() Position { return e.pos }
func (e *FunctionExpr) String() string {
	return fmt.Sprintf("%s(...)", e.Name)
}

// NewFunctionExpr creates a new function expression.
func NewFunctionExpr(name string, args []Expr, pos Position) *FunctionExpr {
	return &FunctionExpr{Name: name, Args: args, pos: pos}
}

// LiteralExpr represents a literal value expression.
type LiteralExpr struct {
	Value Value
	pos   Position
}

func (e *LiteralExpr) exprNode()     {}
func (e *LiteralExpr) Pos() Position { return e.pos }
func (e *LiteralExpr) String() string {
	return e.Value.Raw
}

// NewLiteralExpr creates a new literal expression.
func NewLiteralExpr(value Value, pos Position) *LiteralExpr {
	return &LiteralExpr{Value: value, pos: pos}
}

// SubqueryExpr represents a subquery expression (a nested query).
type SubqueryExpr struct {
	Query *Query
	pos   Position
}

func (e *SubqueryExpr) exprNode()     {}
func (e *SubqueryExpr) Pos() Position { return e.pos }
func (e *SubqueryExpr) String() string {
	return "(SELECT ...)"
}

// NewSubqueryExpr creates a new subquery expression.
func NewSubqueryExpr(query *Query, pos Position) *SubqueryExpr {
	return &SubqueryExpr{Query: query, pos: pos}
}

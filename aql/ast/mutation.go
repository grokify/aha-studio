package ast

// Statement represents any AQL statement (query or mutation).
type Statement interface {
	statementNode()
}

// Ensure Query implements Statement
func (q *Query) statementNode() {}

// InsertStatement represents an INSERT statement.
type InsertStatement struct {
	Entity  EntityType
	Columns []string
	Values  []Value
	Pos     Position
}

func (s *InsertStatement) statementNode() {}

// UpdateStatement represents an UPDATE statement.
type UpdateStatement struct {
	Entity      EntityType
	Assignments []Assignment
	Where       *WhereClause
	Pos         Position
}

func (s *UpdateStatement) statementNode() {}

// Assignment represents a field = value assignment in UPDATE.
type Assignment struct {
	Field string
	Value Value
	Pos   Position
}

// DeleteStatement represents a DELETE statement.
type DeleteStatement struct {
	Entity EntityType
	Where  *WhereClause
	Pos    Position
}

func (s *DeleteStatement) statementNode() {}

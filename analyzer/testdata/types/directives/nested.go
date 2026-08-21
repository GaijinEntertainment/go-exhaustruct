package directives

// A use-site directive is written above a statement, not above every literal
// inside it, so its scope covers the literals nested within that statement.

type NestedInner struct{ X, Y int }

type NestedOuter struct {
	I NestedInner
	Z int
}

func nestedConsume(NestedInner) {}

func shouldPassIgnoreCoversNestedFieldLiteral() {
	//exhaustruct:ignore
	_ = NestedOuter{
		I: NestedInner{},
		Z: 1,
	}
}

func shouldPassIgnoreCoversSliceElements() {
	//exhaustruct:ignore
	_ = []NestedInner{
		{X: 1},
		{},
	}
}

func shouldPassIgnoreCoversMapValues() {
	//exhaustruct:ignore
	_ = map[string]NestedInner{
		"a": {X: 1},
	}
}

func shouldPassIgnoreCoversCallArgument() {
	//exhaustruct:ignore
	nestedConsume(NestedInner{
		X: 1,
	})
}

func shouldPassIgnoreCoversReturn() NestedInner {
	//exhaustruct:ignore
	return NestedInner{
		X: 1,
	}
}

func shouldPassIgnoreCoversVarDeclaration() {
	//exhaustruct:ignore
	var _ = NestedOuter{
		I: NestedInner{},
		Z: 1,
	}
}

func shouldPassIgnoreCoversAddressOf() {
	//exhaustruct:ignore
	_ = &NestedInner{
		X: 1,
	}
}

func shouldFailNestedWithoutDirective() {
	_ = NestedInner{ // want "directives.NestedInner is missing field Y"
		X: 1,
	}
}

// The cases below place the literal on a line of its own, below the statement
// carrying the directive, so the walk has to climb past the node between them.
// gofmt joins these onto one line, which is why they are written out by hand:
// the directive then resolves at the literal itself and proves nothing.

func shouldPassIgnoreCoversMultilineCallArgument() {
	//exhaustruct:ignore
	nestedConsume(
		NestedInner{X: 1},
	)
}

func shouldPassIgnoreCoversParenthesized() {
	//exhaustruct:ignore
	_ = (NestedInner{X: 1})
}

func shouldPassIgnoreCoversIndexed(m map[NestedInner]int) {
	//exhaustruct:ignore
	_ = m[NestedInner{X: 1}]
}

func shouldPassIgnoreCoversSend(ch chan NestedInner) {
	//exhaustruct:ignore
	ch <- NestedInner{X: 1}
}

// NestedExcluded is ignored by pattern, so only an enforce directive on the
// statement brings the literals inside it back under the check.
type NestedExcluded struct{ X, Y int }

func shouldPassExcludedWithoutDirective() {
	_ = NestedOuter{
		I: NestedInner{X: 1, Y: 2},
		Z: 1,
	}
	_ = []NestedExcluded{
		{X: 1},
	}
}

func shouldFailEnforceCoversNestedLiterals() {
	//exhaustruct:enforce
	_ = []NestedExcluded{
		{X: 1}, // want "directives.NestedExcluded is missing field Y"
	}
}

func shouldPassIgnoreCoversMultilineAssign() {
	//exhaustruct:ignore
	_ =
		NestedInner{X: 1}
}

func shouldPassIgnoreCoversMultilineAddressOf() {
	//exhaustruct:ignore
	_ = &NestedInner{X: 1}
}

func shouldPassIgnoreCoversMultilineVarDecl() {
	//exhaustruct:ignore
	var _ = NestedInner{X: 1}
}

func shouldPassIgnoreCoversMultilineSelector() {
	//exhaustruct:ignore
	_ =
		(NestedInner{X: 1}).X
}

func shouldPassIgnoreCoversGoStatement() {
	//exhaustruct:ignore
	go nestedConsume(NestedInner{X: 1})
}

func shouldPassIgnoreCoversDeferStatement() {
	//exhaustruct:ignore
	defer nestedConsume(NestedInner{X: 1})
}

func shouldPassIgnoreCoversTrailingReturnResult() (int, NestedInner) {
	//exhaustruct:ignore
	return 1,
		NestedInner{X: 1}
}

func shouldPassIgnoreCoversVarBlock() {
	//exhaustruct:ignore
	var (
		_ = NestedInner{X: 1}
	)
}

func shouldPassIgnoreCoversMultilineIfCondition() {
	//exhaustruct:ignore
	if nestedEquals(
		NestedInner{X: 1},
	) {
		return
	}
}

func shouldPassIgnoreCoversMultilineBinaryExpr() {
	//exhaustruct:ignore
	_ = NestedInner{X: 1} ==
		NestedInner{X: 1}
}

func shouldPassIgnoreCoversMultilineTypeAssert() {
	//exhaustruct:ignore
	_ = any(
		NestedInner{X: 1},
	).(NestedInner)
}

func shouldPassIgnoreCoversMultilineSwitchTag() {
	//exhaustruct:ignore
	switch nestedIdentity(
		NestedInner{X: 1},
	) {
	default:
	}
}

// A directive on a statement must not reach literals in the block below it:
// those are statements of their own.
func shouldFailDirectiveDoesNotEnterBlock() {
	//exhaustruct:ignore
	if true {
		_ = NestedInner{X: 1} // want "directives.NestedInner is missing field Y"
	}
}

// A type assertion carries the position of the operand it asserts on, so a
// directive is never resolved at the assertion itself. What the walk climbs
// through it for is the statement above, which is where the directive is
// written.
func shouldPassIgnoreCoversLiteralUnderTypeAssert() {
	//exhaustruct:ignore
	_ =
		any(NestedInner{X: 1}).(NestedInner)
}

func nestedEquals(NestedInner) bool            { return true }
func nestedIdentity(v NestedInner) NestedInner { return v }

// An `if` initializer is a statement of its own. Written on the line the
// directive targets it is covered, and gofmt writes it onto that line in any
// case.
func shouldPassIgnoreCoversOneLineIfInitializer() {
	//exhaustruct:ignore
	if v := (NestedInner{X: 1}); v.X > 0 {
		_ = v
	}
}

// A label is a statement of its own, and the statement it names begins on
// another line. The directive above the label answers for the label.
func shouldFailDirectiveAboveLabel() {
	var v NestedInner

	//exhaustruct:ignore
above:
	v = NestedInner{X: 1} // want "directives.NestedInner is missing field Y"

	if v.X < 0 {
		goto above
	}

	_ = v
}

// The statement a label names reads the directive written above itself.
func shouldPassIgnoreAboveLabeledStatement() {
	var v NestedInner

below:
	//exhaustruct:ignore
	v = NestedInner{X: 1}

	if v.X < 0 {
		goto below
	}

	_ = v
}

// Package paren_tags covers a struct written inside parentheses. The
// parentheses stand between the declaration and the type, and the type is
// named all the same, so a pattern naming it still selects it.
package paren_tags

type ExcludedParen (struct {
	Field string `exhaustruct:"optional"` // want `struct tag "exhaustruct" is not supported anymore`
})

type ReportedParen (struct {
	Field string `exhaustruct:"optional"` // want `struct tag "exhaustruct" is not supported anymore`
})

// Issue #168: Anonymous struct keys in map literals.
// https://github.com/GaijinEntertainment/go-exhaustruct/issues/168
//
// Both the key and the value of a map literal may elide their type, so their
// positions must be resolved from opposite sides of the map type. Resolving a
// key from the map's value type poisons the position-keyed struct cache: the
// value literal then inherits the key's field list, which either reports the
// key's fields against the value or panics when the key has fewer fields.
package collections

func shouldPassAnonymousStructMapKey() {
	_ = map[struct{ K int }]struct{ V1, V2 int }{
		{K: 1}: {V1: 1, V2: 2},
	}
}

func shouldPassAnonymousStructPointerMapKey() {
	_ = map[*struct{ K int }]*struct{ V1, V2 int }{
		{K: 1}: {V1: 1, V2: 2},
	}
}

func shouldFailAnonymousStructMapKey() {
	_ = map[struct{ K1, K2 int }]struct{ V1, V2 int }{
		{K1: 1}: {V1: 1}, // want "collections.<anonymous> is missing field K2" "collections.<anonymous> is missing field V2"
	}
}

func shouldFailAnonymousStructMapKeyWiderThanValue() {
	_ = map[struct{ K1, K2, K3 int }]struct{ V int }{
		{K1: 1}: {V: 1}, // want "collections.<anonymous> is missing fields K2, K3"
	}
}

func shouldFailAnonymousStructMapKeyNarrowerThanValue() {
	_ = map[struct{ K int }]struct{ V1, V2 int }{
		{}: {V1: 1}, // want "collections.<anonymous> is missing field K" "collections.<anonymous> is missing field V2"
	}
}

// A directive sits on one of the two declarations, so the position a key
// literal resolves to decides which literal it applies to. Resolving a key from
// the value type hands the key the value's directives, and the other way round.
func shouldPassDirectiveOnTheKeyDeclaration() {
	_ = map[
	//exhaustruct:optional
	struct {
		K1 int
		K2 int
	}]struct {
		V1 int
		V2 int
	}{
		{K1: 1}: {V1: 1, V2: 2},
	}
}

func shouldFailDirectiveOnTheValueDeclaration() {
	_ = map[struct {
		K1 int
		K2 int
	}]struct { //exhaustruct:optional
		V1 int
		V2 int
	}{
		{K1: 1}: {V1: 1}, // want "collections.<anonymous> is missing field K2"
	}
}

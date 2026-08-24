//go:build go1.26

package promoted

// enforcedBelow and HolderBelow repeat the shape of EnforcedInside and
// OptionalHolder in a file that declares Go 1.26, below the version the module
// sets. A literal here cannot write a promoted field as a key, so the embedded
// field is the one key that reaches the field enforced under it, and it is what
// the finding names.
type enforcedBelow struct {
	Loose string
	//exhaustruct:enforce
	Strict string
}

//exhaustruct:optional
type HolderBelow struct {
	enforcedBelow

	Own int
}

func shouldFailEnforcedBelowAGo126File() {
	_ = HolderBelow{Own: 1} // want "promoted.HolderBelow is missing field enforcedBelow"
	_ = HolderBelow{enforcedBelow: enforcedBelow{Loose: "", Strict: "x"}, Own: 1}
}

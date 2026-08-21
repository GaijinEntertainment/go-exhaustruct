//exhaustruct:ignore
package directives

// A directive above the package clause answers for that line and no more. The
// walk out of a literal stops below the file, so a package-level literal reads
// nothing from it -- there is no file-level directive.
var atPackageLevel = NestedInner{X: 1} // want "directives.NestedInner is missing field Y"

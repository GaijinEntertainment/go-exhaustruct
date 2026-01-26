package fix_test

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"

	"dev.gaijin.team/go/exhaustruct/v4/internal/fix"
)

func TestGenerate_EmptyMissing(t *testing.T) {
	t.Parallel()

	result := fix.Generate(nil, nil, nil)
	assert.Nil(t, result)

	result = fix.Generate(nil, nil, []fix.MissingField{})
	assert.Nil(t, result)
}

func TestZeroValueString(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(&packages.Config{ //nolint:exhaustruct
		Mode:       packages.NeedTypes,
		Dir:        "testdata",
		Context:    nil,
		Logf:       nil,
		Env:        nil,
		BuildFlags: nil,
		Fset:       nil,
		ParseFile:  nil,
		Tests:      false,
		Overlay:    nil,
	}, "")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]

	testCases := []struct {
		name     string
		typeName string
		expected string
	}{
		{"string", "stringField", `""`},
		{"int", "intField", "0"},
		{"bool", "boolField", "false"},
		{"float64", "float64Field", "0"},
		{"pointer", "pointerField", "nil"},
		{"slice", "sliceField", "nil"},
		{"map", "mapField", "nil"},
		{"chan", "chanField", "nil"},
		{"interface", "interfaceField", "nil"},
		{"func", "funcField", "nil"},
		{"struct", "structField", "testdata.Nested{}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			obj := pkg.Types.Scope().Lookup("TestStruct")
			require.NotNil(t, obj, "TestStruct not found")

			strct := obj.Type().Underlying().(*types.Struct) //nolint:forcetypeassert

			var fieldType types.Type
			for i := range strct.NumFields() {
				if strct.Field(i).Name() == tc.typeName {
					fieldType = strct.Field(i).Type()
					break
				}
			}
			require.NotNil(t, fieldType, "field %s not found", tc.typeName)

			result := fix.ZeroValueString(fieldType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestZeroValueString_NestedStructs(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(&packages.Config{ //nolint:exhaustruct
		Mode:       packages.NeedTypes,
		Dir:        "testdata",
		Context:    nil,
		Logf:       nil,
		Env:        nil,
		BuildFlags: nil,
		Fset:       nil,
		ParseFile:  nil,
		Tests:      false,
		Overlay:    nil,
	}, "")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)

	pkg := pkgs[0]

	testCases := []struct {
		name     string
		typeName string
		expected string
	}{
		{"nested_struct", "Nested", "testdata.Nested{}"},
		{"deep_nested_struct", "Deep", "testdata.DeepNested{}"},
		{"pointer_to_nested", "Ptr", "nil"},
		{"slice_of_nested", "Slice", "nil"},
		{"map_of_nested", "Map", "nil"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			obj := pkg.Types.Scope().Lookup("NestedStruct")
			require.NotNil(t, obj, "NestedStruct not found")

			strct := obj.Type().Underlying().(*types.Struct) //nolint:forcetypeassert

			var fieldType types.Type
			for i := range strct.NumFields() {
				if strct.Field(i).Name() == tc.typeName {
					fieldType = strct.Field(i).Type()
					break
				}
			}
			require.NotNil(t, fieldType, "field %s not found", tc.typeName)

			result := fix.ZeroValueString(fieldType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

package client_test

import (
	"fmt"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

const internalAPIImportPath = "github.com/klaskosk/report-portal-cli/pkg/internal/api"

// TestArchitecture_NoExportedAPIReferences verifies that no exported
// declaration in pkg/client exposes generated types from pkg/internal/api.
// Generated types may appear in unexported struct fields, unexported interface
// methods, and function bodies, but exported type definitions, function
// signatures, and var/const types must remain free of internal API references.
func TestArchitecture_NoExportedAPIReferences(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedTypes,
	}, ".")
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	require.Empty(t, pkgs[0].Errors, "package loading errors")

	scope := pkgs[0].Types.Scope()

	var violations []string

	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}

		violations = append(violations, checkExportedObject(obj)...)
	}

	assert.Empty(t, violations,
		"exported declarations must not reference pkg/internal/api; "+
			"use unexported interface methods to keep generated types internal")
}

// checkExportedObject inspects an exported package-level object and returns
// descriptions of any references to the internal API package in its public
// surface.
func checkExportedObject(obj types.Object) []string {
	switch obj := obj.(type) {
	case *types.TypeName:
		return checkTypeDeclaration(obj)
	case *types.Func:
		if referencesInternalAPI(obj.Type()) {
			return []string{apiViolation("function", obj.Name())}
		}
	case *types.Var:
		if referencesInternalAPI(obj.Type()) {
			return []string{apiViolation("variable", obj.Name())}
		}
	case *types.Const:
		if referencesInternalAPI(obj.Type()) {
			return []string{apiViolation("constant", obj.Name())}
		}
	}

	return nil
}

// checkTypeDeclaration inspects a type declaration for internal API references
// in its exported surface: exported struct fields, exported interface methods,
// exported receiver methods, and type parameter constraints.
func checkTypeDeclaration(typeName *types.TypeName) []string {
	if typeName.IsAlias() {
		if referencesInternalAPI(typeName.Type()) {
			return []string{apiViolation("type alias", typeName.Name())}
		}

		return nil
	}

	named, ok := typeName.Type().(*types.Named)
	if !ok {
		return nil
	}

	var violations []string

	if typeParamsReferenceInternalAPI(named.TypeParams()) {
		violations = append(violations,
			fmt.Sprintf("exported type %s has type parameter constraint referencing internal api",
				typeName.Name()))
	}

	switch underlying := named.Underlying().(type) {
	case *types.Interface:
		for method := range underlying.Methods() {
			if !method.Exported() {
				continue
			}

			if referencesInternalAPI(method.Type()) {
				violations = append(violations,
					fmt.Sprintf("exported interface %s method %s references internal api",
						typeName.Name(), method.Name()))
			}
		}
	case *types.Struct:
		for field := range underlying.Fields() {
			if field.Exported() && referencesInternalAPI(field.Type()) {
				violations = append(violations,
					fmt.Sprintf("exported struct %s field %s references internal api",
						typeName.Name(), field.Name()))
			}
		}
	default:
		if referencesInternalAPI(underlying) {
			violations = append(violations, apiViolation("type", typeName.Name()))
		}
	}

	for method := range named.Methods() {
		if !method.Exported() {
			continue
		}

		if referencesInternalAPI(method.Type()) {
			violations = append(violations,
				fmt.Sprintf("exported method %s.%s references internal api",
					typeName.Name(), method.Name()))
		}
	}

	return violations
}

// referencesInternalAPI reports whether a type contains any reference to a
// named type from the internal API package. It recurses through composite
// types (pointers, slices, maps, channels, signatures, aliases, unions) and
// inspects type arguments on generic instantiations, embedded types in
// interfaces, and type parameter constraints on signatures. It stops at named
// type boundaries — only checking whether the named type itself (or its type
// arguments) belongs to the internal API package, not its underlying structure.
func referencesInternalAPI(t types.Type) bool {
	switch t := t.(type) {
	case *types.Alias:
		if typeListReferencesInternalAPI(t.TypeArgs()) {
			return true
		}

		return referencesInternalAPI(types.Unalias(t))
	case *types.Named:
		if t.Obj().Pkg() != nil && t.Obj().Pkg().Path() == internalAPIImportPath {
			return true
		}

		return typeListReferencesInternalAPI(t.TypeArgs())
	case *types.Pointer:
		return referencesInternalAPI(t.Elem())
	case *types.Slice:
		return referencesInternalAPI(t.Elem())
	case *types.Array:
		return referencesInternalAPI(t.Elem())
	case *types.Map:
		return referencesInternalAPI(t.Key()) || referencesInternalAPI(t.Elem())
	case *types.Chan:
		return referencesInternalAPI(t.Elem())
	case *types.Signature:
		return typeParamsReferenceInternalAPI(t.TypeParams()) ||
			typeParamsReferenceInternalAPI(t.RecvTypeParams()) ||
			tupleReferencesInternalAPI(t.Params()) ||
			tupleReferencesInternalAPI(t.Results())
	case *types.Interface:
		for etyp := range t.EmbeddedTypes() {
			if referencesInternalAPI(etyp) {
				return true
			}
		}

		for method := range t.Methods() {
			if referencesInternalAPI(method.Type()) {
				return true
			}
		}

		return false
	case *types.Struct:
		for field := range t.Fields() {
			if referencesInternalAPI(field.Type()) {
				return true
			}
		}

		return false
	case *types.Union:
		for term := range t.Terms() {
			if referencesInternalAPI(term.Type()) {
				return true
			}
		}

		return false
	case *types.TypeParam:
		return referencesInternalAPI(t.Constraint())
	}

	return false
}

// tupleReferencesInternalAPI reports whether any element of a parameter or
// result tuple references the internal API package.
func tupleReferencesInternalAPI(tuple *types.Tuple) bool {
	for v := range tuple.Variables() {
		if referencesInternalAPI(v.Type()) {
			return true
		}
	}

	return false
}

// typeParamsReferenceInternalAPI reports whether any type parameter constraint
// references the internal API package. Returns false for a nil list.
func typeParamsReferenceInternalAPI(params *types.TypeParamList) bool {
	if params == nil {
		return false
	}

	for tparam := range params.TypeParams() {
		if referencesInternalAPI(tparam.Constraint()) {
			return true
		}
	}

	return false
}

// typeListReferencesInternalAPI reports whether any type in a TypeList
// (e.g. type arguments of a generic instantiation) references the internal
// API package. Returns false for a nil list.
func typeListReferencesInternalAPI(typeList *types.TypeList) bool {
	if typeList == nil {
		return false
	}

	for t := range typeList.Types() {
		if referencesInternalAPI(t) {
			return true
		}
	}

	return false
}

func apiViolation(kind, name string) string {
	return fmt.Sprintf("exported %s %s references internal api", kind, name)
}

// newInternalNamedType creates a *types.Named residing in the internal API
// package, for use in unit tests of referencesInternalAPI.
func newInternalNamedType(name string) *types.Named {
	pkg := types.NewPackage(internalAPIImportPath, "api")
	tn := types.NewTypeName(token.NoPos, pkg, name, nil)
	named := types.NewNamed(tn, types.Typ[types.Int], nil)

	return named
}

// newExternalNamedType creates a *types.Named outside the internal API
// package, for use in unit tests of referencesInternalAPI.
func newExternalNamedType(name string) *types.Named {
	pkg := types.NewPackage("example.com/external", "external")
	tn := types.NewTypeName(token.NoPos, pkg, name, nil)
	named := types.NewNamed(tn, types.Typ[types.Int], nil)

	return named
}

// newGenericInstantiation creates a *types.Named that is a generic type
// instantiated with the given type argument.
func newGenericInstantiation(typeArg types.Type) *types.Named {
	pkg := types.NewPackage("example.com/generic", "generic")
	tn := types.NewTypeName(token.NoPos, pkg, "Wrapper", nil)
	named := types.NewNamed(tn, nil, nil)

	tptn := types.NewTypeName(token.NoPos, nil, "T", nil)
	tp := types.NewTypeParam(tptn, types.NewInterfaceType(nil, nil))
	named.SetTypeParams([]*types.TypeParam{tp})
	named.SetUnderlying(types.Typ[types.Int])

	inst, err := types.Instantiate(nil, named, []types.Type{typeArg}, false)
	if err != nil {
		panic(fmt.Sprintf("Instantiate: %v", err))
	}

	result, ok := inst.(*types.Named)
	if !ok {
		panic(fmt.Sprintf("Instantiate returned %T, want *types.Named", inst))
	}

	return result
}

func TestReferencesInternalAPI(t *testing.T) {
	t.Parallel()

	internalType := newInternalNamedType("Generated")
	externalType := newExternalNamedType("External")

	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{
			name: "named type from internal api",
			typ:  internalType,
			want: true,
		},
		{
			name: "named type from external package",
			typ:  externalType,
			want: false,
		},
		{
			name: "pointer to internal api type",
			typ:  types.NewPointer(internalType),
			want: true,
		},
		{
			name: "slice of internal api type",
			typ:  types.NewSlice(internalType),
			want: true,
		},
		{
			name: "alias to internal api type",
			typ: func() types.Type {
				tn := types.NewTypeName(token.NoPos, nil, "MyAlias", nil)

				return types.NewAlias(tn, internalType)
			}(),
			want: true,
		},
		{
			name: "alias to external type",
			typ: func() types.Type {
				tn := types.NewTypeName(token.NoPos, nil, "MyAlias", nil)

				return types.NewAlias(tn, externalType)
			}(),
			want: false,
		},
		{
			name: "chained alias to internal api type",
			typ: func() types.Type {
				innerTN := types.NewTypeName(token.NoPos, nil, "Inner", nil)
				innerAlias := types.NewAlias(innerTN, internalType)
				outerTN := types.NewTypeName(token.NoPos, nil, "Outer", nil)

				return types.NewAlias(outerTN, innerAlias)
			}(),
			want: true,
		},
		{
			name: "generic instantiation with internal api type arg",
			typ:  newGenericInstantiation(internalType),
			want: true,
		},
		{
			name: "generic instantiation with external type arg",
			typ:  newGenericInstantiation(externalType),
			want: false,
		},
		{
			name: "interface embedding internal api type",
			typ: func() types.Type {
				iface := types.NewInterfaceType(nil, []types.Type{internalType})
				iface.Complete()

				return iface
			}(),
			want: true,
		},
		{
			name: "interface embedding external type",
			typ: func() types.Type {
				iface := types.NewInterfaceType(nil, []types.Type{externalType})
				iface.Complete()

				return iface
			}(),
			want: false,
		},
		{
			name: "signature with type param constrained by internal api type",
			typ: types.NewSignatureType(
				nil, nil,
				[]*types.TypeParam{
					types.NewTypeParam(
						types.NewTypeName(token.NoPos, nil, "T", nil),
						internalType,
					),
				},
				nil, nil, false,
			),
			want: true,
		},
		{
			name: "signature with type param constrained by external type",
			typ: types.NewSignatureType(
				nil, nil,
				[]*types.TypeParam{
					types.NewTypeParam(
						types.NewTypeName(token.NoPos, nil, "T", nil),
						externalType,
					),
				},
				nil, nil, false,
			),
			want: false,
		},
		{
			name: "signature with internal api param",
			typ: types.NewSignatureType(
				nil, nil, nil,
				types.NewTuple(types.NewParam(token.NoPos, nil, "x", internalType)),
				nil, false,
			),
			want: true,
		},
		{
			name: "basic type",
			typ:  types.Typ[types.String],
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, referencesInternalAPI(tt.typ))
		})
	}
}

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

var architecturePackages = []string{
	"github.com/klaskosk/report-portal-cli/pkg/client",
	"github.com/klaskosk/report-portal-cli/pkg/client/activity",
	"github.com/klaskosk/report-portal-cli/pkg/client/bts",
	"github.com/klaskosk/report-portal-cli/pkg/client/dashboard",
	"github.com/klaskosk/report-portal-cli/pkg/client/demodata",
	"github.com/klaskosk/report-portal-cli/pkg/client/files",
	"github.com/klaskosk/report-portal-cli/pkg/client/filter",
	"github.com/klaskosk/report-portal-cli/pkg/client/groups",
	"github.com/klaskosk/report-portal-cli/pkg/client/integrations",
	"github.com/klaskosk/report-portal-cli/pkg/client/item",
	"github.com/klaskosk/report-portal-cli/pkg/client/launch",
	"github.com/klaskosk/report-portal-cli/pkg/client/log",
	"github.com/klaskosk/report-portal-cli/pkg/client/logtypes",
	"github.com/klaskosk/report-portal-cli/pkg/client/onboarding",
	"github.com/klaskosk/report-portal-cli/pkg/client/plugin",
	"github.com/klaskosk/report-portal-cli/pkg/client/projects",
	"github.com/klaskosk/report-portal-cli/pkg/client/settings",
	"github.com/klaskosk/report-portal-cli/pkg/client/users",
	"github.com/klaskosk/report-portal-cli/pkg/client/widget",
}

// TestArchitecture_NoExportedAPIReferences verifies that no exported
// declaration in pkg/client (facade and domain subpackages) exposes generated
// types from pkg/internal/api. Generated types may appear in unexported struct
// fields, unexported interface methods, and function bodies, but exported type
// definitions, function signatures, and var/const types must remain free of
// internal API references.
func TestArchitecture_NoExportedAPIReferences(t *testing.T) {
	t.Parallel()

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedTypes,
	}, architecturePackages...)
	require.NoError(t, err)

	var violations []string

	for _, pkg := range pkgs {
		require.Empty(t, pkg.Errors, "package loading errors for %s", pkg.PkgPath)

		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if !obj.Exported() {
				continue
			}

			found := checkExportedObject(obj)
			for i := range found {
				found[i] = fmt.Sprintf("%s: %s", pkg.PkgPath, found[i])
			}

			violations = append(violations, found...)
		}
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

// checkNamedUnderlyingViolations inspects a named type's underlying interface or struct
// for internal API references visible on exported members.
func checkNamedUnderlyingViolations(typeName *types.TypeName, named *types.Named) []string {
	switch underlying := named.Underlying().(type) {
	case *types.Interface:
		return checkUnderlyingInterfaceViolations(typeName, underlying)
	case *types.Struct:
		return checkUnderlyingStructViolations(typeName, underlying)
	default:
		if referencesInternalAPI(underlying) {
			return []string{apiViolation("type", typeName.Name())}
		}

		return nil
	}
}

func checkUnderlyingInterfaceViolations(typeName *types.TypeName, iface *types.Interface) []string {
	var violations []string

	for method := range iface.Methods() {
		if !method.Exported() {
			continue
		}

		if referencesInternalAPI(method.Type()) {
			violations = append(violations,
				fmt.Sprintf("exported interface %s method %s references internal api",
					typeName.Name(), method.Name()))
		}
	}

	return violations
}

func checkUnderlyingStructViolations(typeName *types.TypeName, strct *types.Struct) []string {
	var violations []string

	for field := range strct.Fields() {
		if field.Exported() && referencesInternalAPI(field.Type()) {
			violations = append(violations,
				fmt.Sprintf("exported struct %s field %s references internal api",
					typeName.Name(), field.Name()))
		}
	}

	return violations
}

// checkExportedNamedMethods inspects exported methods on a named type.
func checkExportedNamedMethods(typeName *types.TypeName, named *types.Named) []string {
	var violations []string

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

	violations = append(violations, checkNamedUnderlyingViolations(typeName, named)...)
	violations = append(violations, checkExportedNamedMethods(typeName, named)...)

	return violations
}

// referencesInternalAPI reports whether a type contains any reference to a
// named type from the internal API package. It recurses through composite
// types (pointers, slices, maps, channels, signatures, aliases, unions) and
// inspects type arguments on generic instantiations, embedded types in
// interfaces, and type parameter constraints on signatures. It stops at named
// type boundaries — only checking whether the named type itself (or its type
// arguments) belongs to the internal API package, not its underlying structure.
//
//nolint:cyclop // dispatch mirrors go/types composite kinds; helpers handle each branch.
func referencesInternalAPI(typ types.Type) bool {
	switch concrete := typ.(type) {
	case *types.Alias:
		return referencesInternalAPIAlias(concrete)
	case *types.Named:
		return referencesInternalAPINamed(concrete)
	case *types.Pointer:
		return referencesInternalAPI(concrete.Elem())
	case *types.Slice:
		return referencesInternalAPI(concrete.Elem())
	case *types.Array:
		return referencesInternalAPI(concrete.Elem())
	case *types.Map:
		return referencesInternalAPI(concrete.Key()) || referencesInternalAPI(concrete.Elem())
	case *types.Chan:
		return referencesInternalAPI(concrete.Elem())
	case *types.Signature:
		return referencesInternalAPISignature(concrete)
	case *types.Interface:
		return referencesInternalAPIInterface(concrete)
	case *types.Struct:
		return referencesInternalAPIStruct(concrete)
	case *types.Union:
		return referencesInternalAPIUnion(concrete)
	case *types.TypeParam:
		return referencesInternalAPI(concrete.Constraint())
	default:
		return false
	}
}

func referencesInternalAPIAlias(a *types.Alias) bool {
	if typeListReferencesInternalAPI(a.TypeArgs()) {
		return true
	}

	return referencesInternalAPI(types.Unalias(a))
}

func referencesInternalAPINamed(n *types.Named) bool {
	if n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == internalAPIImportPath {
		return true
	}

	return typeListReferencesInternalAPI(n.TypeArgs())
}

func referencesInternalAPISignature(sig *types.Signature) bool {
	return typeParamsReferenceInternalAPI(sig.TypeParams()) ||
		typeParamsReferenceInternalAPI(sig.RecvTypeParams()) ||
		tupleReferencesInternalAPI(sig.Params()) ||
		tupleReferencesInternalAPI(sig.Results())
}

func referencesInternalAPIInterface(iface *types.Interface) bool {
	for etyp := range iface.EmbeddedTypes() {
		if referencesInternalAPI(etyp) {
			return true
		}
	}

	for method := range iface.Methods() {
		if referencesInternalAPI(method.Type()) {
			return true
		}
	}

	return false
}

func referencesInternalAPIStruct(strct *types.Struct) bool {
	for field := range strct.Fields() {
		if referencesInternalAPI(field.Type()) {
			return true
		}
	}

	return false
}

func referencesInternalAPIUnion(u *types.Union) bool {
	for term := range u.Terms() {
		if referencesInternalAPI(term.Type()) {
			return true
		}
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

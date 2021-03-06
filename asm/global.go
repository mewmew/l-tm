package asm

import (
	"fmt"

	"github.com/llir/l/ir"
	"github.com/llir/l/ir/types"
	"github.com/mewmew/l-tm/asm/ll/ast"
	"github.com/mewmew/l-tm/internal/enc"
	"github.com/pkg/errors"
)

// resolveGlobals resolves the global variable and function declarations and
// defintions of the given module. The returned value maps from global
// identifier (without '@' prefix) to the corresponding IR value.
func (gen *generator) resolveGlobals(module *ast.Module) (map[string]ir.Constant, error) {
	// index maps from global identifier to underlying AST value.
	index := make(map[string]ast.LlvmNode)
	// Record order of global variable and function declarations and definitions.
	var globalOrder, funcOrder []string
	// Index global variable and function declarations and definitions.
	for _, entity := range module.TopLevelEntities() {
		switch entity := entity.(type) {
		case *ast.GlobalDecl:
			name := global(entity.Name())
			globalOrder = append(globalOrder, name)
			if prev, ok := index[name]; ok {
				// TODO: don't report error if prev is a declaration (of same type)?
				return nil, errors.Errorf("AST global identifier %q already present; prev `%s`, new `%s`", enc.Global(name), text(prev), text(entity))
			}
			index[name] = entity
		case *ast.GlobalDef:
			name := global(entity.Name())
			globalOrder = append(globalOrder, name)
			if prev, ok := index[name]; ok {
				// TODO: don't report error if prev is a declaration (of same type)?
				return nil, errors.Errorf("AST global identifier %q already present; prev `%s`, new `%s`", enc.Global(name), text(prev), text(entity))
			}
			index[name] = entity
		case *ast.FuncDecl:
			name := global(entity.Header().Name())
			funcOrder = append(funcOrder, name)
			if prev, ok := index[name]; ok {
				// TODO: don't report error if prev is a declaration (of same type)?
				return nil, errors.Errorf("AST global identifier %q already present; prev `%s`, new `%s`", enc.Global(name), text(prev), text(entity))
			}
			index[name] = entity
		case *ast.FuncDef:
			name := global(entity.Header().Name())
			funcOrder = append(funcOrder, name)
			if prev, ok := index[name]; ok {
				// TODO: don't report error if prev is a declaration (of same type)?
				return nil, errors.Errorf("AST global identifier %q already present; prev `%s`, new `%s`", enc.Global(name), text(prev), text(entity))
			}
			index[name] = entity
			// TODO: handle alias definitions and IFuncs.
			//case *ast.AliasDef:
			//case *ast.IFuncDef:
		}
	}

	// Create corresponding IR global variables and functions (without bodies but
	// with type).
	gen.gs = make(map[string]ir.Constant)
	for name, old := range index {
		g, err := gen.newGlobal(name, old)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		gen.gs[name] = g
	}

	// Translate global variables and functions (including bodies).
	for name, old := range index {
		g := gen.gs[name]
		_, err := gen.astToIRGlobal(g, old)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	// Add global variable declarations and definitions to IR module in order of
	// occurrence in input.
	for _, key := range globalOrder {
		g, err := gen.global(key)
		if err != nil {
			// NOTE: panic since this would indicate a bug in the implementation.
			panic(err)
		}
		gen.m.Globals = append(gen.m.Globals, g)
	}

	// Add function declarations and definitions to IR module in order of
	// occurrence in input.
	for _, key := range funcOrder {
		f, err := gen.function(key)
		if err != nil {
			// NOTE: panic since this would indicate a bug in the implementation.
			panic(err)
		}
		gen.m.Funcs = append(gen.m.Funcs, f)
	}

	return gen.gs, nil
}

// newGlobal returns a new IR value (without body but with type) based on the
// given AST global variable or function.
func (gen *generator) newGlobal(name string, old ast.LlvmNode) (ir.Constant, error) {
	switch old := old.(type) {
	case *ast.GlobalDecl:
		g := &ir.Global{GlobalName: name}
		// Content type.
		contentType, err := gen.irType(old.ContentType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		g.ContentType = contentType
		g.Typ = types.NewPointer(g.ContentType)
		return g, nil
	case *ast.GlobalDef:
		g := &ir.Global{GlobalName: name}
		// Content type.
		contentType, err := gen.irType(old.ContentType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		g.ContentType = contentType
		g.Typ = types.NewPointer(g.ContentType)
		return g, nil
	case *ast.FuncDecl:
		f := &ir.Function{GlobalName: name}
		hdr := old.Header()
		sig := &types.FuncType{}
		// Return type.
		retType, err := gen.irType(hdr.RetType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		sig.RetType = retType
		// Function parameters.
		ps := hdr.Params()
		for _, p := range ps.Params() {
			param, err := gen.irType(p.Typ())
			if err != nil {
				return nil, errors.WithStack(err)
			}
			sig.Params = append(sig.Params, param)
		}
		// Variadic.
		sig.Variadic = irOptVariadic(ps.Variadic())
		f.Sig = sig
		f.Typ = types.NewPointer(f.Sig)
		return f, nil
	case *ast.FuncDef:
		f := &ir.Function{GlobalName: name}
		sig := &types.FuncType{}
		hdr := old.Header()
		// Return type.
		retType, err := gen.irType(hdr.RetType())
		if err != nil {
			return nil, errors.WithStack(err)
		}
		sig.RetType = retType
		// Function parameters.
		ps := hdr.Params()
		for _, p := range ps.Params() {
			param, err := gen.irType(p.Typ())
			if err != nil {
				return nil, errors.WithStack(err)
			}
			sig.Params = append(sig.Params, param)
		}
		// Variadic.
		sig.Variadic = irOptVariadic(ps.Variadic())
		f.Sig = sig
		f.Typ = types.NewPointer(f.Sig)
		return f, nil
	default:
		panic(fmt.Errorf("support for global variable or function %T not yet implemented", old))
	}
}

// astToIRGlobal translates the AST global variable or function into an
// equivalent IR value.
func (gen *generator) astToIRGlobal(g ir.Constant, old ast.LlvmNode) (ir.Constant, error) {
	switch old := old.(type) {
	case *ast.GlobalDecl:
		return gen.astToIRGlobalDecl(g, old)
	case *ast.GlobalDef:
		return gen.astToIRGlobalDef(g, old)
	case *ast.FuncDecl:
		return gen.astToIRFuncDecl(g, old)
	case *ast.FuncDef:
		return gen.astToIRFuncDef(g, old)
	default:
		panic(fmt.Errorf("support for type %T not yet implemented", old))
	}
}

// ~~~ [ Global Variable Declaration ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (gen *generator) astToIRGlobalDecl(g ir.Constant, old *ast.GlobalDecl) (*ir.Global, error) {
	global, ok := g.(*ir.Global)
	if !ok {
		panic(fmt.Errorf("invalid IR type for AST global declaration; expected *ir.Global, got %T", g))
	}
	// Linkage.
	global.Linkage = irOptLinkage(old.ExternLinkage())
	// Preemption.
	global.Preemption = irOptPreemption(old.Preemption())
	// Visibility.
	global.Visibility = irOptVisibility(old.Visibility())
	// DLL storage class.
	global.DLLStorageClass = irOptDLLStorageClass(old.DLLStorageClass())
	// Thread local storage model.
	global.TLSModel = irOptTLSModelFromThreadLocal(old.ThreadLocal())
	// Unnamed address.
	global.UnnamedAddr = irOptUnnamedAddr(old.UnnamedAddr())
	// Address space.
	global.Typ.AddrSpace = irOptAddrSpace(old.AddrSpace())
	// Externally initialized.
	global.ExternallyInitialized = irOptExternallyInitialized(old.ExternallyInitialized())
	// Immutable (constant or global).
	global.Immutable = irImmutable(old.Immutable())
	// Content type already stored during index.
	// TODO: handle GlobalAttrs.
	// TODO: handle FuncAttrs.
	return global, nil
}

// ~~~ [ Global Variable Definition ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (gen *generator) astToIRGlobalDef(g ir.Constant, old *ast.GlobalDef) (*ir.Global, error) {
	global, ok := g.(*ir.Global)
	if !ok {
		panic(fmt.Errorf("invalid IR type for AST global definition; expected *ir.Global, got %T", g))
	}
	// Linkage.
	global.Linkage = irOptLinkage(old.Linkage())
	// Preemption.
	global.Preemption = irOptPreemption(old.Preemption())
	// Visibility.
	global.Visibility = irOptVisibility(old.Visibility())
	// DLL storage class.
	global.DLLStorageClass = irOptDLLStorageClass(old.DLLStorageClass())
	// Thread local storage model.
	global.TLSModel = irOptTLSModelFromThreadLocal(old.ThreadLocal())
	// Unnamed address.
	global.UnnamedAddr = irOptUnnamedAddr(old.UnnamedAddr())
	// Address space.
	global.Typ.AddrSpace = irOptAddrSpace(old.AddrSpace())
	// Externally initialized.
	global.ExternallyInitialized = irOptExternallyInitialized(old.ExternallyInitialized())
	// Immutable (constant or global).
	global.Immutable = irImmutable(old.Immutable())
	// Content type already stored during index.
	init, err := gen.irConstant(global.ContentType, old.Init())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	global.Init = init
	// TODO: handle GlobalAttrs.
	// TODO: handle FuncAttrs.
	return global, nil
}

// ~~~ [ Indirect Symbol Definition ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TODO: add alias definition and IFuncs.

// ~~~ [ Function Declaration ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (gen *generator) astToIRFuncDecl(g ir.Constant, old *ast.FuncDecl) (*ir.Function, error) {
	f, ok := g.(*ir.Function)
	if !ok {
		panic(fmt.Errorf("invalid IR type for AST function declaration; expected *ir.Function, got %T", g))
	}
	// Metadata.
	// TODO: translate function metadata.
	if err := gen.astToIRFuncHeader(f, old.Header()); err != nil {
		return nil, errors.WithStack(err)
	}
	return f, nil
}

func (gen *generator) astToIRFuncHeader(f *ir.Function, hdr ast.FuncHeader) error {
	// Linkage.
	f.Linkage = irOptLinkage(hdr.ExternLinkage())
	// Preemption.
	f.Preemption = irOptPreemption(hdr.Preemption())
	// Visibility.
	f.Visibility = irOptVisibility(hdr.Visibility())
	// DLL storage class.
	f.DLLStorageClass = irOptDLLStorageClass(hdr.DLLStorageClass())
	// Calling convention.
	// TODO: translate CallingConv.
	// Return attributes.
	// TODO: handle ReturnAttrs.
	// Return type; already handled.
	// Function name; already handled.
	// Function parameters.
	ps := hdr.Params()
	for _, p := range ps.Params() {
		// Type.
		typ, err := gen.irType(p.Typ())
		if err != nil {
			return errors.WithStack(err)
		}
		// Parameter attributes.
		// TODO: handle Attrs.
		name := optLocal(p.Name())
		param := ir.NewParam(typ, name)
		f.Params = append(f.Params, param)
	}

	// Unnamed address.
	f.UnnamedAddr = irOptUnnamedAddr(hdr.UnnamedAddr())
	// Address space.
	f.Typ.AddrSpace = irOptAddrSpace(hdr.AddrSpace())
	// Function attributes.
	// TODO: handle FuncAttrs.
	// Section.
	// TODO: handle Section.
	// Comdat.
	// TODO: handle Comdat.
	// GC.
	// TODO: handle GC.
	// Prefix.
	// TODO: handle Prefix.
	// Prologue.
	// TODO: handle Prologue.
	// Personality.
	// TODO: handle Personality.
	return nil
}

// ~~~ [ Function Definition ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (gen *generator) astToIRFuncDef(g ir.Constant, old *ast.FuncDef) (*ir.Function, error) {
	f, ok := g.(*ir.Function)
	if !ok {
		panic(fmt.Errorf("invalid IR type for AST function definition; expected *ir.Function, got %T", g))
	}
	if err := gen.astToIRFuncHeader(f, old.Header()); err != nil {
		return nil, errors.WithStack(err)
	}
	// Metadata.
	// TODO: translate function metadata.
	// Basic blocks.
	fgen := newFuncGen(gen, f)
	_, err := fgen.resolveLocals(old.Body())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// Use list orders.
	// TODO: translate use list orders.
	return f, nil
}

// ### [ Helper functions ] ####################################################

// text returns the text of the given node.
func text(n ast.LlvmNode) string {
	if n := n.LlvmNode(); n != nil {
		return n.Text()
	}
	return ""
}

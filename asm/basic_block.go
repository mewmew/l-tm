package asm

import (
	"github.com/llir/l/ir"
	"github.com/mewmew/l-tm/asm/ll/ast"
	"github.com/pkg/errors"
)

func (fgen *funcGen) irBasicBlock(old ast.Label) (*ir.BasicBlock, error) {
	name := local(old.Name())
	v, ok := fgen.ls[name]
	if !ok {
		return nil, errors.Errorf("unable to locate local identifier %q", name)
	}
	block, ok := v.(*ir.BasicBlock)
	if !ok {
		return nil, errors.Errorf("invalid basic block type; expected *ir.BasicBlock, got %T", v)
	}
	return block, nil
}

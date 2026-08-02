// Vendored from github.com/chrishrb/go-grip (pkg/mathjax) under the MIT License.
// Copyright (c) 2024 Christoph Herb. See mathjax.go for the full notice.

package mathjax

import "github.com/yuin/goldmark/ast"

// MathBlock is the AST node for a block-level math expression.
type MathBlock struct {
	ast.BaseBlock
}

// KindMathBlock is the node kind for [MathBlock].
var KindMathBlock = ast.NewNodeKind("MathBlock")

// NewMathBlock returns an empty [MathBlock] node.
func NewMathBlock() *MathBlock {
	return &MathBlock{}
}

// Dump implements ast.Node.
func (n *MathBlock) Dump(source []byte, level int) {
	m := map[string]string{}
	ast.DumpHelper(n, source, level, m, nil)
}

// Kind implements ast.Node.
func (n *MathBlock) Kind() ast.NodeKind {
	return KindMathBlock
}

// IsRaw reports that the node's content is raw and must not be parsed as
// inline markdown.
func (n *MathBlock) IsRaw() bool {
	return true
}

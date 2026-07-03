// Vendored from github.com/chrishrb/go-grip (pkg/mathjax) under the MIT License.
// Copyright (c) 2024 Christoph Herb. See mathjax.go for the full notice.

package mathjax

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"
)

// InlineMath is the AST node for an inline math expression.
type InlineMath struct {
	ast.BaseInline
}

// Inline implements ast.Node.
func (n *InlineMath) Inline() {}

// IsBlank reports whether the inline math has no non-space content.
func (n *InlineMath) IsBlank(source []byte) bool {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		text := c.(*ast.Text).Segment
		if !util.IsBlank(text.Value(source)) {
			return false
		}
	}
	return true
}

// Dump implements ast.Node.
func (n *InlineMath) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// KindInlineMath is the node kind for [InlineMath].
var KindInlineMath = ast.NewNodeKind("InlineMath")

// Kind implements ast.Node.
func (n *InlineMath) Kind() ast.NodeKind {
	return KindInlineMath
}

// NewInlineMath returns an empty [InlineMath] node.
func NewInlineMath() *InlineMath {
	return &InlineMath{
		BaseInline: ast.BaseInline{},
	}
}

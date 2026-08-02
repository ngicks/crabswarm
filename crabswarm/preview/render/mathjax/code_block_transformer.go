// Vendored from github.com/chrishrb/go-grip (pkg/mathjax) under the MIT License.
// Copyright (c) 2024 Christoph Herb. See mathjax.go for the full notice.

package mathjax

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type mathCodeBlockTransformer struct{}

var defaultMathCodeBlockTransformer = &mathCodeBlockTransformer{}

// NewMathCodeBlockTransformer converts ```math fenced code blocks into
// [MathBlock] nodes.
func NewMathCodeBlockTransformer() parser.ASTTransformer {
	return defaultMathCodeBlockTransformer
}

// Transform implements parser.ASTTransformer.
func (t *mathCodeBlockTransformer) Transform(
	node *ast.Document,
	reader text.Reader,
	pc parser.Context,
) {
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Check if this is a fenced code block
		codeBlock, ok := n.(*ast.FencedCodeBlock)
		if !ok || codeBlock.Info == nil {
			return ast.WalkContinue, nil
		}

		// Check if the language is "math"
		language := codeBlock.Info.Value(reader.Source())
		if !bytes.Equal(language, []byte("math")) {
			return ast.WalkContinue, nil
		}

		// Convert to MathBlock
		mathBlock := NewMathBlock()

		// Copy all lines from the code block to the math block
		for i := range codeBlock.Lines().Len() {
			line := codeBlock.Lines().At(i)
			mathBlock.Lines().Append(line)
		}

		// Replace the code block with the math block
		if parent := n.Parent(); parent != nil {
			parent.ReplaceChild(parent, codeBlock, mathBlock)
		}

		return ast.WalkContinue, nil
	})
}

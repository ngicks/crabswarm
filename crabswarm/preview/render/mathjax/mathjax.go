// Package mathjax is a goldmark extension that emits MathJax-compatible
// markup for inline ($...$) and block ($$...$$ or ```math fences) math, to be
// typeset client-side by MathJax (tex-mml-chtml.js).
//
// Vendored from github.com/chrishrb/go-grip (pkg/mathjax), which is itself
// adapted from github.com/litao91/goldmark-mathjax, under the MIT License:
//
//	MIT License
//	Copyright (c) 2024 Christoph Herb
//
//	Permission is hereby granted, free of charge, to any person obtaining a copy
//	of this software and associated documentation files (the "Software"), to deal
//	in the Software without restriction, including without limitation the rights
//	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//	copies of the Software, and to permit persons to whom the Software is
//	furnished to do so, subject to the following conditions:
//
//	The above copyright notice and this permission notice shall be included in all
//	copies or substantial portions of the Software.
//
//	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//	SOFTWARE.
package mathjax

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type mathjax struct {
	inlineStartDelim string
	inlineEndDelim   string
	blockStartDelim  string
	blockEndDelim    string
}

// Option configures a MathJax extension built by [NewMathJax].
type Option interface {
	SetOption(e *mathjax)
}

type withInlineDelim struct {
	start string
	end   string
}

type withBlockDelim struct {
	start string
	end   string
}

// WithInlineDelim overrides the delimiters wrapped around inline math.
func WithInlineDelim(start string, end string) Option {
	return &withInlineDelim{start, end}
}

func (o *withInlineDelim) SetOption(e *mathjax) {
	e.inlineStartDelim = o.start
	e.inlineEndDelim = o.end
}

// WithBlockDelim overrides the delimiters wrapped around block math.
func WithBlockDelim(start string, end string) Option {
	return &withBlockDelim{start, end}
}

func (o *withBlockDelim) SetOption(e *mathjax) {
	e.blockStartDelim = o.start
	e.blockEndDelim = o.end
}

// NewMathJax returns a goldmark extension that renders inline and block math
// as MathJax-compatible markup. Defaults use the \( \) and \[ \] delimiters.
func NewMathJax(opts ...Option) goldmark.Extender {
	r := &mathjax{
		inlineStartDelim: `\(`,
		inlineEndDelim:   `\)`,
		blockStartDelim:  `\[`,
		blockEndDelim:    `\]`,
	}

	for _, o := range opts {
		o.SetOption(r)
	}
	return r
}

func (e *mathjax) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(NewMathJaxBlockParser(), 701),
	))
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(NewInlineMathParser(), 501),
	))
	m.Parser().AddOptions(parser.WithASTTransformers(
		util.Prioritized(NewMathCodeBlockTransformer(), 100),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewMathBlockRenderer(e.blockStartDelim, e.blockEndDelim), 501),
		util.Prioritized(NewInlineMathRenderer(e.inlineStartDelim, e.inlineEndDelim), 502),
	))
}

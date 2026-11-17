// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown

import (
	"context"
	"strings"

	"gitea.dev/modules/markup"
	giteautil "gitea.dev/modules/util"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// FencedCodeBlocks returns the text of each fenced code block whose language is lang. It parses the
// source the way render does, so callers find exactly the blocks the rendered page shows.
func FencedCodeBlocks(ctx context.Context, source []byte, lang string) []string {
	rctx := markup.NewRenderContext(ctx)
	buf := giteautil.NormalizeEOL(source)
	rc := &RenderConfig{Meta: markup.RenderMetaAsDetails}
	buf, _ = ExtractMetadataBytes(buf, rc) // front matter is not part of the rendered body
	pc := newParserContext(rctx)
	pc.Set(renderConfigKey, rc)
	doc := SpecializedMarkdown(rctx).goldmarkMarkdown.Parser().Parse(text.NewReader(buf), parser.WithContext(pc))

	var blocks []string
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		code, ok := n.(*ast.FencedCodeBlock)
		if !entering || !ok {
			return ast.WalkContinue, nil
		}
		// the AST transformer has already cut the info string at its first space, comma or colon
		if string(code.Language(buf)) == lang {
			var sb strings.Builder
			for i := range code.Lines().Len() {
				seg := code.Lines().At(i)
				sb.Write(seg.Value(buf)) // Value keeps the indentation a list or quote leaves inside the block
			}
			blocks = append(blocks, sb.String())
		}
		return ast.WalkSkipChildren, nil
	})
	return blocks
}

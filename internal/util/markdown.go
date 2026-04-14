package util

import (
	"bytes"
	"html"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

var md = goldmark.New()
var htmlPolicy = bluemonday.UGCPolicy()

// MarkdownToHTML converts Markdown to sanitized HTML for embedding in templates.
func MarkdownToHTML(input string) template.HTML {
	s := bytes.TrimSpace([]byte(input))
	if len(s) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := md.Convert([]byte(input), &buf); err != nil {
		return template.HTML(html.EscapeString(input))
	}
	return template.HTML(htmlPolicy.SanitizeBytes(buf.Bytes()))
}

// RenderPostBody applies >>refs then Markdown + sanitize.
func RenderPostBody(raw string) template.HTML {
	raw = PostRefMarkers(raw)
	return MarkdownToHTML(raw)
}

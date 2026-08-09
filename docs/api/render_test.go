package apidoc

import (
	"strings"
	"testing"
)

func TestRenderMarkdownPage(test *testing.T) {
	markdown := []byte(`# API <Demo>

普通文本包含 ` + "`code`" + ` 和 [安全链接](https://example.com/docs)。

- 第一项
- 第二项

| 参数 | 类型 |
| --- | --- |
| code | string |

` + "```json" + `
{"value":"<script>"}
` + "```" + `
`)

	html := string(Render(markdown))
	expectedFragments := []string{
		`<!doctype html>`,
		`<nav class="toc">`,
		`href="#api-demo"`,
		`<h1 id="api-demo">API &lt;Demo&gt;</h1>`,
		`<code>code</code>`,
		`href="https://example.com/docs"`,
		`<ul><li>第一项</li><li>第二项</li></ul>`,
		`<table><thead><tr><th>参数</th><th>类型</th></tr></thead>`,
		`<code class="language-json">`,
		`&lt;script&gt;`,
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(html, fragment) {
			test.Errorf("rendered HTML does not contain %q", fragment)
		}
	}
	if strings.Contains(html, `<script>`) {
		test.Fatal("rendered HTML contains unescaped script tag")
	}
}

func TestEmbeddedDocument(test *testing.T) {
	if !strings.Contains(string(Markdown), "TDX HTTP API Reference") {
		test.Fatal("embedded Markdown is missing API document title")
	}
	if !strings.Contains(string(Markdown), "/kline/minute/241") {
		test.Fatal("embedded Markdown is missing 241-minute K-line API")
	}
	if !strings.Contains(string(Markdown), "adjust=qfq") {
		test.Fatal("embedded Markdown is missing adjusted K-line example")
	}
	if !strings.Contains(string(Markdown), "/code/stocks/detail") {
		test.Fatal("embedded Markdown is missing stock detail API")
	}
	if !strings.Contains(string(Markdown), "StockIndustry") {
		test.Fatal("embedded Markdown is missing stock industry model")
	}
	if !strings.Contains(string(HTML), "TDX HTTP API Reference") {
		test.Fatal("rendered HTML is missing API document title")
	}
}

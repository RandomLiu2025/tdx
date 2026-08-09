package apidoc

import _ "embed"

//go:embed http-api.md
var Markdown []byte

var HTML = Render(Markdown)

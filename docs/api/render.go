package apidoc

import (
	"html"
	"net/url"
	"strings"
	"unicode"
)

type heading struct {
	level int
	id    string
	text  string
}

func Render(markdown []byte) []byte {
	lines := strings.Split(strings.ReplaceAll(string(markdown), "\r\n", "\n"), "\n")
	headings, headingByLine := collectHeadings(lines)
	body := renderBlocks(lines, headingByLine)
	title := "TDX HTTP API Reference"
	if len(headings) > 0 {
		title = headings[0].text
	}

	var page strings.Builder
	page.WriteString("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\">")
	page.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">")
	page.WriteString("<title>")
	page.WriteString(html.EscapeString(title))
	page.WriteString("</title><style>")
	page.WriteString(pageStyles)
	page.WriteString("</style></head><body><div class=\"layout\"><aside>")
	page.WriteString(renderTableOfContents(headings))
	page.WriteString("</aside><main class=\"content\">")
	page.WriteString(body)
	page.WriteString("</main></div></body></html>")
	return []byte(page.String())
}

func collectHeadings(lines []string) ([]heading, map[int]heading) {
	headings := make([]heading, 0)
	headingByLine := make(map[int]heading)
	usedIDs := make(map[string]int)
	insideCode := false
	for lineIndex, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			insideCode = !insideCode
			continue
		}
		if insideCode {
			continue
		}
		level, text, ok := parseHeading(trimmed)
		if !ok {
			continue
		}
		baseID := slug(text)
		usedIDs[baseID]++
		headingID := baseID
		if usedIDs[baseID] > 1 {
			headingID += "-" + integerString(usedIDs[baseID])
		}
		item := heading{level: level, id: headingID, text: text}
		headings = append(headings, item)
		headingByLine[lineIndex] = item
	}
	return headings, headingByLine
}

func renderBlocks(lines []string, headingByLine map[int]heading) string {
	var output strings.Builder
	for lineIndex := 0; lineIndex < len(lines); {
		trimmed := strings.TrimSpace(lines[lineIndex])
		if trimmed == "" {
			lineIndex++
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			language := safeClassName(strings.TrimSpace(strings.TrimPrefix(trimmed, "```")))
			lineIndex++
			codeLines := make([]string, 0)
			for lineIndex < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[lineIndex]), "```") {
				codeLines = append(codeLines, lines[lineIndex])
				lineIndex++
			}
			if lineIndex < len(lines) {
				lineIndex++
			}
			output.WriteString("<pre><code")
			if language != "" {
				output.WriteString(" class=\"language-")
				output.WriteString(language)
				output.WriteString("\"")
			}
			output.WriteString(">")
			output.WriteString(html.EscapeString(strings.Join(codeLines, "\n")))
			output.WriteString("</code></pre>")
			continue
		}

		if item, ok := headingByLine[lineIndex]; ok {
			output.WriteString("<h")
			output.WriteString(integerString(item.level))
			output.WriteString(" id=\"")
			output.WriteString(html.EscapeString(item.id))
			output.WriteString("\">")
			output.WriteString(renderInline(item.text))
			output.WriteString("</h")
			output.WriteString(integerString(item.level))
			output.WriteString(">")
			lineIndex++
			continue
		}

		if isTableStart(lines, lineIndex) {
			headers := splitTableRow(lines[lineIndex])
			lineIndex += 2
			rows := make([][]string, 0)
			for lineIndex < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[lineIndex]), "|") {
				rows = append(rows, splitTableRow(lines[lineIndex]))
				lineIndex++
			}
			output.WriteString("<div class=\"table-wrap\"><table><thead><tr>")
			for _, cell := range headers {
				output.WriteString("<th>")
				output.WriteString(renderInline(cell))
				output.WriteString("</th>")
			}
			output.WriteString("</tr></thead><tbody>")
			for _, row := range rows {
				output.WriteString("<tr>")
				for _, cell := range row {
					output.WriteString("<td>")
					output.WriteString(renderInline(cell))
					output.WriteString("</td>")
				}
				output.WriteString("</tr>")
			}
			output.WriteString("</tbody></table></div>")
			continue
		}

		if strings.HasPrefix(trimmed, "- ") {
			output.WriteString("<ul>")
			for lineIndex < len(lines) {
				listLine := strings.TrimSpace(lines[lineIndex])
				if !strings.HasPrefix(listLine, "- ") {
					break
				}
				output.WriteString("<li>")
				output.WriteString(renderInline(strings.TrimSpace(strings.TrimPrefix(listLine, "- "))))
				output.WriteString("</li>")
				lineIndex++
			}
			output.WriteString("</ul>")
			continue
		}

		paragraphLines := make([]string, 0)
		for lineIndex < len(lines) && !startsBlock(lines, lineIndex, headingByLine) {
			paragraphLines = append(paragraphLines, strings.TrimSpace(lines[lineIndex]))
			lineIndex++
		}
		if len(paragraphLines) > 0 {
			output.WriteString("<p>")
			output.WriteString(renderInline(strings.Join(paragraphLines, " ")))
			output.WriteString("</p>")
		}
	}
	return output.String()
}

func renderTableOfContents(headings []heading) string {
	var output strings.Builder
	output.WriteString("<nav class=\"toc\"><div class=\"toc-title\">API 文档</div><ul>")
	for _, item := range headings {
		if item.level > 3 {
			continue
		}
		output.WriteString("<li class=\"level-")
		output.WriteString(integerString(item.level))
		output.WriteString("\"><a href=\"#")
		output.WriteString(html.EscapeString(item.id))
		output.WriteString("\">")
		output.WriteString(html.EscapeString(item.text))
		output.WriteString("</a></li>")
	}
	output.WriteString("</ul></nav>")
	return output.String()
}

func renderInline(text string) string {
	var output strings.Builder
	for len(text) > 0 {
		codeIndex := strings.Index(text, "`")
		linkIndex := strings.Index(text, "[")
		nextIndex := firstNonNegative(codeIndex, linkIndex)
		if nextIndex < 0 {
			output.WriteString(html.EscapeString(text))
			break
		}
		output.WriteString(html.EscapeString(text[:nextIndex]))
		text = text[nextIndex:]

		if strings.HasPrefix(text, "`") {
			endIndex := strings.Index(text[1:], "`")
			if endIndex < 0 {
				output.WriteString("`")
				text = text[1:]
				continue
			}
			output.WriteString("<code>")
			output.WriteString(html.EscapeString(text[1 : endIndex+1]))
			output.WriteString("</code>")
			text = text[endIndex+2:]
			continue
		}

		labelEnd := strings.Index(text, "](")
		if labelEnd <= 1 {
			output.WriteString("[")
			text = text[1:]
			continue
		}
		urlEnd := strings.Index(text[labelEnd+2:], ")")
		if urlEnd < 0 {
			output.WriteString("[")
			text = text[1:]
			continue
		}
		label := text[1:labelEnd]
		rawURL := text[labelEnd+2 : labelEnd+2+urlEnd]
		if safeURL(rawURL) {
			output.WriteString("<a href=\"")
			output.WriteString(html.EscapeString(rawURL))
			output.WriteString("\">")
			output.WriteString(html.EscapeString(label))
			output.WriteString("</a>")
		} else {
			output.WriteString(html.EscapeString(label))
		}
		text = text[labelEnd+3+urlEnd:]
	}
	return output.String()
}

func parseHeading(line string) (int, string, bool) {
	level := 0
	for level < len(line) && level < 6 && line[level] == '#' {
		level++
	}
	if level == 0 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(line[level+1:]), true
}

func slug(text string) string {
	var output strings.Builder
	lastHyphen := false
	for _, character := range strings.ToLower(text) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			output.WriteRune(character)
			lastHyphen = false
			continue
		}
		if output.Len() > 0 && !lastHyphen {
			output.WriteByte('-')
			lastHyphen = true
		}
	}
	result := strings.Trim(output.String(), "-")
	if result == "" {
		return "section"
	}
	return result
}

func startsBlock(lines []string, lineIndex int, headingByLine map[int]heading) bool {
	if lineIndex >= len(lines) {
		return true
	}
	trimmed := strings.TrimSpace(lines[lineIndex])
	return trimmed == "" || strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "- ") ||
		headingByLine[lineIndex].id != "" || isTableStart(lines, lineIndex)
}

func isTableStart(lines []string, lineIndex int) bool {
	if lineIndex+1 >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[lineIndex]), "|") {
		return false
	}
	separators := splitTableRow(lines[lineIndex+1])
	if len(separators) == 0 {
		return false
	}
	for _, separator := range separators {
		separator = strings.TrimSpace(separator)
		separator = strings.TrimPrefix(separator, ":")
		separator = strings.TrimSuffix(separator, ":")
		if len(separator) < 3 || strings.Trim(separator, "-") != "" {
			return false
		}
	}
	return true
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func safeURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == "" || parsed.Scheme == "http" || parsed.Scheme == "https"
}

func safeClassName(value string) string {
	var output strings.Builder
	for _, character := range value {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '-' || character == '_' {
			output.WriteRune(character)
		}
	}
	return output.String()
}

func firstNonNegative(values ...int) int {
	result := -1
	for _, value := range values {
		if value >= 0 && (result < 0 || value < result) {
			result = value
		}
	}
	return result
}

func integerString(value int) string {
	if value == 0 {
		return "0"
	}
	digits := [20]byte{}
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[position:])
}

const pageStyles = `
:root{color-scheme:light;--bg:#f6f8fb;--surface:#fff;--text:#1f2937;--muted:#64748b;--line:#e2e8f0;--accent:#2563eb;--code:#0f172a}
*{box-sizing:border-box}html{scroll-behavior:smooth}body{margin:0;background:var(--bg);color:var(--text);font:15px/1.7 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif}
.layout{display:grid;grid-template-columns:280px minmax(0,1fr);max-width:1500px;margin:0 auto;min-height:100vh}aside{border-right:1px solid var(--line);background:var(--surface)}
.toc{position:sticky;top:0;height:100vh;overflow:auto;padding:24px 20px}.toc-title{font-size:20px;font-weight:700;margin-bottom:12px}.toc ul{list-style:none;padding:0;margin:0}.toc li{margin:2px 0}.toc .level-2{padding-left:12px}.toc .level-3{padding-left:24px}.toc a{display:block;padding:5px 8px;border-radius:6px;color:var(--muted);text-decoration:none}.toc a:hover{background:#eff6ff;color:var(--accent)}
.content{min-width:0;background:var(--surface);padding:36px 48px 80px}.content h1{font-size:34px;margin:0 0 24px}.content h2{font-size:25px;margin:42px 0 16px;padding-bottom:8px;border-bottom:1px solid var(--line)}.content h3{font-size:19px;margin:30px 0 12px}.content h4{font-size:16px}.content p{margin:10px 0}.content ul{padding-left:24px}
code{padding:2px 5px;border-radius:5px;background:#eef2ff;color:#3730a3;font:13px/1.5 ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace}pre{overflow:auto;padding:16px;border-radius:10px;background:var(--code);color:#e2e8f0}pre code{padding:0;background:transparent;color:inherit}
.table-wrap{overflow:auto;margin:14px 0 22px;border:1px solid var(--line);border-radius:9px}table{width:100%;border-collapse:collapse;background:var(--surface)}th,td{padding:10px 12px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top;white-space:normal}th{background:#f8fafc;font-weight:650}tr:last-child td{border-bottom:0}a{color:var(--accent)}
@media(max-width:900px){.layout{display:block}aside{border-right:0;border-bottom:1px solid var(--line)}.toc{position:relative;height:auto;max-height:320px}.content{padding:24px 18px 60px}.content h1{font-size:28px}}
`

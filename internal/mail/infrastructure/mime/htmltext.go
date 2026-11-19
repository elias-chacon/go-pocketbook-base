package mime

import (
	"html"
	"regexp"
	"strings"
)

var (
	dropBlocks   = regexp.MustCompile(`(?is)<(script|style|head)\b.*?</(script|style|head)\s*>|<!--.*?-->`)
	lineBreaks   = regexp.MustCompile(`(?i)<br\s*/?>|</(div|tr|h[1-6]|section|article)\s*>`)
	paragraphs   = regexp.MustCompile(`(?i)</(p|table|ul|ol|blockquote)\s*>`)
	listItems    = regexp.MustCompile(`(?i)<li\b[^>]*>`)
	anyTag       = regexp.MustCompile(`(?s)<[^>]*>`)
	horizSpace   = regexp.MustCompile(`[ \t\f\v\x{00a0}]+`)
	manyNewlines = regexp.MustCompile(`\n{3,}`)
)

// htmlToText turns the HTML body of a message into plain text for a screen that
// shows no markup: blocks become lines, list items get a bullet, scripts and
// styles disappear and entities are decoded. It is a reader's view, not a faithful
// rendering.
func htmlToText(s string) string {
	s = dropBlocks.ReplaceAllString(s, "")
	s = paragraphs.ReplaceAllString(s, "\n\n")
	s = lineBreaks.ReplaceAllString(s, "\n")
	s = listItems.ReplaceAllString(s, "\n• ")
	s = anyTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")

	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(horizSpace.ReplaceAllString(l, " "))
	}
	out := manyNewlines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	return strings.TrimSpace(out)
}

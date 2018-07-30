package util

// MarkdownLink returns a markdown link
func MarkdownLink(text string, url string) string {
	if url != "" {
		if text == "" {
			text = url
		}
		return "[" + text + "](" + url + ")"
	} else {
		return text
	}
}

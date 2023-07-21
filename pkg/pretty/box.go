package pretty

import (
	"fmt"
	"strings"
	"unicode"
)

func BoxItems(title string, items []string, color string) string {

	var (
		boxStart     = Color(fmt.Sprintf("[%s][bold]├─", color))
		boxLine      = Color(fmt.Sprintf("[%s][bold]│", color))
		boxSeparator = Color(fmt.Sprintf("[%s][bold]├─", color))
		boxEnd       = Color(fmt.Sprintf("[%s][bold]└─", color))
	)

	var boxed strings.Builder

	if len(title) > 0 {
		title = Color(fmt.Sprintf("[%s][bold]%s", color, title))
		boxed.WriteString(boxStart + " " + title + "\n" + boxLine + "\n")
	} else {
		boxed.WriteString(boxStart + "\n")
	}

	for i, item := range items {
		if i > 0 {
			boxed.WriteString(boxSeparator + "\n")
		}
		boxed.WriteString(prefixLines(item, boxLine+" ") + "\n")
	}

	boxed.WriteString(boxEnd)

	return boxed.String()
}

func BoxSection(title, content, color string) string {

	var (
		boxStart = Color(fmt.Sprintf("[%s][bold]┌─", color))
		boxLine  = Color(fmt.Sprintf("[%s][bold]│", color))
		boxEnd   = Color(fmt.Sprintf("[%s][bold]└─", color))
	)

	var boxed strings.Builder

	if len(title) > 0 {
		title = Color(fmt.Sprintf("[%s][bold]%s", color, title))
		boxed.WriteString(boxStart + " " + title + "\n" + boxLine + "\n")
	} else {
		boxed.WriteString(boxStart + "\n")
	}

	boxed.WriteString(prefixLines(content, boxLine+" ") + "\n")
	boxed.WriteString(boxEnd)

	return boxed.String()
}

func prefixLines(text string, prefix string) string {
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		lines[i] = prefix + line
		lines[i] = trimTrailingWhitespace(lines[i])
	}

	return strings.Join(lines, "\n")
}

func trimTrailingWhitespace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

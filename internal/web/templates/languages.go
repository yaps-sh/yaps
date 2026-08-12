package templates

import (
	"github.com/alecthomas/chroma/v3/lexers"
)

var PrimaryLanguages = []string{
	"go", "python3", "javascript", "typescript", "json", "html", "css",
	"bash", "sql", "rust", "java", "c", "c++", "yaml", "markdown",
}

func AllLanguages() []string {
	primary := make(map[string]bool, len(PrimaryLanguages))
	for _, lang := range PrimaryLanguages {
		primary[lang] = true
	}

	var rest []string
	for _, lang := range lexers.Names(false) {
		if !primary[lang] {
			rest = append(rest, lang)
		}
	}
	return rest
}
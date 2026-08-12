package web

import (
	"log/slog"
	"strings"

	"github.com/alecthomas/chroma/v3"
	"github.com/alecthomas/chroma/v3/formatters/html"
	"github.com/alecthomas/chroma/v3/lexers"
	"github.com/alecthomas/chroma/v3/styles"
)

var (
	formatter = html.New(html.WithClasses(true), html.WithLineNumbers(true))
	style     = styles.Get("nord")
)

func resolveLexer(lang, ext string) chroma.Lexer {
	if ext != "" {
		slog.Debug("resolving lexer for extension", "ext", ext)
		if lexer := lexers.Get(ext); lexer != nil {
			slog.Debug("found lexer for extension", "ext", ext)
			return lexer
		}
	}

	if lexer := lexers.Get(lang); lexer != nil {
		slog.Debug("found lexer for language", "lang", lang)
		return lexer
	}

	slog.Debug("no lexer found for language or extension", "lang", lang, "ext", ext)
	return lexers.Fallback
}

func highlight(lang, ext, content string) (string, error) {
	lexer := chroma.Coalesce(resolveLexer(lang, ext))
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func highlightCSS() (string, error) {
	var buf strings.Builder
	if err := formatter.WriteCSS(&buf, style); err != nil {
		return "", err
	}

	return buf.String(), nil
}

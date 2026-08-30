package rest

import (
	"strings"

	"golang.org/x/net/html"
)

var allowedHTMLTags = map[string]bool{
	"p": true, "br": true, "hr": true,
	"em": true, "i": true, "strong": true, "b": true, "u": true, "s": true,
	"blockquote": true,
	"h1":         true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"ul": true, "ol": true, "li": true,
	"div": true, "span": true,
}

var voidHTMLTags = map[string]bool{"br": true, "hr": true}

var dropContentTags = map[string]bool{
	"script": true, "style": true, "head": true, "title": true,
	"noscript": true, "iframe": true, "template": true,
}

func sanitizeNovelHTML(in string) string {
	z := html.NewTokenizer(strings.NewReader(in))
	var b strings.Builder
	skipDepth := 0
	skipTag := ""

	for {
		switch z.Next() {
		case html.ErrorToken:
			return b.String()

		case html.TextToken:
			if skipDepth == 0 {
				b.WriteString(html.EscapeString(string(z.Text())))
			}

		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := z.TagName()
			tag := string(name)
			if skipDepth > 0 {
				if tag == skipTag {
					skipDepth++
				}
				continue
			}
			if dropContentTags[tag] {
				skipDepth = 1
				skipTag = tag
				continue
			}
			if allowedHTMLTags[tag] {
				b.WriteByte('<')
				b.WriteString(tag)
				b.WriteByte('>')
			}

		case html.EndTagToken:
			name, _ := z.TagName()
			tag := string(name)
			if skipDepth > 0 {
				if tag == skipTag {
					skipDepth--
					if skipDepth == 0 {
						skipTag = ""
					}
				}
				continue
			}
			if allowedHTMLTags[tag] && !voidHTMLTags[tag] {
				b.WriteString("</")
				b.WriteString(tag)
				b.WriteByte('>')
			}
		}
	}
}

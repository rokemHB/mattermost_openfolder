package main

import (
	"net/url"
	"regexp"
	"strings"
)

// Converter turns UNC paths on the allowed servers into Markdown links that
// show the original path but point to openfolder://, e.g.
//
//	\\fileserver.example.com\share\Projects 2026
//
// becomes
//
//	[`\\fileserver.example.com\share\Projects 2026`](openfolder://fileserver.example.com/share/Projects%202026)
//
// Recognised forms:
//   - bare path: runs to the end of the line (spaces allowed); it stops early
//     before another path or URL, and trailing punctuation is trimmed
//   - path in "quotes", „German quotes“ or `backticks`: exactly the quoted text
//   - file://server/... URLs
//
// Fenced code blocks and existing Markdown links are left untouched, so
// converting an already converted message changes nothing.
type Converter struct {
	reLink     *regexp.Regexp
	reCode     *regexp.Regexp
	reQuoted   *regexp.Regexp
	reFile     *regexp.Regexp
	reBare     *regexp.Regexp
	rePath     *regexp.Regexp // whole string is a UNC path on an allowed server
	reFileFull *regexp.Regexp
	reCut      *regexp.Regexp
}

func NewConverter(hosts []string) *Converter {
	var quoted []string
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h != "" {
			quoted = append(quoted, regexp.QuoteMeta(h))
		}
	}
	if len(quoted) == 0 {
		return nil
	}
	host := `(?:` + strings.Join(quoted, "|") + `)`
	unc := `\\\\` + host + `\\[^\\\s]`

	return &Converter{
		reLink:     regexp.MustCompile("^!?\\[((?:`[^`]*`|\\\\.|[^\\]\\\\`])*)\\]\\(([^)\\s]*)\\)"),
		reCode:     regexp.MustCompile("^`([^`]+)`"),
		reQuoted:   regexp.MustCompile(`(?i)^(["“„»«])(` + unc + `[^"“”„»«]*?)(["”“«»])`),
		reFile:     regexp.MustCompile(`(?i)^file:/{2,5}` + host + `/[^\s<>"'\]]+`),
		reBare:     regexp.MustCompile(`(?i)^` + unc),
		rePath:     regexp.MustCompile(`(?i)^` + unc + `.*$`),
		reFileFull: regexp.MustCompile(`(?i)^file:/{2,5}` + host + `/\S+$`),
		reCut:      regexp.MustCompile(`(?i)\s+(?:\\\\|file:/|https?://|openfolder:)`),
	}
}

func (c *Converter) Convert(msg string) string {
	if c == nil || msg == "" {
		return msg
	}
	lines := strings.SplitAfter(msg, "\n")
	var out strings.Builder
	fence := ""
	for _, line := range lines {
		body := strings.TrimRight(line, "\r\n")
		nl := line[len(body):]
		trimmed := strings.TrimLeft(body, " ")
		if len(body)-len(trimmed) <= 3 {
			if fence == "" && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
				fence = trimmed[:3]
				out.WriteString(line)
				continue
			}
			if fence != "" && strings.HasPrefix(trimmed, fence) {
				fence = ""
				out.WriteString(line)
				continue
			}
		}
		if fence != "" {
			out.WriteString(line)
			continue
		}
		out.WriteString(c.convertLine(body))
		out.WriteString(nl)
	}
	return out.String()
}

func (c *Converter) convertLine(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		rest := s[i:]

		if m := c.reLink.FindStringSubmatch(rest); m != nil {
			out.WriteString(c.upgradeLink(m[0], m[1], m[2]))
			i += len(m[0])
			continue
		}

		if m := c.reCode.FindStringSubmatch(rest); m != nil {
			inner := strings.TrimSpace(m[1])
			switch {
			case c.rePath.MatchString(inner):
				out.WriteString(link(inner))
			case c.reFileFull.MatchString(inner):
				out.WriteString(link(fileURLToUNC(inner)))
			default:
				out.WriteString(m[0])
			}
			i += len(m[0])
			continue
		}

		if m := c.reQuoted.FindStringSubmatch(rest); m != nil {
			out.WriteString(m[1] + link(m[2]) + m[3])
			i += len(m[0])
			continue
		}

		if m := c.reFile.FindString(rest); m != "" {
			m = trimTrailing(m)
			out.WriteString(link(fileURLToUNC(m)))
			i += len(m)
			continue
		}

		if c.reBare.MatchString(rest) {
			p := rest
			if loc := c.reCut.FindStringIndex(p); loc != nil {
				p = p[:loc[0]]
			}
			p = trimTrailing(p)
			out.WriteString(link(p))
			i += len(p)
			continue
		}

		// copy one byte; multi-byte runes are copied byte by byte unchanged
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

// trimTrailing removes whitespace and punctuation that most likely belongs to
// the sentence, not to the path. Closing brackets are only removed when they
// have no matching opening bracket inside the path, so "Projekt (alt)" stays.
func trimTrailing(p string) string {
	for {
		before := p
		p = strings.TrimRight(p, " \t.,;:!?*_~\"'”“»«")
		if strings.HasSuffix(p, ")") && strings.Count(p, "(") < strings.Count(p, ")") {
			p = p[:len(p)-1]
		}
		if strings.HasSuffix(p, "]") && strings.Count(p, "[") < strings.Count(p, "]") {
			p = p[:len(p)-1]
		}
		if p == before {
			return p
		}
	}
}

func fileURLToUNC(u string) string {
	p := u[strings.Index(u, ":")+1:]
	p = strings.TrimLeft(p, "/")
	if dec, err := url.PathUnescape(p); err == nil {
		p = dec
	}
	return `\\` + strings.ReplaceAll(p, "/", `\`)
}

// link builds [`\\server\share\...`](openfolder://server/share/...).
//
// The path is shown as inline code: inside a code span Markdown takes every
// backslash literally, so the path is displayed exactly as typed. (Escaping
// the backslashes instead is rendered inconsistently by Mattermost.)
func link(unc string) string {
	parts := strings.Split(strings.TrimLeft(unc, `\`), `\`)
	var segs []string
	for _, s := range parts {
		if s != "" {
			segs = append(segs, url.PathEscape(s))
		}
	}
	return "[" + codeSpan(unc) + "](openfolder://" + strings.Join(segs, "/") + ")"
}

func codeSpan(s string) string {
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	return "`` " + s + " ``"
}

var reUnescape = regexp.MustCompile(`\\([!-/:-@\[-` + "`" + `{-~])`)

// upgradeLink leaves Markdown links alone, except openfolder links written by
// version 1.0.0 with backslash-escaped text (\\\\server\\share): those are
// rewritten in the new format when the message is edited.
func (c *Converter) upgradeLink(whole, text, target string) string {
	if !strings.HasPrefix(strings.ToLower(target), "openfolder:") || !strings.HasPrefix(text, `\\\\`) {
		return whole
	}
	unc := reUnescape.ReplaceAllString(text, "$1")
	if !c.rePath.MatchString(unc) {
		return whole
	}
	return link(unc)
}

package organize

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultNameTemplate is the canonical folder layout: a studio folder holding a
// scene folder named Studio.Date.Cast.Title.XXX.FOV.Heightp.
const DefaultNameTemplate = "{studio}/{studio}.{date}.{cast}.{title}.XXX.{fov}.{height}p"

// layoutTokens lists the tokens accepted by ValidateTemplate.
var layoutTokens = map[string]bool{
	"studio": true, "site": true,
	"date": true, "yy": true, "mm": true, "dd": true, "year": true,
	"cast": true, "title": true, "fov": true, "height": true,
}

var reToken = regexp.MustCompile(`\{([A-Za-z]+)\}`)
var reDotRun = regexp.MustCompile(`\.{2,}`)

// templateValues holds the per-scene values substituted into a name template.
type templateValues struct {
	Studio string
	Date   time.Time
	Cast   string
	Title  string
	FOV    string
	Height int
}

// tokenValue returns the replacement for one layout token (name already lower-cased).
func (v templateValues) tokenValue(name string) string {
	switch name {
	case "studio", "site":
		return v.Studio
	case "date":
		return v.Date.Format("06.01.02")
	case "yy":
		return v.Date.Format("06")
	case "mm":
		return v.Date.Format("01")
	case "dd":
		return v.Date.Format("02")
	case "year":
		return v.Date.Format("2006")
	case "cast":
		return v.Cast
	case "title":
		return v.Title
	case "fov":
		return v.FOV
	case "height":
		return strconv.Itoa(v.Height)
	}
	return ""
}

// renderTemplate renders a relative folder path template for one scene. It returns
// the slash-joined path and its final segment (the scene folder name). Empty
// segments are dropped so optional tokens such as {cast} don't leave stray dots or
// slashes. The bool is false when nothing usable remains.
func renderTemplate(tmpl string, v templateValues) (rel, dirName string, ok bool) {
	if tmpl == "" {
		tmpl = DefaultNameTemplate
	}
	var segs []string
	for _, raw := range strings.Split(tmpl, "/") {
		seg := reToken.ReplaceAllStringFunc(raw, func(m string) string {
			return v.tokenValue(strings.ToLower(m[1 : len(m)-1]))
		})
		// Collapse dot runs left by an empty token between separators and trim the
		// edges so a folder name never starts or ends with a dot.
		seg = reDotRun.ReplaceAllString(seg, ".")
		seg = strings.Trim(seg, " .")
		if seg == "" || seg == "." || seg == ".." {
			continue
		}
		segs = append(segs, seg)
	}
	if len(segs) == 0 {
		return "", "", false
	}
	return strings.Join(segs, "/"), segs[len(segs)-1], true
}

// ValidateTemplate checks a user-supplied layout template before it is stored.
// An empty template means the default layout. It rejects absolute paths, empty
// segments, dot segments, and unknown tokens.
func ValidateTemplate(tmpl string) error {
	if tmpl == "" {
		return nil
	}
	if strings.HasPrefix(strings.TrimSpace(tmpl), "/") {
		return fmt.Errorf("layout must be a relative path")
	}
	for _, raw := range strings.Split(tmpl, "/") {
		seg := strings.TrimSpace(raw)
		if seg == "" {
			return fmt.Errorf("layout contains an empty folder name")
		}
		if seg == "." || seg == ".." {
			return fmt.Errorf("layout may not contain %q segments", seg)
		}
		var unknown string
		out := reToken.ReplaceAllStringFunc(seg, func(m string) string {
			name := strings.ToLower(m[1 : len(m)-1])
			if !layoutTokens[name] {
				if unknown == "" {
					unknown = name
				}
				return m
			}
			return ""
		})
		if unknown != "" {
			return fmt.Errorf("unknown layout token {%s}", unknown)
		}
		if strings.ContainsAny(out, "{}") {
			return fmt.Errorf("unrecognised token in %q", seg)
		}
	}
	return nil
}

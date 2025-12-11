package ttml

import (
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Types mapped from the original TypeScript definitions

// TTMLMetadata represents a key with multiple values.
type TTMLMetadata struct {
	Key   string   `json:"key"`
	Value []string `json:"value"`
}

// TTMLLyric is the parsed result.
type TTMLLyric struct {
	Metadata   []TTMLMetadata `json:"metadata"`
	LyricLines []LyricLine    `json:"lyricLines"`
}

// LyricWord is a single word (or token) with timing.
type LyricWord struct {
	StartTime int    `json:"startTime"`           // milliseconds
	EndTime   int    `json:"endTime"`             // milliseconds
	Word      string `json:"word"`                // the text
	EmptyBeat *int   `json:"emptyBeat,omitempty"` // optional
}

// LyricLine is a line composed of words and additional information.
type LyricLine struct {
	Words           []LyricWord `json:"words"`
	TranslatedLyric string      `json:"translatedLyric"`
	RomanLyric      string      `json:"romanLyric"`
	IsBG            bool        `json:"isBG"`
	IsDuet          bool        `json:"isDuet"`
	StartTime       int         `json:"startTime"` // milliseconds
	EndTime         int         `json:"endTime"`   // milliseconds
	BGs             []LyricLine `json:"bgs"`
}

// ParseTTML parses a TTML string (XML) into a TTMLLyric structure.
//
// This is a conversion of the provided TypeScript parser into Go. It builds
// a simple DOM-like tree from the XML and walks it to extract metadata,
// agents, and <body><p begin end> lines with words, translations, romanizations,
// and background (x-bg) spans.
//
// Note: XML namespace prefixes are not preserved by encoding/xml (prefixes are
// mapped to namespace URIs). This implementation matches element and attribute
// names by their local name (e.g. "meta", "agent", "begin", "end", "role")
// which is sufficient for typical TTML from Apple Music.
func ParseTTML(ttmlText string) (TTMLLyric, error) {
	decoder := xml.NewDecoder(strings.NewReader(ttmlText))
	decoder.Strict = false

	root, err := buildTree(decoder)
	if err != nil {
		return TTMLLyric{}, err
	}

	// Validate root has tt element at top-level
	if root == nil || !hasElementChild(root, "tt") {
		return TTMLLyric{}, errors.New("不是有效的 TTML 文档")
	}

	mainAgentId := "v1"

	// metadata
	metadata := []TTMLMetadata{}
	for _, meta := range findAll(root, "meta") {
		// original TypeScript checked meta.tagName === "amll:meta". Here we match local name "meta"
		key := attr(meta, "key")
		if key != "" {
			value := attr(meta, "value")
			if value != "" {
				idx := -1
				for i := range metadata {
					if metadata[i].Key == key {
						idx = i
						break
					}
				}
				if idx >= 0 {
					metadata[idx].Value = append(metadata[idx].Value, value)
				} else {
					metadata = append(metadata, TTMLMetadata{
						Key:   key,
						Value: []string{value},
					})
				}
			}
		}
	}

	// find main agent id from agent elements with type="person"
	for _, agent := range findAll(root, "agent") {
		if attr(agent, "type") == "person" {
			// xml:id usually parsed as attribute with local name "id"
			if id := attr(agent, "id"); id != "" {
				mainAgentId = id
				break
			}
			// fallback: attribute named "xml:id" may not appear as such; try "xml:id" key
			if id := attr(agent, "xml:id"); id != "" {
				mainAgentId = id
				break
			}
		}
	}

	lyricLines := []LyricLine{}

	// find all <p> elements under body with begin and end attributes
	for _, p := range findAll(root, "p") {
		if attr(p, "begin") != "" && attr(p, "end") != "" {
			parseParseLine(p, &lyricLines, mainAgentId, false, false)
		}
	}

	/*return TTMLLyric{
		Metadata:   metadata,
		LyricLines: lyricLines,
	}, nil*/
	var merged []LyricLine
	for i := 0; i < len(lyricLines); {
		line := lyricLines[i]

		if line.IsBG {
			// 如果遇到单独的 BG 行而前面没有主行，就跳过
			i++
			continue
		}

		// 收集后面连续的 BG 行
		var bgs []LyricLine
		j := i + 1
		for j < len(lyricLines) && lyricLines[j].IsBG {
			bgs = append(bgs, lyricLines[j])
			j++
		}

		line.BGs = bgs
		merged = append(merged, line)
		i = j
	}

	return TTMLLyric{
		Metadata:   metadata,
		LyricLines: merged,
	}, nil

}

// ---------- internal node tree representation & helpers ----------

type nodeType int

const (
	elementNode nodeType = iota
	textNode
)

type node struct {
	Typ      nodeType
	Name     string            // local name for elements
	Attrs    map[string]string // attribute local names -> values
	Children []*node
	Text     string // for text nodes
}

func buildTree(decoder *xml.Decoder) (*node, error) {
	root := &node{Typ: elementNode, Name: "root", Attrs: map[string]string{}}
	stack := []*node{root}

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			n := &node{
				Typ:   elementNode,
				Name:  t.Name.Local,
				Attrs: map[string]string{},
			}
			for _, a := range t.Attr {
				// store attributes by local name
				// if multiple attrs share same local name across namespaces, later one will overwrite.
				// This matches a pragmatic approach used in the TS version which matched by local names.
				n.Attrs[a.Name.Local] = a.Value
				// Also keep the raw "prefix:local" if the original Name contains a colon in the input
				// encoding/xml doesn't provide the prefix. So we cannot reconstruct it here.
			}
			// append to parent
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, n)
			// push
			stack = append(stack, n)

		case xml.EndElement:
			// pop stack
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			text := string([]byte(t))
			// trim preserving whitespace inside words (we will use text content directly)
			if len(text) > 0 {
				txt := &node{Typ: textNode, Text: text}
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, txt)
			}
		case xml.Comment:
			// ignore
		default:
			_ = t
		}
	}

	return root, nil
}

func findAll(n *node, name string) []*node {
	var out []*node
	var walk func(*node)
	walk = func(cur *node) {
		if cur.Typ == elementNode && cur.Name == name {
			out = append(out, cur)
		}
		for _, c := range cur.Children {
			if c.Typ == elementNode {
				walk(c)
			}
		}
	}
	if n != nil {
		walk(n)
	}
	return out
}

func attr(n *node, key string) string {
	if n == nil {
		return ""
	}
	// common variations
	if v, ok := n.Attrs[key]; ok {
		return v
	}
	// some inputs might include the prefix in attribute name (unlikely with encoding/xml),
	// check a few likely alternatives:
	if v, ok := n.Attrs["xml:"+key]; ok {
		return v
	}
	if v, ok := n.Attrs["ttm:"+key]; ok {
		return v
	}
	if v, ok := n.Attrs["amll:"+key]; ok {
		return v
	}
	return ""
}

func innerText(n *node) string {
	var b strings.Builder
	var walk func(*node)
	walk = func(cur *node) {
		if cur.Typ == textNode {
			b.WriteString(cur.Text)
		}
		for _, c := range cur.Children {
			walk(c)
		}
	}
	if n != nil {
		walk(n)
	}
	return b.String()
}

func hasElementChild(n *node, name string) bool {
	for _, c := range n.Children {
		if c.Typ == elementNode && c.Name == name {
			return true
		}
	}
	return false
}

// ---------- parsing logic converted from TypeScript ----------

var timeRegexp = regexp.MustCompile(`^(.+)$`) // not used; kept for clarity

// parseTimespan accepts formats like "hh:mm:ss.sss", "mm:ss.sss", "ss.sss"
// uses '.' or ':' as separators between seconds and fractional part.
func parseTimespan(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty timespan")
	}
	// trim spaces
	ts := strings.TrimSpace(s)
	// sometimes TTML might use comma as decimal separator
	ts = strings.ReplaceAll(ts, ",", ".")
	parts := strings.Split(ts, ":")

	var hours, mins int64
	var secs float64
	var err error

	switch len(parts) {
	case 3:
		hours, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, err
		}
		mins, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, err
		}
		secs, err = strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return 0, err
		}
	case 2:
		hours = 0
		mins, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, err
		}
		secs, err = strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, err
		}
	case 1:
		hours = 0
		mins = 0
		secs, err = strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
	default:
		// fallback attempt: try to parse as float seconds
		f, ferr := strconv.ParseFloat(ts, 64)
		if ferr == nil {
			secs = f
		} else {
			return 0, errors.New("时间戳字符串解析失败：" + s)
		}
	}

	ms := int((hours*3600+mins*60)*1000 + int64(secs*1000.0))
	return ms, nil
}

// parseParseLine is analogous to the TypeScript parseParseLine function.
// It appends parsed lines to lyricLines slice.
func parseParseLine(lineEl *node, lyricLines *[]LyricLine, mainAgentId string, isBG bool, isDuet bool) {
	line := LyricLine{
		Words:           []LyricWord{},
		TranslatedLyric: "",
		RomanLyric:      "",
		IsBG:            isBG,
		IsDuet:          false,
		StartTime:       0,
		EndTime:         0,
	}

	// initial duet detection: presence of ttm:agent attribute and not equal to mainAgentId
	if a := attr(lineEl, "agent"); a != "" && a != mainAgentId {
		line.IsDuet = true
	}
	// override if provided by caller (for background spans)
	if isBG {
		line.IsDuet = isDuet
	}

	haveBg := false

	for _, child := range lineEl.Children {
		if child.Typ == textNode {
			// push text nodes as words (may contain whitespace)
			w := LyricWord{
				Word:      child.Text,
				StartTime: 0,
				EndTime:   0,
			}
			line.Words = append(line.Words, w)
			continue
		}

		// element node
		role := attr(child, "role")
		// handle span with role
		if child.Name == "span" && role != "" {
			if role == "x-bg" {
				// recursively parse bg span
				parseParseLine(child, lyricLines, mainAgentId, true, line.IsDuet)
				haveBg = true
			} else if role == "x-translation" {
				// set translatedLyric to inner content (we use innerText)
				line.TranslatedLyric = innerText(child)
			} else if role == "x-roman" {
				line.RomanLyric = innerText(child)
			} else {
				// other span roles - attempt to treat as word if it has begin & end
				if attr(child, "begin") != "" && attr(child, "end") != "" {
					beginStr := attr(child, "begin")
					endStr := attr(child, "end")
					startMs, err1 := parseTimespan(beginStr)
					endMs, err2 := parseTimespan(endStr)
					if err1 == nil && err2 == nil {
						w := LyricWord{
							Word:      innerText(child),
							StartTime: startMs,
							EndTime:   endMs,
						}
						if eb := attr(child, "empty-beat"); eb != "" {
							if v, err := strconv.Atoi(eb); err == nil {
								w.EmptyBeat = &v
							}
						}
						line.Words = append(line.Words, w)
					} else {
						// fallback: push as plain text
						line.Words = append(line.Words, LyricWord{
							Word: innerText(child),
						})
					}
				} else {
					// span without begin/end - treat content as plain text words (concatenate)
					txt := innerText(child)
					if txt != "" {
						line.Words = append(line.Words, LyricWord{
							Word: txt,
						})
					}
				}
			}
			continue
		}

		// element with begin & end (e.g. <span begin="..." end="..."> or <p> inside)
		if attr(child, "begin") != "" && attr(child, "end") != "" {
			beginStr := attr(child, "begin")
			endStr := attr(child, "end")
			startMs, err1 := parseTimespan(beginStr)
			endMs, err2 := parseTimespan(endStr)
			w := LyricWord{
				Word: innerText(child),
			}
			if err1 == nil {
				w.StartTime = startMs
			}
			if err2 == nil {
				w.EndTime = endMs
			}
			if eb := attr(child, "empty-beat"); eb != "" {
				if v, err := strconv.Atoi(eb); err == nil {
					w.EmptyBeat = &v
				}
			}
			line.Words = append(line.Words, w)
			continue
		}

		// Other element types: recursively pull their inner text as a plain word
		txt := innerText(child)
		if txt != "" {
			line.Words = append(line.Words, LyricWord{
				Word: txt,
			})
		}
	}

	// BG trim parentheses as TS code did
	if line.IsBG && len(line.Words) > 0 {
		// trim leading "(" from first word
		first := &line.Words[0]
		if strings.HasPrefix(first.Word, "(") {
			first.Word = strings.TrimPrefix(first.Word, "(")
			if len(strings.TrimSpace(first.Word)) == 0 {
				// remove it
				if len(line.Words) > 1 {
					line.Words = line.Words[1:]
				} else {
					line.Words = []LyricWord{}
				}
			}
		}
		// trim trailing ")" from last word
		if len(line.Words) > 0 {
			lastIdx := len(line.Words) - 1
			last := &line.Words[lastIdx]
			if strings.HasSuffix(last.Word, ")") {
				last.Word = strings.TrimSuffix(last.Word, ")")
				if len(strings.TrimSpace(last.Word)) == 0 {
					// pop
					line.Words = line.Words[:lastIdx]
				}
			}
		}
	}

	// determine startTime and endTime for the line
	if attr(lineEl, "begin") != "" && attr(lineEl, "end") != "" {
		if st, err := parseTimespan(attr(lineEl, "begin")); err == nil {
			line.StartTime = st
		}
		if et, err := parseTimespan(attr(lineEl, "end")); err == nil {
			line.EndTime = et
		}
	} else {
		// compute from words with non-empty trimmed words
		minStart := int(^uint(0) >> 1) // large int
		maxEnd := 0
		hasAny := false
		for _, w := range line.Words {
			if strings.TrimSpace(w.Word) == "" {
				continue
			}
			hasAny = true
			if w.StartTime > 0 && w.StartTime < minStart {
				minStart = w.StartTime
			}
			if w.EndTime > maxEnd {
				maxEnd = w.EndTime
			}
		}
		if hasAny {
			if minStart == int(^uint(0)>>1) {
				minStart = 0
			}
			line.StartTime = minStart
			line.EndTime = maxEnd
		}
	}

	if haveBg {
		// the TypeScript logic popped the last line and inserted the bg line before it.
		// Emulate by moving the bg line before the previously appended line.
		// Here we append the bg line in place of the last pushed, then reappend the previous last.
		lns := *lyricLines
		var last *LyricLine
		if len(lns) > 0 {
			lastVal := lns[len(lns)-1]
			last = &lastVal
			lns = lns[:len(lns)-1]
		}
		lns = append(lns, line)
		if last != nil {
			lns = append(lns, *last)
		}
		*lyricLines = lns
	} else {
		*lyricLines = append(*lyricLines, line)
	}
}

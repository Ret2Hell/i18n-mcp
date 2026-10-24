package validate

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxICUBytes = 1 << 20
const maxICUDepth = 32
const maxICUNodes = 4096

// icuNode is an argument or literal message fragment. Branch bodies are parsed
// as messages, never reinterpreted as standalone brace placeholders.
type icuNode struct {
	text              string
	name, kind, style string
	offset            uint64
	branches          map[string][]icuNode
}

type icuParser struct {
	ctx   context.Context
	input string
	pos   int
	nodes int
}

func parseICU(ctx context.Context, input string) ([]icuNode, error) {
	if len(input) > maxICUBytes {
		return nil, fmt.Errorf("ICU message exceeds %d bytes", maxICUBytes)
	}
	if !utf8.ValidString(input) {
		return nil, fmt.Errorf("ICU message is not valid UTF-8")
	}
	p := icuParser{ctx: ctx, input: input}
	nodes, err := p.message(0, false, false)
	if err != nil {
		return nil, err
	}
	return nodes, ctx.Err()
}

func (p *icuParser) fail(message string) error { return fmt.Errorf("%s at byte %d", message, p.pos) }
func (p *icuParser) space() {
	for p.pos < len(p.input) {
		r, n := utf8.DecodeRuneInString(p.input[p.pos:])
		if !unicode.IsSpace(r) {
			break
		}
		p.pos += n
	}
}
func (p *icuParser) take(c byte) bool {
	if p.pos < len(p.input) && p.input[p.pos] == c {
		p.pos++
		return true
	}
	return false
}
func (p *icuParser) identifier() string {
	start := p.pos
	for p.pos < len(p.input) {
		r, n := utf8.DecodeRuneInString(p.input[p.pos:])
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		p.pos += n
	}
	return p.input[start:p.pos]
}

func (p *icuParser) message(depth int, nested, plural bool) ([]icuNode, error) {
	if depth > maxICUDepth {
		return nil, p.fail("ICU nesting limit exceeded")
	}
	var nodes []icuNode
	start := p.pos
	flush := func() {
		if p.pos > start {
			nodes = append(nodes, icuNode{text: p.input[start:p.pos]})
		}
	}
	for p.pos < len(p.input) {
		if err := p.ctx.Err(); err != nil {
			return nil, err
		}
		switch p.input[p.pos] {
		case '\'':
			p.quoted(plural)
		case '{':
			flush()
			node, err := p.argument(depth, plural)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
			start = p.pos
		case '}':
			if !nested {
				return nil, p.fail("unexpected closing brace")
			}
			flush()
			p.pos++
			return nodes, nil
		default:
			p.pos++
		}
	}
	if nested {
		return nil, p.fail("unclosed branch body")
	}
	flush()
	return nodes, nil
}

// ICU apostrophe-friendly quoting starts only before syntax; ordinary apostrophes
// in contractions remain literal. A doubled apostrophe always represents one.
func (p *icuParser) quoted(plural bool) {
	p.pos++
	if p.pos == len(p.input) {
		return
	}
	next := p.input[p.pos]
	if next == '\'' {
		p.pos++
		return
	}
	if next != '{' && next != '}' && (!plural || next != '#') {
		return
	}
	for p.pos < len(p.input) {
		if p.input[p.pos] == '\'' {
			p.pos++
			if p.pos < len(p.input) && p.input[p.pos] == '\'' {
				p.pos++
				continue
			}
			return
		}
		p.pos++
	}
	// ICU closes a quote implicitly at the end of the message.
}

func (p *icuParser) argument(depth int, inPlural bool) (icuNode, error) {
	p.nodes++
	if p.nodes > maxICUNodes {
		return icuNode{}, p.fail("ICU node limit exceeded")
	}
	p.pos++
	p.space()
	node := icuNode{name: p.identifier(), kind: "argument"}
	if node.name == "" {
		return node, p.fail("expected argument name")
	}
	p.space()
	if p.take('}') {
		return node, nil
	}
	if !p.take(',') {
		return node, p.fail("expected comma or closing brace")
	}
	p.space()
	node.kind = p.identifier()
	p.space()
	switch node.kind {
	case "number", "date", "time":
		if p.take('}') {
			return node, nil
		}
		if !p.take(',') {
			return node, p.fail("expected format style")
		}
		p.space()
		start := p.pos
		for p.pos < len(p.input) && p.input[p.pos] != '}' {
			if p.input[p.pos] == '{' {
				return node, p.fail("braces are not supported inside format styles")
			}
			p.pos++
		}
		node.style = strings.TrimSpace(p.input[start:p.pos])
		if node.style == "" {
			return node, p.fail("empty format style")
		}
		if !p.take('}') {
			return node, p.fail("unclosed formatted argument")
		}
		return node, nil
	case "plural", "selectordinal", "select":
		if !p.take(',') {
			return node, p.fail("expected selector list")
		}
	default:
		return node, p.fail("unsupported ICU argument type " + node.kind)
	}
	p.space()
	if node.kind != "select" && strings.HasPrefix(p.input[p.pos:], "offset:") {
		p.pos += len("offset:")
		p.space()
		start := p.pos
		for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
			p.pos++
		}
		offset, err := strconv.ParseUint(p.input[start:p.pos], 10, 64)
		if err != nil {
			return node, p.fail("invalid plural offset")
		}
		node.offset = offset
		if p.pos < len(p.input) && p.input[p.pos] != ' ' && p.input[p.pos] != '\n' && p.input[p.pos] != '\t' && p.input[p.pos] != '\r' {
			return node, p.fail("expected whitespace after offset")
		}
	}
	node.branches = map[string][]icuNode{}
	for {
		if err := p.ctx.Err(); err != nil {
			return node, err
		}
		p.space()
		if p.take('}') {
			break
		}
		start := p.pos
		for p.pos < len(p.input) && p.input[p.pos] != '{' && p.input[p.pos] != '}' {
			r, size := utf8.DecodeRuneInString(p.input[p.pos:])
			if unicode.IsSpace(r) {
				break
			}
			p.pos += size
		}
		selector := p.input[start:p.pos]
		if selector == "" {
			return node, p.fail("expected selector")
		}
		if node.kind != "select" {
			if strings.HasPrefix(selector, "=") {
				number, err := strconv.ParseInt(selector[1:], 10, 64)
				if err != nil {
					return node, p.fail("invalid exact numeric selector")
				}
				selector = "=" + strconv.FormatInt(number, 10)
			} else if selector != "zero" && selector != "one" && selector != "two" && selector != "few" && selector != "many" && selector != "other" {
				return node, p.fail("invalid plural category")
			}
		}
		if _, exists := node.branches[selector]; exists {
			return node, p.fail("duplicate selector " + selector)
		}
		p.space()
		if !p.take('{') {
			return node, p.fail("expected selector body")
		}
		body, err := p.message(depth+1, true, inPlural || node.kind != "select")
		if err != nil {
			return node, err
		}
		p.nodes++
		if p.nodes > maxICUNodes {
			return node, p.fail("ICU node limit exceeded")
		}
		node.branches[selector] = body
	}
	if _, ok := node.branches["other"]; !ok {
		return node, p.fail("missing required other selector")
	}
	return node, nil
}

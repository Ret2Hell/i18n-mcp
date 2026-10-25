package validate

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

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
		return p.formattedArgument(node)
	case "plural", "selectordinal", "select":
		return p.selectorArgument(node, depth, inPlural)
	default:
		return node, p.fail("unsupported ICU argument type " + node.kind)
	}
}

func (p *icuParser) formattedArgument(node icuNode) (icuNode, error) {
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
}

func (p *icuParser) selectorArgument(node icuNode, depth int, inPlural bool) (icuNode, error) {
	if !p.take(',') {
		return node, p.fail("expected selector list")
	}
	p.space()
	if node.kind != "select" && strings.HasPrefix(p.input[p.pos:], "offset:") {
		offset, err := p.pluralOffset()
		if err != nil {
			return node, err
		}
		node.offset = offset
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
		selector, err := p.selector(node.kind)
		if err != nil {
			return node, err
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

func (p *icuParser) pluralOffset() (uint64, error) {
	p.pos += len("offset:")
	p.space()
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
		p.pos++
	}
	offset, err := strconv.ParseUint(p.input[start:p.pos], 10, 64)
	if err != nil {
		return 0, p.fail("invalid plural offset")
	}
	if p.pos < len(p.input) && p.input[p.pos] != ' ' && p.input[p.pos] != '\n' && p.input[p.pos] != '\t' && p.input[p.pos] != '\r' {
		return 0, p.fail("expected whitespace after offset")
	}
	return offset, nil
}

func (p *icuParser) selector(kind string) (string, error) {
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
		return "", p.fail("expected selector")
	}
	if kind == "select" {
		return selector, nil
	}
	if strings.HasPrefix(selector, "=") {
		number, err := strconv.ParseInt(selector[1:], 10, 64)
		if err != nil {
			return "", p.fail("invalid exact numeric selector")
		}
		return "=" + strconv.FormatInt(number, 10), nil
	}
	switch selector {
	case "zero", "one", "two", "few", "many", "other":
		return selector, nil
	default:
		return "", p.fail("invalid plural category")
	}
}

package jsonedit

import (
	"encoding/json"
	"fmt"
	"strconv"
	"unicode"
)

type parser struct {
	data []byte
	pos  int
}

func parseOrderedValue(data []byte) (*Value, error) {
	p := &parser{data: data}
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.pos != len(p.data) {
		return nil, p.errorf("unexpected trailing data")
	}
	return value, nil
}

func (p *parser) parseValue() (*Value, error) {
	p.skipWhitespace()
	if p.pos >= len(p.data) {
		return nil, p.errorf("unexpected end of JSON")
	}
	switch p.data[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		text, _, err := p.parseStringLiteral()
		if err != nil {
			return nil, err
		}
		return &Value{Kind: KindString, String: text}, nil
	default:
		return p.parseRaw()
	}
}

func (p *parser) parseObject() (*Value, error) {
	p.pos++
	value := &Value{Kind: KindObject}
	p.skipWhitespace()
	if p.consume('}') {
		return value, nil
	}
	for {
		p.skipWhitespace()
		if p.pos >= len(p.data) || p.data[p.pos] != '"' {
			return nil, p.errorf("expected object key")
		}
		key, _, err := p.parseStringLiteral()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if !p.consume(':') {
			return nil, p.errorf("expected ':' after object key")
		}
		memberValue, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		value.Object = append(value.Object, Member{Key: key, Value: memberValue})
		p.skipWhitespace()
		if p.consume('}') {
			return value, nil
		}
		if !p.consume(',') {
			return nil, p.errorf("expected ',' or '}'")
		}
	}
}

func (p *parser) parseArray() (*Value, error) {
	p.pos++
	value := &Value{Kind: KindArray}
	p.skipWhitespace()
	if p.consume(']') {
		return value, nil
	}
	for {
		item, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		value.Array = append(value.Array, item)
		p.skipWhitespace()
		if p.consume(']') {
			return value, nil
		}
		if !p.consume(',') {
			return nil, p.errorf("expected ',' or ']'")
		}
	}
}

func (p *parser) parseStringLiteral() (string, string, error) {
	start := p.pos
	p.pos++
	escaped := false
	for p.pos < len(p.data) {
		ch := p.data[p.pos]
		if ch < 0x20 {
			return "", "", p.errorf("invalid control character in string")
		}
		if escaped {
			escaped = false
			p.pos++
			continue
		}
		switch ch {
		case '\\':
			escaped = true
		case '"':
			p.pos++
			raw := string(p.data[start:p.pos])
			decoded, err := strconv.Unquote(raw)
			if err != nil {
				return "", "", err
			}
			return decoded, raw, nil
		}
		p.pos++
	}
	return "", "", p.errorf("unterminated string")
}

func (p *parser) parseRaw() (*Value, error) {
	start := p.pos
	for p.pos < len(p.data) && !isValueTerminator(p.data[p.pos]) {
		p.pos++
	}
	raw := string(p.data[start:p.pos])
	if raw == "" {
		return nil, p.errorf("expected JSON value")
	}
	if !validRawJSON(raw) {
		return nil, p.errorf("invalid JSON value %q", raw)
	}
	return &Value{Kind: KindRaw, RawJSON: raw}, nil
}

func validRawJSON(raw string) bool {
	return json.Valid([]byte(raw))
}

func isValueTerminator(ch byte) bool {
	return ch == ',' || ch == '}' || ch == ']' || unicode.IsSpace(rune(ch))
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.data) && unicode.IsSpace(rune(p.data[p.pos])) {
		p.pos++
	}
}

func (p *parser) consume(ch byte) bool {
	if p.pos < len(p.data) && p.data[p.pos] == ch {
		p.pos++
		return true
	}
	return false
}

func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("json offset %d: %s", p.pos, fmt.Sprintf(format, args...))
}

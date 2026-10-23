// Package pagination provides bounded, snapshot-bound pages without retaining full results.
package pagination

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/security"
)

// Defaults and hard budgets for network-facing pages.
const (
	DefaultLimit = 50
	MaxLimit     = 200
	// ItemBudget leaves room for summaries and cursor metadata in a 128 KiB response.
	ItemBudget     = 64 << 10
	ResponseBudget = 128 << 10
)

// Input controls a bounded page request.
type Input struct {
	Limit  int    `json:"limit,omitzero" jsonschema:"page size; default 50; maximum 200"`
	Cursor string `json:"cursor,omitzero" jsonschema:"opaque nextCursor from the previous page; keep filters unchanged"`
}

// Metadata makes omitted results and continuation explicit.
type Metadata struct {
	Returned     int    `json:"returned"`
	TotalMatched int    `json:"totalMatched"`
	Truncated    bool   `json:"truncated"`
	NextCursor   string `json:"nextCursor,omitzero"`
}

type cursorData struct {
	Offset int `json:"offset"`
}

// Builder hashes the complete matching stream but retains only one bounded page.
// Cursors bind the owner, project, operation, filters, page size and stream content.
type Builder[T any] struct {
	Items                            []T
	input                            Input
	signer                           *security.RequestStateSigner
	ctx                              context.Context
	operation, queryDigest, expected string
	offset, total, bytes             int
	blocked                          bool
	digest                           hash.Hash
}

// New authenticates a cursor and validates request limits.
func New[T any](ctx context.Context, signer *security.RequestStateSigner, operation, project string, query any, in Input) (*Builder[T], error) {
	if in.Limit == 0 {
		in.Limit = DefaultLimit
	}
	if in.Limit < 1 || in.Limit > MaxLimit {
		return nil, fmt.Errorf("limit must be between 1 and %d", MaxLimit)
	}
	raw, err := json.Marshal(struct {
		Project string
		Query   any
		Limit   int
	}{project, query, in.Limit})
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	b := &Builder[T]{input: in, signer: signer, ctx: ctx, operation: operation, queryDigest: hex.EncodeToString(sum[:]), digest: sha256.New(), Items: []T{}}
	if in.Cursor != "" {
		claims, err := signer.Verify(in.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		if claims.Subject != security.SubjectFromContext(ctx) || claims.Operation != operation || claims.InputDigest != b.queryDigest {
			return nil, fmt.Errorf("cursor does not match project, subject, filters or limit")
		}
		var data cursorData
		if err := json.Unmarshal(claims.Data, &data); err != nil || data.Offset < 0 {
			return nil, fmt.Errorf("invalid cursor offset")
		}
		b.offset = data.Offset
		b.expected = claims.PlanDigest
	}
	return b, nil
}

// Add hashes a matching result and retains it only if it fits this page.
func (b *Builder[T]) Add(item T) error {
	if err := b.ctx.Err(); err != nil {
		return err
	}
	raw, err := json.Marshal(item)
	if err != nil {
		return err
	}
	_, _ = b.digest.Write(raw)
	_, _ = b.digest.Write([]byte{'\n'})
	b.total++
	if b.total <= b.offset || b.blocked {
		return nil
	}
	if len(b.Items) == b.input.Limit || b.bytes+len(raw)+1 > ItemBudget {
		if len(b.Items) == 0 {
			return fmt.Errorf("one result exceeds the %d-byte page item budget; use a local file/CLI export for this value", ItemBudget)
		}
		b.blocked = true
		return nil
	}
	b.Items = append(b.Items, item)
	b.bytes += len(raw) + 1
	return nil
}

// Finish checks snapshot consistency and signs the next cursor.
func (b *Builder[T]) Finish() (Metadata, error) {
	if err := b.ctx.Err(); err != nil {
		return Metadata{}, err
	}
	snapshot := hex.EncodeToString(b.digest.Sum(nil))
	if b.expected != "" && b.expected != snapshot {
		return Metadata{}, fmt.Errorf("cursor is stale; project results changed; restart from the first page")
	}
	if b.offset > b.total {
		return Metadata{}, fmt.Errorf("cursor offset exceeds matching results")
	}
	m := Metadata{Returned: len(b.Items), TotalMatched: b.total, Truncated: b.offset+len(b.Items) < b.total}
	if m.Truncated {
		data, _ := json.Marshal(cursorData{Offset: b.offset + len(b.Items)})
		token, err := b.signer.Sign(security.RequestStateClaims{Subject: security.SubjectFromContext(b.ctx), Operation: b.operation, InputDigest: b.queryDigest, PlanDigest: snapshot, ExpiresAt: time.Now().Add(time.Hour).Unix(), Data: data})
		if err != nil {
			return Metadata{}, err
		}
		m.NextCursor = token
	}
	return m, nil
}

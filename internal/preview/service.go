// Package preview stores bounded, immutable patch snapshots for explicit review.
package preview

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Ret2Hell/i18n-mcp/internal/security"
)

// Storage and inline/page response limits.
const (
	// InlineBytes bounds the combined inline patch text, not each file separately.
	InlineBytes    = 16 << 10
	MaxInlineFiles = 50
	ChunkBytes     = 16 << 10
	MaxBytes       = 64 << 20
	maxSnapshots   = 16
	ttl            = 15 * time.Minute
)

// File describes a file change and any explicitly truncated inline diff.
type File struct {
	Path          string `json:"path"`
	Diff          string `json:"diff,omitzero"`
	Changed       bool   `json:"changed"`
	Written       bool   `json:"written,omitzero"`
	DiffBytes     int    `json:"diffBytes,omitzero"`
	DiffSHA256    string `json:"diffSHA256,omitzero"`
	DiffTruncated bool   `json:"diffTruncated,omitzero"`
	PatchURI      string `json:"patchUri,omitzero"`
}

// Metadata identifies a complete immutable review snapshot and its approval token.
type Metadata struct {
	Token                string    `json:"token"`
	ExpiresAt            time.Time `json:"expiresAt"`
	TotalFiles           int       `json:"totalFiles"`
	ReturnedFiles        int       `json:"returnedFiles"`
	TotalDiffBytes       int       `json:"totalDiffBytes"`
	Truncated            bool      `json:"truncated"`
	DiagnosticsTruncated bool      `json:"diagnosticsTruncated,omitzero"`
	ManifestURI          string    `json:"manifestUri"`
}

// View is the bounded inline representation of a complete snapshot.
type View struct {
	Metadata *Metadata
	Files    []File
}

type snapshot struct {
	id, owner, operation, inputDigest, digest, token string
	expires                                          time.Time
	files                                            []File
	bytes                                            int
}

// Service retains at most 64 MiB and 16 snapshots across all callers. Reads never
// access arbitrary filesystem paths; expired/evicted snapshots require re-preview.
type Service struct {
	mu      sync.Mutex
	signer  *security.RequestStateSigner
	project string
	records map[string]snapshot
	bytes   int
	now     func() time.Time
}

// NewService creates a project-scoped snapshot store.
func NewService(project string, signer *security.RequestStateSigner) *Service {
	return &Service{project: sum(project), signer: signer, records: map[string]snapshot{}, now: time.Now}
}
func sum(s string) string { digest := sha256.Sum256([]byte(s)); return hex.EncodeToString(digest[:]) }
func digestValue(value any) string {
	h := sha256.New()
	_ = json.NewEncoder(h).Encode(value)
	return hex.EncodeToString(h.Sum(nil))
}
func planDigest(input string, files []File, extra any) string {
	h := sha256.New()
	_ = json.NewEncoder(h).Encode(input)
	for _, file := range files {
		_ = json.NewEncoder(h).Encode([]string{file.Path, sum(file.Diff), strconv.FormatBool(file.Changed)})
	}
	_ = json.NewEncoder(h).Encode(extra)
	return hex.EncodeToString(h.Sum(nil))
}

// Prepare publishes a new snapshot, or validates/reuses an existing review token.
// Failure is returned before callers perform writes, including storage-size errors.
func (s *Service) Prepare(ctx context.Context, operation string, input any, files []File, extra any, token string) (View, error) {
	if err := ctx.Err(); err != nil {
		return View{}, err
	}
	if token != "" {
		record, err := s.verify(ctx, operation, input, files, extra, token)
		if err != nil {
			return View{}, err
		}
		return view(record), nil
	}
	total := 0
	for _, file := range files {
		total += len(file.Diff) + len(file.Path)
		if total > MaxBytes {
			return View{}, fmt.Errorf("complete preview exceeds 64 MiB; narrow the selected locales/keys before applying")
		}
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return View{}, err
	}
	id := hex.EncodeToString(idBytes)
	now := s.now()
	inputDigest := digestValue(input)
	record := snapshot{id: id, owner: security.SubjectFromContext(ctx), operation: operation, inputDigest: inputDigest, digest: planDigest(inputDigest, files, extra), expires: now.Add(ttl), files: append([]File(nil), files...), bytes: total}
	data, _ := json.Marshal(id)
	signed, err := s.signer.Sign(security.RequestStateClaims{Subject: record.owner, Operation: operation, InputDigest: s.project + ":" + inputDigest, PlanDigest: record.digest, ExpiresAt: record.expires.Unix(), Data: data})
	if err != nil {
		return View{}, err
	}
	record.token = signed
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purge(now)
	for len(s.records) >= maxSnapshots || s.bytes+total > MaxBytes {
		oldest := ""
		var age time.Time
		for id, r := range s.records {
			if oldest == "" || r.expires.Before(age) {
				oldest = id
				age = r.expires
			}
		}
		s.bytes -= s.records[oldest].bytes
		delete(s.records, oldest)
	}
	for i := range record.files {
		record.files[i].DiffBytes = len(record.files[i].Diff)
		record.files[i].DiffSHA256 = sum(record.files[i].Diff)
	}
	s.records[id] = record
	s.bytes += total
	return view(record), nil
}

func (s *Service) purge(now time.Time) {
	for id, r := range s.records {
		if !now.Before(r.expires) {
			s.bytes -= r.bytes
			delete(s.records, id)
		}
	}
}
func (s *Service) owned(ctx context.Context, id string) (snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purge(s.now())
	record, ok := s.records[id]
	if !ok || record.owner != security.SubjectFromContext(ctx) {
		return snapshot{}, fmt.Errorf("preview not found, expired or evicted; create a new preview")
	}
	return record, nil
}
func (s *Service) verify(ctx context.Context, operation string, input any, files []File, extra any, token string) (snapshot, error) {
	claims, err := s.signer.Verify(token)
	if err != nil {
		return snapshot{}, fmt.Errorf("invalid preview token: %w", err)
	}
	inputDigest := digestValue(input)
	if claims.Subject != security.SubjectFromContext(ctx) || claims.Operation != operation || claims.InputDigest != s.project+":"+inputDigest || claims.PlanDigest != planDigest(inputDigest, files, extra) {
		return snapshot{}, fmt.Errorf("preview is stale or belongs to a different operation; review a new preview before applying")
	}
	var id string
	if err := json.Unmarshal(claims.Data, &id); err != nil {
		return snapshot{}, fmt.Errorf("invalid preview token data")
	}
	record, err := s.owned(ctx, id)
	if err != nil {
		return snapshot{}, err
	}
	if record.digest != claims.PlanDigest {
		return snapshot{}, fmt.Errorf("preview snapshot mismatch")
	}
	return record, nil
}

// Verify checks the actual plan immediately before its file edits are written.
func (s *Service) Verify(ctx context.Context, operation string, input any, files []File, extra any, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := s.verify(ctx, operation, input, files, extra, token)
	return err
}

func view(record snapshot) View {
	count := min(len(record.files), MaxInlineFiles)
	files := append([]File(nil), record.files[:count]...)
	metadata := &Metadata{Token: record.token, ExpiresAt: record.expires, TotalFiles: len(record.files), ReturnedFiles: count, Truncated: count < len(record.files), ManifestURI: "i18n://patches/" + record.id + "/manifest/0"}
	remaining := InlineBytes
	for _, file := range record.files {
		metadata.TotalDiffBytes += len(file.Diff)
	}
	for i := range files {
		files[i].DiffBytes = len(files[i].Diff)
		files[i].PatchURI = fmt.Sprintf("i18n://patches/%s/file/%d/0", record.id, i)
		if len(files[i].Diff) > remaining || !utf8.ValidString(files[i].Diff) {
			n := min(len(files[i].Diff), remaining)
			for n > 0 && !utf8.ValidString(files[i].Diff[:n]) {
				n--
			}
			files[i].Diff = files[i].Diff[:n]
			files[i].DiffTruncated = true
			metadata.Truncated = true
		}
		remaining -= len(files[i].Diff)
	}
	return View{Metadata: metadata, Files: files}
}

// Page returns either bounded file metadata or a byte-exact base64 patch chunk.
// A chunk's SHA-256 covers the complete patch, not just the returned data.
func (s *Service) Page(ctx context.Context, id, kind string, index, offset int) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	record, err := s.owned(ctx, id)
	if err != nil {
		return nil, err
	}
	if offset < 0 {
		return nil, fmt.Errorf("negative preview offset")
	}
	if kind == "manifest" {
		if offset > len(record.files) {
			return nil, fmt.Errorf("preview manifest offset out of range")
		}
		end := min(len(record.files), offset+MaxInlineFiles)
		files := append([]File(nil), record.files[offset:end]...)
		for i := range files {
			files[i].DiffBytes = len(files[i].Diff)
			files[i].Diff = ""
			files[i].PatchURI = fmt.Sprintf("i18n://patches/%s/file/%d/0", id, offset+i)
		}
		next := ""
		if end < len(record.files) {
			next = fmt.Sprintf("i18n://patches/%s/manifest/%d", id, end)
		}
		return struct {
			Files        []File `json:"files"`
			TotalFiles   int    `json:"totalFiles"`
			NextResource string `json:"nextResource,omitzero"`
		}{files, len(record.files), next}, nil
	}
	if kind != "file" || index < 0 || index >= len(record.files) {
		return nil, fmt.Errorf("preview file index out of range")
	}
	file := record.files[index]
	if offset > len(file.Diff) {
		return nil, fmt.Errorf("preview chunk offset out of range")
	}
	end := min(len(file.Diff), offset+ChunkBytes)
	next := ""
	if end < len(file.Diff) {
		next = fmt.Sprintf("i18n://patches/%s/file/%d/%d", id, index, end)
	}
	return struct {
		Path         string `json:"path"`
		Offset       int    `json:"offset"`
		TotalBytes   int    `json:"totalBytes"`
		SHA256       string `json:"sha256"`
		Encoding     string `json:"encoding"`
		Data         string `json:"data"`
		NextResource string `json:"nextResource,omitzero"`
	}{file.Path, offset, len(file.Diff), file.DiffSHA256, "base64", base64.StdEncoding.EncodeToString([]byte(file.Diff[offset:end])), next}, nil
}

//go:build jsonbench

package jsonbench

import (
	"bytes"
	stdjson "encoding/json"
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/bytedance/sonic"
)

var (
	resultBytes []byte
	resultValue any
	resultBool  bool
)

type codec struct {
	name          string
	marshal       func(any) ([]byte, error)
	marshalIndent func(any, string, string) ([]byte, error)
	unmarshal     func([]byte, any) error
	valid         func([]byte) bool
}

var codecs = []codec{
	{
		name:          "stdlib",
		marshal:       stdjson.Marshal,
		marshalIndent: stdjson.MarshalIndent,
		unmarshal:     stdjson.Unmarshal,
		valid:         stdjson.Valid,
	},
	{
		name:          "sonic_std",
		marshal:       sonic.ConfigStd.Marshal,
		marshalIndent: sonic.ConfigStd.MarshalIndent,
		unmarshal:     sonic.ConfigStd.Unmarshal,
		valid:         sonic.ConfigStd.Valid,
	},
	{
		name:          "sonic_default",
		marshal:       sonic.ConfigDefault.Marshal,
		marshalIndent: sonic.ConfigDefault.MarshalIndent,
		unmarshal:     sonic.ConfigDefault.Unmarshal,
		valid:         sonic.ConfigDefault.Valid,
	},
}

func TestSonicNativeImplementation(t *testing.T) {
	if sonic.APIKind != sonic.UseSonicJSON {
		t.Fatal("Sonic is using its encoding/json fallback; native Sonic does not support this Go/architecture combination")
	}
}

func TestCompatibility(t *testing.T) {
	batch := makeBatch(100)
	wantJSON, err := stdjson.Marshal(batch)
	if err != nil {
		t.Fatalf("marshal fixture with encoding/json: %v", err)
	}

	for _, candidate := range codecs[1:] {
		t.Run(candidate.name+"/typed_round_trip", func(t *testing.T) {
			gotJSON, err := candidate.marshal(batch)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var want, got translate.Batch
			if err := stdjson.Unmarshal(wantJSON, &want); err != nil {
				t.Fatalf("stdlib unmarshal: %v", err)
			}
			if err := candidate.unmarshal(gotJSON, &got); err != nil {
				t.Fatalf("candidate unmarshal: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatal("candidate changed the typed batch after a round trip")
			}
		})
	}

	localeJSON := makeLocaleJSON(t, 100)
	want := decodeLocaleStd(t, localeJSON)
	got := decodeLocaleSonic(t, localeJSON)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("sonic decoder with UseNumber changed dynamically decoded locale values")
	}
}

func BenchmarkMarshalBatch(b *testing.B) {
	requireNativeSonic(b)
	for _, size := range []int{10, 100, 1000} {
		batch := makeBatch(size)
		payload, err := stdjson.Marshal(batch)
		if err != nil {
			b.Fatal(err)
		}
		for _, candidate := range codecs {
			b.Run(fmt.Sprintf("items_%04d/%s", size, candidate.name), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(payload)))
				for b.Loop() {
					out, err := candidate.marshal(batch)
					if err != nil {
						b.Fatal(err)
					}
					resultBytes = out
				}
			})
		}
	}
}

func BenchmarkMarshalIndentBatch(b *testing.B) {
	requireNativeSonic(b)
	for _, size := range []int{10, 100, 1000} {
		batch := makeBatch(size)
		payload, err := stdjson.MarshalIndent(batch, "", "  ")
		if err != nil {
			b.Fatal(err)
		}
		for _, candidate := range codecs {
			b.Run(fmt.Sprintf("items_%04d/%s", size, candidate.name), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(payload)))
				for b.Loop() {
					out, err := candidate.marshalIndent(batch, "", "  ")
					if err != nil {
						b.Fatal(err)
					}
					resultBytes = out
				}
			})
		}
	}
}

func BenchmarkUnmarshalBatch(b *testing.B) {
	requireNativeSonic(b)
	for _, size := range []int{10, 100, 1000} {
		payload, err := stdjson.Marshal(makeBatch(size))
		if err != nil {
			b.Fatal(err)
		}
		for _, candidate := range codecs {
			b.Run(fmt.Sprintf("items_%04d/%s", size, candidate.name), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(payload)))
				for b.Loop() {
					var out translate.Batch
					if err := candidate.unmarshal(payload, &out); err != nil {
						b.Fatal(err)
					}
					resultValue = out
				}
			})
		}
	}
}

func BenchmarkDecodeDynamicLocale(b *testing.B) {
	requireNativeSonic(b)
	for _, size := range []int{10, 100, 1000} {
		payload := makeLocaleJSON(b, size)
		decoders := []struct {
			name string
			fn   func([]byte) (any, error)
		}{
			{name: "stdlib", fn: decodeLocaleStdBytes},
			{name: "sonic_std", fn: decodeLocaleSonicBytes},
		}
		for _, decoder := range decoders {
			b.Run(fmt.Sprintf("keys_%04d/%s", size, decoder.name), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(payload)))
				for b.Loop() {
					out, err := decoder.fn(payload)
					if err != nil {
						b.Fatal(err)
					}
					resultValue = out
				}
			})
		}
	}
}

func BenchmarkMarshalString(b *testing.B) {
	requireNativeSonic(b)
	values := []struct {
		name  string
		value string
	}{
		{name: "short", value: "Save changes"},
		{name: "escaped", value: `Delete <strong>{count}</strong> items? "This cannot be undone."`},
		{name: "unicode", value: "こんにちは世界 — Καλημέρα κόσμε — مرحبا بالعالم"},
	}
	for _, value := range values {
		for _, candidate := range codecs {
			b.Run(value.name+"/"+candidate.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					out, err := candidate.marshal(value.value)
					if err != nil {
						b.Fatal(err)
					}
					resultBytes = out
				}
			})
		}
	}
}

func BenchmarkValidLocale(b *testing.B) {
	requireNativeSonic(b)
	for _, size := range []int{10, 100, 1000} {
		payload := makeLocaleJSON(b, size)
		for _, candidate := range codecs {
			b.Run(fmt.Sprintf("keys_%04d/%s", size, candidate.name), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(payload)))
				for b.Loop() {
					resultBool = candidate.valid(payload)
				}
			})
		}
	}
}

func requireNativeSonic(b *testing.B) {
	b.Helper()
	if sonic.APIKind != sonic.UseSonicJSON {
		b.Skip("Sonic is using its encoding/json fallback; comparison would be misleading")
	}
}

func makeBatch(itemCount int) translate.Batch {
	items := make([]translate.Item, itemCount)
	for i := range items {
		items[i] = translate.Item{
			ID:             fmt.Sprintf("fr:common:checkout.section_%02d.label_%04d", i%10, i),
			Locale:         "fr",
			Namespace:      "common",
			Key:            fmt.Sprintf("checkout.section_%02d.label_%04d", i%10, i),
			Status:         diff.Missing,
			SourceValue:    fmt.Sprintf("Hello {name}, you have %d items in your cart", i),
			SourceHash:     fmt.Sprintf("sha256:%064x", i+1),
			Placeholders:   []string{"name"},
			SourceFilePath: "messages/en/common.json",
			TargetFilePath: "messages/fr/common.json",
		}
	}
	return translate.Batch{
		BatchID:         "benchmark-batch",
		SourceLocale:    "en",
		TargetLocales:   []string{"fr"},
		Items:           items,
		ValidationRules: []string{"preserve placeholders exactly", "preserve HTML-like tag structure"},
		ResourceLinks:   []string{"i18n://analysis/diff", "i18n://translation/plan/latest"},
		CreatedAt:       time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
	}
}

func makeLocaleJSON(tb testing.TB, keyCount int) []byte {
	tb.Helper()
	sections := make(map[string]map[string]any, (keyCount+9)/10)
	for i := range keyCount {
		sectionName := fmt.Sprintf("section_%02d", i/10)
		section := sections[sectionName]
		if section == nil {
			section = make(map[string]any, 10)
			sections[sectionName] = section
		}
		section[fmt.Sprintf("label_%04d", i)] = fmt.Sprintf("Hello {name}, item %d costs %.2f €", i, float64(i)+0.99)
	}
	payload, err := stdjson.Marshal(sections)
	if err != nil {
		tb.Fatalf("marshal locale fixture: %v", err)
	}
	return payload
}

func decodeLocaleStd(tb testing.TB, payload []byte) any {
	tb.Helper()
	out, err := decodeLocaleStdBytes(payload)
	if err != nil {
		tb.Fatalf("decode locale with encoding/json: %v", err)
	}
	return out
}

func decodeLocaleSonic(tb testing.TB, payload []byte) any {
	tb.Helper()
	out, err := decodeLocaleSonicBytes(payload)
	if err != nil {
		tb.Fatalf("decode locale with sonic: %v", err)
	}
	return out
}

func decodeLocaleStdBytes(payload []byte) (any, error) {
	decoder := stdjson.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	return decodeOneValue(decoder.Decode)
}

func decodeLocaleSonicBytes(payload []byte) (any, error) {
	decoder := sonic.ConfigStd.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	return decodeOneValue(decoder.Decode)
}

func decodeOneValue(decode func(any) error) (any, error) {
	var value any
	if err := decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return value, nil
}

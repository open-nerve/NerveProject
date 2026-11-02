package content

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// Trusting a declaration or omitting the full stream hash would accept unrelated bytes.
func TestContentValidation(t *testing.T) {
	picture := image.NewRGBA(image.Rect(0, 0, 2, 3))
	var pngBytes, jpegBytes, gifBytes bytes.Buffer
	if err := png.Encode(&pngBytes, picture); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&jpegBytes, picture, nil); err != nil {
		t.Fatal(err)
	}
	if err := gif.Encode(&gifBytes, picture, nil); err != nil {
		t.Fatal(err)
	}
	webp, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, mime string
		data       []byte
		w, h       int
	}{
		{"png", "image/png", pngBytes.Bytes(), 2, 3}, {"jpeg", "image/jpeg", jpegBytes.Bytes(), 2, 3},
		{"gif", "image/gif", gifBytes.Bytes(), 2, 3}, {"webp", "image/webp", webp, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proof, err := (Validator{}).Validate(context.Background(), bytes.NewReader(tc.data), int64(len(tc.data)), tc.mime, true)
			digest := sha256.Sum256(tc.data)
			if err != nil || proof.Width == nil || proof.Height == nil || *proof.Width != tc.w || *proof.Height != tc.h || proof.SHA256 != hex.EncodeToString(digest[:]) {
				t.Fatalf("proof=%+v err=%v", proof, err)
			}
		})
	}
	for _, tc := range []struct {
		name, mime string
		data       []byte
		size       int64
		image      bool
		want       error
	}{
		{"plain attachment", "application/octet-stream", []byte("abc"), 3, false, nil},
		{"another same size", "application/octet-stream", []byte("abd"), 3, false, nil},
		{"untrusted image attachment", "image/png", []byte("<svg/>"), 6, false, nil},
		{"disguised svg", "image/png", []byte("<svg/>"), 6, true, domain.ErrAssetContentInvalid},
		{"svg image", "image/svg+xml", []byte("<svg/>"), 6, true, domain.ErrAssetTypeUnsupported},
		{"png declared jpeg", "image/jpeg", pngBytes.Bytes(), int64(pngBytes.Len()), true, domain.ErrAssetContentInvalid},
		{"short stream", "application/octet-stream", []byte("ab"), 3, false, domain.ErrAssetContentInvalid},
		{"long stream", "application/octet-stream", []byte("abcd"), 3, false, domain.ErrAssetContentInvalid},
		{"empty", "application/octet-stream", nil, 0, false, domain.ErrAssetContentInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := (Validator{}).Validate(context.Background(), bytes.NewReader(tc.data), tc.size, tc.mime, tc.image)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want %v", err, tc.want)
			}
			if tc.want == nil {
				digest := sha256.Sum256(tc.data)
				if got.SHA256 != hex.EncodeToString(digest[:]) || got.Width != nil || got.Height != nil {
					t.Fatalf("attachment proof=%+v", got)
				}
			}
		})
	}
}

// A valid PNG header is enough for DecodeConfig, not a claim that all pixels decode.
func pngHeader(width, height uint32) []byte {
	out := []byte("\x89PNG\r\n\x1a\n")
	payload := make([]byte, 13)
	binary.BigEndian.PutUint32(payload, width)
	binary.BigEndian.PutUint32(payload[4:], height)
	payload[8] = 8
	payload[9] = 6
	size := make([]byte, 4)
	binary.BigEndian.PutUint32(size, uint32(len(payload)))
	out = append(out, size...)
	chunk := append([]byte("IHDR"), payload...)
	out = append(out, chunk...)
	sum := make([]byte, 4)
	binary.BigEndian.PutUint32(sum, crc32.ChecksumIEEE(chunk))
	return append(out, sum...)
}
func TestContentDimensionsAndHeaderBudget(t *testing.T) {
	for _, tc := range []struct {
		w, h  uint32
		valid bool
	}{{1, 1, true}, {16384, 1, true}, {16385, 1, false}, {10000, 4000, true}, {10001, 4000, false}, {0, 1, false}, {0xffffffff, 0xffffffff, false}} {
		data := pngHeader(tc.w, tc.h)
		_, err := (Validator{}).Validate(context.Background(), bytes.NewReader(data), int64(len(data)), "image/png", true)
		if (err == nil) != tc.valid {
			t.Errorf("%dx%d: error=%v valid=%v", tc.w, tc.h, err, tc.valid)
		}
	}
	// JPEG APP segments push the actual frame header beyond the allowed 1 MiB.
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil); err != nil {
		t.Fatal(err)
	}
	data := []byte{0xff, 0xd8}
	segment := append([]byte{0xff, 0xe1, 0xff, 0xff}, make([]byte, 65533)...)
	for range 17 {
		data = append(data, segment...)
	}
	data = append(data, encoded.Bytes()[2:]...)
	// The fixture really is a decodable header absent our limit.
	if _, err := jpeg.DecodeConfig(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if _, err := (Validator{}).Validate(context.Background(), bytes.NewReader(data), int64(len(data)), "image/jpeg", true); !errors.Is(err, domain.ErrAssetContentInvalid) {
		t.Fatalf("oversized header=%v", err)
	}
}

type failedReader struct{ err error }

func (r failedReader) Read([]byte) (int, error) { return 0, r.err }

type countedReader struct{ n int }

func (r *countedReader) Read(b []byte) (int, error) { r.n += len(b); clear(b); return len(b), nil }
func TestContentValidationIOBounds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reader := &countedReader{}
	if _, err := (Validator{}).Validate(ctx, reader, 3, "text/plain", false); !errors.Is(err, context.Canceled) || reader.n != 0 {
		t.Fatalf("canceled read %d, %v", reader.n, err)
	}
	lost := errors.New("lost read")
	if _, err := (Validator{}).Validate(context.Background(), failedReader{lost}, 3, "text/plain", false); !errors.Is(err, lost) {
		t.Fatalf("swallowed failure: %v", err)
	}
	if _, err := (Validator{}).Validate(context.Background(), reader, 3, "text/plain", false); !errors.Is(err, domain.ErrAssetContentInvalid) || reader.n != 4 {
		t.Fatalf("limit+one read=%d err=%v", reader.n, err)
	}
	if _, err := (Validator{}).Validate(context.Background(), io.LimitReader(&countedReader{}, 104857601), 104857601, "text/plain", false); !errors.Is(err, domain.ErrAssetTooLarge) {
		t.Fatalf("hard size bound: %v", err)
	}
}

// Hashing only the first chunk misses equal-length changes at the end of an attachment.
func TestContentFullStreamHash(t *testing.T) {
	first := bytes.Repeat([]byte("a"), 131072)
	second := bytes.Clone(first)
	second[len(second)-1] = 'b'
	var hashes []string
	for _, data := range [][]byte{first, second} {
		proof, err := (Validator{}).Validate(context.Background(), bytes.NewReader(data), int64(len(data)), "application/octet-stream", false)
		sum := sha256.Sum256(data)
		if err != nil || proof.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatalf("full stream proof=%+v err=%v", proof, err)
		}
		hashes = append(hashes, proof.SHA256)
	}
	if hashes[0] == hashes[1] {
		t.Fatal("different tail has same proof")
	}
}

type canceledOnEOF struct{ cancel context.CancelFunc }

func (r canceledOnEOF) Read(b []byte) (int, error) { n := copy(b, "abc"); r.cancel(); return n, io.EOF }

type noProgressReader struct{}

func (noProgressReader) Read([]byte) (int, error) { return 0, nil }
func TestContentCancellationAtEOF(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := (Validator{}).Validate(ctx, canceledOnEOF{cancel}, 3, "text/plain", false); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled result published: %v", err)
	}
	if _, err := (Validator{}).Validate(context.Background(), noProgressReader{}, 3, "text/plain", false); !errors.Is(err, io.ErrNoProgress) {
		t.Fatalf("no-progress read=%v", err)
	}
}

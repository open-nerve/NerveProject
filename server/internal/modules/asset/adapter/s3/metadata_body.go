package s3

import (
	"errors"
	"io"
)

type metadataBudgetKey struct{}
type objectBodyKey struct{}

// Ordinary protocol replies are small XML. Inventory has its own page-sized
// budget, and only a successful explicit object read may stream file bytes.
const protocolMetadataLimit int64 = 64 << 10

// Limit metadata allocation inside SDK XML parsing, without buffering the
// response. The byte after the budget is only a sentinel; overflow is not EOF.
type metadataBody struct {
	io.ReadCloser
	left int64
}

func (b *metadataBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.left == 0 {
		var sentinel [1]byte
		n, err := b.ReadCloser.Read(sentinel[:])
		if n > 0 {
			return 0, errors.New("S3 metadata response exceeds limit")
		}
		return 0, err
	}
	n, err := b.ReadCloser.Read(p[:min(int64(len(p)), b.left)])
	b.left -= int64(n)
	return n, err
}

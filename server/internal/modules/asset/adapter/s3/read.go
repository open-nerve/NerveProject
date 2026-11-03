package s3

import (
	"context"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) Open(ctx context.Context, key domain.ObjectKey, expectedETag string) (io.ReadCloser, error) {
	if _, err := domain.ParseObjectKey(string(key)); err != nil {
		return nil, err
	}
	op, done, err := s.operation(ctx)
	if err != nil {
		return nil, err
	}
	if expectedETag == "" {
		_, err := s.inspect(op, key)
		done()
		if err != nil {
			return nil, err
		}
		return nil, domain.ErrObjectChanged
	}
	if !validETag(expectedETag) {
		done()
		return nil, domain.ErrObjectChanged
	}
	readContext := context.WithValue(op, objectBodyKey{}, true)
	out, err := s.client.GetObject(readContext, &awss3.GetObjectInput{Bucket: aws.String(s.cfg.Bucket), Key: aws.String(s.cfg.Prefix + string(key)), IfMatch: &expectedETag})
	if err != nil {
		converted := s.objectError(op, "get object", err)
		done()
		return nil, converted
	}
	info, err := objectInfo(key, out.ContentLength, out.ContentType, out.ETag, out.LastModified, out.Metadata)
	if err == nil && info.ETag != expectedETag {
		err = domain.ErrObjectChanged
	}
	if canceled := op.Err(); canceled != nil {
		err = canceled
	}
	if err != nil {
		done()
		_ = out.Body.Close()
		return nil, err
	}
	r := &objectReader{body: out.Body, ctx: op, finish: done}
	r.stop = context.AfterFunc(op, func() { done(); _ = r.closeBody() })
	return r, nil
}

// The returned reader owns its operation context until Close. Cancellation
// closes the actual body even if no further Read happens, including after EOF.
type objectReader struct {
	body   io.ReadCloser
	ctx    context.Context
	finish func()
	stop   func() bool
	once   sync.Once
	err    error
}

func (r *objectReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.body.Read(p)
	if canceled := r.ctx.Err(); canceled != nil {
		return n, canceled
	}
	if err == io.EOF {
		return n, io.EOF
	}
	return n, storageError("read object", err)
}

func (r *objectReader) closeBody() error {
	r.once.Do(func() { r.err = storageError("close object", r.body.Close()) })
	return r.err
}

func (r *objectReader) Close() error {
	r.stop()
	r.finish()
	return r.closeBody()
}

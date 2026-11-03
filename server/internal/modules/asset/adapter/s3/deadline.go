package s3

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type deadlinePresigner struct {
	deadline time.Time
	limit    time.Duration
	invalid  error
}

// The SDK supplies its actual signing time. Set the exact whole-second
// deadline before the official signer signs the query; never rewrite a signed
// URL or fake the signing clock to compensate for duration truncation.
func (p deadlinePresigner) PresignHTTP(ctx context.Context, c aws.Credentials, req *http.Request, hash, service, region string, at time.Time, options ...func(*v4.SignerOptions)) (string, http.Header, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	seconds := p.deadline.Unix() - at.Unix()
	if seconds <= 0 || seconds > int64(p.limit/time.Second) {
		return "", nil, p.invalid
	}
	query := req.URL.Query()
	query.Set("X-Amz-Expires", strconv.FormatInt(seconds, 10))
	req.URL.RawQuery = query.Encode()
	signer := v4.NewSigner(func(o *v4.SignerOptions) { o.DisableURIPathEscaping = true })
	return signer.PresignHTTP(ctx, c, req, hash, service, region, at, options...)
}

func exactDeadline(deadline time.Time, limit time.Duration, invalid error) func(*awss3.PresignOptions) {
	return func(o *awss3.PresignOptions) {
		// The SDK's positive-duration check still applies. Its temporary expiry
		// is replaced with the absolute deadline at the real signing instant.
		o.Expires = limit
		o.Presigner = deadlinePresigner{deadline: deadline, limit: limit, invalid: invalid}
	}
}

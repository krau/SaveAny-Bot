package s3

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/pkg/storagetypes"
)

type Client struct {
	endpoint   string
	region     string
	bucket     string
	accessKey  string
	secretKey  string
	httpClient *http.Client
	pathStyle  bool
}

const (
	maxRequestAttempts   = 4
	multipartThreshold   = 64 << 20
	multipartPartSize    = 16 << 20
	minMultipartPartSize = 5 << 20
)

type initiateMultipartUploadResult struct {
	UploadID string `xml:"UploadId"`
}

type completeMultipartUpload struct {
	XMLName xml.Name        `xml:"CompleteMultipartUpload"`
	Parts   []completedPart `xml:"Part"`
}

type completedPart struct {
	PartNumber int    `xml:"PartNumber"`
	ETag       string `xml:"ETag"`
}

type Config struct {
	Endpoint        string
	Region          string
	BucketName      string
	AccessKeyID     string
	SecretAccessKey string
	PathStyle       bool
	HttpClient      *http.Client
}

func (c *Config) ApplyDefaults() {
	if c.HttpClient == nil {
		transport, ok := http.DefaultTransport.(*http.Transport)
		if ok {
			cloned := transport.Clone()
			cloned.ResponseHeaderTimeout = 30 * time.Second
			cloned.IdleConnTimeout = 90 * time.Second
			c.HttpClient = &http.Client{Transport: cloned}
		} else {
			c.HttpClient = &http.Client{
				Transport: &http.Transport{ResponseHeaderTimeout: 30 * time.Second},
			}
		}
	}
	if c.Endpoint == "" {
		switch c.Region {
		case "us-east-1", "":
			c.Endpoint = "https://s3.amazonaws.com"
		default:
			c.Endpoint = fmt.Sprintf("https://s3.%s.amazonaws.com", c.Region)
		}
	}
}

func NewClient(cfg *Config) (*Client, error) {
	cfg.ApplyDefaults()
	return &Client{
		endpoint:   cfg.Endpoint,
		region:     cfg.Region,
		bucket:     cfg.BucketName,
		accessKey:  cfg.AccessKeyID,
		secretKey:  cfg.SecretAccessKey,
		httpClient: cfg.HttpClient,
		pathStyle:  cfg.PathStyle,
	}, nil
}

func (c *Client) HeadBucket(ctx context.Context) error {
	url, err := c.buildURL("")
	if err != nil {
		return err
	}
	resp, err := c.doRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		if err != nil {
			return nil, err
		}
		if err := signRequest(req, c.region, c.accessKey, c.secretKey, hashSHA256(nil)); err != nil {
			return nil, err
		}
		return req, nil
	}, true)
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("head bucket failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	url, err := c.buildURL(key)
	if err != nil {
		return false, err
	}
	resp, err := c.doRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		if err != nil {
			return nil, err
		}
		if err := signRequest(req, c.region, c.accessKey, c.secretKey, hashSHA256(nil)); err != nil {
			return nil, err
		}
		return req, nil
	}, true)
	if err != nil {
		return false, err
	}
	defer closeBody(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, responseError("head object", resp)
	}
}

func (c *Client) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	return c.put(ctx, key, r, size, false)
}

// PutIfAbsent creates an object only when the key does not exist yet.
func (c *Client) PutIfAbsent(ctx context.Context, key string, r io.Reader, size int64) error {
	return c.put(ctx, key, r, size, true)
}

func (c *Client) put(ctx context.Context, key string, r io.Reader, size int64, ifAbsent bool) error {
	if size > multipartThreshold {
		return c.putMultipart(ctx, key, r, size, ifAbsent)
	}
	return c.putSingle(ctx, key, r, size, ifAbsent)
}

func (c *Client) putSingle(ctx context.Context, key string, r io.Reader, size int64, ifAbsent bool) error {
	url, err := c.buildURL(key)
	if err != nil {
		return err
	}

	var start int64
	seeker, canRewind := r.(io.Seeker)
	if canRewind {
		start, err = seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			canRewind = false
		}
	}

	resp, err := c.doRetry(ctx, func() (*http.Request, error) {
		if canRewind {
			if _, err := seeker.Seek(start, io.SeekStart); err != nil {
				return nil, fmt.Errorf("rewind S3 request body: %w", err)
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, r)
		if err != nil {
			return nil, err
		}
		if size >= 0 {
			req.ContentLength = size
		}
		if ifAbsent {
			req.Header.Set("If-None-Match", "*")
		}
		if err := signRequest(req, c.region, c.accessKey, c.secretKey, "UNSIGNED-PAYLOAD"); err != nil {
			return nil, err
		}
		return req, nil
	}, canRewind)
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)

	if resp.StatusCode >= 300 {
		if ifAbsent && (resp.StatusCode == http.StatusPreconditionFailed || resp.StatusCode == http.StatusConflict) {
			return storagetypes.ErrAlreadyExists
		}
		return responseError("put object", resp)
	}
	return nil
}

func (c *Client) putMultipart(ctx context.Context, key string, r io.Reader, size int64, ifAbsent bool) error {
	uploadID, err := c.createMultipartUpload(ctx, key)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		if complete {
			return
		}
		abortCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		if err := c.abortMultipartUpload(abortCtx, key, uploadID); err != nil {
			log.FromContext(ctx).Warnf("Failed to abort S3 multipart upload: %v", err)
		}
	}()

	partSize := int64(multipartPartSize)
	if size <= partSize {
		partSize = size
	}
	if partSize < minMultipartPartSize && size > partSize {
		partSize = minMultipartPartSize
	}
	// S3 allows at most 10,000 parts; grow the part size for very large objects.
	if (size+partSize-1)/partSize > 10000 {
		partSize = (size + 9999) / 10000
		partSize = ((partSize + minMultipartPartSize - 1) / minMultipartPartSize) * minMultipartPartSize
	}
	parts := make([]completedPart, 0, (size+partSize-1)/partSize)
	for offset, number := int64(0), 1; offset < size; number++ {
		length := min(partSize, size-offset)
		buf := make([]byte, length)
		if _, err := io.ReadFull(r, buf); err != nil {
			return fmt.Errorf("read multipart body at part %d: %w", number, err)
		}
		etag, err := c.uploadPart(ctx, key, uploadID, number, buf)
		if err != nil {
			return err
		}
		parts = append(parts, completedPart{PartNumber: number, ETag: etag})
		offset += length
	}
	if err := c.completeMultipartUpload(ctx, key, uploadID, parts, ifAbsent); err != nil {
		return err
	}
	complete = true
	return nil
}

func (c *Client) createMultipartUpload(ctx context.Context, key string) (string, error) {
	url, err := c.buildURLWithQuery(key, url.Values{"uploads": {""}})
	if err != nil {
		return "", err
	}
	resp, err := c.doRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, http.NoBody)
		if err != nil {
			return nil, err
		}
		if err := signRequest(req, c.region, c.accessKey, c.secretKey, hashSHA256(nil)); err != nil {
			return nil, err
		}
		return req, nil
	}, true)
	if err != nil {
		return "", err
	}
	defer closeBody(resp.Body)
	if resp.StatusCode >= 300 {
		return "", responseError("create multipart upload", resp)
	}
	var result initiateMultipartUploadResult
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode multipart upload response: %w", err)
	}
	if result.UploadID == "" {
		return "", errors.New("S3 returned an empty multipart upload ID")
	}
	return result.UploadID, nil
}

func (c *Client) uploadPart(ctx context.Context, key, uploadID string, number int, body []byte) (string, error) {
	url, err := c.buildURLWithQuery(key, url.Values{
		"partNumber": {strconv.Itoa(number)},
		"uploadId":   {uploadID},
	})
	if err != nil {
		return "", err
	}
	resp, err := c.doRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.ContentLength = int64(len(body))
		req.Header.Set("Content-MD5", md5Base64(body))
		if err := signRequest(req, c.region, c.accessKey, c.secretKey, "UNSIGNED-PAYLOAD"); err != nil {
			return nil, err
		}
		return req, nil
	}, true)
	if err != nil {
		return "", err
	}
	defer closeBody(resp.Body)
	if resp.StatusCode >= 300 {
		return "", responseError(fmt.Sprintf("upload multipart part %d", number), resp)
	}
	etag := strings.TrimSpace(resp.Header.Get("ETag"))
	if etag == "" {
		return "", fmt.Errorf("S3 did not return an ETag for part %d", number)
	}
	return etag, nil
}

func (c *Client) completeMultipartUpload(ctx context.Context, key, uploadID string, parts []completedPart, ifAbsent bool) error {
	url, err := c.buildURLWithQuery(key, url.Values{"uploadId": {uploadID}})
	if err != nil {
		return err
	}
	payload, err := xml.Marshal(completeMultipartUpload{
		XMLName: xml.Name{Local: "CompleteMultipartUpload"},
		Parts:   parts,
	})
	if err != nil {
		return fmt.Errorf("encode complete multipart upload: %w", err)
	}
	resp, err := c.doRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.ContentLength = int64(len(payload))
		req.Header.Set("Content-Type", "application/xml")
		if ifAbsent {
			req.Header.Set("If-None-Match", "*")
		}
		if err := signRequest(req, c.region, c.accessKey, c.secretKey, hashSHA256(payload)); err != nil {
			return nil, err
		}
		return req, nil
	}, true)
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	if resp.StatusCode >= 300 {
		if ifAbsent && (resp.StatusCode == http.StatusPreconditionFailed || resp.StatusCode == http.StatusConflict) {
			return storagetypes.ErrAlreadyExists
		}
		return responseError("complete multipart upload", resp)
	}
	return nil
}

func (c *Client) abortMultipartUpload(ctx context.Context, key, uploadID string) error {
	url, err := c.buildURLWithQuery(key, url.Values{"uploadId": {uploadID}})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	if err := signRequest(req, c.region, c.accessKey, c.secretKey, hashSHA256(nil)); err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return responseError("abort multipart upload", resp)
	}
	return nil
}

func (c *Client) doRetry(ctx context.Context, build func() (*http.Request, error), canRetryBody bool) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < maxRequestAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		req, err := build()
		if err != nil {
			return nil, err
		}
		resp, err := c.httpClient.Do(req)
		if err == nil && !retryableStatus(resp.StatusCode) {
			return resp, nil
		}
		if err != nil {
			if resp != nil {
				closeBody(resp.Body)
			}
			lastErr = err
		} else {
			lastErr = responseError(req.Method+" "+req.URL.Path, resp)
			closeBody(resp.Body)
		}
		if !canRetryBody {
			return nil, lastErr
		}
		if attempt+1 >= maxRequestAttempts {
			break
		}
		delay := retryDelay(attempt)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, lastErr
}

func retryableStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func retryDelay(attempt int) time.Duration {
	return time.Duration(200*(1<<attempt)) * time.Millisecond
}

func closeBody(body io.ReadCloser) {
	if err := discardAndClose(body); err != nil {
		log.Warnf("Failed to close S3 response body: %v", err)
	}
}

func discardAndClose(body io.ReadCloser) error {
	_, readErr := io.Copy(io.Discard, io.LimitReader(body, 4096))
	return errors.Join(readErr, body.Close())
}

func md5Base64(data []byte) string {
	sum := md5.Sum(data)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func (c *Client) buildURLWithQuery(key string, query url.Values) (string, error) {
	raw, err := c.buildURL(key)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	rawQuery := query.Encode()
	if len(query) == 1 {
		if values, ok := query["uploads"]; ok && len(values) == 1 && values[0] == "" {
			rawQuery = "uploads"
		}
	}
	u.RawQuery = rawQuery
	return u.String(), nil
}

func (c *Client) buildURL(key string) (string, error) {
	if c.pathStyle {
		return fmt.Sprintf("%s/%s/%s", c.endpoint, c.bucket, escapePath(key)), nil
	}
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return "", err
	}
	u.Host = c.bucket + "." + u.Host
	u.Path = "/" + key
	u.RawPath = "/" + escapePath(key)
	return u.String(), nil
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func hashSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func signRequest(req *http.Request, region, accessKey, secretKey string, payloadHash string) error {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	date := now.Format("20060102")

	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	// Canonical headers. Host is required by SigV4, but Go stores it on
	// Request.Host/URL.Host rather than in Request.Header.
	headerValues := map[string]string{
		"host": req.URL.Host,
	}
	if req.Host != "" {
		headerValues["host"] = req.Host
	}
	for k := range req.Header {
		headerValues[strings.ToLower(k)] = strings.TrimSpace(req.Header.Get(k))
	}

	var headers []string
	for k := range headerValues {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	var canonicalHeaders strings.Builder
	for _, k := range headers {
		canonicalHeaders.WriteString(k)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(headerValues[k])
		canonicalHeaders.WriteString("\n")
	}

	signedHeaders := strings.Join(headers, ";")

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI(req.URL.Path),
		req.URL.RawQuery,
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := fmt.Sprintf("%s/%s/s3/aws4_request", date, region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hashSHA256([]byte(canonicalRequest)),
	}, "\n")

	kDate := hmacSHA256([]byte("AWS4"+secretKey), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, "s3")
	kSigning := hmacSHA256(kService, "aws4_request")

	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))

	auth := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, scope, signedHeaders, signature,
	)

	req.Header.Set("Authorization", auth)
	return nil
}

func responseError(operation string, resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("%s failed: %s", operation, resp.Status)
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		return fmt.Errorf("%s failed: %s", operation, resp.Status)
	}

	return fmt.Errorf("%s failed: %s: %s", operation, resp.Status, message)
}

func canonicalURI(path string) string {
	if path == "" {
		return "/"
	}
	return escapePath(path)
}

func escapePath(path string) string {
	var b strings.Builder
	for i := 0; i < len(path); i++ {
		c := path[i]
		if shouldEscapePathByte(c) {
			b.WriteByte('%')
			b.WriteByte("0123456789ABCDEF"[c>>4])
			b.WriteByte("0123456789ABCDEF"[c&15])
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func shouldEscapePathByte(c byte) bool {
	if c >= 'A' && c <= 'Z' {
		return false
	}
	if c >= 'a' && c <= 'z' {
		return false
	}
	if c >= '0' && c <= '9' {
		return false
	}
	switch c {
	case '-', '.', '_', '~', '/':
		return false
	default:
		return true
	}
}

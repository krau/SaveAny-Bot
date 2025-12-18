package s3

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	storconfig "github.com/krau/SaveAny-Bot/config/storage"
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

func NewClient(cfg storconfig.S3StorageConfig) (*Client, error) {
	endpoint := cfg.Endpoint
	if !strings.HasPrefix(endpoint, "http") {
		if cfg.UseSSL {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}

	return &Client{
		endpoint:   endpoint,
		region:     cfg.Region,
		bucket:     cfg.BucketName,
		accessKey:  cfg.AccessKeyID,
		secretKey:  cfg.SecretAccessKey,
		pathStyle:  !cfg.VirtualHost,
		httpClient: http.DefaultClient,
	}, nil
}

func (c *Client) HeadBucket(ctx context.Context) error {
	url := c.buildURL("")
	req, _ := http.NewRequestWithContext(ctx, "HEAD", url, nil)

	signRequest(req, c.region, c.accessKey, c.secretKey, hashSHA256(nil))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("head bucket failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) Exists(ctx context.Context, key string) bool {
	req, _ := http.NewRequestWithContext(ctx, "HEAD", c.buildURL(key), nil)
	signRequest(req, c.region, c.accessKey, c.secretKey, hashSHA256(nil))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (c *Client) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	req, _ := http.NewRequestWithContext(ctx, "PUT", c.buildURL(key), r)
	if size >= 0 {
		req.ContentLength = size
	}

	signRequest(req, c.region, c.accessKey, c.secretKey, "UNSIGNED-PAYLOAD")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("put object failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) buildURL(key string) string {
	if c.pathStyle {
		return fmt.Sprintf("%s/%s/%s", c.endpoint, c.bucket, key)
	}
	u, _ := url.Parse(c.endpoint)
	u.Host = c.bucket + "." + u.Host
	u.Path = "/" + key
	return u.String()
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

	// Canonical headers
	var headers []string
	for k := range req.Header {
		headers = append(headers, strings.ToLower(k))
	}
	sort.Strings(headers)

	var canonicalHeaders strings.Builder
	for _, k := range headers {
		canonicalHeaders.WriteString(k)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(req.Header.Get(k)))
		canonicalHeaders.WriteString("\n")
	}

	signedHeaders := strings.Join(headers, ";")

	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
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

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
		c.HttpClient = http.DefaultClient
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
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
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
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("head bucket failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) Exists(ctx context.Context, key string) bool {
	url, err := c.buildURL(key)
	if err != nil {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false
	}
	if err := signRequest(req, c.region, c.accessKey, c.secretKey, hashSHA256(nil)); err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (c *Client) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	url, err := c.buildURL(key)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "PUT", url, r)
	if err != nil {
		return err
	}
	if size >= 0 {
		req.ContentLength = size
	}

	if err := signRequest(req, c.region, c.accessKey, c.secretKey, "UNSIGNED-PAYLOAD"); err != nil {
		return err
	}

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

func (c *Client) buildURL(key string) (string, error) {
	if c.pathStyle {
		return fmt.Sprintf("%s/%s/%s", c.endpoint, c.bucket, key), nil
	}
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return "", err
	}
	u.Host = c.bucket + "." + u.Host
	u.Path = "/" + key
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

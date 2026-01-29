package directlinks

import (
	"mime"
	"net/url"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// parseFilename extracts filename from Content-Disposition header
// It handles multiple encoding scenarios:
// 1. RFC 5987/RFC 2231 format: filename*=UTF-8”%E6%B5%8B%E8%AF%95.zip (preferred, checked first)
// 2. MIME encoded-word: filename="=?UTF-8?B?5rWL6K+VLnppcA==?="
// 3. URL-encoded: filename="%E6%B5%8B%E8%AF%95.zip"
// 4. Plain ASCII filename
//
// The key fix is checking filename*= first before mime.ParseMediaType, because
// some servers send Content-Disposition headers with invalid characters that cause
// mime.ParseMediaType to fail, but the filename*= parameter is still valid.
func parseFilename(contentDisposition string) string {
	// First, try to find filename*= (RFC 5987 format, most reliable for non-ASCII)
	if filename := parseFilenameExtended(contentDisposition); filename != "" {
		return filename
	}

	// Try standard MIME parsing for regular filename= parameter
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err == nil {
		if filename := params["filename"]; filename != "" {
			return decodeFilenameParam(filename)
		}
	}

	// Fallback: manual parsing if mime.ParseMediaType fails
	return parseFilenameFallback(contentDisposition)
}

// parseFilenameExtended parses RFC 5987/RFC 2231 extended parameter format
// Format: filename*=charset'language'value (e.g., UTF-8”%E6%B5%8B%E8%AF%95.zip)
func parseFilenameExtended(cd string) string {
	// Look for filename*= (case-insensitive)
	lower := strings.ToLower(cd)
	idx := strings.Index(lower, "filename*=")
	if idx == -1 {
		return ""
	}

	// Extract the value after filename*=
	value := cd[idx+len("filename*="):]

	// Find the end of the value (next ; or end of string)
	if endIdx := strings.Index(value, ";"); endIdx != -1 {
		value = value[:endIdx]
	}
	value = strings.TrimSpace(value)

	// Parse charset'language'encoded-value format
	// Common format: UTF-8''%E6%B5%8B%E8%AF%95.zip
	parts := strings.SplitN(value, "''", 2)
	if len(parts) == 2 {
		// parts[0] is charset (e.g., "UTF-8")
		// parts[1] is percent-encoded value
		decoded, err := url.QueryUnescape(parts[1])
		if err == nil {
			return decoded
		}
	}

	// Try with single quote delimiter as well (some servers use this)
	parts = strings.SplitN(value, "'", 3)
	if len(parts) >= 3 {
		decoded, err := url.QueryUnescape(parts[2])
		if err == nil {
			return decoded
		}
	}

	return ""
}

// TryUrlQueryUnescape tries to unescape a URL-encoded string.
//
// If unescaping fails, it returns the original string.
func tryUrlQueryUnescape(s string) string {
	if decoded, err := url.QueryUnescape(s); err == nil {
		return decoded
	}
	return s
}

// decodeFilenameParam decodes a filename parameter value
// Handles MIME encoded-word, URL encoding, and GBK encoding fallback
func decodeFilenameParam(filename string) string {
	// Check if the filename is MIME encoded-word (e.g., =?UTF-8?B?...?=)
	if strings.HasPrefix(filename, "=?") {
		decoder := new(mime.WordDecoder)
		// Some servers use "UTF8" instead of "UTF-8", create a normalized copy
		normalizedFilename := strings.Replace(filename, "UTF8", "UTF-8", 1)
		if decoded, err := decoder.Decode(normalizedFilename); err == nil {
			return decoded
		}
	}

	// Try URL decoding
	decoded := tryUrlQueryUnescape(filename)

	// Check if the result is valid UTF-8. If not, try GBK decoding.
	// This handles the case where Chinese Windows servers send GBK-encoded filenames
	// which appear as garbled characters (e.g., "下载地址.zip" -> "���ص�ַ.zip")
	if !utf8.ValidString(decoded) {
		if gbkDecoded := tryDecodeGBK(decoded); gbkDecoded != "" {
			return gbkDecoded
		}
	}

	return decoded
}

// gbkDecoder is a reusable GBK decoder for better performance
var gbkDecoder = simplifiedchinese.GBK.NewDecoder()

// tryDecodeGBK attempts to decode a string as GBK/GB2312/GB18030 encoding
// Returns empty string if decoding fails or result is not valid UTF-8
func tryDecodeGBK(s string) string {
	// GBK uses 1-2 bytes per character. Single-byte chars are 0x00-0x7F (ASCII compatible).
	// Double-byte chars have first byte 0x81-0xFE and second byte 0x40-0xFE.
	// Skip if string is empty or all ASCII (valid UTF-8)
	if len(s) == 0 {
		return ""
	}

	// Create a fresh decoder since the transform state may be corrupted
	decoder := gbkDecoder
	decoded, err := decoder.Bytes([]byte(s))
	if err != nil {
		return ""
	}
	result := string(decoded)
	if utf8.ValidString(result) {
		return result
	}
	return ""
}

// parseFilenameFromURL extracts filename from URL path
// This is used as a fallback when Content-Disposition is not available
func parseFilenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	// Get the path part and extract the last segment
	path := parsed.Path
	if path == "" {
		return ""
	}

	// URL decode the path first
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		decodedPath = path
	}

	// Get the last segment of the path
	lastSlash := strings.LastIndex(decodedPath, "/")
	if lastSlash == -1 {
		return decodedPath
	}
	filename := decodedPath[lastSlash+1:]

	// Remove query string if somehow still present
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	return filename
}

// parseFilenameFallback manually parses filename= when mime.ParseMediaType fails
func parseFilenameFallback(cd string) string {
	// Look for filename= (case-insensitive)
	lower := strings.ToLower(cd)
	idx := strings.Index(lower, "filename=")
	if idx == -1 {
		return ""
	}

	// Skip "filename=" prefix
	value := cd[idx+len("filename="):]

	// Find the end of the value
	if endIdx := strings.Index(value, ";"); endIdx != -1 {
		value = value[:endIdx]
	}
	value = strings.TrimSpace(value)

	// Remove quotes if present
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	return decodeFilenameParam(value)
}

var progressUpdatesLevels = []struct {
	size        int64 // 文件大小阈值
	stepPercent int   // 每多少 % 更新一次
}{
	{10 << 20, 100},
	{50 << 20, 50},
	{200 << 20, 20},
	{500 << 20, 10},
}

func shouldUpdateProgress(total, downloaded int64, lastUpdatePercent int) bool {
	if total <= 0 || downloaded <= 0 {
		return false
	}

	percent := int((downloaded * 100) / total)
	if percent <= lastUpdatePercent {
		return false
	}

	step := progressUpdatesLevels[len(progressUpdatesLevels)-1].stepPercent
	for _, lvl := range progressUpdatesLevels {
		if total < lvl.size {
			step = lvl.stepPercent
			break
		}
	}

	return percent >= lastUpdatePercent+step
}

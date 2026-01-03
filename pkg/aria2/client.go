package aria2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

var (
	ErrInvalidURL      = errors.New("aria2: invalid URL")
	ErrRPCFailed       = errors.New("aria2: RPC call failed")
	ErrInvalidResponse = errors.New("aria2: invalid response")
)

// Client represents an aria2 JSON-RPC client
type Client struct {
	url    string
	secret string
	client *http.Client
	id     atomic.Int64
}

// rpcRequest represents a JSON-RPC 2.0 request
type rpcRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

// rpcResponse represents a JSON-RPC 2.0 response
type rpcResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError represents a JSON-RPC 2.0 error
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("aria2 RPC error %d: %s", e.Code, e.Message)
}

// Options for download
type Options map[string]any

// Status represents the status of a download
type Status struct {
	GID             string   `json:"gid"`
	Status          string   `json:"status"`
	TotalLength     string   `json:"totalLength"`
	CompletedLength string   `json:"completedLength"`
	UploadLength    string   `json:"uploadLength"`
	Bitfield        string   `json:"bitfield,omitempty"`
	DownloadSpeed   string   `json:"downloadSpeed"`
	UploadSpeed     string   `json:"uploadSpeed"`
	InfoHash        string   `json:"infoHash,omitempty"`
	NumSeeders      string   `json:"numSeeders,omitempty"`
	Seeder          string   `json:"seeder,omitempty"`
	PieceLength     string   `json:"pieceLength,omitempty"`
	NumPieces       string   `json:"numPieces,omitempty"`
	Connections     string   `json:"connections"`
	ErrorCode       string   `json:"errorCode,omitempty"`
	ErrorMessage    string   `json:"errorMessage,omitempty"`
	FollowedBy      []string `json:"followedBy,omitempty"`
	Following       string   `json:"following,omitempty"`
	BelongsTo       string   `json:"belongsTo,omitempty"`
	Dir             string   `json:"dir"`
	Files           []File   `json:"files"`
	BitTorrent      struct {
		AnnounceList [][]string `json:"announceList,omitempty"`
		Comment      string     `json:"comment,omitempty"`
		CreationDate int64      `json:"creationDate,omitempty"`
		Mode         string     `json:"mode,omitempty"`
		Info         struct {
			Name string `json:"name,omitempty"`
		} `json:"info"`
	} `json:"bittorrent"`
	VerifiedLength         string `json:"verifiedLength,omitempty"`
	VerifyIntegrityPending string `json:"verifyIntegrityPending,omitempty"`
}

// File represents a file in the download
type File struct {
	Index           string `json:"index"`
	Path            string `json:"path"`
	Length          string `json:"length"`
	CompletedLength string `json:"completedLength"`
	Selected        string `json:"selected"`
	URIs            []URI  `json:"uris"`
}

// URI represents a URI for a file
type URI struct {
	URI    string `json:"uri"`
	Status string `json:"status"`
}

// GlobalStat represents global statistics
type GlobalStat struct {
	DownloadSpeed   string `json:"downloadSpeed"`
	UploadSpeed     string `json:"uploadSpeed"`
	NumActive       string `json:"numActive"`
	NumWaiting      string `json:"numWaiting"`
	NumStopped      string `json:"numStopped"`
	NumStoppedTotal string `json:"numStoppedTotal"`
}

// Version represents aria2 version information
type Version struct {
	Version         string   `json:"version"`
	EnabledFeatures []string `json:"enabledFeatures"`
}

// NewClient creates a new aria2 client
// url: aria2 RPC URL (e.g., "http://localhost:6800/jsonrpc")
// secret: aria2 RPC secret token (optional, use empty string if not set)
func NewClient(url, secret string) (*Client, error) {
	if url == "" {
		return nil, ErrInvalidURL
	}

	return &Client{
		url:    url,
		secret: secret,
		client: &http.Client{},
	}, nil
}

// NewClientWithHTTPClient creates a new aria2 client with custom HTTP client
func NewClientWithHTTPClient(url, secret string, httpClient *http.Client) (*Client, error) {
	if url == "" {
		return nil, ErrInvalidURL
	}

	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{
		url:    url,
		secret: secret,
		client: httpClient,
	}, nil
}

// call makes a JSON-RPC call to aria2
func (c *Client) call(ctx context.Context, method string, params []any, result any) error {
	// Prepare params with secret token if set
	var rpcParams []any
	if c.secret != "" {
		rpcParams = append([]any{fmt.Sprintf("token:%s", c.secret)}, params...)
	} else {
		rpcParams = params
	}

	// Create request
	reqID := fmt.Sprintf("%d", c.id.Add(1))
	req := &rpcRequest{
		Jsonrpc: "2.0",
		ID:      reqID,
		Method:  method,
		Params:  rpcParams,
	}

	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal request: %v", ErrRPCFailed, err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("%w: failed to create request: %v", ErrRPCFailed, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: failed to send request: %v", ErrRPCFailed, err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: failed to read response: %v", ErrRPCFailed, err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: HTTP %d: %s", ErrRPCFailed, resp.StatusCode, string(body))
	}

	// Parse response
	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("%w: failed to unmarshal response: %v", ErrInvalidResponse, err)
	}

	// Check for RPC error
	if rpcResp.Error != nil {
		return rpcResp.Error
	}

	// Check response ID
	if rpcResp.ID != reqID {
		return fmt.Errorf("%w: response ID mismatch", ErrInvalidResponse)
	}

	// Unmarshal result if needed
	if result != nil {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("%w: failed to unmarshal result: %v", ErrInvalidResponse, err)
		}
	}

	return nil
}

// AddURI adds a new download with URIs
func (c *Client) AddURI(ctx context.Context, uris []string, options Options) (string, error) {
	var gid string
	params := []any{uris}
	if options != nil {
		params = append(params, options)
	}
	err := c.call(ctx, "aria2.addUri", params, &gid)
	return gid, err
}

// AddTorrent adds a new download with torrent file content
func (c *Client) AddTorrent(ctx context.Context, torrent []byte, uris []string, options Options) (string, error) {
	var gid string
	params := []any{torrent}
	if len(uris) > 0 {
		params = append(params, uris)
	}
	if options != nil {
		params = append(params, options)
	}
	err := c.call(ctx, "aria2.addTorrent", params, &gid)
	return gid, err
}

// AddMetalink adds a new download with metalink file content
func (c *Client) AddMetalink(ctx context.Context, metalink []byte, options Options) ([]string, error) {
	var gids []string
	params := []any{metalink}
	if options != nil {
		params = append(params, options)
	}
	err := c.call(ctx, "aria2.addMetalink", params, &gids)
	return gids, err
}

// Remove removes the download denoted by gid
func (c *Client) Remove(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.remove", []any{gid}, &result)
	return result, err
}

// ForceRemove removes the download denoted by gid forcefully
func (c *Client) ForceRemove(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.forceRemove", []any{gid}, &result)
	return result, err
}

// Pause pauses the download denoted by gid
func (c *Client) Pause(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.pause", []any{gid}, &result)
	return result, err
}

// PauseAll pauses all downloads
func (c *Client) PauseAll(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.pauseAll", []any{}, &result)
	return result, err
}

// ForcePause pauses the download denoted by gid forcefully
func (c *Client) ForcePause(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.forcePause", []any{gid}, &result)
	return result, err
}

// ForcePauseAll pauses all downloads forcefully
func (c *Client) ForcePauseAll(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.forcePauseAll", []any{}, &result)
	return result, err
}

// Unpause unpauses the download denoted by gid
func (c *Client) Unpause(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.unpause", []any{gid}, &result)
	return result, err
}

// UnpauseAll unpauses all downloads
func (c *Client) UnpauseAll(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.unpauseAll", []any{}, &result)
	return result, err
}

// TellStatus returns the progress of the download denoted by gid
func (c *Client) TellStatus(ctx context.Context, gid string, keys ...string) (*Status, error) {
	var status Status
	params := []any{gid}
	if len(keys) > 0 {
		params = append(params, keys)
	}
	err := c.call(ctx, "aria2.tellStatus", params, &status)
	return &status, err
}

// GetURIs returns the URIs used in the download denoted by gid
func (c *Client) GetURIs(ctx context.Context, gid string) ([]URI, error) {
	var uris []URI
	err := c.call(ctx, "aria2.getUris", []any{gid}, &uris)
	return uris, err
}

// GetFiles returns the file list of the download denoted by gid
func (c *Client) GetFiles(ctx context.Context, gid string) ([]File, error) {
	var files []File
	err := c.call(ctx, "aria2.getFiles", []any{gid}, &files)
	return files, err
}

// GetPeers returns a list of peers of the download denoted by gid
func (c *Client) GetPeers(ctx context.Context, gid string) ([]any, error) {
	var peers []any
	err := c.call(ctx, "aria2.getPeers", []any{gid}, &peers)
	return peers, err
}

// GetServers returns currently connected HTTP(S)/FTP/SFTP servers of the download denoted by gid
func (c *Client) GetServers(ctx context.Context, gid string) ([]any, error) {
	var servers []any
	err := c.call(ctx, "aria2.getServers", []any{gid}, &servers)
	return servers, err
}

// TellActive returns a list of active downloads
func (c *Client) TellActive(ctx context.Context, keys ...string) ([]Status, error) {
	var statuses []Status
	params := []any{}
	if len(keys) > 0 {
		params = append(params, keys)
	}
	err := c.call(ctx, "aria2.tellActive", params, &statuses)
	return statuses, err
}

// TellWaiting returns a list of waiting downloads
func (c *Client) TellWaiting(ctx context.Context, offset, num int, keys ...string) ([]Status, error) {
	var statuses []Status
	params := []any{offset, num}
	if len(keys) > 0 {
		params = append(params, keys)
	}
	err := c.call(ctx, "aria2.tellWaiting", params, &statuses)
	return statuses, err
}

// TellStopped returns a list of stopped downloads
func (c *Client) TellStopped(ctx context.Context, offset, num int, keys ...string) ([]Status, error) {
	var statuses []Status
	params := []any{offset, num}
	if len(keys) > 0 {
		params = append(params, keys)
	}
	err := c.call(ctx, "aria2.tellStopped", params, &statuses)
	return statuses, err
}

// ChangePosition changes the position of the download denoted by gid
func (c *Client) ChangePosition(ctx context.Context, gid string, pos int, how string) (int, error) {
	var result int
	err := c.call(ctx, "aria2.changePosition", []any{gid, pos, how}, &result)
	return result, err
}

// ChangeURI changes the URI of the download denoted by gid
func (c *Client) ChangeURI(ctx context.Context, gid string, fileIndex int, delURIs []string, addURIs []string) ([]int, error) {
	var result []int
	params := []any{gid, fileIndex, delURIs, addURIs}
	err := c.call(ctx, "aria2.changeUri", params, &result)
	return result, err
}

// GetOption returns options of the download denoted by gid
func (c *Client) GetOption(ctx context.Context, gid string) (Options, error) {
	var options Options
	err := c.call(ctx, "aria2.getOption", []any{gid}, &options)
	return options, err
}

// ChangeOption changes options of the download denoted by gid dynamically
func (c *Client) ChangeOption(ctx context.Context, gid string, options Options) (string, error) {
	var result string
	err := c.call(ctx, "aria2.changeOption", []any{gid, options}, &result)
	return result, err
}

// GetGlobalOption returns the global options
func (c *Client) GetGlobalOption(ctx context.Context) (Options, error) {
	var options Options
	err := c.call(ctx, "aria2.getGlobalOption", []any{}, &options)
	return options, err
}

// ChangeGlobalOption changes global options dynamically
func (c *Client) ChangeGlobalOption(ctx context.Context, options Options) (string, error) {
	var result string
	err := c.call(ctx, "aria2.changeGlobalOption", []any{options}, &result)
	return result, err
}

// GetGlobalStat returns global statistics such as the overall download and upload speed
func (c *Client) GetGlobalStat(ctx context.Context) (*GlobalStat, error) {
	var stat GlobalStat
	err := c.call(ctx, "aria2.getGlobalStat", []any{}, &stat)
	return &stat, err
}

// PurgeDownloadResult purges completed/error/removed downloads
func (c *Client) PurgeDownloadResult(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.purgeDownloadResult", []any{}, &result)
	return result, err
}

// RemoveDownloadResult removes a completed/error/removed download denoted by gid
func (c *Client) RemoveDownloadResult(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.removeDownloadResult", []any{gid}, &result)
	return result, err
}

// GetVersion returns the version of aria2 and the list of enabled features
func (c *Client) GetVersion(ctx context.Context) (*Version, error) {
	var version Version
	err := c.call(ctx, "aria2.getVersion", []any{}, &version)
	return &version, err
}

// GetSessionInfo returns session information
func (c *Client) GetSessionInfo(ctx context.Context) (map[string]any, error) {
	var info map[string]any
	err := c.call(ctx, "aria2.getSessionInfo", []any{}, &info)
	return info, err
}

// Shutdown shuts down aria2
func (c *Client) Shutdown(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.shutdown", []any{}, &result)
	return result, err
}

// ForceShutdown shuts down aria2 forcefully
func (c *Client) ForceShutdown(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.forceShutdown", []any{}, &result)
	return result, err
}

// SaveSession saves the current session to a file
func (c *Client) SaveSession(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.saveSession", []any{}, &result)
	return result, err
}

// MultiCall executes multiple method calls in a single request (system.multicall)
func (c *Client) MultiCall(ctx context.Context, calls []map[string]any) ([]any, error) {
	var results []any
	err := c.call(ctx, "system.multicall", []any{calls}, &results)
	return results, err
}

// ListMethods lists all available RPC methods
func (c *Client) ListMethods(ctx context.Context) ([]string, error) {
	var methods []string
	err := c.call(ctx, "system.listMethods", []any{}, &methods)
	return methods, err
}

// ListNotifications lists all available RPC notifications
func (c *Client) ListNotifications(ctx context.Context) ([]string, error) {
	var notifications []string
	err := c.call(ctx, "system.listNotifications", []any{}, &notifications)
	return notifications, err
}

// IsDownloadComplete checks if the download is complete
func (s *Status) IsDownloadComplete() bool {
	return s.Status == "complete"
}

// IsDownloadActive checks if the download is active
func (s *Status) IsDownloadActive() bool {
	return s.Status == "active"
}

// IsDownloadWaiting checks if the download is waiting
func (s *Status) IsDownloadWaiting() bool {
	return s.Status == "waiting"
}

// IsDownloadPaused checks if the download is paused
func (s *Status) IsDownloadPaused() bool {
	return s.Status == "paused"
}

// IsDownloadError checks if the download has an error
func (s *Status) IsDownloadError() bool {
	return s.Status == "error"
}

// IsDownloadRemoved checks if the download is removed
func (s *Status) IsDownloadRemoved() bool {
	return s.Status == "removed"
}

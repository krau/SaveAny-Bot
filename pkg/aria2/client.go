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

type Options map[string]any

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

type File struct {
	Index           string `json:"index"`
	Path            string `json:"path"`
	Length          string `json:"length"`
	CompletedLength string `json:"completedLength"`
	Selected        string `json:"selected"`
	URIs            []URI  `json:"uris"`
}

type URI struct {
	URI    string `json:"uri"`
	Status string `json:"status"`
}

type GlobalStat struct {
	DownloadSpeed   string `json:"downloadSpeed"`
	UploadSpeed     string `json:"uploadSpeed"`
	NumActive       string `json:"numActive"`
	NumWaiting      string `json:"numWaiting"`
	NumStopped      string `json:"numStopped"`
	NumStoppedTotal string `json:"numStoppedTotal"`
}

type Version struct {
	Version         string   `json:"version"`
	EnabledFeatures []string `json:"enabledFeatures"`
}

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

func (c *Client) call(ctx context.Context, method string, params []any, result any) error {
	var rpcParams []any
	if c.secret != "" {
		rpcParams = append([]any{fmt.Sprintf("token:%s", c.secret)}, params...)
	} else {
		rpcParams = params
	}

	reqID := fmt.Sprintf("%d", c.id.Add(1))
	req := &rpcRequest{
		Jsonrpc: "2.0",
		ID:      reqID,
		Method:  method,
		Params:  rpcParams,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal request: %v", ErrRPCFailed, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("%w: failed to create request: %v", ErrRPCFailed, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: failed to send request: %v", ErrRPCFailed, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: failed to read response: %v", ErrRPCFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: HTTP %d: %s", ErrRPCFailed, resp.StatusCode, string(body))
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("%w: failed to unmarshal response: %v", ErrInvalidResponse, err)
	}

	if rpcResp.Error != nil {
		return rpcResp.Error
	}

	if rpcResp.ID != reqID {
		return fmt.Errorf("%w: response ID mismatch", ErrInvalidResponse)
	}

	if result != nil {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("%w: failed to unmarshal result: %v", ErrInvalidResponse, err)
		}
	}

	return nil
}

func (c *Client) AddURI(ctx context.Context, uris []string, options Options) (string, error) {
	var gid string
	params := []any{uris}
	if options != nil {
		params = append(params, options)
	}
	err := c.call(ctx, "aria2.addUri", params, &gid)
	return gid, err
}

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

func (c *Client) AddMetalink(ctx context.Context, metalink []byte, options Options) ([]string, error) {
	var gids []string
	params := []any{metalink}
	if options != nil {
		params = append(params, options)
	}
	err := c.call(ctx, "aria2.addMetalink", params, &gids)
	return gids, err
}

func (c *Client) Remove(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.remove", []any{gid}, &result)
	return result, err
}

func (c *Client) ForceRemove(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.forceRemove", []any{gid}, &result)
	return result, err
}

func (c *Client) Pause(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.pause", []any{gid}, &result)
	return result, err
}

func (c *Client) PauseAll(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.pauseAll", []any{}, &result)
	return result, err
}

func (c *Client) ForcePause(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.forcePause", []any{gid}, &result)
	return result, err
}

func (c *Client) ForcePauseAll(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.forcePauseAll", []any{}, &result)
	return result, err
}

func (c *Client) Unpause(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.unpause", []any{gid}, &result)
	return result, err
}

func (c *Client) UnpauseAll(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.unpauseAll", []any{}, &result)
	return result, err
}

func (c *Client) TellStatus(ctx context.Context, gid string, keys ...string) (*Status, error) {
	var status Status
	params := []any{gid}
	if len(keys) > 0 {
		params = append(params, keys)
	}
	err := c.call(ctx, "aria2.tellStatus", params, &status)
	return &status, err
}

func (c *Client) GetURIs(ctx context.Context, gid string) ([]URI, error) {
	var uris []URI
	err := c.call(ctx, "aria2.getUris", []any{gid}, &uris)
	return uris, err
}

func (c *Client) GetFiles(ctx context.Context, gid string) ([]File, error) {
	var files []File
	err := c.call(ctx, "aria2.getFiles", []any{gid}, &files)
	return files, err
}

func (c *Client) GetPeers(ctx context.Context, gid string) ([]any, error) {
	var peers []any
	err := c.call(ctx, "aria2.getPeers", []any{gid}, &peers)
	return peers, err
}

func (c *Client) GetServers(ctx context.Context, gid string) ([]any, error) {
	var servers []any
	err := c.call(ctx, "aria2.getServers", []any{gid}, &servers)
	return servers, err
}

func (c *Client) TellActive(ctx context.Context, keys ...string) ([]Status, error) {
	var statuses []Status
	params := []any{}
	if len(keys) > 0 {
		params = append(params, keys)
	}
	err := c.call(ctx, "aria2.tellActive", params, &statuses)
	return statuses, err
}

func (c *Client) TellWaiting(ctx context.Context, offset, num int, keys ...string) ([]Status, error) {
	var statuses []Status
	params := []any{offset, num}
	if len(keys) > 0 {
		params = append(params, keys)
	}
	err := c.call(ctx, "aria2.tellWaiting", params, &statuses)
	return statuses, err
}

func (c *Client) TellStopped(ctx context.Context, offset, num int, keys ...string) ([]Status, error) {
	var statuses []Status
	params := []any{offset, num}
	if len(keys) > 0 {
		params = append(params, keys)
	}
	err := c.call(ctx, "aria2.tellStopped", params, &statuses)
	return statuses, err
}

func (c *Client) ChangePosition(ctx context.Context, gid string, pos int, how string) (int, error) {
	var result int
	err := c.call(ctx, "aria2.changePosition", []any{gid, pos, how}, &result)
	return result, err
}

func (c *Client) ChangeURI(ctx context.Context, gid string, fileIndex int, delURIs []string, addURIs []string) ([]int, error) {
	var result []int
	params := []any{gid, fileIndex, delURIs, addURIs}
	err := c.call(ctx, "aria2.changeUri", params, &result)
	return result, err
}

func (c *Client) GetOption(ctx context.Context, gid string) (Options, error) {
	var options Options
	err := c.call(ctx, "aria2.getOption", []any{gid}, &options)
	return options, err
}

func (c *Client) ChangeOption(ctx context.Context, gid string, options Options) (string, error) {
	var result string
	err := c.call(ctx, "aria2.changeOption", []any{gid, options}, &result)
	return result, err
}

func (c *Client) GetGlobalOption(ctx context.Context) (Options, error) {
	var options Options
	err := c.call(ctx, "aria2.getGlobalOption", []any{}, &options)
	return options, err
}

func (c *Client) ChangeGlobalOption(ctx context.Context, options Options) (string, error) {
	var result string
	err := c.call(ctx, "aria2.changeGlobalOption", []any{options}, &result)
	return result, err
}

func (c *Client) GetGlobalStat(ctx context.Context) (*GlobalStat, error) {
	var stat GlobalStat
	err := c.call(ctx, "aria2.getGlobalStat", []any{}, &stat)
	return &stat, err
}

func (c *Client) PurgeDownloadResult(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.purgeDownloadResult", []any{}, &result)
	return result, err
}

func (c *Client) RemoveDownloadResult(ctx context.Context, gid string) (string, error) {
	var result string
	err := c.call(ctx, "aria2.removeDownloadResult", []any{gid}, &result)
	return result, err
}

func (c *Client) GetVersion(ctx context.Context) (*Version, error) {
	var version Version
	err := c.call(ctx, "aria2.getVersion", []any{}, &version)
	return &version, err
}

func (c *Client) GetSessionInfo(ctx context.Context) (map[string]any, error) {
	var info map[string]any
	err := c.call(ctx, "aria2.getSessionInfo", []any{}, &info)
	return info, err
}

func (c *Client) Shutdown(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.shutdown", []any{}, &result)
	return result, err
}

func (c *Client) ForceShutdown(ctx context.Context) (string, error) {
	var result string
	err := c.call(ctx, "aria2.forceShutdown", []any{}, &result)
	return result, err
}

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

func (c *Client) ListMethods(ctx context.Context) ([]string, error) {
	var methods []string
	err := c.call(ctx, "system.listMethods", []any{}, &methods)
	return methods, err
}

func (c *Client) ListNotifications(ctx context.Context) ([]string, error) {
	var notifications []string
	err := c.call(ctx, "system.listNotifications", []any{}, &notifications)
	return notifications, err
}

func (s *Status) IsDownloadComplete() bool {
	return s.Status == "complete"
}

func (s *Status) IsDownloadActive() bool {
	return s.Status == "active"
}

func (s *Status) IsDownloadWaiting() bool {
	return s.Status == "waiting"
}

func (s *Status) IsDownloadPaused() bool {
	return s.Status == "paused"
}

func (s *Status) IsDownloadError() bool {
	return s.Status == "error"
}

func (s *Status) IsDownloadRemoved() bool {
	return s.Status == "removed"
}

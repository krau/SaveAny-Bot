package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/log"
	ytdlp "github.com/lrstanley/go-ytdlp"

	"github.com/krau/SaveAny-Bot/config"
)

// noProxy is the candidate for a direct run without --proxy.
const noProxy = ""

// RunWithProxyFailover runs cmd once per configured proxy, in order, until a
// run succeeds. With no proxies configured it runs once without --proxy.
// Cancellation stops the rotation immediately.
func RunWithProxyFailover(ctx context.Context, cmd *ytdlp.Command, cfg config.YtdlpConfig, args ...string) (*ytdlp.Result, error) {
	logger := log.FromContext(ctx)
	candidates := proxyCandidates(cfg)

	var lastErr error
	for attempt, proxy := range candidates {
		// Each Run rebuilds the command line, so replacing the proxy on the same
		// builder never stacks --proxy flags.
		if proxy == noProxy {
			cmd = cmd.UnsetProxy()
		} else {
			cmd = cmd.Proxy(proxy)
		}

		result, err := cmd.Run(ctx, args...)
		switch {
		case err == nil && result.ExitCode == 0:
			if attempt > 0 {
				logger.Infof("yt-dlp succeeded via proxy %d/%d (%s)", attempt+1, len(candidates), redactProxy(proxy))
			}
			return result, nil
		case err != nil:
			lastErr = err
		default:
			lastErr = fmt.Errorf("yt-dlp exited with code %d: %s", result.ExitCode, result.Stderr)
		}
		lastErr = redactProxiesInError(lastErr, candidates)

		if errors.Is(lastErr, context.Canceled) || ctx.Err() != nil {
			return nil, lastErr
		}
		if len(candidates) > 1 {
			logger.Warnf("yt-dlp failed via proxy %d/%d (%s): %v", attempt+1, len(candidates), redactProxy(proxy), lastErr)
		}
	}

	if len(candidates) > 1 {
		return nil, fmt.Errorf("all %d proxies failed, last error: %w", len(candidates), lastErr)
	}
	return nil, lastErr
}

// proxyCandidates returns the configured proxies, trimmed and deduplicated in
// their configured order, or a single direct candidate when none are set.
func proxyCandidates(cfg config.YtdlpConfig) []string {
	seen := make(map[string]struct{}, len(cfg.Proxies))
	candidates := make([]string, 0, len(cfg.Proxies))
	for _, proxy := range cfg.Proxies {
		proxy = strings.TrimSpace(proxy)
		if proxy == "" {
			continue
		}
		if _, dup := seen[proxy]; dup {
			continue
		}
		seen[proxy] = struct{}{}
		candidates = append(candidates, proxy)
	}
	if len(candidates) == 0 {
		return []string{noProxy}
	}
	return candidates
}

// redactProxy hides the password in a proxy URL before it is logged.
func redactProxy(proxy string) string {
	if proxy == noProxy {
		return "direct"
	}
	u, err := url.Parse(proxy)
	if err != nil || u.Host == "" {
		return "<proxy>"
	}
	if u.User != nil {
		if _, ok := u.User.Password(); ok {
			u.User = url.UserPassword(u.User.Username(), "xxxxx")
		}
	}
	return u.Redacted()
}

// redactedError keeps the original error for errors.Is/As while replacing its
// message.
type redactedError struct {
	msg string
	err error
}

func (e *redactedError) Error() string { return e.msg }
func (e *redactedError) Unwrap() error { return e.err }

// redactProxiesInError replaces proxy URLs that yt-dlp echoed into err (for
// example in connection errors) with their redacted form.
func redactProxiesInError(err error, proxies []string) error {
	msg := err.Error()
	redacted := msg
	for _, proxy := range proxies {
		if proxy == noProxy {
			continue
		}
		if safe := redactProxy(proxy); safe != proxy {
			redacted = strings.ReplaceAll(redacted, proxy, safe)
		}
	}
	if redacted == msg {
		return err
	}
	return &redactedError{msg: redacted, err: err}
}

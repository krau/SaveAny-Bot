//go:build !no_jsparser

package js

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/MoeForge/moejs"
	"github.com/blang/semver"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/common/utils/netutil"
	"github.com/krau/SaveAny-Bot/parsers/parsers"
)

func jsRegisterParser(vm *moejs.Runtime) moejs.NativeFunc {
	return func(_ *moejs.Realm, _ moejs.Value, args []moejs.Value) (moejs.Value, error) {
		jsObj := moejs.Arg(args, 0)
		if !jsObj.IsObject() {
			return moejs.Undefined(), errors.New("registerParser expects an object { canHandle, parse }")
		}

		metaValue, err := vm.Get(jsObj, "metadata")
		if err != nil {
			return moejs.Undefined(), err
		}
		if metaValue.IsNullish() {
			return moejs.Undefined(), errors.New("parser must provide metadata")
		}
		var metadata PluginMeta
		if err := vm.Unmarshal(metaValue, &metadata); err != nil {
			return moejs.Undefined(), fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		if metadata.APIVersion == "" {
			metadata.APIVersion = SupportedAPIVersion
		}
		if metadata.APIVersion != SupportedAPIVersion {
			return moejs.Undefined(), fmt.Errorf("parser apiVersion %s is not supported, expected %s", metadata.APIVersion, SupportedAPIVersion)
		}
		pluginV, err := semver.Parse(metadata.Version)
		if err != nil {
			return moejs.Undefined(), fmt.Errorf("invalid parser version %q: %w", metadata.Version, err)
		}
		if pluginV.LT(MinimumParserVersion) {
			return moejs.Undefined(), fmt.Errorf("parser version %s is not supported, must be at least %s", metadata.Version, MinimumParserVersion)
		}
		if pluginV.Major > LatestParserVersion.Major {
			log.Printf("warning: parser major version %d is newer than latest supported major version %d", pluginV.Major, LatestParserVersion.Major)
		}

		handleFn, err := vm.Get(jsObj, "canHandle")
		if err != nil {
			return moejs.Undefined(), err
		}
		parseFn, err := vm.Get(jsObj, "parse")
		if err != nil {
			return moejs.Undefined(), err
		}
		if handleFn.IsNullish() {
			return moejs.Undefined(), errors.New("parser must provide a canHandle function")
		}
		if parseFn.IsNullish() {
			return moejs.Undefined(), errors.New("parser must provide a parse function")
		}
		parsers.Add(newJSParser(vm, handleFn, parseFn, metadata))
		return moejs.Undefined(), nil
	}
}

var jsConsole = func(logger *log.Logger) map[string]any {
	format := func(args []moejs.Value) []any {
		out := make([]any, len(args))
		for i, arg := range args {
			out[i] = arg.String()
		}
		return out
	}
	return map[string]any{
		"log": moejs.NativeFunc(func(_ *moejs.Realm, _ moejs.Value, args []moejs.Value) (moejs.Value, error) {
			if len(args) == 0 {
				return moejs.Undefined(), nil
			}
			values := format(args)
			logger.Info(values[0], values[1:]...)
			return moejs.Undefined(), nil
		}),
		"error": moejs.NativeFunc(func(_ *moejs.Realm, _ moejs.Value, args []moejs.Value) (moejs.Value, error) {
			if len(args) == 0 {
				return moejs.Undefined(), nil
			}
			values := format(args)
			logger.Error(values[0], values[1:]...)
			return moejs.Undefined(), nil
		}),
	}
}

/*
jsGhttp provides a http helper for js plugins

It provides the following functions:
  - get(url): performs a GET request and returns the response body as string
  - getJSON(url): performs a GET request and returns the response body parsed as JSON
  - head(url): performs a HEAD request and returns the response headers and status code
*/
var jsGhttp = func(vm *moejs.Runtime) map[string]any {
	client := netutil.DefaultParserHTTPClient()
	return map[string]any{
		"get": moejs.NativeFunc(func(_ *moejs.Realm, _ moejs.Value, args []moejs.Value) (moejs.Value, error) {
			url := moejs.Arg(args, 0).String()
			resp, err := client.Get(url)
			if err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Sprintf("failed to fetch %s: %v", url, err),
				})
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return vm.FromGo(map[string]any{
					"error":  fmt.Sprintf("failed to fetch %s: %s", url, resp.Status),
					"status": resp.StatusCode,
				})
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Errorf("failed to read response body: %w", err).Error(),
				})
			}
			return moejs.String(string(body)), nil
		}),
		"getJSON": moejs.NativeFunc(func(_ *moejs.Realm, _ moejs.Value, args []moejs.Value) (moejs.Value, error) {
			url := moejs.Arg(args, 0).String()

			resp, err := client.Get(url)
			if err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Sprintf("failed to fetch %s: %v", url, err),
				})
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return vm.FromGo(map[string]any{
					"error":  fmt.Sprintf("failed to fetch %s: %s", url, resp.Status),
					"status": resp.StatusCode,
				})
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Errorf("failed to read response body: %w", err).Error(),
				})
			}
			var jsonData map[string]any
			if err := json.Unmarshal(body, &jsonData); err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Errorf("failed to unmarshal JSON: %w", err).Error(),
				})
			}
			return vm.FromGo(map[string]any{
				"data": jsonData,
			})
		}),
		"head": moejs.NativeFunc(func(_ *moejs.Realm, _ moejs.Value, args []moejs.Value) (moejs.Value, error) {
			url := moejs.Arg(args, 0).String()
			resp, err := client.Head(url)
			if err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Sprintf("failed to fetch %s: %v", url, err),
				})
			}
			defer resp.Body.Close()
			headers := make(map[string]string)
			for k, v := range resp.Header {
				headers[k] = v[0]
			}
			return vm.FromGo(map[string]any{
				"status":  resp.StatusCode,
				"headers": headers,
			})
		}),
	}
}

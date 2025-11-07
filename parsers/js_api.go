package parsers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/blang/semver"
	"github.com/charmbracelet/log"
	"github.com/dop251/goja"
	"github.com/krau/SaveAny-Bot/common/utils/netutil"
	"github.com/playwright-community/playwright-go"
)

func jsRegisterParser(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		jsObj := call.Argument(0)
		if jsObj == nil || goja.IsUndefined(jsObj) || goja.IsNull(jsObj) {
			panic("registerParser expects an object { canHandle, parse }")
		}

		obj := jsObj.ToObject(vm)
		if obj == nil {
			panic("registerParser: cannot convert argument to object")
		}
		metaValue := obj.Get("metadata")
		if metaValue == nil || goja.IsUndefined(metaValue) {
			panic("parser must provide metadata")
		}
		var metadata PluginMeta
		if exported := metaValue.Export(); exported != nil {
			data, err := json.Marshal(exported)
			if err != nil {
				panic(fmt.Sprintf("failed to marshal metadata to JSON: %v", err))
			}
			if err := json.Unmarshal(data, &metadata); err != nil {
				panic(fmt.Sprintf("failed to unmarshal JSON to PluginMeta: %v", err))
			}
		} else {
			panic("metadata cannot be null or undefined")
		}

		pluginV := semver.MustParse(metadata.Version)
		if pluginV.LT(MinimumParserVersion) {
			panic(fmt.Sprintf("parser version %s is not supported, must be at least %s", metadata.Version, MinimumParserVersion))
		}
		if pluginV.Major > LatestParserVersion.Major {
			panic(fmt.Sprintf("parser major version %d is too new, latest supported major version is %d", pluginV.Major, LatestParserVersion.Major))
		}

		handleFn := obj.Get("canHandle")
		parseFn := obj.Get("parse")
		if parseFn == nil || goja.IsUndefined(parseFn) {
			panic("parser must provide a parse function")
		}

		parsers = append(parsers, newJSParser(vm, handleFn, parseFn, metadata))
		return goja.Undefined()
	}
}

var jsConsole = func(logger *log.Logger) map[string]any {
	return map[string]any{
		"log": func(args ...any) {
			if len(args) == 0 {
				return
			}
			if len(args) > 1 {
				logger.Info(args[0], args[1:]...)
			} else {
				logger.Info(args[0])
			}
		},
	}
}

/*
jsGhttp provides a http helper for js plugins

It provides the following functions:
  - get(url): performs a GET request and returns the response body as string
  - getJSON(url): performs a GET request and returns the response body parsed as JSON
  - head(url): performs a HEAD request and returns the response headers and status code
*/
var jsGhttp = func(vm *goja.Runtime) *goja.Object {
	ghttp := vm.NewObject()
	client := netutil.DefaultParserHTTPClient()
	ghttp.Set("get", func(call goja.FunctionCall) goja.Value {
		url := call.Argument(0).String()
		resp, err := client.Get(url)
		if err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Sprintf("failed to fetch %s: %v", url, err),
			})
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return vm.ToValue(map[string]any{
				"error":  fmt.Sprintf("failed to fetch %s: %s", url, resp.Status),
				"status": resp.StatusCode,
			})
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Errorf("failed to read response body: %w", err).Error(),
			})
		}
		return vm.ToValue(string(body))
	})
	ghttp.Set("getJSON", func(call goja.FunctionCall) goja.Value {
		url := call.Argument(0).String()

		resp, err := client.Get(url)
		if err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Sprintf("failed to fetch %s: %v", url, err),
			})
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return vm.ToValue(map[string]any{
				"error":  fmt.Sprintf("failed to fetch %s: %s", url, resp.Status),
				"status": resp.StatusCode,
			})
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Errorf("failed to read response body: %w", err).Error(),
			})
		}
		var jsonData map[string]any
		if err := json.Unmarshal(body, &jsonData); err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Errorf("failed to unmarshal JSON: %w", err).Error(),
			})
		}
		return vm.ToValue(map[string]any{
			"data": jsonData,
		})
	})
	ghttp.Set("head", func(call goja.FunctionCall) goja.Value {
		url := call.Argument(0).String()
		resp, err := client.Head(url)
		if err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Sprintf("failed to fetch %s: %v", url, err),
			})
		}
		defer resp.Body.Close()
		headers := make(map[string]string)
		for k, v := range resp.Header {
			headers[k] = v[0]
		}
		return vm.ToValue(map[string]any{
			"status":  resp.StatusCode,
			"headers": headers,
		})
	})
	return ghttp
}

var jsPlaywright = func(vm *goja.Runtime, logger *log.Logger) *goja.Object {
	pwObj := vm.NewObject()
	var installOnce sync.Once
	slogger := slog.New(logger)
	pwObj.Set("get", func(call goja.FunctionCall) goja.Value {
		url := call.Argument(0).String()
		var installErr error
		installOnce.Do(func() {
			installErr = playwright.Install(&playwright.RunOptions{
				Browsers:        []string{"chromium"},
				DriverDirectory: "./playwright",
				Logger:          slogger,
			})
		})
		if installErr != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Sprintf("failed to install playwright: %v", installErr),
			})
		}

		pw, err := playwright.Run(&playwright.RunOptions{
			DriverDirectory: "./playwright",
			Logger:          slogger,
		})
		if err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Sprintf("failed to start playwright: %v", err),
			})
		}
		defer pw.Stop()

		browser, err := pw.Chromium.Launch()
		if err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Sprintf("failed to launch browser: %v", err),
			})
		}
		defer browser.Close()

		page, err := browser.NewPage()
		if err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Sprintf("failed to create page: %v", err),
			})
		}

		resp, err := page.Goto(url, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
			Timeout:   playwright.Float(60000),
		})
		if err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Sprintf("failed to navigate: %v", err),
			})
		}
		if resp != nil && resp.Status() >= 400 {
			return vm.ToValue(map[string]any{
				"error": fmt.Sprintf("bad status code: %d", resp.Status()),
			})
		}
		content, err := page.Content()
		if err != nil {
			return vm.ToValue(map[string]any{
				"error": fmt.Sprintf("failed to get page content: %v", err),
			})
		}
		return vm.ToValue(content)
	})
	return pwObj
}

//go:build !no_jsparser && !no_playwright

package js

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/dop251/goja"
	"github.com/playwright-community/playwright-go"
)

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

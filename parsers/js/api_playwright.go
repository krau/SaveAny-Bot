//go:build !no_jsparser && !no_playwright

package js

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/MoeForge/moejs"
	"github.com/charmbracelet/log"
	"github.com/mxschmitt/playwright-go"
)

var jsPlaywright = func(vm *moejs.Runtime, logger *log.Logger) map[string]any {
	var installOnce sync.Once
	slogger := slog.New(logger)
	return map[string]any{
		"get": moejs.NativeFunc(func(_ *moejs.Realm, _ moejs.Value, args []moejs.Value) (moejs.Value, error) {
			url := moejs.Arg(args, 0).String()
			var installErr error
			installOnce.Do(func() {
				installErr = playwright.Install(&playwright.RunOptions{
					Browsers:        []string{"chromium"},
					DriverDirectory: "./playwright",
					Logger:          slogger,
				})
			})
			if installErr != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Sprintf("failed to install playwright: %v", installErr),
				})
			}

			pw, err := playwright.Run(&playwright.RunOptions{
				DriverDirectory: "./playwright",
				Logger:          slogger,
			})
			if err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Sprintf("failed to start playwright: %v", err),
				})
			}
			defer pw.Stop()

			browser, err := pw.Chromium.Launch()
			if err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Sprintf("failed to launch browser: %v", err),
				})
			}
			defer browser.Close()

			page, err := browser.NewPage()
			if err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Sprintf("failed to create page: %v", err),
				})
			}

			resp, err := page.Goto(url, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateNetworkidle,
				Timeout:   playwright.Float(60000),
			})
			if err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Sprintf("failed to navigate: %v", err),
				})
			}
			if resp != nil && resp.Status() >= 400 {
				return vm.FromGo(map[string]any{
					"error": fmt.Sprintf("bad status code: %d", resp.Status()),
				})
			}
			content, err := page.Content()
			if err != nil {
				return vm.FromGo(map[string]any{
					"error": fmt.Sprintf("failed to get page content: %v", err),
				})
			}
			return moejs.String(content), nil
		}),
	}
}

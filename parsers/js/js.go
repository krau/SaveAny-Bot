//go:build !no_jsparser

package js

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/dop251/goja"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

type jsParser struct {
	meta  PluginMeta
	vm    *goja.Runtime
	reqCh chan jsParserReq
}

type jsParserReq struct {
	method ParserMethod
	url    string
	respCh chan jsParserResp
}

type jsParserResp struct {
	item *parser.Item
	ok   bool
	err  error
}

func (p *jsParser) CanHandle(url string) bool {
	respCh := make(chan jsParserResp, 1)
	p.reqCh <- jsParserReq{method: ParserMethodCanHandle, url: url, respCh: respCh}
	resp := <-respCh
	return resp.ok && resp.err == nil
}

func (p *jsParser) Parse(ctx context.Context, url string) (*parser.Item, error) {
	respCh := make(chan jsParserResp, 1)
	p.reqCh <- jsParserReq{method: ParserMethodParse, url: url, respCh: respCh}
	select {
	case resp := <-respCh:
		return resp.item, resp.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func newJSParser(vm *goja.Runtime, canHandleFunc, parseFunc goja.Value, metadata PluginMeta) *jsParser {
	p := &jsParser{
		vm:    vm,
		reqCh: make(chan jsParserReq, 10),
		meta:  metadata,
	}

	go func() {
		for req := range p.reqCh {
			switch req.method {
			case ParserMethodCanHandle:
				fn, _ := goja.AssertFunction(canHandleFunc)
				res, err := fn(goja.Undefined(), p.vm.ToValue(req.url))
				if err != nil {
					req.respCh <- jsParserResp{ok: false, err: err}
					continue
				}
				req.respCh <- jsParserResp{ok: res.ToBoolean()}
			case ParserMethodParse:
				fn, _ := goja.AssertFunction(parseFunc)
				result, err := fn(goja.Undefined(), p.vm.ToValue(req.url))
				if err != nil {
					req.respCh <- jsParserResp{err: err}
					continue
				}

				var item parser.Item
				if exported := result.Export(); exported != nil {
					data, err := json.Marshal(exported)
					if err != nil {
						req.respCh <- jsParserResp{err: fmt.Errorf("failed to marshal result to JSON: %w", err)}
						continue
					}

					if err := json.Unmarshal(data, &item); err != nil {
						req.respCh <- jsParserResp{err: fmt.Errorf("failed to unmarshal JSON to Item: %w", err)}
						continue
					}
				} else {
					req.respCh <- jsParserResp{err: fmt.Errorf("JS function returned null or undefined")}
					continue
				}
				req.respCh <- jsParserResp{item: &item}
			}
		}
	}()

	return p
}

// 加载指定文件夹下的所有 JS 解析器插件
func LoadPlugins(ctx context.Context, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".js" {
			continue
		}
		scriptPath := filepath.Join(dir, e.Name())
		code, err := os.ReadFile(scriptPath)
		if err != nil {
			return err
		}

		vm := goja.New()
		vm.Set("registerParser", jsRegisterParser(vm))
		// Inject some utils to vm
		logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("[plugin|parser]/%s", e.Name()))
		vm.Set("console", jsConsole(logger))
		// http fetch funcs
		vm.Set("ghttp", jsGhttp(vm))
		// playwright fetch func
		vm.Set("playwright", jsPlaywright(vm, logger))

		if _, err := vm.RunString(string(code)); err != nil {
			return fmt.Errorf("error loading plugin %s: %w", e.Name(), err)
		}
	}
	return nil
}

var (
	pluginNameMu sync.Map
)

func AddPlugin(ctx context.Context, code string, name string) error {
	value, _ := pluginNameMu.LoadOrStore(name, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	return addPlugin(ctx, code, name)
}

func addPlugin(ctx context.Context, code string, name string) error {
	logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("[plugin|parser]/%s", name))
	vm := goja.New()
	vm.Set("registerParser", jsRegisterParser(vm))
	vm.Set("console", jsConsole(logger))
	vm.Set("ghttp", jsGhttp(vm))
	vm.Set("playwright", jsPlaywright(vm, logger))
	if _, err := vm.RunString(code); err != nil {
		return fmt.Errorf("error loading plugin %s: %w", name, err)
	}
	dir := "plugins"
	configuredDirs := config.C().Parser.PluginDirs
	if len(configuredDirs) > 0 {
		dir = configuredDirs[0]
	}
	if err := os.MkdirAll(dir, 0755); err == nil {
		pluginPath := filepath.Join(dir, name)
		if err := os.WriteFile(pluginPath, []byte(code), 0644); err != nil {
			logger.Warn("Failed to save plugin file: " + err.Error())
		}
	}
	return nil
}

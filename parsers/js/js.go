//go:build !no_jsparser

package js

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MoeForge/moejs"
	"github.com/MoeForge/moejs/engine"
	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

type jsParser struct {
	meta  PluginMeta
	vm    *moejs.Runtime
	reqCh chan jsParserReq
}

type jsParserReq struct {
	ctx    context.Context
	method ParserMethod
	url    string
	respCh chan jsParserResp
}

type jsParserResp struct {
	item *parser.Item
	ok   bool
	err  error
}

const (
	canHandleTimeout = 10 * time.Second
	parseTimeout     = 2 * time.Minute
)

func (p *jsParser) CanHandle(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), canHandleTimeout)
	defer cancel()
	resp, ok := p.request(ctx, ParserMethodCanHandle, url)
	return ok && resp.err == nil && resp.ok
}

func (p *jsParser) Parse(ctx context.Context, url string) (*parser.Item, error) {
	resp, ok := p.request(ctx, ParserMethodParse, url)
	if !ok {
		return nil, ctx.Err()
	}
	return resp.item, resp.err
}

func (p *jsParser) request(ctx context.Context, method ParserMethod, url string) (jsParserResp, bool) {
	respCh := make(chan jsParserResp, 1)
	req := jsParserReq{ctx: ctx, method: method, url: url, respCh: respCh}
	select {
	case p.reqCh <- req:
	case <-ctx.Done():
		return jsParserResp{}, false
	}
	select {
	case resp := <-respCh:
		return resp, true
	case <-ctx.Done():
		return jsParserResp{}, false
	}
}

func newJSParser(vm *moejs.Runtime, canHandleFunc, parseFunc moejs.Value, metadata PluginMeta) *jsParser {
	p := &jsParser{
		vm:    vm,
		reqCh: make(chan jsParserReq, 10),
		meta:  metadata,
	}
	go func() {
		for req := range p.reqCh {
			resp := p.handle(req, canHandleFunc, parseFunc)
			select {
			case req.respCh <- resp:
			case <-req.ctx.Done():
			}
		}
	}()
	return p
}

func (p *jsParser) handle(req jsParserReq, canHandleFunc, parseFunc moejs.Value) (resp jsParserResp) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Errorf("JS parser %q panicked while handling method %d: %v", p.meta.Name, req.method, recovered)
			resp = jsParserResp{err: fmt.Errorf("JS parser %q panicked: %v", p.meta.Name, recovered)}
		}
	}()

	execCtx, cancel := context.WithTimeout(req.ctx, parseTimeout)
	defer cancel()
	done := make(chan struct{})
	defer p.vm.ClearInterrupt()
	defer close(done)
	go func() {
		select {
		case <-execCtx.Done():
			p.vm.Interrupt(execCtx.Err())
		case <-done:
		}
	}()

	arg, err := p.vm.FromGo(req.url)
	if err != nil {
		return jsParserResp{err: fmt.Errorf("failed to convert URL: %w", err)}
	}

	switch req.method {
	case ParserMethodCanHandle:
		result, err := p.vm.CallFunction(canHandleFunc, moejs.Undefined(), arg)
		if err != nil {
			return jsParserResp{ok: false, err: jsCallError(execCtx, p.meta.Name, err)}
		}
		return jsParserResp{ok: engine.ToBoolean(result)}
	case ParserMethodParse:
		result, err := p.vm.CallFunction(parseFunc, moejs.Undefined(), arg)
		if err != nil {
			return jsParserResp{err: jsCallError(execCtx, p.meta.Name, err)}
		}
		if result.IsNullish() {
			return jsParserResp{err: errors.New("JS function returned null or undefined")}
		}
		var item parser.Item
		if err := p.vm.Unmarshal(result, &item); err != nil {
			return jsParserResp{err: jsCallError(execCtx, p.meta.Name, fmt.Errorf("failed to unmarshal JS result to Item: %w", err))}
		}
		return jsParserResp{item: &item}
	default:
		return jsParserResp{err: fmt.Errorf("unknown JS parser method %d", req.method)}
	}
}

func jsCallError(ctx context.Context, name string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("JS parser %q stopped: %w", name, ctxErr)
	}
	return err
}

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
			log.Warnf("Failed to read plugin file %s: %v", e.Name(), err)
			continue
		}

		logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("[plugin|parser]/%s", e.Name()))
		script, err := moejs.CompileScript(e.Name(), string(code))
		if err != nil {
			logger.Warnf("Failed to compile plugin %s: %v", e.Name(), err)
			continue
		}
		vm := moejs.NewRuntime(moejs.Options{})
		if err := installPluginGlobals(vm, logger); err != nil {
			logger.Warnf("Failed to initialize plugin %s: %v", e.Name(), err)
			continue
		}
		if _, err := vm.RunScript(script); err != nil {
			logger.Warnf("Failed to load plugin %s: %v", e.Name(), err)
			continue
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
	script, err := moejs.CompileScript(name, code)
	if err != nil {
		return fmt.Errorf("error compiling plugin %s: %w", name, err)
	}
	vm := moejs.NewRuntime(moejs.Options{})
	if err := installPluginGlobals(vm, logger); err != nil {
		return fmt.Errorf("error initializing plugin %s: %w", name, err)
	}
	if _, err := vm.RunScript(script); err != nil {
		return fmt.Errorf("error loading plugin %s: %w", name, err)
	}
	dir := "plugins"
	configuredDirs := config.C().Parser.PluginDirs
	if len(configuredDirs) > 0 {
		dir = configuredDirs[0]
	}
	fileName := filepath.Base(name)
	if fileName == "" || fileName == "." || fileName == ".." {
		return fmt.Errorf("invalid plugin name %q", name)
	}
	if err := os.MkdirAll(dir, 0755); err == nil {
		pluginPath := filepath.Join(dir, fileName)
		if err := os.WriteFile(pluginPath, []byte(code), 0644); err != nil {
			logger.Warn("Failed to save plugin file: " + err.Error())
		}
	}
	return nil
}

func installPluginGlobals(vm *moejs.Runtime, logger *log.Logger) error {
	if err := vm.SetGlobal("registerParser", jsRegisterParser(vm)); err != nil {
		return err
	}
	if err := vm.SetGlobal("console", jsConsole(logger)); err != nil {
		return err
	}
	if err := vm.SetGlobal("ghttp", jsGhttp(vm)); err != nil {
		return err
	}
	if err := vm.SetGlobal("playwright", jsPlaywright(vm, logger)); err != nil {
		return err
	}
	return nil
}

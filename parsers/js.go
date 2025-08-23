package parsers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/blang/semver"
	"github.com/charmbracelet/log"
	"github.com/dop251/goja"
	"github.com/krau/SaveAny-Bot/pkg/parser"
)

var (
	LatestParserVersion  = semver.MustParse("1.0.0")
	MinimumParserVersion = semver.MustParse("1.0.0")
)

type PluginMeta struct {
	Name        string `json:"name"`
	Version     string `json:"version"` // [TODO] 分版本解析, 但是我们现在只有 v1 所以先不写
	Description string `json:"description"`
	Author      string `json:"author"`
}

type jsParser struct {
	meta  PluginMeta
	vm    *goja.Runtime
	reqCh chan jsParserReq
}

type jsParserReq struct {
	method string
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
	p.reqCh <- jsParserReq{method: "canHandle", url: url, respCh: respCh}
	resp := <-respCh
	return resp.ok && resp.err == nil
}

func (p *jsParser) Parse(url string) (*parser.Item, error) {
	respCh := make(chan jsParserResp, 1)
	p.reqCh <- jsParserReq{method: "parse", url: url, respCh: respCh}
	resp := <-respCh
	return resp.item, resp.err
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
			case "canHandle":
				fn, _ := goja.AssertFunction(canHandleFunc)
				res, err := fn(goja.Undefined(), p.vm.ToValue(req.url))
				if err != nil {
					req.respCh <- jsParserResp{ok: false, err: err}
					continue
				}
				req.respCh <- jsParserResp{ok: res.ToBoolean()}
			case "parse":
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

func registerParser(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
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
		if pluginV.LT(MinimumParserVersion) || pluginV.GT(LatestParserVersion) {
			panic(fmt.Sprintf("parser version %s is not supported, must be between %s and %s", metadata.Version, MinimumParserVersion, LatestParserVersion))
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
		vm.Set("registerParser", registerParser(vm))
		// Inject some utils to vm
		logger := log.FromContext(ctx).WithPrefix(fmt.Sprintf("[plugin|parser]/%s", e.Name()))
		vm.Set("console", map[string]any{
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
		})
		// http fetch funcs
		ghttp := vm.NewObject()
		ghttp.Set("get", func(call goja.FunctionCall) goja.Value {
			url := call.Argument(0).String()
			resp, err := http.Get(url)
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

			resp, err := http.Get(url)
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
		vm.Set("ghttp", ghttp)

		if _, err := vm.RunString(string(code)); err != nil {
			return fmt.Errorf("error loading plugin %s: %w", e.Name(), err)
		}
	}
	return nil
}

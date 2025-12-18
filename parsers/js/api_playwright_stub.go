//go:build no_playwright && !no_jsparser

package js

import (
	"github.com/charmbracelet/log"
	"github.com/dop251/goja"
)

var jsPlaywright = func(vm *goja.Runtime, _ *log.Logger) *goja.Object {
	pwObj := vm.NewObject()
	unsupported := vm.ToValue(map[string]any{
		"error": "playwright is not supported in this build",
	})
	pwObj.Set("get", func(call goja.FunctionCall) goja.Value {
		return unsupported
	})
	return pwObj
}

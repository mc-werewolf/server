package addon

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/dop251/goja"
)

// evalTimeout bounds how long a properties.js snippet is allowed to run.
// This code comes from a third-party GitHub release, so a runaway/malicious
// script must not be able to hang the sync process.
const evalTimeout = 2 * time.Second

// exportStmtRe strips the trailing ES module "export { ... };" statement
// that esbuild emits. goja's RunString evaluates its input as a classic
// script (not an ES module), so a bare "export" is a syntax error.
var exportStmtRe = regexp.MustCompile(`(?s)export\s*\{[^}]*\}\s*;?\s*$`)

// ParsePropertiesJS evaluates the JS source of a BP pack's
// scripts/properties.js (an esbuild-bundled "var properties = {...};
// export { properties };" snippet) and returns the properties object as
// JSON.
func ParsePropertiesJS(src string) (json.RawMessage, error) {
	cleaned := exportStmtRe.ReplaceAllString(src, "")

	vm := goja.New()

	timer := time.AfterFunc(evalTimeout, func() {
		vm.Interrupt("properties.js evaluation timed out")
	})
	defer timer.Stop()

	if _, err := vm.RunString(cleaned); err != nil {
		return nil, fmt.Errorf("evaluate properties.js: %w", err)
	}

	val := vm.Get("properties")
	if val == nil || goja.IsUndefined(val) {
		return nil, fmt.Errorf("properties.js did not define a global \"properties\"")
	}

	data, err := json.Marshal(val.Export())
	if err != nil {
		return nil, fmt.Errorf("marshal properties: %w", err)
	}

	return json.RawMessage(data), nil
}

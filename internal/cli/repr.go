package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Repr formats a JSON value as Python's repr() does, so error messages read as the Python's.
func Repr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case bool:
		if x {
			return "True"
		}
		return "False"
	case float64:
		if x == math.Trunc(x) && math.Abs(x) < 1e16 {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'g', -1, 64)
	case int:
		return strconv.Itoa(x)
	case string:
		if strings.Contains(x, "'") && !strings.Contains(x, `"`) {
			return `"` + x + `"`
		}
		return "'" + strings.ReplaceAll(strings.ReplaceAll(x, `\`, `\\`), "'", `\'`) + "'"
	case []any:
		parts := make([]string, len(x))
		for n, e := range x {
			parts[n] = Repr(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		b, _ := json.Marshal(x)
		return string(b)
	}
	return fmt.Sprint(v)
}

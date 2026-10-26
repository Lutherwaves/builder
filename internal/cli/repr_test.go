package cli

import "testing"

func TestPyrepr(t *testing.T) {
	for v, want := range map[any]string{"a": "'a'", "it's": `"it's"`, nil: "None", true: "True", false: "False", 5.0: "5", 1.5: "1.5"} {
		if got := Repr(v); got != want {
			t.Fatalf("%v: got %s want %s", v, got, want)
		}
	}
}

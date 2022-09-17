package dom

import (
	"github.com/psilva261/sparkle/js"
	"testing"
)

func TestLocation(t *testing.T) {
	vm = js.New()
	htm := `<body></body>`
	_, err := Init(vm, "https://example.com", htm, "window.location.hash = 'test'")
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := vm.RunString(`
location.hash;
	`)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if v := res.Export(); v != "test" {
		t.Fatalf("%v", v)
	}
}

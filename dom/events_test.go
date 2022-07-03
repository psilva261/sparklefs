package dom

import (
	"github.com/psilva261/sparkle/js"
	"testing"
)

func TestNewEvent(t *testing.T) {
	vm := js.New()
	_, err := Init(vm, "<body></body>", "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, _ := vm.RunString("new Event('click')")
	e := res.Export().(*Event)
	if e.Type != "click" || e.Bubbles {
		t.Fatalf("%v", e)
	}
	res, _ = vm.RunString("new Event('click', {bubbles: true})")
	e = res.Export().(*Event)
	if e.Type != "click" || !e.Bubbles {
		t.Fatalf("%v", e)
	}
}

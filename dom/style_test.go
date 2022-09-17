package dom

import (
	"github.com/psilva261/sparkle/js"
	"testing"
)

func TestKebabCamel(t *testing.T) {
	res := kebab("backgroundColor")
	t.Logf("res: %v", res)
	if res != "background-color" {
		t.Fail()
	}
	res = camel(res)
	if res != "backgroundColor" {
		t.Fail()
	}
}

func TestElementStyle(t *testing.T) {

	j := `
document.getElementById("demo").innerHTML =
"The title of this document is: " + document.title;
	`
	vm := js.New()
	d, err := Init(vm, "https://example.com", htm, j)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := vm.RunString(`document.getElementById("demo").innerHTML`)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	p := grep(d.doc, "p")
	t.Logf("p=%v", p)
	el := Element{n: p}
	if s := el.Style(); s.Get("font-weight").String() != "bold" {
		t.Fatalf("%v", s.Get("font-weight").String())
	}
	res, err = vm.RunString(`
		var p = document.getElementById("demo");
		p.style.display = 'none';
		p.style.display;
	`)
	t.Logf("res='%v'", res)
	if err != nil || res.String() != "none" {
		t.Fatalf("%v", err)
	}
}

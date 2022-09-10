package dom

import (
	"github.com/psilva261/sparkle/js"
)

type SVG struct{}

func NewSVG(doc string) *SVG {
	return &SVG{}
}

func (s *SVG) Obj() *js.Object {
	return vm.NewDynamicObject(s)
}

func (s *SVG) Getters() map[string]bool {
	return map[string]bool{
	}
}

func (s *SVG) Props() map[string]bool {
	return map[string]bool{}
}

func (s *SVG) Get(k string) (v js.Value) {
	if res, ok := GetCall(s, k); ok {
		return res
	}
	return vm.ToValue(nil)
}

func (s *SVG) Set(k string, desc js.PropertyDescriptor) bool {
	return true
}

func (s *SVG) Has(k string) bool {
	if yes := HasCall(s, k); yes {
		return true
	}
	return false
}

func (s *SVG) Delete(k string) bool {
	return false
}

func (s *SVG) Keys() []string {
	return []string{""}
}

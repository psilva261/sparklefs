package dom

import (
	"github.com/psilva261/sparkle/js"
)

type DOMParser struct{}

func NewDOMParser() *DOMParser {
	return &DOMParser{}
}

func (dp *DOMParser) Obj() *js.Object {
	return vm.NewDynamicObject(dp)
}

func (dp *DOMParser) Getters() map[string]bool {
	return map[string]bool{
	}
}

func (dp *DOMParser) Props() map[string]bool {
	return map[string]bool{}
}

func (dp *DOMParser) Get(k string) (v js.Value) {
	if res, ok := GetCall(dp, k); ok {
		return res
	}
	return vm.ToValue(nil)
}

func (dp *DOMParser) Set(k string, desc js.PropertyDescriptor) bool {
	return true
}

func (dp *DOMParser) Has(k string) bool {
	if yes := HasCall(dp, k); yes {
		return true
	}
	return false
}

func (dp *DOMParser) Delete(k string) bool {
	return false
}

func (dp *DOMParser) Keys() []string {
	return []string{""}
}

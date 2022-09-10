package dom

import (
	"github.com/psilva261/sparkle/js"
)

type TreeWalker struct{}

func NewTreeWalker() *TreeWalker {
	return &TreeWalker{}
}

func (tw *TreeWalker) Obj() *js.Object {
	return vm.NewDynamicObject(tw)
}

func (tw *TreeWalker) Getters() map[string]bool {
	return map[string]bool{
	}
}

func (tw *TreeWalker) Props() map[string]bool {
	return map[string]bool{}
}

func (tw *TreeWalker) Get(k string) (v js.Value) {
	if res, ok := GetCall(tw, k); ok {
		return res
	}
	return vm.ToValue(nil)
}

func (tw *TreeWalker) Set(k string, desc js.PropertyDescriptor) bool {
	return true
}

func (tw *TreeWalker) Has(k string) bool {
	if yes := HasCall(tw, k); yes {
		return true
	}
	return false
}

func (tw *TreeWalker) Delete(k string) bool {
	return false
}

func (tw *TreeWalker) Keys() []string {
	return []string{""}
}

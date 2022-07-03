package dom

import (
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparklefs/logger"
)

type Animation struct {
}

func (a *Animation) Obj() *js.Object {
	return vm.NewDynamicObject(a)
}

func (a *Animation) Getters() map[string]bool {
	return map[string]bool{}
}

func (a *Animation) Props() map[string]bool {
	return map[string]bool{}
}

func (a *Animation) Get(k string) (v js.Value) {
	if res, ok := GetCall(a, k); ok {
		return res
	}
	return vm.ToValue(nil)
}

func (a *Animation) Set(k string, desc js.PropertyDescriptor) bool {
	log.Printf("animation set %v", k)
	return true
}

func (a *Animation) Has(key string) bool {
	return true
}

func (a *Animation) Delete(key string) bool {
	return false
}

func (a *Animation) Keys() []string {
	return []string{""}
}

func (a *Animation) Cancel() {}

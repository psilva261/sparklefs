package dom

import (
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparklefs/logger"
	"golang.org/x/net/html"
	"strconv"
)

type HTMLCollection struct {
	d *Document
	f func() []*html.Node
}

func (hc *HTMLCollection) Obj() *js.Object {
	obj, ok := hcObjRefs[hc]
	if ok {
		return obj
	}
	obj = vm.NewDynamicObject(hc)
	hcObjRefs[hc] = obj
	return obj
}

func (hc *HTMLCollection) Getters() map[string]bool {
	return map[string]bool{
		"length": true,
	}
}

func (hc *HTMLCollection) Props() map[string]bool {
	return map[string]bool{}
}

func (hc *HTMLCollection) Get(k string) (v js.Value) {
	if res, ok := GetCall(hc, k); ok {
		return res
	}
	switch k {
	default:
		i, err := strconv.Atoi(k)
		if err == nil {
			c := hc.f()
			if i >= len(c) {
				return nil
			}
			return hc.d.getEl(c[i]).Obj()
		}
		c := hc.f()
		for _, n := range c {
			if attr(*n, "id") == k || attr(*n, "name") == k {
				return hc.d.getEl(n).Obj()
			}
		}
		log.Printf("html collection get unknown %v", k)
	}
	return vm.ToValue(nil)
}

func (hc *HTMLCollection) Set(k string, desc js.PropertyDescriptor) bool {
	return false
}

func (hc *HTMLCollection) Has(k string) bool {
	if yes := HasCall(hc, k); yes {
		return true
	}
	if i, err := strconv.Atoi(k); err == nil {
		c := hc.f()
		return 0 <= i && i < len(c)
	}
	return false
}

func (hc *HTMLCollection) Delete(k string) bool {
	return false
}

func (hc *HTMLCollection) Keys() []string {
	ks := Calls(hc)
	c := hc.f()
	for i := range c {
		ks = append(ks, strconv.Itoa(i))
	}
	return ks
}

func (hc *HTMLCollection) ChildNodes() (es []*Element) {
	c := hc.f()
	es = make([]*Element, 0, len(c))
	for _, n := range c {
		es = append(es, hc.d.getEl(n))
	}
	return
}

func (hc *HTMLCollection) Length() int {
	c := hc.f()
	return len(c)
}

func (hc *HTMLCollection) Item(j any) *Element {
	i, ok := j.(int)
	if !ok {
		log.Errorf("html collection item: %T %v", j, j)
		return nil
	}
	c := hc.f()
	if i >= len(c) {
		return nil
	}
	return hc.d.getEl(c[i])
}

func (hc *HTMLCollection) ToString() string {
	return "[object HTMLCollection]"
}

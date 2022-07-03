package dom

import (
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparklefs/logger"
	"golang.org/x/net/html"
)

type NamedNodeMap struct {
	n *html.Node
}

func (nm *NamedNodeMap) Obj() *js.Object {
	return vm.NewDynamicObject(nm)
}

func (nm *NamedNodeMap) Get(k string) (v js.Value) {
	switch k {
	case "item":
		return vm.ToValue(func(x interface{}) (o *js.Object) {
			i, ok := x.(int64)
			if ok {
				if int(i) >= len(nm.n.Attr) {
					return nil
				}
				a := &Attr{&nm.n.Attr[i]}
				return a.Obj()
			}
			return nil
		})
	case "length":
		return vm.ToValue(len(nm.n.Attr))
	}
	if a := attr(*nm.n, k); a != "" {
		return vm.ToValue(a)
	}
	return vm.ToValue(nil)
}

func (nm *NamedNodeMap) Set(k string, desc js.PropertyDescriptor) bool {
	v := desc.Value
	log.Printf("named node map set %v => %v", k, v)
	return true
}

func (nm *NamedNodeMap) Has(k string) bool {
	log.Printf("named node has? %v", k)
	return true
}

func (nm *NamedNodeMap) Delete(k string) bool {
	return false
}

func (nm *NamedNodeMap) Keys() []string {
	return []string{""}
}

type Attr struct {
	a *html.Attribute
}

func (a *Attr) Obj() *js.Object {
	return vm.NewDynamicObject(a)
}

func (a *Attr) Get(k string) (v js.Value) {
	log.Printf("attr get %v", k)
	switch k {
	case "name":
		return vm.ToValue(a.a.Key)
	case "value":
		return vm.ToValue(a.a.Val)
	case "toString":
		return vm.ToValue(func() string {
			return "[object Attr]"
		})
	case "valueOf":
		return vm.ToValue(func() string {
			return a.a.Key + `="` + a.a.Val + `"`
		})
	}
	return vm.ToValue(nil)
}

func (a *Attr) Set(k string, desc js.PropertyDescriptor) bool {
	v := desc.Value
	log.Printf("attr set %v => %v", k, v)
	return true
}

func (a *Attr) Has(k string) bool {
	log.Printf("attr has? %v", k)
	return true
}

func (a *Attr) Delete(k string) bool {
	return false
}

func (a *Attr) Keys() []string {
	return []string{""}
}

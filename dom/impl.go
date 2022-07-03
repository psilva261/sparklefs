package dom

import (
	"fmt"
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparklefs/logger"
	"golang.org/x/net/html"
	"strings"
)

type Implementation struct{}

func (impl *Implementation) Obj() *js.Object {
	return vm.NewDynamicObject(impl)
}

func (impl *Implementation) Getters() map[string]bool {
	return map[string]bool{}
}

func (impl *Implementation) Props() map[string]bool {
	return map[string]bool{}
}

func (impl *Implementation) Get(k string) (v js.Value) {
	if res, ok := GetCall(impl, k); ok {
		return res
	}
	return vm.ToValue(nil)
}

func (impl *Implementation) Set(k string, desc js.PropertyDescriptor) bool {
	log.Printf("impl.Set(%v, %v)", k, desc.Value)
	return true
}

func (impl *Implementation) Has(key string) bool {
	return true
}

func (impl *Implementation) Delete(key string) bool {
	return false
}

func (impl *Implementation) Keys() []string {
	return []string{""}
}

func (impl *Implementation) CreateHTMLDocument(title ...string) (d *Document) {
	h := fmt.Sprintf("<html><head><title></title><head><body></body></html>")
	doc, err := html.Parse(strings.NewReader(h))
	if err != nil {
		log.Printf("parse error")
	}
	d = NewDocument(doc)
	return
}

func (impl *Implementation) HasFeature(args ...any) bool {
	var feature string
	if len(args) > 0 {
		feature, _ = args[0].(string)
	}
	return feature != "XML"
}

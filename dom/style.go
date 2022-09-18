package dom

import (
	"embed"
	"encoding/csv"
	"fmt"
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparklefs/logger"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
	"golang.org/x/net/html"
	"strings"
)

// CSS properties from https://www.w3.org/Style/CSS/all-properties.en.tab

//go:embed all-properties.en.tab
var allPropertiesTab embed.FS

var allProperties = make(map[string]bool)

func init() {
	f, err := allPropertiesTab.Open("all-properties.en.tab")
	if err != nil {
		panic(err.Error())
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = '\t'
	rcs, err := r.ReadAll()
	if err != nil {
		panic(err.Error())
	}
	for _, rc := range rcs {
		allProperties[rc[0]] = true
	}
}

// Style represents a CSSStyleDeclaration object
type Style struct {
	n *html.Node
}

func (s *Style) Obj() *js.Object {
	return vm.NewDynamicObject(s)
}

func (s *Style) Getters() map[string]bool {
	return map[string]bool{
		"length": true,
	}
}

func (s *Style) Props() map[string]bool {
	return map[string]bool{}
}

func (s *Style) Get(k string) (v js.Value) {
	if res, ok := GetCall(s, k); ok {
		return res
	}
	k = kebab(k)
	st := attr(*s.n, "style")
	m := parseStyle(st)
	if s, ok := m[k]; ok && s != "" {
		return vm.ToValue(s)
	}
	if _, ok := allProperties[k]; ok {
		return vm.ToValue("")
	}
	return vm.ToValue(nil)
}

func (s *Style) Length() int {
	st := attr(*s.n, "style")
	m := parseStyle(st)
	return len(m)
}

func kebab(k string) (res string) {
	if strings.Contains(k, "-") {
		return k
	}
	for i := len(k) - 1; i >= 0; i-- {
		s := k[i : i+1]
		if s == strings.ToUpper(s) {
			k = k[:i] + "-" + strings.ToLower(s) + k[i+1:]
		}
	}
	return k
}

func camel(k string) (res string) {
	if !strings.Contains(k, "-") {
		return k
	}
	tmp := strings.Split(k, "-")
	for i, s := range tmp {
		if i > 0 {
			s = strings.Title(s)
		}
		res += s
	}
	return
}

func (s *Style) Set(k string, desc js.PropertyDescriptor) bool {
	v := desc.Value
	if k == "cssText" {
		setAttr(s.n, "style", v.String())
		return true
	}
	st := attr(*s.n, "style")
	m := parseStyle(st)
	k = kebab(k)
	m[k] = v.String()
	st = ""
	for k, v := range m {
		st += fmt.Sprintf("%v: %v; ", k, v)
	}
	setAttr(s.n, "style", st)
	return true
}

func (s *Style) Has(k string) (yes bool) {
	log.Printf("style has? %v", k)
	k = kebab(k)
	st := attr(*s.n, "style")
	m := parseStyle(st)
	if s, ok := m[k]; ok && s != "" {
		return true
	}
	return HasCall(s, k)
}

func (s *Style) Delete(key string) bool {
	log.Printf("style delete %v", key)
	return false
}

func (s *Style) Keys() []string {
	log.Printf("style get keys")
	return []string{""}
}

func (s *Style) GetPropertyValue(p string) string {
	st := attr(*s.n, "style")
	m := parseStyle(st)
	v, _ := m[p]
	return v
}

func parseStyle(st string) (m map[string]string) {
	m = make(map[string]string)
	p := css.NewParser(parse.NewInputString(st), true)
	for {
		gt, _, data := p.Next()
		if gt == css.ErrorGrammar {
			break
		} else if gt == css.AtRuleGrammar || gt == css.BeginAtRuleGrammar || gt == css.BeginRulesetGrammar || gt == css.DeclarationGrammar {
			k := string(data)
			v := ""
			for _, val := range p.Values() {
				v += string(val.Data)
			}
			m[k] = v
		}
	}
	return
}

type ComputedStyle struct {
	el *Element
}

func (cs *ComputedStyle) Obj() *js.Object {
	return vm.NewDynamicObject(cs)
}

func (cs *ComputedStyle) Getters() map[string]bool {
	return map[string]bool{}
}

func (cs *ComputedStyle) Props() map[string]bool {
	return map[string]bool{}
}

func (cs *ComputedStyle) Get(k string) (v js.Value) {
	if res, ok := GetCall(cs, k); ok {
		return res
	}
	res := cs.GetPropertyValue(k)
	if res != "" {
		return vm.ToValue(v)
	}
	return vm.ToValue(nil)
}

func (cs *ComputedStyle) Set(k string, desc js.PropertyDescriptor) bool {
	log.Errorf("computed style is read-only")
	return false
}

func (cs *ComputedStyle) Has(k string) bool {
	if yes := HasCall(cs, k); yes {
		return true
	}
	return false
}

func (cs *ComputedStyle) Delete(k string) bool {
	return false
}

func (cs *ComputedStyle) Keys() []string {
	return []string{""}
}

func (cs *ComputedStyle) GetPropertyValue(k string) string {
	if Query == nil {
		log.Errorf("nil Query func")
		return ""
	}
	p, ok := path(cs.el)
	if !ok {
		log.Errorf("path lookup failed")
		return ""
	}
	res, err := Query(p, k)
	if err != nil {
		log.Errorf("query: %v", err)
		return ""
	}
	return res
}

type DOMRect struct{}

func (dr *DOMRect) Obj() *js.Object {
	return vm.NewDynamicObject(dr)
}

func (dr *DOMRect) Getters() map[string]bool {
	return map[string]bool{
		"length": true,
	}
}

func (dr *DOMRect) Props() map[string]bool {
	return map[string]bool{}
}

func (dr *DOMRect) Get(k string) (v js.Value) {
	if res, ok := GetCall(dr, k); ok {
		return res
	}
	return vm.ToValue(nil)
}

func (dr *DOMRect) Set(k string, desc js.PropertyDescriptor) bool {
	return true
}

func (dr *DOMRect) Has(k string) bool {
	if yes := HasCall(dr, k); yes {
		return true
	}
	return false
}

func (dr *DOMRect) Delete(k string) bool {
	return false
}

func (dr *DOMRect) Keys() []string {
	return []string{""}
}

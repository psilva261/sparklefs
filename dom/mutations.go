package dom

import (
	"github.com/psilva261/sparkle/js"
	"golang.org/x/net/html"
	"time"
)

type MutationType int

const (
	Value  = 1
	ChAttr = 2
	RmAttr = 3
	Rm     = 4
	Mv     = 5
	Insert = 6
)

func (t MutationType) String() string {
	switch t {
	case Value:
		return "Value"
	case ChAttr:
		return "Attr"
	case RmAttr:
		return "RmAttr"
	case Rm:
		return "Rm"
	case Mv:
		return "Mv"
	case Insert:
		return "Insert"
	}
	return ""
}

type Mutation struct {
	Time time.Time
	Type MutationType
	Path string
	Tag  string
	Node map[string]string
}

// addMutation can be called after changing the node tree
func addMutation(d *Document, t MutationType, n *html.Node) {
	m := Mutation{
		Time: time.Now(),
		Type: t,
		Path: "",
		Node: map[string]string{},
	}
	if n != nil {
		if n.Type == html.ElementNode {
			m.Tag = n.Data
		}
		for _, a := range n.Attr {
			m.Node[a.Key] = a.Val
		}
		if d != nil {
			m.Node["innerHTML"] = d.getEl(n).InnerHTML()
		}
	}
	select {
	case mutations <- m:
	default:
	}
}

type MutObserver struct{}

func NewMutObserver() *js.Object {
	m := &MutObserver{}
	return m.Obj()
}

func (m *MutObserver) Obj() *js.Object {
	return vm.NewDynamicObject(m)
}

func (m *MutObserver) Getters() map[string]bool {
	return map[string]bool{
	}
}

func (m *MutObserver) Props() map[string]bool {
	return map[string]bool{}
}

func (m *MutObserver) Get(k string) (v js.Value) {
	if res, ok := GetCall(m, k); ok {
		return res
	}
	return vm.ToValue(nil)
}

func (m *MutObserver) Set(k string, desc js.PropertyDescriptor) bool {
	return true
}

func (m *MutObserver) Has(k string) bool {
	if yes := HasCall(m, k); yes {
		return true
	}
	return false
}

func (m *MutObserver) Delete(k string) bool {
	return false
}

func (m *MutObserver) Keys() []string {
	return []string{""}
}

func (m *MutObserver) Observe(target *Element, opts map[string]any) {
}

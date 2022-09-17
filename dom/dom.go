package dom

import (
	"bytes"
	"fmt"
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparklefs/dom/sel"
	"github.com/psilva261/sparklefs/logger"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	vm        *js.Runtime
	mutations = make(chan Mutation, 10000)
)

var (
	elObjRefs = make(map[*Element]*js.Object)
	evObjRefs = make(map[*Event]*js.Object)
	dfObjRefs = make(map[*DocumentFragment]*js.Object)
	hcObjRefs = make(map[*HTMLCollection]*js.Object)
)

var (
	elVars = make(map[*Element]map[string]js.Value)
	evVars = make(map[*Event]map[string]js.Value)
)

var (
	Geom  func(sel string) (string, error)
	Query func(sel, prop string) (val string, err error)
)

func Mutations() <-chan Mutation {
	return mutations
}

type Console struct {
	Log func(xs ...interface{}) `json:"log"`
}

func NewConsole() (c *Console) {
	c = &Console{}
	c.Log = c.log
	return
}

func (c *Console) log(xs ...interface{}) {
	s := ""
	for _, x := range xs {
		switch v := x.(type) {
		case string:
			s += v
		case map[string]interface{}:
			s += "{ "
			for k, _ := range v {
				s += k + ": ..., "
			}
			s += " }"
		default:
			s += fmt.Sprintf("%v", x)
		}
	}
	log.Infof("[console] %v", s)
}

type Navigator struct {
	UserAgent string `json:"userAgent"`
}

type History struct{}

var NodePrototype js.Value
var TextPrototype js.Value
var HTMLElementPrototype js.Value
var HTMLInputElementPrototype js.Value

type Window struct {
	*Document
	*Location
	Navigator
	History

	obj  *js.Object
	vars map[string]js.Value

	animationFrame func(js.FunctionCall) js.Value

	builtinThis    *js.Object
	eventListeners map[string][]func(js.FunctionCall) js.Value
}

func NewWindow(url string, builtinThis *js.Object, d *Document) *Window {
	w := &Window{
		Document: d,
		Location: NewLocation(url),
		Navigator: Navigator{
			UserAgent: "udom",
		},
	}
	w.builtinThis = builtinThis
	w.vars = make(map[string]js.Value)
	w.eventListeners = make(map[string][]func(js.FunctionCall) js.Value)
	return w
}

func (w *Window) Obj() *js.Object {
	if w.obj == nil {
		w.obj = vm.NewDynamicObject(w)
	}
	return w.obj
}

func (w *Window) Get(k string) js.Value {
	log.Printf("window get k=%v", k)
	if w.Document == nil {
		log.Errorf("nil document")
		return js.Undefined()
	}
	if v, ok := w.vars[k]; ok {
		log.Printf("window get as var")
		return v
	}
	switch k {
	case "console":
		return vm.ToValue(NewConsole())
	case "window", "self", "parent", "top", "frames":
		return w.Obj()
	case "frameElement":
		return js.Null()
	case "document":
		return w.Document.Obj()
	case "location":
		return vm.NewDynamicObject(w.Location)
	case "navigator":
		return vm.ToValue(w.Navigator)
	case "addEventListener":
		return vm.ToValue(w.addEventListener)
	case "removeEventListener":
		return vm.ToValue(w.removeEventListener)
	case "dispatchEvent":
		return vm.ToValue(w.dispatchEvent)
	case "Node":
		return NodePrototype
	case "Text":
		return TextPrototype
	case "HTMLElementPrototype":
		return HTMLElementPrototype
	case "HTMLInputElement":
		return HTMLInputElementPrototype
	case "getComputedStyle":
		return vm.ToValue(func(args ...any) js.Value {
			el := args[0].(*Element)
			log.Printf("getComputedStyle(%v)", el)
			s := &ComputedStyle{el: el}
			return s.Obj()
		})
	case "requestAnimationFrame":
		return vm.ToValue(func(f func(js.FunctionCall) js.Value) {
			w.animationFrame = f
		})
	case "SVGElement":
		return vm.ToValue(func(call js.ConstructorCall) *js.Object {
			doc := call.Argument(0).String()
			s := NewSVG(doc)
			sv := vm.ToValue(s).(*js.Object)
			sv.SetPrototype(call.This.Prototype())
			return sv
		})
	case "DOMParser":
		return vm.ToValue(func(call js.ConstructorCall) *js.Object {
			dp := NewDOMParser()
			dpv := vm.ToValue(dp).(*js.Object)
			dpv.SetPrototype(call.This.Prototype())
			return dpv
		})
	case "MutationObserver":
		return vm.ToValue(func(call js.ConstructorCall) *js.Object {
			m := NewMutObserver()
			mv := vm.ToValue(m).(*js.Object)
			mv.SetPrototype(call.This.Prototype())
			return mv
		})
	case "Event":
		return vm.ToValue(func(call js.ConstructorCall) *js.Object {
			var opts map[string]interface{}
			typ := call.Argument(0).String()
			if len(call.Arguments) >= 2 {
				opts = call.Argument(1).Export().(map[string]interface{})
			}
			e := NewEvent(typ, opts)
			ev := vm.ToValue(e).(*js.Object)
			ev.SetPrototype(call.This.Prototype())
			return ev
		})
	case "MouseEvent":
		return vm.ToValue(func(call js.ConstructorCall) *js.Object {
			var opts map[string]interface{}
			if len(call.Arguments) >= 2 {
				opts = call.Argument(1).Export().(map[string]interface{})
			}
			e := NewMouseEvent(call.Argument(0).String(), opts)
			ev := vm.ToValue(e).(*js.Object)
			ev.SetPrototype(call.This.Prototype())
			return ev
		})
	case "CustomEvent":
		return vm.ToValue(func(call js.ConstructorCall) *js.Object {
			e := NewEvent(call.Argument(0).String(), map[string]any{})
			ev := vm.ToValue(e).(*js.Object)
			ev.SetPrototype(call.This.Prototype())
			return ev
		})
	case "Comment":
		return vm.ToValue(func(call js.ConstructorCall) *js.Object {
			var data string
			if len(call.Arguments) >= 1 {
				data = call.Argument(0).String()
			}
			el := w.Document.CreateComment(data)
			return el.Obj()
		})
	case "Image":
		return vm.ToValue(func(call js.ConstructorCall) *js.Object {
			el := w.Document.CreateElement("img")
			if len(call.Arguments) >= 1 {
				w := call.Argument(0).String()
				setAttr(el.n, "width", w)
			}
			if len(call.Arguments) >= 1 {
				h := call.Argument(1).String()
				setAttr(el.n, "height", h)
			}
			return el.Obj()
		})
	case "Document":
		return vm.ToValue(func(call js.ConstructorCall) *js.Object {
			impl := &Implementation{}
			return impl.CreateHTMLDocument("").Obj()
		})
	case "DocumentFragment":
		return vm.ToValue(func(call js.ConstructorCall) *js.Object {
			return w.Document.CreateDocumentFragment().Obj()
		})
	default:
		res := w.builtinThis.Get(k)
		if res == nil {
			return js.Undefined()
		}
		return res
	}
	return js.Undefined()
}

func (w *Window) Set(key string, desc js.PropertyDescriptor) bool {
	switch key {
	case "Promise":
		// noop
	default:
		w.vars[key] = desc.Value
	}
	return true
}

func (w *Window) Has(key string) bool {
	if _, ok := w.vars[key]; ok {
		return true
	}
	return true
}

func (w *Window) Delete(key string) bool {
	if _, ok := w.vars[key]; ok {
		delete(w.vars, key)
		return true
	}
	return false
}

func (w *Window) Keys() []string {
	return []string{""}
}

func (w *Window) RenderAnimationFrame() (ok bool) {
	if w.animationFrame == nil {
		return false
	}
	fn, ok := js.AssertFunction(vm.ToValue(w.animationFrame))
	if !ok {
		log.Errorf("request animation frame assert function: %v", ok)
		return
	}
	t := time.Now().UnixMilli()
	_, err := fn(nil, vm.ToValue(float64(t)))
	if err != nil {
		log.Infof("run anim cb: %v", err)
	}
	return true
}

func (w *Window) addEventListener(e string, f func(js.FunctionCall) js.Value) {
	c := &Call{
		recv:  "Window",
		k:     "addEventListener",
		found: true,
	}
	calls = append(calls, c)
	w.eventListeners[e] = append(w.eventListeners[e], f)
}

func (w *Window) removeEventListener(e string, f func(js.FunctionCall) js.Value) {
	c := &Call{
		recv:  "Window",
		k:     "removeEventListener",
		found: true,
	}
	calls = append(calls, c)
	for i, ff := range w.eventListeners[e] {
		if fmt.Sprintf("%p", f) == fmt.Sprintf("%p", ff) {
			w.eventListeners[e] = append(w.eventListeners[e][:i], w.eventListeners[e][i+1:]...)
			break
		}
	}
}

func (w *Window) dispatchEvent(e *Event) {
	c := &Call{
		recv:  "Window",
		k:     "dispatchEvent",
		found: true,
	}
	calls = append(calls, c)
	for _, f := range w.eventListeners[e.Type] {
		fn, ok := js.AssertFunction(vm.ToValue(f))
		if !ok {
			log.Errorf("win assert function: %v", ok)
			e.Consumed = true
			continue
		}
		that, err := vm.RunString("this")
		if err != nil {
			log.Fatalf("oh no: %v", err)
			continue
		}
		_, err = fn(that, e.Obj())
		if err != nil {
			log.Errorf("win event handler fn: %v", err)
			e.Consumed = true
			continue
		}
	}
}

type Document struct {
	Window *Window
	doc    *html.Node
	obj    *js.Object
	vars   map[string]js.Value
	elRefs map[*html.Node]*Element

	eventListeners map[string][]func(js.FunctionCall) js.Value
}

func NewDocument(doc *html.Node) (d *Document) {
	d = &Document{
		doc: doc,
	}
	d.vars = make(map[string]js.Value)
	d.elRefs = make(map[*html.Node]*Element)
	d.eventListeners = make(map[string][]func(js.FunctionCall) js.Value)
	return
}

func (d *Document) Element() *Element {
	return d.getEl(d.doc)
}

func (d *Document) Doc() *html.Node {
	return d.doc
}

func (d *Document) Obj() (o *js.Object) {
	if d.obj == nil {
		d.obj = vm.NewDynamicObject(d)
	}
	return d.obj
}

func (d *Document) Getters() map[string]bool {
	return map[string]bool{
		"domain":          true,
		"location":        true,
		"referrer":        true,
		"cookie":          true,
		"implementation":  true,
		"defaultView":     true,
		"documentElement": true,
		"all":             true,
		"body":            true,
		"head":            true,
		"title":           true,
		"scripts":         true,
		"styleSheets":     true,
		"activeElement":   true,
		"nodeType":        true,
		"parentNode":      true,
		"childNodes":      true,
		"children":        true,
	}
}

func (d *Document) Props() map[string]bool {
	return map[string]bool{
		"window": true,
	}
}

func (d *Document) Get(key string) js.Value {
	if d == nil {
		log.Errorf("nil document (unreachable)")
		return js.Undefined()
	}
	if v, ok := d.vars[key]; ok {
		log.Printf("document found var %v", key)
		return v
	}
	if res, ok := GetCall(d, key); ok {
		return res
	}
	if key == "nodeName" {
		// TODO: weird error preventing to factor nodeName into a function
		return vm.ToValue("#document")
	}
	return js.Undefined()
}

func (d *Document) Set(key string, desc js.PropertyDescriptor) bool {
	switch key {
	case "nodeValue":
		// no effect
	default:
		d.vars[key] = desc.Value
	}
	log.Printf("document set %v => %v", key, desc.Value)
	return true
}

func (d *Document) Has(key string) bool {
	if _, ok := d.vars[key]; ok {
		return true
	}
	if key == "addEventListener" || key == "attachEvent" {
		return true
	}
	return HasCall(d, key)
}

func (d *Document) Delete(key string) bool {
	if _, ok := d.vars[key]; ok {
		delete(d.vars, key)
		return true
	}
	return false
}

func (d *Document) Keys() []string {
	return []string{""}
}

func (d *Document) Domain() string {
	return d.Window.Location.Hostname
}

func (d *Document) Location() *js.Object {
	return d.Window.Location.Obj()
}

func (d *Document) Referrer() string {
	return "https://example.com"
}

func (d *Document) Cookie() string {
	return ""
}

func (d *Document) Implementation() js.Value {
	return vm.NewDynamicObject(&Implementation{})
}

func (d *Document) DefaultView() *Window {
	return d.Window
}

func (d *Document) DocumentElement() *js.Object {
	return d.getEl(d.doc).Obj()
}

func (d *Document) All() *js.Object {
	hc := &HTMLCollection{
		d: d,
		f: func() []*html.Node {
			return grepAll(d.doc, "*", false)
		},
	}
	return hc.Obj()
}

func (d *Document) Body() *js.Object {
	return d.getEl(grep(d.doc, "body")).Obj()
}

func (d *Document) Head() *js.Object {
	return d.getEl(grep(d.doc, "head")).Obj()
}

func (d *Document) Title() js.Value {
	return vm.ToValue(grep(d.doc, "title").FirstChild.Data)
}

// Scripts just returns an empty list
func (d *Document) Scripts() *js.Object {
	return d.GetElementsByTagName("script").Obj()
}

// StyleSheets just returns an empty list
func (d *Document) StyleSheets() js.Value {
	return vm.ToValue([]any{})
}

func (d *Document) ActiveElement() *js.Object {
	return d.Body()
}

func (d *Document) CreateDocumentFragment(opts ...any) *DocumentFragment {
	log.Printf("CreateDocumentFragment: opts=%+v", opts)
	return NewDocumentFragment(d)
}

func (d *Document) CreateElement(tag string) *Element {
	return CreateElement(d, tag)
}

func (d *Document) CreateElementNS(uri, qn string, opts ...any) *Element {
	return CreateElementNS(d, uri, qn)
}

func (d *Document) CreateEvent(t string) *Event {
	if t != "Event" {
		log.Errorf("unsupported event type %v", t)
	}
	return &Event{}
}

func (d *Document) CreateTextNode(args ...string) *Element {
	var data string
	if len(args) > 0 {
		data = args[0]
	}
	n := &html.Node{}
	n.Data = data
	n.Type = html.TextNode
	return d.getEl(n)
}

func (d *Document) CreateComment(data string) *Element {
	n := &html.Node{}
	n.Data = data
	n.Type = html.CommentNode
	return d.getEl(n)
}

func (d *Document) CreateProcessingInstruction(target, data string) *Element {
	n := &html.Node{}
	n.Data = target
	n.Type = html.ElementNode
	n.Attr = []html.Attribute{
		html.Attribute{
			Key: data,
		},
	}
	return d.getEl(n)
}

func (d *Document) CreateTreeWalker(opts ...any) *TreeWalker {
	return NewTreeWalker()
}

func (d *Document) CloneNode(deep ...bool) *Element {
	e := d.getEl(d.doc).CloneNode()
	if e == nil {
		return nil
	} else {
		return e
	}
}

func (d *Document) NodeType() int {
	return 9
}

func (d *Document) ToString() (s string) {
	return "[object HTMLDocument]"
}

func (d *Document) ParentNode() js.Value {
	return vm.ToValue(nil)
}

func (d *Document) ChildNodes() *js.Object {
	return d.Children()
}

func (d *Document) Children() *js.Object {
	return d.getEl(d.doc).Children()
}

func (d *Document) GetElementById(id string) *Element {
	return d.getEl(d.doc).getElementById(id)
}

func (d *Document) GetElementsByClassName(cl ...string) *HTMLCollection {
	return d.getEl(d.doc).GetElementsByClassName(strings.Join(cl, " "))
}

func (d *Document) GetElementsByName(nm string) (els []*Element) {
	return d.getEl(d.doc).getElementsByName(nm)
}

func (d *Document) GetElementsByTagName(tag string) *HTMLCollection {
	return &HTMLCollection{
		d: d,
		f: func() []*html.Node {
			return grepAll(d.doc, tag, true)
		},
	}
}

func (d *Document) QuerySelector(s string) *Element {
	return d.getEl(d.doc).QuerySelector(s)
}

func (d *Document) QuerySelectorAll(s string) []*Element {
	return d.getEl(d.doc).QuerySelectorAll(s)
}

func (d *Document) Write(s string) {
	// TODO: check if closed
	body := grep(d.doc, "body")
	f, err := html.ParseFragment(strings.NewReader(s), body)
	if err != nil {
		log.Errorf("write: %v", err)
		return
	}
	for _, c := range f {
		log.Printf("write: append: type=%v %v", c.Type, c.Data)
		body.AppendChild(c)
		addMutation(d, Insert, c)
	}
}

func (d *Document) AttachEvent(e string, f func(js.FunctionCall) js.Value, opts ...any) {
	d.AddEventListener(e, f, opts...)
}

func (d *Document) AddEventListener(e string, fn any, opts ...any) {
	f, ok := fn.(func(js.FunctionCall) js.Value)
	if !ok {
		log.Errorf("document add event listener unexpected %T %+v", fn, fn)
	}
	log.Printf("Document AddEventListener(%v, %v)", e, fn)
	d.eventListeners[e] = append(d.eventListeners[e], f)
}

func (d *Document) RemoveEventListener(e string, f func(js.FunctionCall) js.Value, opts ...any) {
	for i, ff := range d.eventListeners[e] {
		if fmt.Sprintf("%p", f) == fmt.Sprintf("%p", ff) {
			if len(d.eventListeners[e]) > i+1 {
				d.eventListeners[e] = append(d.eventListeners[e][:i], d.eventListeners[e][i+1:]...)
			} else {
				d.eventListeners[e] = d.eventListeners[e][:i]
			}
			break
		}
	}
}

func (d *Document) DispatchEvent(e *Event) {
	for _, f := range d.eventListeners[e.Type] {
		fn, ok := js.AssertFunction(vm.ToValue(f))
		if !ok {
			log.Errorf("doc assert function: %v", ok)
			e.Consumed = true
			continue
		}
		/*that, err := vm.RunString("this")
		if err != nil {
			log.Fatalf("oh no: %v", err)
			continue
		}*/
		_, err := fn(d.Obj(), e.Obj())
		if err != nil {
			log.Errorf("doc event handler fn: %v", err)
			e.Consumed = true
			continue
		}
	}
	if e.Bubbles {
		d.Window.dispatchEvent(e)
	}
}

func (d *Document) Close() (err error) {
	d.vars["readyState"] = vm.ToValue("interactive")
	d.DispatchEvent(&Event{Type: "readystatechange"})
	d.DispatchEvent(&Event{Type: "DOMContentLoaded"})
	d.vars["readyState"] = vm.ToValue("complete")
	d.DispatchEvent(&Event{Type: "readystatechange"})
	d.Window.DispatchEvent(&Event{Type: "load"})
	return
}

type DocumentFragment struct {
	children []*html.Node
	d        *Document
	vars     map[string]js.Value
}

func NewDocumentFragment(d *Document) (df *DocumentFragment) {
	df = &DocumentFragment{}
	df.vars = make(map[string]js.Value)
	df.d = d
	return
}

func (df *DocumentFragment) Obj() *js.Object {
	obj, ok := dfObjRefs[df]
	if ok {
		return obj
	}
	obj = vm.NewDynamicObject(df)
	dfObjRefs[df] = obj
	return obj
}

func (df *DocumentFragment) CreateElement(tag string) *Element {
	return CreateElement(df.d, tag)
}

func (df *DocumentFragment) GetElementById(id string) *Element {
	for _, c := range df.children {
		if el := df.d.getEl(c).getElementById(id); el != nil {
			return el
		}
	}
	return nil
}

func (df *DocumentFragment) GetElementsByTagName(tag string) *HTMLCollection {
	hc := &HTMLCollection{
		d: df.d,
		f: func() (res []*html.Node) {
			for _, c := range df.children {
				res = append(res, grepAll(c, tag, true)...)
			}
			return
		},
	}
	return hc
}

func (df *DocumentFragment) QuerySelectorAll(s string) (els []*Element) {
	for _, c := range df.children {
		els = append(els, df.d.getEl(c).QuerySelectorAll(s)...)
	}
	return
}

func (df *DocumentFragment) AppendChild(o interface{}) *Element {
	el := o.(*Element)
	el.df = df
	df.children = append(df.children, el.n)
	return df.d.getEl(el.n)
}

func (df *DocumentFragment) InsertBefore(nu, ol any) *Element {
	nue := nu.(*Element)
	ole, ok := ol.(*Element)
	if ok {
		for i, c := range df.children {
			if c == ole.n {
				var rest []*html.Node
				if i+1 < len(df.children) {
					rest = df.children[i+1:]
				}
				df.children = append(df.children[:i], nue.n)
				df.children = append(df.children, rest...)
				return nue
			}
		}
	}
	return df.AppendChild(nu)
}

func (df *DocumentFragment) CloneNode(deep ...bool) *DocumentFragment {
	cl := &DocumentFragment{
		d:        df.d,
		children: make([]*html.Node, 0, len(df.children)),
	}
	for _, c := range df.children {
		cel := df.d.getEl(c).CloneNode()
		cl.children = append(cl.children, cel.n)
	}
	return cl
}

func (df *DocumentFragment) ChildNodes() js.Value {
	return df.Children()
}

var dfChildren = make(map[*DocumentFragment]*HTMLCollection)

func (df *DocumentFragment) Children() js.Value {
	hc, ok := dfChildren[df]
	if !ok {
		hc = &HTMLCollection{
			d: df.d,
			f: func() []*html.Node {
				return df.children
			},
		}
		dfChildren[df] = hc
	}
	return hc.Obj()
}

func (df *DocumentFragment) RemoveChild(c interface{}) *Element {
	if c == nil {
		return nil
	}
	ce := c.(*Element)
	for i, cc := range df.children {
		if ce.n == cc {
			df.children = append(df.children[:i], df.children[:i+1]...)
			return df.d.getEl(ce.n)
		}
	}
	return nil
}

func (df *DocumentFragment) FirstChild() *js.Object {
	if len(df.children) == 0 {
		return nil
	}
	o := df.d.getEl(df.children[0]).Obj()
	return o
}

func (df *DocumentFragment) LastChild() *js.Object {
	if len(df.children) == 0 {
		return nil
	}
	o := df.d.getEl(df.children[len(df.children)-1]).Obj()
	return o
}

func (df *DocumentFragment) Getters() map[string]bool {
	return map[string]bool{
		"ownerDocument": true,
		"nodeName":      true,
		"nodeType":      true,
		"nodeValue":     true,
		"childNodes":    true,
		"children":      true,
		"firstChild":    true,
		"lastChild":     true,
	}
}

func (df *DocumentFragment) Props() map[string]bool {
	return map[string]bool{}
}

func (df *DocumentFragment) Get(key string) js.Value {
	if res, ok := GetCall(df, key); ok {
		return res
	}
	if v, ok := df.vars[key]; ok {
		return v
	}
	return vm.ToValue(nil)
}

func (df *DocumentFragment) Set(key string, desc js.PropertyDescriptor) bool {
	val := desc.Value
	switch key {
	case "textContent":
		df.children = []*html.Node{}
		if s := val.String(); s != "" {
			tn := &html.Node{
				Type: html.TextNode,
				Data: val.String(),
			}
			df.children = append(df.children, tn)
		}
		addMutation(df.d, Value, nil) // TODO
	default:
		df.vars[key] = val
	}
	return true
}

func (df *DocumentFragment) Has(key string) bool {
	if _, ok := df.vars[key]; ok {
		return true
	}
	return HasCall(df, key)
}

func (df *DocumentFragment) Delete(key string) bool {
	if _, ok := df.vars[key]; ok {
		delete(df.vars, key)
		return true
	}
	return false
}

func (df *DocumentFragment) Keys() []string {
	ks := Calls(df)
	for k := range df.vars {
		ks = append(ks, k)
	}
	return ks
}

func (df *DocumentFragment) OwnerDocument() js.Value {
	return df.d.Obj()
}

func (df *DocumentFragment) NodeName() string {
	return "#document-fragment"
}

func (df *DocumentFragment) NodeType() int {
	return 11
}

func (df *DocumentFragment) NodeValue() any {
	return nil
}

type Element struct {
	d  *Document
	df *DocumentFragment
	n  *html.Node
}

func (d *Document) getEl(n *html.Node) (el *Element) {
	if n == nil {
		return nil
	}
	el, ok := d.elRefs[n]
	if ok {
		return
	}
	el = &Element{n: n}
	if el.d == nil {
		el.d = d
	}
	d.elRefs[n] = el
	return
}

func CreateElement(d *Document, tagName string) *Element {
	n := &html.Node{}
	n.Data = tagName
	n.Type = html.ElementNode
	n.DataAtom = atom.Lookup([]byte(tagName))
	el := d.getEl(n)
	el.d = d
	return el
}

func CreateElementNS(d *Document, uri, qn string) *Element {
	n := &html.Node{}
	n.Data = qn
	n.Type = html.ElementNode
	n.DataAtom = atom.Lookup([]byte(qn))
	n.Namespace = uri
	el := d.getEl(n)
	el.d = d
	return el
}

var elEventListener = make(map[*html.Node]map[string][]*js.Object)

func (el *Element) Obj() (obj *js.Object) {
	obj, ok := elObjRefs[el]
	if ok {
		return
	}
	obj = vm.NewDynamicObject(el)
	elObjRefs[el] = obj
	err := obj.SetPrototype(NodePrototype.(*js.Object))
	if err != nil {
		panic(err.Error())
	}
	/*err = obj.SetPrototype(HTMLElementPrototype.(*js.Object))
	if err != nil {
		panic(err.Error())
	}*/
	return
}

func (el *Element) ClassName() string {
	return attr(*el.n, "class")
}

func (el *Element) Name() js.Value {
	if !hasAttr(*el.n, "name") {
		return js.Undefined()
	}
	return vm.ToValue(attr(*el.n, "name"))
}

func (el *Element) TagName() string {
	if el.n.Type == html.DocumentNode {
		return "HTML"
	}
	return strings.ToUpper(el.n.Data)
}

func (el *Element) NodeName() string {
	switch el.n.Type {
	case html.CommentNode:
		return "#comment"
	case html.TextNode:
		return "#text"
	case html.DocumentNode:
		return "HTML"
	}
	return strings.ToUpper(el.n.Data)
}

func (el *Element) NodeValue() js.Value {
	if el.n.Type == html.CommentNode {
		return vm.ToValue(el.n.Data)
	} else if el.n.Type == html.TextNode {
		return vm.ToValue(el.n.Data)
	} else {
		return vm.ToValue(nil)
	}
}

func (el *Element) NodeType() (i int) {
	switch el.n.Type {
	case html.ElementNode:
		i = 1
	case html.TextNode:
		i = 3
	case html.DocumentNode:
		i = 9
	case html.CommentNode:
		i = 8
	}
	return
}

func (el *Element) LocalName() string {
	return el.n.Data
}

func (el *Element) Content() js.Value {
	if el.n.Data == "template" {
		df := el.d.CreateDocumentFragment()
		for n := el.n.FirstChild; n != nil; n = n.NextSibling {
			if n.Type != html.ElementNode {
				continue
			}
			df.AppendChild(el.d.getEl(n).CloneNode(true))
		}
		return df.Obj()
	} else {
		log.Errorf("content called for non-template element")
	}
	return vm.ToValue(el.text())
}

func (el *Element) TextContent() string {
	return el.text()
}

func (el *Element) ToString() (s string) {
	s += "[object HTML"
	if el.n.Data != "html" {
		s += strings.Title(el.n.Data)
	}
	s += "Element]"
	return
}

func (el *Element) Attributes() js.Value {
	if el.n.Type == html.CommentNode || el.n.Type == html.TextNode {
		return nil
	}
	nm := &NamedNodeMap{n: el.n}
	return nm.Obj()
}

func (el *Element) RemoveAttribute(a string) {
	rmAttr(el.n, a)
}

func (el *Element) ContentWindow() js.Value {
	res, err := vm.RunString("this")
	if err != nil {
		log.Fatalf("getting this: %v", err)
	}
	return res
}

func (el *Element) Window() *Window {
	return el.d.Window
}

func (el *Element) OwnerDocument() js.Value {
	return el.d.Obj()
}

func (el *Element) AddEventListener(e string, fn *js.Object, opts ...any) {
	id := el.n.Data
	if a := attr(*el.n, "id"); a != "" {
		id += " id=" + a
	}
	if _, ok := elEventListener[el.n]; !ok {
		elEventListener[el.n] = make(map[string][]*js.Object)
	}
	elEventListener[el.n][e] = append(elEventListener[el.n][e], fn)
}

func (el *Element) RemoveEventListener(e string, f func(js.FunctionCall) js.Value, opts ...any) {
	_, ok := elEventListener[el.n]
	if !ok {
		log.Errorf("nothing to remove")
		return
	}

	for i, ff := range elEventListener[el.n][e] {
		if fmt.Sprintf("%p", f) == fmt.Sprintf("%p", ff) {
			elEventListener[el.n][e] = append(elEventListener[el.n][e][:i], elEventListener[el.n][e][i+1:]...)
			break
		}
	}
}

func (el *Element) buttonClick(ignDisabled bool) (consumed bool) {
	var p *html.Node
	for p = el.n.Parent; p != nil && p.Data != "form"; p = p.Parent {
	}
	if p != nil {
		form := el.d.getEl(p)
		if evs, ok := elVars[form]; ok {
			f, ok := evs["onsubmit"]
			if ok {
				consumed = true
				fn, ok := js.AssertFunction(vm.ToValue(f))
				if !ok {
					log.Errorf("el assert function: %v", ok)
				} else {
					_, err := fn(el.Obj())
					if err != nil {
						log.Errorf("el event handler fn: %v", err)
					}
				}
			}
		}
	}
	return consumed
}

func (el *Element) inputClick(ignDisabled bool) {
	if attr(*el.n, "type") == "checkbox" {
		if !hasAttr(*el.n, "disabled") || ignDisabled {
			if hasAttr(*el.n, "checked") {
				rmAttr(el.n, "checked")
			} else {
				setAttr(el.n, "checked", "true")
			}
		}
	} else if attr(*el.n, "type") == "radio" {
		setAttr(el.n, "checked", "true")
	}
}

// TODO: https://datastation.multiprocess.io/blog/2022-04-26-event-handler-attributes.html
func (el *Element) attachEvent(e string, fn *js.Object) {
	el.AddEventListener(e, fn)
}

func (el *Element) DispatchEvent(ei any) bool {
	var e *Event
	switch v := ei.(type) {
	case *Event:
		e = v
	case *MouseEvent:
		e = &v.Event
	default:
		log.Fatalf("unknown type %T", ei)
	}
	if e.Target == nil {
		e.Target = el
	}
	e.CurrentTarget = el
	e.SrcElement = el
	if e.CancelBubble {
		e.CancelBubble = false
		return true
	}
	if e.propagationStopped {
		e.propagationStopped = false
		return true
	}
	if e.Type == "click" {
		if el.n.Data == "button" {
			if c := el.buttonClick(true); c {
				e.Consumed = true
			}
		}
		if el.n.Data == "input" && !e.DefaultPrevented {
			if _, ok := ei.(*MouseEvent); ok {
				el.inputClick(true)
			} else {
				log.Errorf("dispatch event called with wrong Event Class (type %v)", e.Type)
				return false
			}
		}
		if onclick := attr(*el.n, "onclick"); onclick != "" {
			_, err := vm.RunString(onclick)
			if err != nil {
				log.Errorf("onclick '%v': %v", onclick, err)
			}
			e.Consumed = true
		}
	}
	if evs, ok := elVars[el]; ok {
		f, ok := evs["on"+e.Type]
		if ok {
			fn, ok := js.AssertFunction(vm.ToValue(f))
			if !ok {
				log.Errorf("el assert function: %v", ok)
			} else {
				_, err := fn(el.Obj())
				if err != nil {
					log.Errorf("el event handler fn: %v", err)
				}
			}
		}
	}
	hs, ok := elEventListener[el.n]
	if ok {
		for _, x := range hs[e.Type] {
			var f func(js.FunctionCall) js.Value
			var this = el.Obj()
			switch v := x.Export().(type) {
			case func(js.FunctionCall) js.Value:
				f = v
			case map[string]any:
				f, ok = v["handleEvent"].(func(js.FunctionCall) js.Value)
				if !ok {
					log.Errorf("handleEvent is not a function")
					continue
				}
				this = x
			default:
				log.Errorf("unexpected event handler %+v %T", x, x) // TODO Errorf
			}
			fn, ok := js.AssertFunction(vm.ToValue(f))
			if !ok {
				log.Errorf("el assert function: %v", ok)
				e.Consumed = true
				continue
			}
			e.Consumed = true
			_, err := fn(this, e.Obj())
			if err != nil {
				log.Errorf("el event handler fn: %v", err)
				e.Consumed = true
				continue
			}
		}
	}
	if e.Bubbles {
		if p := el.n.Parent; p != nil {
			el.d.getEl(p).DispatchEvent(e)
		} else if el.df == nil {
			el.d.DispatchEvent(e)
		}
	}
	return e.Consumed
}

func (el *Element) Click(xs ...interface{}) js.Value {
	el.Clic()
	return vm.ToValue(nil)
}

func (el *Element) Clic() (consumed bool) {
	e := &Event{
		Type:    "click",
		Target:  el,
		Bubbles: true,
	}
	return el.DispatchEvent(e)
	/*if hasAttr(*el.n, "disabled") {
		return
	}
	if el.n.Data == "button" {
		el.buttonClick(false)
	}
	if el.n.Data == "input" {
		el.inputClick(false)
	}
	if onclick := attr(*el.n, "onclick"); onclick == "" {
		// noop
	} else {
		_, err := vm.RunString(onclick)
		if err != nil {
			log.Errorf("onclick' '%v': %v", onclick, err)
		}
		consumed = true
	}
	if evs, ok := elVars[el]; ok {
		f, ok := evs["onclick"]
		if ok {
			fn, ok := js.AssertFunction(vm.ToValue(f))
			if !ok {
				log.Errorf("el assert function: %v", ok)
			} else {
				_, err := fn(this.Obj())
				if err != nil {
					log.Errorf("el event handler fn: %v", err)
				}
				consumed = true
			}
		}
	}*/
	return
}

func (el *Element) BubbledClick() {
	for e := el; e != nil; e = el.d.getEl(el.n.Parent) {

	}
}

func (el *Element) Getters() map[string]bool {
	return map[string]bool{
		"className":       true,
		"name":            true,
		"style":           true,
		"tagName":         true,
		"nodeValue":       true,
		"nodeType":        true,
		"nodeName":        true,
		"localName":       true,
		"attributes":      true,
		"id":              true,
		"type":            true,
		"value":           true,
		"selected":        true,
		"checked":         true,
		"content":         true,
		"textContent":     true,
		"innerHTML":       true,
		"outerHTML":       true,
		"contentWindow":   true,
		"window":          true,
		"ownerDocument":   true,
		"parentNode":      true,
		"parentElement":   true,
		"firstChild":      true,
		"previousSibling": true,
		"nextSibling":     true,
		"lastChild":       true,
		"childNodes":      true,
		"children":        true,
		"hash":            true,
		"href":            true,
		"src":             true,
		"hostname":        true,
		"pathname":        true,
		"data":            true,
		"length":          true,
		"offsetHeight":    true,
		"offsetWidth":     true,
	}
}

func (el *Element) Props() map[string]bool {
	return map[string]bool{}
}

func (el *Element) Get(key string) js.Value {
	if el == nil {
		log.Errorf("nil element")
		return js.Undefined()
	}
	if el.n == nil {
		log.Errorf("element with nil node")
		return js.Undefined()
	}
	if vs, ok := elVars[el]; ok {
		if v, ok := vs[key]; ok {
			return v
		}
	}
	if key == "addEventListener" || key == "attachEvent" {
		// dispatch here because 2nd parameter can be a function or an object
		c := &Call{
			recv:  "*dom.Element",
			k:     key,
			found: true,
		}
		calls = append(calls, c)
		return vm.ToValue(func(e string, fn *js.Object, opts ...any) {
			el.AddEventListener(e, fn, opts)
		})
	}
	if res, ok := GetCall(el, key); ok {
		return res
	}
	return js.Undefined()
}

func (el *Element) GetAttribute(k string, opts ...any) interface{} {
	if !hasAttr(*el.n, k) {
		return nil
	}
	return attr(*el.n, k)
}

func (el *Element) HasAttribute(k string) bool {
	return hasAttr(*el.n, k)
}

func (el *Element) SetAttribute(k, v string) {
	setAttr(el.n, k, v)
}

func (el *Element) Id() string {
	return attr(*el.n, "id")
}

func (el *Element) Type() string {
	return attr(*el.n, "type")
}

// Options of a <select> element
func (el *Element) Options() *HTMLCollection {
	return &HTMLCollection{
		d: el.d,
		f: func() []*html.Node {
			return grepAll(el.n, "option", false)
		},
	}
}

func (el *Element) Value() string {
	if el.n.Data == "select" {
		opts := grepAll(el.n, "option", false)
		var sel *html.Node
		for _, opt := range opts {
			if sel == nil || hasAttr(*sel, "selected") {
				sel = opt
			}
		}
		if sel == nil {
			return ""
		}
		v := sel.Data
		if hasAttr(*sel, "value") {
			return attr(*sel, "value")
		}
		return v
	}
	v := attr(*el.n, "value")
	return v
}

func (el *Element) Selected() string {
	return attr(*el.n, "selected")
}

func (el *Element) Checked() bool {
	return hasAttr(*el.n, "checked")
}

func (el *Element) Hash() js.Value {
	if el.n.Data == "a" {
		href := attr(*el.n, "href")
		i := strings.Index(href, "#")
		if i >= 0 {
			return vm.ToValue(href[i:])
		} else {
			return vm.ToValue("")
		}
	}
	return vm.ToValue(nil)
}

func (el *Element) Href() js.Value {
	href := attr(*el.n, "href")
	if strings.HasPrefix(href, "#") {
		return vm.ToValue("https://example.com" + href)
	}
	return vm.ToValue(href)
}

func (el *Element) Src() js.Value {
	src := attr(*el.n, "src")
	if strings.HasPrefix(src, "#") {
		return vm.ToValue("https://example.com" + src)
	}
	return vm.ToValue(src)
}

func (el *Element) Hostname() js.Value {
	h := attr(*el.n, "href")
	if h != "" {
		if u, err := url.Parse(h); err == nil {
			h = u.Path
		} else {
			log.Errorf("hostname %v: %v", h, err)
		}
	}
	return vm.ToValue(h)
}

func (el *Element) Pathname() js.Value {
	p := attr(*el.n, "href")
	if p != "" {
		if u, err := url.Parse(p); err == nil {
			p = u.Path
		} else {
			log.Errorf("pathname %v: %v", p, err)
		}
	}
	return vm.ToValue(p)
}

func (el *Element) Set(key string, desc js.PropertyDescriptor) bool {
	val := desc.Value
	c := &Call{
		recv:  "Element",
		k:     "XSet" + key,
		found: true,
	}
	calls = append(calls, c)
	switch key {
	case "nodeValue":
		switch el.n.Type {
		case html.ElementNode:
			// no effect
		case html.CommentNode, html.TextNode:
			el.n.Data = val.String()
		}
	case "className":
		setAttr(el.n, "class", val.String())
	case "href", "id", "type", "value", "selected", "src":
		setAttr(el.n, key, val.String())
	case "textContent":
		el.setText(val.String())
	case "innerHTML":
		el.setInnerHTML(val.String())
	case "outerHTML":
		el.setOuterHTML(val.String())
	case "disabled":
		if val.ToBoolean() {
			setAttr(el.n, "disabled", "true")
		} else {
			rmAttr(el.n, "disabled")
		}
	default:
		if _, ok := elVars[el]; !ok {
			elVars[el] = make(map[string]js.Value)
		}
		elVars[el][key] = val
	}
	return true
}

func (el *Element) Has(key string) bool {
	if vs, ok := elVars[el]; ok {
		if _, ok := vs[key]; ok {
			return true
		}
	}
	return HasCall(el, key)
}

func (el *Element) Delete(key string) bool {
	if vs, ok := elVars[el]; ok {
		if _, ok := vs[key]; ok {
			delete(vs, key)
			return true
		}
	}
	return false
}

func (el *Element) Keys() []string {
	ks := Calls(el)
	for k := range elVars[el] {
		ks = append(ks, k)
	}
	return ks
}

func (el *Element) setInnerHTML(h string) {
	for el.n.FirstChild != nil {
		el.n.RemoveChild(el.n.FirstChild)
	}
	ns := el.n.Namespace
	if ns != "" {
		el.n.Namespace = ""
		defer func() {
			var f func(*html.Node)
			f = func(n *html.Node) {
				n.Namespace = ns
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			}
			f(el.n)
		}()
	}
	f, err := html.ParseFragment(strings.NewReader(h), el.n)
	if err != nil {
		log.Errorf("set inner html: %v", err)
		return
	}
	for _, c := range f {
		el.n.AppendChild(c)
	}
	i := 0
	for c := el.n.FirstChild; c != nil; c = c.NextSibling {
		i++
	}
	addMutation(el.d, Value, el.n)
}

func (el *Element) setOuterHTML(h string) {
	f, err := html.ParseFragment(strings.NewReader(h), el.n)
	if err != nil {
		log.Errorf("set outer html: %v", err)
		return
	}
	if len(f) != 1 {
		panic("...")
	}
	el.d.getEl(el.n.Parent).ReplaceChild(el.d.getEl(f[0]), el)
	addMutation(el.d, Value, el.n)
}

func (el *Element) text() string {
	var f func(*html.Node, int) string
	f = func(n *html.Node, r int) (t string) {
		if r > 30 {
			log.Errorf("element text: recursion limit exceeded")
			return
		}
		if n.Type == html.TextNode || n.Type == html.ElementNode {
			t += n.Data
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			t += f(c, r+1)
		}
		return
	}
	return f(el.n, 0)
}

func (el *Element) setText(t string) {
	for el.n.FirstChild != nil {
		el.n.RemoveChild(el.n.FirstChild)
	}
	if t != "" {
		tn := &html.Node{
			Type: html.TextNode,
			Data: t,
		}
		el.n.AppendChild(tn)
	}
	addMutation(el.d, Value, el.n)
}

func (el *Element) InnerHTML() string {
	buf := bytes.NewBufferString("")
	for c := el.n.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(buf, c); err != nil {
			log.Errorf("render: %v", err)
			return ""
		}
	}
	return buf.String()
}

func (el *Element) OuterHTML() string {
	buf := bytes.NewBufferString("")
	if err := html.Render(buf, el.n); err != nil {
		log.Errorf("render: %v", err)
		return ""
	}
	return buf.String()
}

func (el *Element) getElementById(id string) (e *Element) {
	n := grepById(el.n, id)
	return el.d.getEl(n)
}

func (el *Element) GetElementsByClassName(class string) (hc *HTMLCollection) {
	f := func() []*html.Node {
		class = strings.TrimSpace(class)
		return grepByClass(el.n, class, true)
	}
	hc = &HTMLCollection{d: el.d, f: f}
	return
}

func (el *Element) GetElementsByTagName(tag string) (hc *HTMLCollection) {
	f := func() []*html.Node {
		return grepAll(el.n, tag, true)
	}
	hc = &HTMLCollection{d: el.d, f: f}
	return
}

func (el *Element) getElementsByName(name string) (els []*Element) {
	ns := grepByName(el.n, name)
	els = make([]*Element, 0, len(ns))
	for _, n := range ns {
		els = append(els, el.d.getEl(n))
	}
	return
}

func (el *Element) Contains(o *Element) bool {
	if el == o {
		return false
	}
	for ch := el.n.FirstChild; ch != nil; ch = ch.NextSibling {
		if chEl := el.d.getEl(ch); chEl == o || chEl.Contains(o) {
			return true
		}
	}
	return false
}

func (el *Element) Matches(s string) bool {
	res, err := sel.Select(s, el.n, false, true)
	if err != nil {
		log.Errorf("select %s: %v", s, err)
		return false
	}
	return len(res) > 0
}

func (el *Element) QuerySelector(s string) *Element {
	es := el.QuerySelectorAll(s)
	if len(es) == 0 {
		return nil
	}
	return es[0]
}

func (el *Element) QuerySelectorAll(s string) (els []*Element) {
	res, err := sel.Select(s, el.n, true, false)
	if err != nil {
		log.Errorf("select %s: %v", s, err)
		return
	}
	els = make([]*Element, 0, len(res))
	for _, n := range res {
		els = append(els, el.d.getEl(n))
	}
	return
}

func (el *Element) GetRootNode(opts ...any) js.Value {
	if el.df != nil {
		return el.df.Obj()
	}
	if el.n.Parent == nil {
		return js.Undefined()
	}
	r := el.n
	for {
		if r.Parent == nil {
			break
		}
		r = r.Parent
	}
	return el.d.getEl(r).Obj()
}

func (el *Element) ParentNode() js.Value {
	if el.df != nil {
		return el.df.Obj()
	}
	if el.n.Type == html.DocumentNode {
		return el.d.Obj()
	}
	e := el.ParentElement()
	if e == nil {
		return js.Null()
	}
	return e
}

func (el *Element) ParentElement() *js.Object {
	if el.n.Parent == nil {
		return nil
	}
	o := el.d.getEl(el.n.Parent).Obj()
	return o
}

func (el *Element) FirstChild() *js.Object {
	if el.n.FirstChild == nil {
		return nil
	}
	o := el.d.getEl(el.n.FirstChild).Obj()
	return o
}

func (el *Element) PreviousSibling() *js.Object {
	if el.n.PrevSibling == nil {
		return nil
	}
	return el.d.getEl(el.n.PrevSibling).Obj()
}

func (el *Element) NextSibling() *js.Object {
	if el.n.NextSibling == nil {
		return nil
	}
	return el.d.getEl(el.n.NextSibling).Obj()
}

func (el *Element) LastChild() *js.Object {
	if el.n.LastChild == nil {
		return nil
	}
	o := el.d.getEl(el.n.LastChild).Obj()
	return o
}

func (el *Element) HasChildNodes() bool {
	return el.n.FirstChild != nil
}

func (el *Element) ChildNodes() *js.Object {
	return el.Children()
}

var elChildren = make(map[*Element]*HTMLCollection)

func (el *Element) Children() *js.Object {
	hc, ok := elChildren[el]
	if !ok {
		hc = &HTMLCollection{
			d: el.d,
			f: func() []*html.Node {
				nodes := make([]*html.Node, 0, 2)
				for c := el.n.FirstChild; c != nil; c = c.NextSibling {
					nodes = append(nodes, c)
				}
				return nodes
			},
		}
		elChildren[el] = hc
	}
	return hc.Obj()
}

func (el *Element) Normalize() {
	normalize(el.n)
}

func (el *Element) SplitText(i int) *Element {
	n := &html.Node{}
	n.Data = el.n.Data[i:]
	n.Type = html.TextNode
	el.n.Parent.InsertBefore(n, el.n.NextSibling)
	el.n.Data = el.n.Data[:i]
	addMutation(el.d, Value, el.n.Parent)
	return el.d.getEl(n)
}

func (el *Element) Data() string {
	return el.n.Data
}

func (el *Element) SubstringData(i, n int) string {
	if i+n > len(el.n.Data) {
		n = len(el.n.Data) - i
	}
	return el.n.Data[i : i+n]
}

func (el *Element) AppendData(s string) {
	el.n.Data += s
	addMutation(el.d, Value, el.n)
}

func (el *Element) DeleteData(i, n int) {
	if i >= len(el.n.Data) {
		i = len(el.n.Data) - 1
	}
	if i < 0 {
		i = 0
	}
	if i+n > len(el.n.Data) {
		n = len(el.n.Data) - i
	}
	if n < 0 {
		n = 0
	}
	el.n.Data = el.n.Data[:i] + el.n.Data[i+n:]
	addMutation(el.d, Value, el.n)
}

func (el *Element) InsertData(i int, s string) {
	el.n.Data = el.n.Data[:i] + s + el.n.Data[i:]
	addMutation(el.d, Value, el.n)
}

func (el *Element) ReplaceData(i, n int, s string) {
	rem := i + n
	if rem < 0 {
		rem = 0
	}
	if rem > len(el.n.Data) {
		rem = len(el.n.Data)
	}
	if n > len(s) {
		n = len(s)
	}
	el.n.Data = el.n.Data[:i] + s[:n] + el.n.Data[rem:]
	addMutation(el.d, Value, el.n)
}

func (el *Element) Length() int {
	return len(el.n.Data)
}

func (el *Element) Remove() js.Value {
	if p := el.n.Parent; p != nil {
		el.d.getEl(p).RemoveChild(el)
		addMutation(el.d, Value, p)
	}
	return js.Undefined()
}

func (el *Element) CloneNode(deep ...bool) *Element {
	var d bool
	if len(deep) == 1 {
		d = deep[0]
	}
	if d {
		h := render(el.n)
		// TODO: does this create references to el.n? (it shouldn't!)
		cs, err := html.ParseFragment(strings.NewReader(h), el.n)
		if err != nil {
			log.Errorf("parge fragment: %v", err)
			return nil
		}
		var c *html.Node
		for _, cc := range cs {
			if cc.Type == html.TextNode {
				continue
			}
			if c == nil {
				c = cc
			} else {
				log.Errorf("parge fragment %v: has unexpected len %v", h, len(cs))
				return nil
			}
		}
		if c == nil {
			log.Errorf("parge fragment %v: has unexpected len %v", h, len(cs))
			return nil
		}
		o := el.d.getEl(cs[0])
		return o
	} else {
		cl := &html.Node{
			Type:      el.n.Type,
			DataAtom:  el.n.DataAtom,
			Data:      el.n.Data,
			Namespace: el.n.Namespace,
			Attr:      append([]html.Attribute{}, el.n.Attr...),
		}
		o := el.d.getEl(cl)
		return o
	}
}

func (el *Element) IsEqualNode(n any) bool {
	e, ok := n.(*Element)
	if !ok {
		return false
	}
	if len(el.n.Attr) != len(e.n.Attr) {
		return false
	}
	allAttrsMatch := true
	for _, a := range el.n.Attr {
		found := false
		for _, b := range e.n.Attr {
			if a.Namespace == b.Namespace && a.Key == b.Key && a.Val == b.Val {
				found = true
				break
			}
		}
		if !found {
			allAttrsMatch = false
		}
	}
	if !allAttrsMatch {
		return false
	}
	if el.n.Data != e.n.Data || el.n.Type != e.n.Type || el.n.Namespace != e.n.Namespace {
		return false
	}
	c := el.n.FirstChild
	d := e.n.FirstChild
	for {
		if (c == nil) != (d == nil) {
			return false
		}
		if c == nil {
			break
		}
		if !el.d.getEl(c).IsEqualNode(el.d.getEl(d)) {
			return false
		}
		c = el.d.getEl(c).n.NextSibling
		d = el.d.getEl(d).n.NextSibling
	}
	return true
}

func (el *Element) IsSameNode(n ...any) bool {
	if len(n) == 0 {
		return false
	}
	e, ok := n[0].(*Element)
	if !ok {
		return false
	}
	return e == el
}

func (el *Element) InsertBefore(nu, old any) any {
	switch v := nu.(type) {
	case *html.Node:
		return el.insertElement(el.d.getEl(v), old)
	case *Element:
		return el.insertElement(v, old)
	case *DocumentFragment:
		for _, cc := range v.children {
			el.d.getEl(cc).df = nil // cache
			el.InsertBefore(cc, old)
		}
		return v
	}
	log.Fatalf("not implemented %T", nu)
	return nil
}

func (el *Element) insertElement(nue *Element, old any) *Element {
	if nue.n.Parent != nil {
		nue.n.Parent.RemoveChild(nue.n)
	}
	var oe *Element
	oe, ok := old.(*Element)
	if !ok {
		el.n.InsertBefore(nue.n, nil)
	} else {
		el.n.InsertBefore(nue.n, oe.n)
	}
	addMutation(el.d, Insert, nue.n)
	return nue
}

func (el *Element) ReplaceChild(nue, ole *Element) *Element {
	for cc := el.n.FirstChild; cc != nil; cc = cc.NextSibling {
		if cc == nue.n {
			el.n.RemoveChild(nue.n)
			break
		}
	}
	for cc := el.n.FirstChild; cc != nil; cc = cc.NextSibling {
		if cc == ole.n {
			nx := ole.n.NextSibling
			el.n.RemoveChild(ole.n)
			el.n.InsertBefore(nue.n, nx)
			return ole
		}
	}
	return nil
}

func (el *Element) AppendChild(c any) any {
	switch v := c.(type) {
	case *html.Node:
		return el.appendElement(el.d.getEl(v))
	case *Element:
		return el.appendElement(v)
	case *DocumentFragment:
		for _, cc := range v.children {
			el.d.getEl(cc).df = nil // cache
			el.AppendChild(cc)
		}
		return v
	case map[string]any:
		// TODO
		log.Errorf("appendChild called with map[string]any")
		return nil
	}
	log.Fatalf("not implemented %T, %+v", c, c)
	return nil
}

func (el *Element) appendElement(e *Element) *Element {
	ce := e
	for cc := el.n.FirstChild; cc != nil; cc = cc.NextSibling {
		if cc == ce.n {
			el.n.RemoveChild(cc)
			break
		}
	}
	el.n.AppendChild(ce.n)
	addMutation(el.d, Insert, ce.n)
	return ce
}

func (el *Element) RemoveChild(c any) *Element {
	ce, ok := c.(*Element)
	if !ok || ce == nil || ce.n == nil {
		log.Errorf("remove child: nil arg")
		return nil
	}
	found := false
	for cc := el.n.FirstChild; cc != nil; cc = cc.NextSibling {
		if cc == ce.n {
			found = true
			break
		}
	}
	if !found {
		log.Errorf("child to remove not found")
		return nil
	}
	el.n.RemoveChild(ce.n)
	addMutation(el.d, Rm, el.n)
	return ce
}

func (el *Element) Node() *html.Node {
	return el.n
}

func (el *Element) Animate(opts ...any) *Animation {
	return &Animation{}
}

func (el *Element) Style() *js.Object {
	return vm.NewDynamicObject(&Style{
		n: el.n,
	})
}

func (el *Element) OffsetHeight() int {
	x1, _, x2, _ := el.geom()
	return x2 - x1
}

func (el *Element) OffsetWidth() int {
	_, y1, _, y2 := el.geom()
	return y2 - y1
}

func (el *Element) geom() (x1, y1, x2, y2 int) {
	if Geom == nil {
		log.Errorf("Geom is nil")
		return
	}
	p, ok := path(el)
	if !ok {
		log.Errorf("path lookup failed")
		return
	}
	geom, err := Geom(p)
	if err != nil {
		log.Errorf("geom %v: %v", p, err)
		return
	}
	items := strings.Split(geom, ",")
	x1, _ = strconv.Atoi(items[0])
	y1, _ = strconv.Atoi(items[1])
	x2, _ = strconv.Atoi(items[2])
	y2, _ = strconv.Atoi(items[3])
	return
}

func (el *Element) GetClientRects() *js.Object {
	return vm.NewDynamicObject(&DOMRect{})
}

func (el *Element) GetBoundingClientRect() *DOMRect {
	return &DOMRect{}
}

func Init(r *js.Runtime, url, htm, script string) (d *Document, err error) {
	vm = r
	doc, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		return
	}
	vm.SetFieldNameMapper(js.TagFieldNameMapper("json", true))

	_, err = vm.RunString(`
		function queueMicrotask(callback) {
			// https://developer.mozilla.org/en-US/docs/Web/API/queueMicrotask
		    Promise.resolve()
		      .then(callback)
		      .catch(e => setTimeout(() => { throw e; })); // report exceptions
		};

		function CDATASection() {}
		function CharacterData() {}
		function HTMLIFrameElement() {}
		function HTMLSlotElement() {}
		function HTMLTemplateElement() {}

		function NodeFilter() {}
		function ProcessingInstruction() {}
		function Window() {}

		NodeFilter.FILTER_ACCEPT = 1;
		NodeFilter.FILTER_REJECT = 2;
		NodeFilter.FILTER_SKIP = 3;

		NodeFilter.SHOW_ALL = 0xFFFFFFFF;
		NodeFilter.SHOW_ELEMENT = 0x1;
		NodeFilter.SHOW_ATTRIBUTE = 0x2;
		NodeFilter.SHOW_TEXT = 0x4;
		NodeFilter.SHOW_CDATA_SECTION = 0x8;
		NodeFilter.SHOW_ENTITY_REFERENCE = 0x10;
		NodeFilter.SHOW_ENTITY = 0x20;
		NodeFilter.SHOW_PROCESSING_INSTRUCTION = 0x40;
		NodeFilter.SHOW_COMMENT = 0x80;
		NodeFilter.SHOW_DOCUMENT = 0x100;
		NodeFilter.SHOW_DOCUMENT_TYPE = 0x200;
		NodeFilter.SHOW_DOCUMENT_FRAGMENT = 0x400;
		NodeFilter.SHOW_NOTATION = 0x800;
	`)
	if err != nil {
		return nil, fmt.Errorf("define misc entities: %v", err)
	}
	NodePrototype, err = vm.RunString(`
		function Node() {};
		Node.ELEMENT_NODE = 1;
		Node.ATTRIBUTE_NODE = 2;
		Node.TEXT_NODE = 3;
		Node.CDATA_SECTION_NODE = 4;
		Node.PROCESSING_INSTRUCTION_NODE = 7;
		Node.COMMENT_NODE = 8;
		Node.DOCUMENT_NODE = 9;
		Node.DOCUMENT_TYPE_NODE = 10;
		Node.DOCUMENT_FRAGMENT_NODE = 11;
		// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Symbol/hasInstance
		Object.defineProperty(Node, Symbol.hasInstance, {
			value: function(instance) { return instance.nodeType === 1; }
		});
		Object.defineProperty(Window, Symbol.hasInstance, {
			value: function(instance) { return instance === window; }
		});
		Element = Node;
		Node;`)
	if err != nil {
		return nil, fmt.Errorf("define NodePrototype: %v", err)
	}
	TextPrototype, err = vm.RunString(`function Text() {}; Text;`)
	if err != nil {
		return nil, fmt.Errorf("define Text: %v", err)
	}
	HTMLElementPrototype, err = vm.RunString(`function HTMLElement() {}; HTMLElement;`)
	if err != nil {
		return nil, fmt.Errorf("define HTMLElementPrototype: %v", err)
	}
	HTMLInputElementPrototype, err = vm.RunString(`function HTMLInputElement() {}; HTMLInputElement;`)
	if err != nil {
		return nil, fmt.Errorf("define HTMLInputElementPrototype: %v", err)
	}
	d = NewDocument(doc)
	builtinThis := vm.GlobalObject()
	w := NewWindow(url, builtinThis, d)
	d.Window = w
	vm.SetGlobalObject(w.Obj())
	_, err = vm.RunString(`
		console.log("running...");
	` + script)
	if err != nil {
		return
	}
	return
}

func grep(n *html.Node, tag string) *html.Node {
	var t *html.Node

	if n.Type == html.ElementNode {
		if n.Data == tag {
			return n
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		res := grep(c, tag)
		if res != nil {
			t = res
		}
	}

	return t
}

func grepAll(n *html.Node, tag string, skipRoot bool) (all []*html.Node) {
	tag = strings.ToLower(tag)
	if n.Type == html.ElementNode {
		if (strings.ToLower(n.Data) == tag || tag == "*") && !skipRoot {
			all = append(all, n)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		res := grepAll(c, tag, false)
		all = append(all, res...)
	}

	return all
}

func grepByClass(n *html.Node, class string, skipRoot bool) (all []*html.Node) {
	qs := classes(class)
	if n.Type == html.ElementNode && !skipRoot {
		if matchesClasses(n, qs) {
			all = append(all, n)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		res := grepByClass(c, class, false)
		all = append(all, res...)
	}
	return all
}

func matchesClasses(n *html.Node, qs []string) bool {
	s := attr(*n, "class")
	cls := classes(s)
	matchesAll := true
	for _, q := range qs {
		found := false
		for _, cl := range cls {
			if cl == q {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return matchesAll
}

func classes(cls string) (res []string) {
	cls = strings.ReplaceAll(cls, "\n", " ")
	cls = strings.ReplaceAll(cls, "\t", " ")
	cls = strings.ReplaceAll(cls, "\f", " ")
	cls = strings.ReplaceAll(cls, "\r", " ")
	tmp := strings.Split(cls, " ")
	res = make([]string, 0, len(tmp))
	for _, c := range tmp {
		if c != "" {
			res = append(res, c)
		}
	}
	return
}

func grepByName(n *html.Node, name string) (all []*html.Node) {
	if n.Type == html.ElementNode {
		if attr(*n, "name") == name {
			all = append(all, n)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		res := grepByName(c, name)
		all = append(all, res...)
	}

	return all
}

func grepById(n *html.Node, id string) *html.Node {
	var t *html.Node

	if id == "" {
		return nil
	}

	if n.Type == html.ElementNode {
		if attr(*n, "id") == id {
			return n
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		res := grepById(c, id)
		if res != nil {
			t = res
			break
		}
	}

	return t
}

func attr(n html.Node, key string) (val string) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return
}

func hasAttr(n html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func setAttr(n *html.Node, key, val string) {
	newAttr := html.Attribute{
		Key: key,
		Val: val,
	}
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i] = newAttr
			addMutation(nil, ChAttr, n)
			return
		}
	}
	n.Attr = append(n.Attr, newAttr)
}

func rmAttr(n *html.Node, key string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
			addMutation(nil, RmAttr, n)
			return
		}
	}
}

func render(n *html.Node) string {
	buf := bytes.NewBufferString("")
	if err := html.Render(buf, n); err != nil {
		log.Errorf("render: %v", err)
		return ""
	}
	return buf.String()
}

func renderInner(n *html.Node) string {
	buf := bytes.NewBufferString("")
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(buf, c); err != nil {
			log.Errorf("render inner: %v", err)
			return ""
		}
	}
	return buf.String()
}

func normalize(n *html.Node) {
	for {
		correction := false
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			normalize(c)
			if c.Type != html.TextNode {
				continue
			}
			if c.Data == "" {
				n.RemoveChild(c)
				correction = true
				break
			} else if nx := c.NextSibling; nx != nil && nx.Type == html.TextNode {
				c.Data += nx.Data
				n.RemoveChild(nx)
				correction = true
				break
			}
		}
		if !correction {
			break
		}
	}
	// TODO: mutation
}

func path(el *Element) (pth string, ok bool) {
	var p *Element

	if el == nil {
		return
	}
	if el.TagName() == "BODY" {
		return "/0", true
	}
	p = el.d.getEl(el.n.Parent)

	if p != nil {
		i := 0
		for n := p.n.FirstChild; n != nil; n = n.NextSibling {
			if n == el.n {
				pre, ok := path(p)
				if ok {
					return pre + "/" + strconv.Itoa(i), true
				}
			}
			if n.Type == html.ElementNode || (n.Type == html.TextNode && strings.TrimSpace(n.Data) != "") {
				i++
			}
		}
	}
	return
}

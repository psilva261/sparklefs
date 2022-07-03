package dom

import (
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparklefs/logger"
	"strings"
)

type Event struct {
	Type               string
	Consumed           bool
	DefaultPrevented   bool
	Phase              int
	Bubbles            bool
	CancelBubble       bool
	Cancelable         bool
	IsTrusted          bool
	propagationStopped bool
	CurrentTarget      *Element
	Target             *Element
	SrcElement         *Element
}

const (
	EvPhNone = iota
)

func NewEvent(t string, opts map[string]any) (o *js.Object) {
	e := &Event{
		Type: t,
	}
	if yes, ok := opts["bubbles"]; ok {
		e.Bubbles, _ = yes.(bool)
	}
	return e.Obj()
}

func (e *Event) InitEvent(t string, opts ...any) {
	var bubbles, cancelable bool
	if len(opts) >= 1 {
		bubbles, _ = opts[0].(bool)
	}
	if len(opts) >= 2 {
		cancelable, _ = opts[1].(bool)
	}
	if e.Consumed {
		log.Errorf("init on consumed event")
		return
	}
	e.Type = strings.ToLower(t)
	e.Bubbles = bubbles
	e.Cancelable = cancelable
	e.DefaultPrevented = false
	e.propagationStopped = false
}

func (e *Event) PreventDefault() {
	if e.Cancelable {
		e.DefaultPrevented = true
	}
}

func (e *Event) ReturnValue() bool {
	return !e.DefaultPrevented
}

func (e *Event) StopPropagation() {
	e.propagationStopped = true
}

func (e *Event) StopImmediatePropagation() {
	e.propagationStopped = true
}

func (e *Event) Getters() map[string]bool {
	return map[string]bool{
		"length":      true,
		"returnValue": true,
	}
}

func (e *Event) Props() map[string]bool {
	return map[string]bool{
		"bubbles":          true,
		"cancelBubble":     true,
		"cancelable":       true,
		"eventPhase":       true,
		"isTrusted":        true,
		"defaultPrevented": true,
		"type":             true,
		"currentTarget":    true,
		//"target": true,
		"srcElement": true,
	}
}

func (e *Event) Obj() (o *js.Object) {
	o, ok := evObjRefs[e]
	if ok {
		return
	}
	o = vm.NewDynamicObject(e)
	evObjRefs[e] = o
	return
}

func (e *Event) Get(key string) js.Value {
	if key == "target" { // TODO: reflect dyn obj. also fails here
		if e.Target == nil {
			return js.Null()
		}
		return e.Target.Obj()
	}
	if key == "currentTarget" { // TODO: reflect dyn obj. also fails here
		if e.CurrentTarget == nil {
			return js.Null()
		}
		return e.CurrentTarget.Obj()
	}
	if vs, ok := evVars[e]; ok {
		if v, ok := vs[key]; ok {
			return v
		}
	}
	if res, ok := GetCall(e, key); ok {
		return res
	}
	return js.Undefined()
}

func (e *Event) Set(key string, desc js.PropertyDescriptor) bool {
	val := desc.Value
	switch key {
	case "cancelBubble":
		e.CancelBubble = val.ToBoolean()
	case "returnValue":
		if e.Cancelable {
			e.DefaultPrevented = !val.ToBoolean()
		}
	case "target":
		// r/o
		/*log.Infof("event set target to %v", val)
		e.Target = val.Export().(*Element)*/
	default:
		if _, ok := evVars[e]; !ok {
			evVars[e] = make(map[string]js.Value)
		}
		evVars[e][key] = val
	}
	return true
}

func (e *Event) Has(key string) bool {
	if vs, ok := evVars[e]; ok {
		if _, ok := vs[key]; ok {
			return true
		}
	}
	return HasCall(e, key)
}

func (e *Event) Delete(key string) (ok bool) {
	if vs, ok := evVars[e]; ok {
		if _, ok := vs[key]; ok {
			delete(vs, key)
			return true
		}
	}
	return false
}

func (e *Event) Keys() []string {
	ks := Calls(e)
	for k := range evVars[e] {
		ks = append(ks, k)
	}
	return ks
}

type MouseEvent struct {
	Event
}

func NewMouseEvent(t string, opts map[string]any) (o *js.Object) {
	e := Event{
		Type: t,
	}
	if v, ok := opts["cancelable"]; ok {
		e.Cancelable = v.(bool)
	}
	o = vm.NewDynamicObject(&MouseEvent{
		Event: e,
	})
	return
}

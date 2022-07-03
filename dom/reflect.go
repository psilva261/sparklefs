package dom

import (
	"fmt"
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparklefs/logger"
	"reflect"
	"strconv"
	"strings"
)

var (
	calls []*Call = nil
)

type Call struct {
	recv   string
	k      string
	prop   bool
	getter bool
	found  bool
	//res string

	// aggregated by recv, k
	n int
}

func ResetCalls() {
	calls = make([]*Call, 0, 1000)
}

func PrintCalls() {
	byRecvK := make(map[string]map[string]*Call)
	for _, c := range calls {
		if _, err := strconv.Atoi(c.k); err == nil {
			c.k = "#num"
		}
		if _, ok := byRecvK[c.recv]; !ok {
			byRecvK[c.recv] = make(map[string]*Call)
		}
		if cc, ok := byRecvK[c.recv][c.k]; ok {
			cc.n++
		} else {
			cc := c
			byRecvK[c.recv][c.k] = cc
			cc.n = 1
		}
	}
	for recv, byK := range byRecvK {
		for k, c := range byK {
			s := ""
			s += fmt.Sprintf("found=%v\tn=%v\t", c.found, c.n)
			s += fmt.Sprintf("%v\t%v", recv, k)
			if !c.getter && !c.prop && c.found {
				s += fmt.Sprintf("(..)")
			}
			log.Printf("%v", s)
		}
	}
}

type Gettable interface {
	Obj() *js.Object
	Getters() map[string]bool
	Props() map[string]bool
}

// GetCall translates a js method or getter call into a Go
// method call.
func GetCall(recv Gettable, k string) (res js.Value, ok bool) {
	c := Call{}
	defer func() {
		if calls != nil {
			calls = append(calls, &c)
		}
	}()
	log.Printf("%T.%v", recv, k)
	c.recv = fmt.Sprintf("%T", recv)
	c.k = k
	hct := reflect.TypeOf(recv)
	hcr := reflect.ValueOf(recv)
	t := strings.Title(k)
	if _, ok := recv.Props()[k]; ok {
		f := reflect.Indirect(hcr).FieldByName(t)
		c.found = true
		c.prop = true
		if !f.IsValid() {
			log.Errorf("invalid prop %v", k)
			return js.Undefined(), false
		}
		return vm.ToValue(f.Interface()), true
	} else if m, ok := hct.MethodByName(t); ok && exported(t) {
		c.found = true
		if _, ok := recv.Getters()[k]; ok {
			res := m.Func.Call([]reflect.Value{hcr})
			c.getter = true
			return vm.ToValue(res[0].Interface()), true
		} else {
			return vm.ToValue(func(args ...any) js.Value {
				mt := m.Type
				as := make([]reflect.Value, 0, len(args)+1)
				as = append(as, hcr)
				for i, a := range args {
					rv, err := reflectVal(mt.In(i), a)
					if err != nil {
						log.Errorf("get call: reflect val %v: %v", a, err)
						return js.Undefined()
					}
					as = append(as, rv)
				}
				res := m.Func.Call(as)
				if len(res) == 0 {
					return vm.ToValue(nil)
				}
				rv, err := jsVal(res[0].Interface())
				if err != nil {
					log.Errorf("get call: js val %v: %v", res[0], err)
					return js.Undefined()
				}
				return rv
			}), true
		}
	}
	return vm.ToValue(nil), false
}

func reflectVal(typ reflect.Type, a any) (rv reflect.Value, err error) {
	var aa any
	switch v := a.(type) {
	case int64:
		aa = int(v)
	case *DocumentFragment, *Element, *Event, *MouseEvent, string, bool, func(js.FunctionCall) js.Value, map[string]any, []any:
		aa = a
	default:
		if v != nil {
			return rv, fmt.Errorf("unhandled arg type %T (%v)", v, v)
		} else {
			aa = reflect.New(typ).Elem()
		}
	}
	return reflect.ValueOf(aa), nil
}

func jsVal(v any) (vv js.Value, err error) {
	switch rv := v.(type) {
	case *Element:
		if rv == nil {
			break
		}
		return rv.Obj(), nil
	case []*Element:
		objs := make([]*js.Object, 0, len(rv))
		for _, el := range rv {
			objs = append(objs, el.Obj())
		}
		return vm.ToValue(objs), nil
	case *Document:
		if rv == nil {
			break
		}
		return rv.Obj(), nil
	case *DocumentFragment:
		if rv == nil {
			break
		}
		return rv.Obj(), nil
	case *Animation:
		if rv == nil {
			break
		}
		return rv.Obj(), nil
	case *DOMRect:
		if rv == nil {
			break
		}
		return rv.Obj(), nil
	case *Event:
		if rv == nil {
			break
		}
		return rv.Obj(), nil
	case *HTMLCollection:
		if rv == nil {
			break
		}
		return rv.Obj(), nil
	case *NamedNodeMap:
		if rv == nil {
			break
		}
		return vm.NewDynamicObject(rv), nil
	case bool, string:
		return vm.ToValue(rv), nil
	case js.Value:
		return rv, nil
	default:
		if rv != nil {
			return vv, fmt.Errorf("unhandled return type %T (%v)", rv, rv)
		}
	}
	return js.Null(), nil
}

func HasCall(recv Gettable, k string) bool {
	hct := reflect.TypeOf(recv)
	t := strings.Title(k)
	_, ok := recv.Props()[k]
	if ok {
		return true
	}
	_, ok = hct.MethodByName(t)
	return ok && exported(t)
}

func Calls(recv Gettable) (cs []string) {
	hct := reflect.TypeOf(recv)
	cs = make([]string, 0, len(recv.Props())+hct.NumMethod())
	for k := range recv.Props() {
		cs = append(cs, untitle(k))
	}
	for i := 0; i < hct.NumMethod(); i++ {
		if nm := hct.Method(i).Name; exported(nm) {
			cs = append(cs, untitle(nm))
		}
	}
	return
}

// exported is true if method with name m it is
func exported(m string) bool {
	m = strings.Title(m)
	return m != "Get" && m != "Set" && m != "Delete" && m != "Has" && m != "Keys"
}

// untitle is the reverse of strings.Title
func untitle(s string) string {
	s = strings.ToLower(s[0:1]) + s[1:]
	return s
}

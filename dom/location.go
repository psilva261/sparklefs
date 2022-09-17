package dom

import (
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparklefs/logger"
	"net/url"
)

type Location struct {
	Protocol string
	Host     string
	Hostname string
	Port     string
	Href     string
	Pathname string
	Search   string
	Hash     string
}

func NewLocation(origin string) (l *Location) {
	l = &Location{
		Host:     "example.com",
		Hostname: "example.com",
		Href:     "https://example.com",
		Pathname: "/",
		Port:     "443",
	}
	u, err := url.Parse(origin)
	if err != nil {
		log.Errorf("parse %v: %v", origin, err)
		return
	}
	l.Host = u.Host
	l.Hostname = u.Hostname()
	l.Port = u.Port()
	l.Pathname = u.Path
	return
}

func (l *Location) Obj() *js.Object {
	return vm.NewDynamicObject(l)
}

func (l *Location) Getters() map[string]bool {
	return map[string]bool{}
}

func (l *Location) Props() map[string]bool {
	return map[string]bool{
		"hash":     true,
		"host":     true,
		"hostname": true,
		"href":     true,
		"pathname": true,
		"port":     true,
		"protocol": true,
		"search":   true,
	}
}

func (l *Location) Get(k string) (v js.Value) {
	if res, ok := GetCall(l, k); ok {
		return res
	}
	return vm.ToValue(nil)
}

func (l *Location) Set(k string, desc js.PropertyDescriptor) bool {
	v := desc.Value
	log.Printf("location set %v", k)
	switch k {
	case "hash":
		l.Hash = v.String()
	}
	return true
}

func (l *Location) Has(key string) bool {
	return HasCall(l, key)
}

func (l *Location) Delete(key string) bool {
	return false
}

func (l *Location) Keys() []string {
	return Calls(l)
}

func (l *Location) Replace(u string) {}

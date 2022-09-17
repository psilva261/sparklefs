package dom

import (
	"errors"
	"fmt"
	"github.com/psilva261/sparkle/console"
	"github.com/psilva261/sparkle/eventloop"
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparkle/require"
	"github.com/psilva261/sparklefs/logger"
	"golang.org/x/net/html"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func Main(dir, fn string) (err error) {
	log.Debug = true
	bs, err := os.ReadFile(dir + "/" + fn)
	if err != nil {
		return
	}
	htm := string(bs)
	scripts, err := loadScripts(dir, htm)
	if err != nil {
		return
	}
	_ = scripts
	l := eventloop.NewEventLoop()
	l.Start()
	defer l.Stop()
	l.RunOnLoop(func(vm *js.Runtime) {
		registry := require.NewRegistry(
			require.WithLoader(
				require.SourceLoader(srcLoader),
			),
		)
		console.Enable(vm)
		registry.Enable(vm)
		d, err := Init(vm, "https://example.com", htm, "")
		if err != nil {
			log.Fatalf(err.Error())
		}
		_ = d
		fmt.Printf("")
		vm.Set("____assert_fail", vm.ToValue(func(msg string) {
			//fmt.Printf("%v\n", render(d.Doc()))
			log.Fatalf(msg)
		}))
		for _, s := range scripts {
			_, err := vm.RunString(s)
			if err != nil {
				log.Fatalf("run script: %v", err)
			}
		}
	})
	return
}

func loadScripts(dir, htm string) (scripts []string, err error) {
	doc, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		return
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			if hasAttr(*n, "src") {
				fn := attr(*n, "src")
				if strings.HasPrefix(fn, "/") {
					fn = "test/wpt" + fn
				} else {
					fn = dir + "/" + fn
				}
				bs, err := os.ReadFile(fn)
				if err != nil {
					log.Fatalf("read %v: %v", fn, err)
				}
				scripts = append(scripts, string(bs))
			} else {
				s := ""
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					s += c.Data
				}
				scripts = append(scripts, s)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return
}

func srcLoader(fn string) ([]byte, error) {
	path := filepath.FromSlash("w3c/" + fn)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, syscall.EISDIR) {
			log.Printf("fvffd2 %v", path)
			err = require.ModuleFileDoesNotExistError
		} else {
			log.Printf("srcLoader: handling of require('%v') is not implemented", fn)
		}
	}
	return data, err
}

func main() {
	if err := Main("test/wpt/dom/nodes", "Element-remove.html"); err != nil {
		log.Fatalf("%v", err)
	}
}

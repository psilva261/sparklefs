// js package as separate program (very wip)
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/knusbaum/go9p/fs"
	"github.com/psilva261/sparklefs"
	"github.com/psilva261/sparklefs/logger"
	"github.com/psilva261/sparklefs/runner"
	"io"
	"net"
	"net/http"
	"os"
	"os/user"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	d       *runner.Runner
	service string
	mtpt    string
	htm     string
	js      []string
	mu      sync.Mutex
)

func usage() {
	log.Printf("usage: sparklefs [-v] [-s service] [-m mtpt] [-h htmlfile jsfile1 [jsfile2] [..]]")
	os.Exit(1)
}

func Main(r io.Reader, w io.Writer) (err error) {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get user: %v", err)
	}
	un := u.Username
	gn, err := sparklefs.Group(u)
	if err != nil {
		return fmt.Errorf("get group: %v", err)
	}

	sparklefs, root := fs.NewFS(un, gn, 0500)
	c := fs.NewListenFile(sparklefs.NewStat("ctl", un, gn, 0600))
	root.AddChild(c)
	lctl := (*fs.ListenFileListener)(c)
	go AssertParent()
	go Ctl(lctl)
	log.Printf("post fs...\n")
	return post(sparklefs.Server())
}

func AssertParent() {
	for {
		<-time.After(time.Second)
		if !stat() {
			os.Exit(1)
		}
	}
}

func Ctl(lctl *fs.ListenFileListener) {
	for {
		conn, err := lctl.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go ctl(conn)
	}
}

func ctl(conn net.Conn) {
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	defer conn.Close()

	l, err := r.ReadString('\n')
	if err != nil {
		log.Printf("sparklefs: read string: %v", err)
		return
	}
	l = strings.TrimSpace(l)

	mu.Lock()
	defer mu.Unlock()

	switch l {
	case "start":
		if len(htm) > 50 {
			log.Printf("htm=%v...", htm[:50])
		} else {
			log.Printf("htm=%v", htm)
		}
		d = runner.New(htm, xhr, geom, query)
		d.Start()
		initialized := false
		for i, s := range js {
			if len(s) > 50 {
				log.Printf("call d.Exec(%v...%v, %v)", s[:25], s[len(s)-25:], !initialized)
			} else {
				log.Printf("call d.Exec(%v, %v)", s, !initialized)
			}
			if _, err := d.Exec /*56*/ (s, !initialized); err != nil {
				if strings.Contains(err.Error(), "halt at") {
					log.Printf("execution halted: %v", err)
					return
				}
				log.Printf("exec <script> %d: %v", i, err)
			}
			initialized = true
		}
		if err := d.CloseDoc(); err != nil {
			log.Printf("close doc: %v", err)
			return
		}
		resHtm, changed, err := d.TrackChanges()
		if err != nil {
			log.Printf("track changes: %v", err)
			return
		}
		log.Printf("print calls1")
		runner.PrintCalls()
		log.Printf("sparklefs: processJS: changed = %v", changed)
		if changed {
			w.WriteString(resHtm)
			w.Flush()
		}
	case "stop":
		if d != nil {
			d.Stop()
			d = nil
		}
	case "click":
		runner.ResetCalls()
		sel, err := r.ReadString('\n')
		if err != nil {
			log.Printf("sparklefs: click: read string: %v", err)
			return
		}
		sel = strings.TrimSpace(sel)
		resHtm, changed, err := d.TriggerClick(sel)
		if err != nil {
			log.Printf("track changes: %v", err)
			return
		}

		runner.PrintCalls()
		log.Printf("sparklefs: processJS: changed = %v", changed)
		if changed {
			w.WriteString(resHtm)
			w.Flush()
		}
	default:
		log.Printf("unknown cmd")
	}
}

var reFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var reAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func kebab(s string) string {
	k := reFirstCap.ReplaceAllString(s, "${1}-${2}")
	k = reAllCap.ReplaceAllString(k, "${1}-${2}")
	return strings.ToLower(k)
}

func geom(sel string) (val string, err error) {
	log.Printf("get geom(%v)", sel)
	fn := sel + "/geom"
	rwc, err := open(fn)
	if err != nil {
		log.Printf("get geom(%v): failed", sel)
		return "", fmt.Errorf("open %v: %v", fn, err)
	}
	defer rwc.Close()
	bs, err := io.ReadAll(rwc)
	val = string(bs)
	log.Printf("get geom(%v) = %v", sel, val)
	return
}

func query(sel, prop string) (val string, err error) {
	log.Printf("run query(%v, %v)", sel, prop)
	fn := sel + "/style/" + kebab(prop)
	rwc, err := open(fn)
	if err != nil {
		log.Printf("run query(%v, %v): failed", sel, prop)
		return "", fmt.Errorf("open %v: %v", fn, err)
	}
	defer rwc.Close()
	bs, err := io.ReadAll(rwc)
	val = string(bs)
	log.Printf("run query(%v, %v) = %v", sel, prop, val)
	return
}

func xhr(req *http.Request) (resp *http.Response, err error) {
	rwc, err := open("xhr")
	if err != nil {
		return nil, fmt.Errorf("open xhr: %w", err)
	}
	// defer rwc.Close()
	if err := req.Write(rwc); err != nil {
		return nil, fmt.Errorf("write: %v", err)
	}
	buf := bytes.NewBufferString("")
	if n, err := io.Copy(buf, rwc); err != nil {
		if n == 0 {
			return nil, fmt.Errorf("io copy %v: %v", n, err)
		} else {
			log.Printf("io copy (read %v): %v", n, err)
		}
	}
	log.Printf("xhr resp: %v", buf.String())
	r := bufio.NewReader(bytes.NewBufferString(buf.String()))
	if resp, err = http.ReadResponse(r, req); err != nil {
		return nil, fmt.Errorf("read resp: %v", err)
	}
	return
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
	}

	htmlfile := ""
	jsfiles := make([]string, 0, len(args))

	for len(args) > 0 {
		switch args[0] {
		case "-v":
			args = args[1:]
			log.Debug = true
		case "-s":
			service, args = args[1], args[2:]
		case "-h":
			htmlfile, args = args[1], args[2:]
		default:
			var jsfile string
			jsfile, args = args[0], args[1:]
			jsfiles = append(jsfiles, jsfile)
		}
	}

	js = make([]string, 0, len(jsfiles))
	if htmlfile != "" {
		b, err := os.ReadFile(htmlfile)
		if err != nil {
			log.Fatalf(err.Error())
		}
		htm = string(b)
	}
	for _, jsfile := range jsfiles {
		b, err := os.ReadFile(jsfile)
		if err != nil {
			log.Fatalf(err.Error())
		}
		js = append(js, string(b))
	}

	if err := Init(); err != nil {
		log.Fatalf("Init: %+v", err)
	}

	if err := Main(os.Stdin, os.Stdout); err != nil {
		log.Fatalf("Main: %+v", err)
	}
	select {}
}

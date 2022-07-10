package runner

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"github.com/psilva261/sparkle/console"
	"github.com/psilva261/sparkle/eventloop"
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparkle/js/parser"
	"github.com/psilva261/sparklefs/dom"
	"github.com/psilva261/sparklefs/logger"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	convert6to5 = os.Getenv("SPARKLEFS_6TO5")
	origin      = "https://example.com"
	timeout     = 60 * time.Second
)

//go:embed domintf.js
var domIntfJs embed.FS

//go:embed regen-runtime.js
var regenRtJs embed.FS

var domIntf string
var regenRt string

func init() {
	data, err := domIntfJs.ReadFile("domintf.js")
	if err != nil {
		panic(err.Error())
	}
	domIntf = string(data)
	data, err = regenRtJs.ReadFile("regen-runtime.js")
	if err != nil {
		panic(err.Error())
	}
	regenRt = string(data)
}

type Runner struct {
	loop       *eventloop.EventLoop
	html       string
	outputHtml string
	doc        *dom.Document
	geom       func(sel string) (val string, err error)
	query      func(sel, prop string) (val string, err error)
	xhrq       func(req *http.Request) (resp *http.Response, err error)
}

func New(
	html string,
	xhr func(req *http.Request) (resp *http.Response, err error),
	geom func(sel string) (val string, err error),
	query func(sel, prop string) (val string, err error),
) (r *Runner) {
	r = &Runner{
		html:  html,
		xhrq:  xhr,
		geom:  geom,
		query: query,
	}
	return
}

func (r *Runner) Start() {
	dom.ResetCalls()
	log.Printf("Start event loop")
	r.loop = eventloop.NewEventLoop()

	r.loop.Start()
	log.Printf("event loop started")
}

func (r *Runner) Stop() {
	r.loop.Stop()
	for len(dom.Mutations()) > 0 {
		<-dom.Mutations()
	}
}

func IntrospectError(err error, script string) {
	prefix := "Line "
	i := strings.Index(err.Error(), prefix)
	if i > 0 {
		i += len(prefix)
		s := err.Error()[i:]
		yxStart := strings.Split(s, " ")[0]
		yx := strings.Split(yxStart, ":")
		y, _ := strconv.Atoi(yx[0])
		x, _ := strconv.Atoi(yx[1])
		lines := strings.Split(script, "\n")

		if y-1 > len(lines)-1 {
			y = len(lines)
		}

		if wholeLine := lines[y-1]; len(wholeLine) > 100 {
			from := x - 50
			to := x + 50
			if from < 0 {
				from = 0
			}
			if to >= len(wholeLine) {
				to = len(wholeLine) - 1
			}
			log.Printf("the line: %v", wholeLine[from:to])
		} else {
			if y > 0 && len(lines[y-1]) < 120 {
				log.Printf("%v: %v", y-1, lines[y-1])
			}
			if y < len(lines) {
				log.Printf("%v: %v", y, lines[y])
			}
			if y+1 < len(lines) && len(lines[y+1]) < 120 {
				log.Printf("%v: %v", y+1, lines[y+1])
			}
		}
	}
}

func printCode(code string, maxWidth int) {
	if maxWidth > len(code) {
		maxWidth = len(code)
	}
	log.Printf("js code: %v", code[:maxWidth])
}

func ResetCalls() {
	dom.ResetCalls()
}

func PrintCalls() {
	dom.PrintCalls()
}

func (r *Runner) initVM(vm *js.Runtime) (err error) {
	vm.SetParserOptions(parser.WithDisableSourceMaps)

	console.Enable(vm)
	r.doc, err = dom.Init(vm, r.html, "")
	if err != nil {
		return fmt.Errorf("init dom: %w", err)
	}

	type S struct {
		Buf      string                                                                `json:"buf"`
		HTML     string                                                                `json:"html"`
		Origin   string                                                                `json:"origin"`
		Referrer func() string                                                         `json:"referrer"`
		Style    func(string, string, string, string) string                           `json:"style"`
		XHR      func(string, string, map[string]string, string, func(string, string)) `json:"xhr"`
		Mutated  func(t int, target string, tag string, node map[string]string)        `json:"mutated"`
		Btoa     func([]byte) string                                                   `json:"btoa"`
	}

	//vm.SetFieldNameMapper(js.TagFieldNameMapper("json", true))
	dom.Geom = r.geom
	vm.Set("opossum", S{
		HTML:     r.html,
		Origin:   origin,
		Referrer: func() string { return origin },
		Style: func(sel, pseudo, prop, prop2 string) string {
			v, err := r.query(sel, prop)
			if err != nil {
				log.Printf("sparkle fs: runner: query %v: %v", sel, err)
				return ""
			}
			log.Printf("call query(%v, %v)=%v", sel, prop, v)
			return v
		},
		XHR:  r.xhr,
		Btoa: Btoa,
	})

	return
}

var (
	reCompatCommentOpen = regexp.MustCompile(`^\s*<!--`)
	reCompatCommentClose = regexp.MustCompile(`-->\s*$`)
)

func (r *Runner) Exec(script string, initial bool) (res string, err error) {
	script = reCompatCommentOpen.ReplaceAllString(script, "//")
	script = reCompatCommentClose.ReplaceAllString(script, "//")
	SCRIPT := domIntf + /*regenRt +*/ script
	if !initial {
		SCRIPT = script
	}

	resCh := make(chan string, 1)
	errCh := make(chan error, 1)

	log.Printf("Start even loop...")
	r.loop.RunOnLoop(func(vm *js.Runtime) {
		log.Printf("RunOnLoop")

		if initial {
			log.Printf("exec: init vm")
			r.initVM(vm)
		}

		log.Printf("exec: run script")
		vv, err := vm.RunString(SCRIPT)
		if err != nil {
			log.Printf("exec: error occurred")
			IntrospectError(err, script)
			errCh <- fmt.Errorf("run program: %w", err)
		} else {
			log.Printf("exec: writing result")
			resCh <- vv.String()
		}
	})

	select {
	case err := <-errCh:
		return "", err
	case res := <-resCh:
		return res, nil
	case <-time.After(10 * time.Second):
		return "", fmt.Errorf("timeout")
	}
	return "", fmt.Errorf("unreachable state")
}

func Btoa(bs []byte) string {
	return base64.StdEncoding.EncodeToString(bs)
}

func (r *Runner) Exec56(script string, initial bool) (res string, err error) {
	if convert6to5 != "" {
		return r.Exec6(script, initial)
	} else {
		return r.Exec(script, initial)
	}
}

func (r *Runner) Exec6(script string, initial bool) (res string, err error) {
	cmd := exec.Command("6to5")
	cmd.Stdin = strings.NewReader(script)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("6to5: %w", err)
	}
	return r.Exec(out.String(), initial)
}

// CloseDoc fires DOMContentLoaded to trigger $(document).ready(..)
func (r *Runner) CloseDoc() (err error) {
	log.Printf("close doc")
	if r.doc == nil {
		return
	}
	errCh := make(chan error, 1)
	r.loop.RunOnLoop(func(vm *js.Runtime) {
		errCh <- r.doc.Close()
	})
	return <-errCh
}

// TriggerClick, and return the result html
func (r *Runner) TriggerClick(selector string) (newHTML string, ok bool, err error) {
	consumedCh := make(chan bool, 1)
	errCh := make(chan error, 1)
	r.loop.RunOnLoop(func(vm *js.Runtime) {
		el := r.doc.Element()
		el = el.QuerySelector(selector)
		if el == nil {
			errCh <- fmt.Errorf("could not find '%v'", selector)
			return
		}
		var consumed bool
		var h string
		var e *dom.MouseEvent
		var e2 *dom.Event
		var e3 *dom.Event
		if consumed = el.Clic(); consumed {
			goto done
		}
		e = &dom.MouseEvent{
			Event: dom.Event{
				Type: "click",
			},
		}
		el.DispatchEvent(e)
		if consumed = e.Consumed; consumed {
			goto done
		}
		e2 = &dom.Event{
			Type: "mouseup",
		}
		el.DispatchEvent(e2)
		if consumed = e2.Consumed; consumed {
			goto done
		}
		e3 = &dom.Event{
			Type: "focus",
		}
		el.DispatchEvent(e3)
		if consumed = e3.Consumed; consumed {
			goto done
		}
		if h = el.OuterHTML(); len(h) > 20 {
			h = h[:20] + "..."
		}
	done:
		consumedCh <- consumed
		errCh <- nil
	})
	if err := <-errCh; err != nil {
		return "", false, err
	}

	if <-consumedCh {
		log.Printf("event consumed")
		newHTML, ok, err = r.TrackChanges()
	} else {
		log.Printf("event not consumed")
	}

	return
}

// Put change into html (e.g. from input field mutation)
func (r *Runner) PutAttr(selector, attr, val string) (ok bool, err error) {
	res, err := r.Exec(`
		var sel = '`+selector+`';
		var el = document.querySelector(sel);
		el.attr('`+attr+`', '`+val+`');
		!!el;
	`, false)

	ok = res == "true"

	return
}

func (r *Runner) TrackChanges() (html string, changed bool, err error) {
outer:
	for {
		// TODO: either add other change types like ajax begin/end or
		// just have one channel for all events worth waiting for.
		select {
		case m := <-dom.Mutations():
			changed = true
			if strings.ToLower(m.Tag) == "script" {
				s := ""
				src, ok := m.Node["src"]
				if ok {
					ch := make(chan string)
					log.Printf("<script> GET %v", src)
					r.xhr("GET", src, make(map[string]string), "", func(data, err string) {
						if err != "" {
							log.Printf("xhr %v: %v", src, err)
							log.Printf("data: %v", data)
						}
						ch <- data
					})
					s = <-ch
				} else if inner, ok := m.Node["innerHTML"]; ok {
					s = inner
				}
				if strings.TrimSpace(s) != "" {
					sp := s
					if len(sp) > 20 {
						sp = sp[:20] + "..."
					}
					if _, err := r.Exec56(s, false); err != nil {
						log.Printf("exec %v: %v", src, err)
					}
				}
			} else {
				changed = true
			}
		case <-time.After(time.Second):
			break outer
		}
	}

	if changed {
		html = r.doc.Element().OuterHTML()
	}
	r.outputHtml = html
	return
}

func (r *Runner) xhr(method, uri string, h map[string]string, data string, cb func(data string, err string)) {
	uri = strings.TrimPrefix(uri, ".")
	if !strings.HasPrefix(uri, "http") && !strings.HasPrefix(uri, "/") {
		// TODO: use instead origin/url prefix
		uri = "/" + uri
	}
	req, err := http.NewRequest(method /*u.String()*/, uri, strings.NewReader(data))
	if err != nil {
		err = fmt.Errorf("new http req: %v", err)
		cb("", err.Error())
		return
	}
	for k, v := range h {
		req.Header.Add(k, v)
	}
	go func() {
		resp, err := r.xhrq(req)
		if err != nil {
			err = fmt.Errorf("xhrq: %v", err)
			cb("", err.Error())
			return
		}
		//defer resp.Body.Close()
		bs, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("read all: %v", err)
			cb("", err.Error())
			return
		}
		r.loop.RunOnLoop(func(*js.Runtime) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("recovered in xhr: %v", r)
				}
			}()
			cb(string(bs), "")
		})
	}()
}

func (r *Runner) docPath(path string) (dp string, err error) {
	if !strings.HasPrefix(path, "/0") {
		return "", fmt.Errorf("malformed path %v", path)
	}
	path = strings.TrimPrefix(path, "/0")
	q := "document.getElementsByTagName('body')[0]"
	for _, el := range strings.Split(path, "/") {
		if el == "" {
			continue
		}
		i, err := strconv.Atoi(el)
		if err == nil {
			q += fmt.Sprintf(".children[%d]", i)
		} else {
			q += fmt.Sprintf(".%v", el)
		}
	}
	return q, nil
}

func (r *Runner) Retrieve(path string) string {
	tmp := strings.Split(path, "/")
	path = strings.Join(tmp[:len(tmp)-1], "/")
	k := tmp[len(tmp)-1]
	if !strings.HasPrefix(path, "/0") {
		log.Printf("malformed path %v", path)
		return ""
	}
	dp, err := r.docPath(path)
	if err != nil {
		log.Printf("doc path %v: %v", path, err)
		return ""
	}
	res, err := r.Exec(dp+"."+k, false)
	if err != nil {
		log.Printf("exec %v: %v", dp+"."+k, err)
		return ""
	}
	return res
}

func (r *Runner) Write(path, val string) (err error) {
	tmp := strings.Split(path, "/")
	path = strings.Join(tmp[:len(tmp)-1], "/")
	k := tmp[len(tmp)-1]
	if !strings.HasPrefix(path, "/0") {
		return fmt.Errorf("malformed path %v", path)
	}
	dp, err := r.docPath(path)
	if err != nil {
		return fmt.Errorf("doc path %v: %v", path, err)
	}
	_, err = r.Exec(dp+`.`+k+` = '`+val+`'`, false)
	return
}

func (r *Runner) List(path string) (l []string) {
	l = make([]string, 0, 10)
	dp, err := r.docPath(path)
	if err != nil {
		log.Printf("doc path %v: %v", path, err)
		return
	}
	q := `
	(function() {
		let items = [];
		let q = ` + dp + `;
		let ks = getProperties(q);
		let ms = getMethods(q);
		let i;
		if (q.children) {
			for (i = 0; i < q.children.length; i++) {
				items.push(i);
			}
		}
		for (i = 0; i < ks.length; i++) {
			items.push(ks[i]);
		}
		for (i = 0; i < ms.length; i++) {
			items.push(ms[i] + '()');
		}
		return items;
	})()
	`
	res, err := r.Exec(q, false)
	if err != nil {
		log.Printf("exec %v: %v", q, err)
		return
	}
	return strings.Split(res, ",")
}

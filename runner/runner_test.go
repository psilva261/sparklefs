package runner

import (
	"fmt"
	"github.com/psilva261/sparklefs/dom"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const simpleHTML = `
<html>
<body>
<h1 id="title">Hello</h1>
</body>
</html>
`

func TestSimple(t *testing.T) {
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	s := `
	var state = 'empty';
	var a = 1;
	b = 2;
	`
	_, err := d.Exec(s, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	s2 := `
	(function() {
		if (state !== 'empty') throw new Exception(state);

		state = a + b;
	})()
	var a = 1;
	b = 2;
	`
	_, err = d.Exec(s2, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	d.Stop()
}

func TestGlobals(t *testing.T) {
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
}

func TestTrackChanges(t *testing.T) {
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	_, err := d.Exec(``, true)
	if err != nil {
		t.Fatalf(err.Error())
	}
	// 0th time: init
	if _, _, err = d.TrackChanges(); err != nil {
		t.Fatalf(err.Error())
	}
	// 1st time: no change
	html, changed, err := d.TrackChanges()
	if err != nil {
		t.Fatalf(err.Error())
	}
	if changed == true {
		t.Fatal()
	}
	// 2nd time: no change
	html, changed, err = d.TrackChanges()
	if err != nil {
		t.Fatalf(err.Error())
	}
	if changed == true {
		t.Fatal()
	}
	_, err = d.Exec("document.getElementById('title').innerHTML='new title'; true;", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	// 3rd time: yes change
	html, changed, err = d.TrackChanges()
	if err != nil {
		t.Fatalf(err.Error())
	}
	if changed == false {
		t.Fatalf("%v", changed)
	}
	if html == "" {
		t.Fatalf(html)
	}
	if !strings.Contains(html, "new title") {
		t.Fatalf(html)
	}
	d.Stop()
}

/*func TestWindowEqualsGlobal(t *testing.T) {
	const h = `
	<html>
	<body>
	<script>
	a = 2;
	window.b = 5;
	</script>
	<script>
	console.log('window.a=', window.a);
	console.log('wot');
	console.log('window.b=', window.b);
	console.log('wit');
	window.a++;
	b++;
	</script>
	</body>
	</html>
	`
	d := New(h)
	d.Start()
	err := d.ExecInlinedScripts()
	if err != nil {
		t.Fatalf(err.Error())
	}
	res, err := d.Export("window.a")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !strings.Contains(res, "3") {
		t.Fatalf(res)
	}
	res, err = d.Export("window.b")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !strings.Contains(res, "6") {
		t.Fatalf(res)
	}
	d.Stop()
}*/

func TestES6(t *testing.T) {
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	script := `
	var foo = function(data={}) {}
	var h = {
		a: 1,
		b: 11
	};
	var {a, b} = h;
	`
	_, err := d.Exec6(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("a+b", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "12" {
		t.Fatal()
	}
	d.Stop()
}

func TestWindowParent(t *testing.T) {
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	script := `
	console.log('Hello!!')
	`
	_, err := d.Exec(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("window === window.parent", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "true" {
		t.Fatal()
	}
	d.Stop()
}

func TestReferrer(t *testing.T) {
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	script := `
	document.referrer;
	`
	res, err := d.Exec(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if res != "https://example.com" {
		t.Fatal()
	}
	d.Stop()
}

func handler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "<html><body>Hello World!</body></html>")
}

func xhr(req *http.Request) (resp *http.Response, err error) {
	fmt.Printf("xhr: %+v\n", *req)
	w := httptest.NewRecorder()
	handler(w, req)
	resp = w.Result()
	return resp, nil
}

func TestXMLHttpRequest(t *testing.T) {
	d := New("https://example.com", simpleHTML, xhr, nil, nil)
	d.Start()
	script := `
		var oReq = new XMLHttpRequest();
		var loaded = false;
		oReq.addEventListener("load", function() {
			loaded = true;
		});
		oReq.open("GET", "http://www.example.org/example.txt");
		oReq.send();
	`
	_, err := d.Exec(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	<-time.After(time.Second)
	res, err := d.Exec("oReq.responseText;", false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if !strings.Contains(res, "<html") {
		t.Fatal()
	}
	d.Stop()
}

func TestFetch(t *testing.T) {
	d := New("https://example.com", simpleHTML, xhr, nil, nil)
	d.Start()
	script := `
		var oHeaders = new Headers();
		var oReq = new Request('http://www.example.org/example.txt', {
			method: 'GET',
			headers: oHeaders
		});
		var loaded = false;
		fetch(oReq)
		  .then(resp => {
		  	return 123;
		  })
		  .then(magic => {
		  	loaded = (magic == 123);
		  });
	`
	_, err := d.Exec(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	<-time.After(time.Second)
	res, err := d.Exec("loaded;", false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if !strings.Contains(res, "true") {
		t.Fatal()
	}
	d.Stop()
}

func TestJQueryAjax(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := New("https://example.com", simpleHTML, xhr, nil, nil)
	d.Start()
	script := `
	var res;
	$.ajax({
		url: '/',
		success: function() {
			res = 'success';
		},
		error: function() {
			res = 'err';
		}
	});
	`
	_, err = d.Exec(string(buf)+";"+script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err = d.CloseDoc(); err != nil {
		t.Fatalf("%v", err)
	}
	<-time.After(time.Second)
	res, err := d.Exec("res;", false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if res != "success" {
		t.Fatalf(res)
	}
	d.Stop()
}

func TestJQueryAjax182(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-1.8.2.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := New("https://example.com", simpleHTML, xhr, nil, nil)
	d.Start()
	script := `
	var res;
	$.ajax({
		url: '/',
		success: function() {
			res = 'success';
		},
		error: function() {
			res = 'err';
		}
	});
	`
	_, err = d.Exec(string(buf)+";"+script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err = d.CloseDoc(); err != nil {
		t.Fatalf("%v", err)
	}
	<-time.After(5 * time.Second)
	res, err := d.Exec("res;", false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if res != "success" {
		t.Fatalf(res)
	}
	d.Stop()
}

func TestNoJsCompatComment(t *testing.T) {
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	script := `

<!-- This is an actual comment

	''.replace(/^\s*<!--/g, '');
	const a = 1;
	a + 7;
-->
	`
	res, err := d.Exec(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	if res != "8" {
		t.Fatal()
	}
	d.Stop()
}

func TestJQuery(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	script := `
	$(document).ready(function() {
		undefinedExpr
	});
	setTimeout(function() {
		console.log("ok");
	}, 1000);
	var a = 1;
	`
	_, err = d.Exec(string(buf)+";"+script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("a+1", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "2" {
		t.Fatal()
	}
	d.Stop()
}

func TestJQueryCss(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	h := `
	<html>
	<body>
	<h1 id="title" style="display: inline-block;">Hello</h1>
	</body>
	</html>
	`
	q := func(sel, prop string) (val string, err error) {
		if sel != "/0/0" {
			panic(sel)
		}
		if prop != "display" {
			panic(prop)
		}
		return "inline-block", nil
	}
	d := New("https://example.com", h, nil, nil, q)
	d.Start()
	_, err = d.Exec(string(buf), true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("$('h1').css('display')", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "inline-block" {
		t.Fatal()
	}
	d.Stop()
}

func TestJqueryUI(t *testing.T) {
	buf, err := ioutil.ReadFile("jqueryui/tabs.html")
	if err != nil {
		t.Fatalf("%v", err)
	}
	q := func(sel, prop string) (val string, err error) {
		t.Logf("query(%v, %v)", sel, prop)
		return "", nil
	}
	d := New("https://example.com", string(buf), xhr, nil, q)
	d.Start()
	for i, fn := range []string{"jquery-3.6.0.js", "jquery-ui.js", "tabs.js"} {
		buf, err := ioutil.ReadFile("jqueryui/" + fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
		_, err = d.Exec(string(buf), i == 0)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	d.CloseDoc()
	if _, _, err = d.TrackChanges(); err != nil {
		t.Fatalf(err.Error())
	}
	_, changed, err := d.TriggerClick(`#ui-id-3`)
	if err != nil {
		t.Logf(d.doc.Element().Get("innerHTML").String())
		t.Fatalf(err.Error())
	}
	if !changed {
		t.Logf(d.doc.Element().Get("innerHTML").String())
		t.Fail()
	}
	t.Logf(d.doc.Element().Get("innerHTML").String())
	n := d.doc.Element().QuerySelector("ul li:nth-child(3)").Node()
	if a := attr(*n, "aria-selected"); a != "true" {
		t.Fatalf(a)
	}
	d.Stop()
}

func TestRun(t *testing.T) {
	jQuery, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	SCRIPT := string(jQuery) + `
	setTimeout(function() {
		var h = document.querySelector('html');
    	console.log(h.innerHTML);
	}, 1000);
	Object.assign(this, window);
	`
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	_, err = d.Exec(SCRIPT, true)
	if err != nil {
		t.Fatalf(err.Error())
	}

	res, err := d.Exec("$('h1').html()", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != "Hello" {
		t.Fatalf(res)
	}
	d.Stop()
}

func TestTriggerClick(t *testing.T) {
	jQuery, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	SCRIPT := string(jQuery) + `
	var clicked = false;
    $(document).ready(function() {
    	$('h1').click(function() {
    		clicked = true;
    	});
    });
	`
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	_, err = d.Exec(SCRIPT, true)
	if err != nil {
		t.Fatalf(err.Error())
	}
	d.CloseDoc()

	res, err := d.Exec("$('h1').html()", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != "Hello" {
		t.Fatalf(res)
	}

	if _, _, err = d.TrackChanges(); err != nil {
		t.Fatalf(err.Error())
	}
	_, changed, err := d.TriggerClick("h1")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if changed {
		t.Fatal()
	}
	res, err = d.Exec("clicked", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != "true" {
		t.Fatalf(res)
	}
	d.Stop()
}

func TestTriggerClick2(t *testing.T) {
	h := `
<html>
<body>
<h1 id="title">Hello</h1>
</body>
</html>
	`
	jQuery, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	SCRIPT := string(jQuery) + `
	var clicked = false;
            var h1 = $('h1');
            h1.click(function() {
                clicked = true;
            });
	`
	d := New("https://example.com", h, nil, nil, nil)
	d.Start()
	_, err = d.Exec(SCRIPT, true)
	if err != nil {
		t.Fatalf(err.Error())
	}
	d.CloseDoc()

	res, err := d.Exec("$('h1').html()", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != "Hello" {
		t.Fatalf(res)
	}

	if _, _, err = d.TrackChanges(); err != nil {
		t.Fatalf(err.Error())
	}
	_, changed, err := d.TriggerClick("h1")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if changed {
		t.Fatal()
	}
	res, err = d.Exec("clicked", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if res != "true" {
		t.Fatalf(res)
	}
	d.Stop()
}

func TestTriggerClickSubmit(t *testing.T) {
	for _, sel := range []string{"#btn" /*, "#submit"*/} {
		jQuery, err := ioutil.ReadFile("jquery-3.5.1.js")
		if err != nil {
			t.Fatalf("%v", err)
		}
		h := `
		<html>
		<body>
		<h1 id="title" style="display: inline-block;">Hello</h1>
		<form id="the-form">
			<input type="text" id="info">
			<input type="submit" id="submit">Submit</button>
			<button type="button" id="btn">Submit</button>
		</form>
		</body>
		</html>
		`
		SCRIPT := string(jQuery) + `
		var clicked = false;
		const form = document.getElementById('the-form');
		form.onsubmit = function(event) {
			clicked = true;
			event.preventDefault();
		};
		`
		d := New("https://example.com", h, nil, nil, nil)
		d.Start()
		_, err = d.Exec(SCRIPT, true)
		if err != nil {
			t.Fatalf(err.Error())
		}
		d.CloseDoc()

		res, err := d.Exec("$('button').html()", false)
		if err != nil {
			t.Fatalf(err.Error())
		}
		if res != "Submit" {
			t.Fatalf(res)
		}

		if _, _, err = d.TrackChanges(); err != nil {
			t.Fatalf(err.Error())
		}
		_, changed, err := d.TriggerClick(sel)
		if err != nil {
			t.Fatalf(err.Error())
		}
		if changed {
			t.Fatal()
		}
		res, err = d.Exec("clicked", false)
		if err != nil {
			t.Fatalf(err.Error())
		}
		if res != "true" {
			t.Fatalf(res)
		}
		d.Stop()
	}
}

func TestDomChanged(t *testing.T) {
	jQuery, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	SCRIPT := string(jQuery) + `
	setTimeout(function() {
		var h = document.querySelector('html');
	}, 1000);
	Object.assign(this, window);
	`
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	_, err = d.Exec(SCRIPT, true)
	if err != nil {
		t.Fatalf(err.Error())
	}

	res, err := d.Exec("$('h1').html()", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	_ = res
	res, err = d.Exec("$('h1').html('minor updates :-)'); $('h1').html();", false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("new res=%v", res)
	d.Stop()
}

func TestMutationEvents(t *testing.T) {
	buf, err := ioutil.ReadFile("jquery-3.5.1.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	q := func(sel, prop string) (val string, err error) {
		if sel != "/0/0" {
			t.Fatalf(sel)
		}
		if prop != "display" {
			panic(prop)
		}
		return "inline-block", nil
	}
	d := New("https://example.com", simpleHTML, nil, nil, q)
	d.Start()
	script := `
	$('h1').hide();
	$('h1').show();
	`
	_, err = d.Exec(string(buf)+";"+script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err = d.CloseDoc(); err != nil {
		t.Fatalf("%v", err)
	}
	res, err := d.Exec("$('h1').attr('style')", false)
	t.Logf("res=%v", res)
	if err != nil {
		t.Fatalf("%v", err)
	}
	d.Stop()
}

func TestMutationEventsAddScript(t *testing.T) {
	q := func(string, string) (string, error) {
		return "", nil
	}
	d := New("https://example.com", simpleHTML, nil, nil, q)
	d.Start()
	script := `
		var div = document.createElement('div');
		div.id = 'foo';
		div.innerHTML = 'bar';
		document.body.appendChild(div);
		document.getElementById('foo').id = 'baz';
	`
	_, err := d.Exec(script, true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err = d.CloseDoc(); err != nil {
		t.Fatalf("%v", err)
	}
	n := 0
	foundIns := false
	foundAtr := false
	outer:
	for {
		select {
		case m := <-dom.Mutations():
			t.Logf("m=%+v",m)
			n++
			if m.Type == dom.Insert && m.Tag == "div" && m.Node["id"] == "foo" &&
				m.Node["innerHTML"] == "bar" /*&& m.Path == "/0"*/ {
				foundIns = true
			}
			if m.Type == dom.ChAttr && /*m.Path == "/0/1" &&*/ m.Node["id"] == "baz" {
				foundAtr = true
			}
		case <-time.After(time.Second):
			break outer
		}
	}
	d.Stop()
	if n != 3 || !foundIns || !foundAtr {
		t.Fatalf("%v, %v, %v", n, foundIns, foundAtr)
	}
}

func TestBtoa(t *testing.T) {
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	res, err := d.Exec("btoa('a')", true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if res != "YQ==" {
		t.Fatalf("%x", res)
	}
	d.Stop()
}

func TestList(t *testing.T) {
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	_, err := d.Exec("true", true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	l := d.List("/0")
	t.Logf("%v", l)
	foundH1 := false
	for _, t := range l {
		if t == "0" {
			foundH1 = true
		}
	}
	if !foundH1 {
		t.Fail()
	}
	res := d.Retrieve("/0/tagName")
	if res != "BODY" {
		t.Fatalf("%v", res)
	}
	l = d.List("/0/0")
	res = d.Retrieve("/0/1/tagName")
	if res != "H1" {
		t.Fatalf("%v", res)
	}
	t.Logf("%v", l)
	d.Stop()
}

func TestWrite(t *testing.T) {
	d := New("https://example.com", simpleHTML, nil, nil, nil)
	d.Start()
	_, err := d.Exec("true", true)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err = d.Write("/0/1/innerHTML", "Hello2"); err != nil {
		t.Fatalf("%v", err)
	}
	res := d.Retrieve("/0/1/outerHTML")
	if res != `<h1 id="title">Hello2</h1>` {
		t.Fatalf("%v", res)
	}
	d.Stop()
}

func attr(n html.Node, key string) (val string) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return
}

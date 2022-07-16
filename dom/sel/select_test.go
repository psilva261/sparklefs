package sel

import (
	"bytes"
	"github.com/psilva261/sparklefs/logger"
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func init() {
	log.Debug = true
}

func TestSelect(t *testing.T) {
	htm := `
<html>
<body>
<p id="a"></p>
</body>
</html>
	`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select("#a", body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	t.Logf("res=%v %T", e, e)
	if e.Data != "p" {
		t.Fail()
	}
}

func TestSelect1(t *testing.T) {
	htm := `
<html>
<body id="b">
<p id="a"></p>
</body>
</html>
	`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select("#b #a", body, false, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	t.Logf("res=%v %T", e, e)
	if e.Data != "p" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect2(t *testing.T) {
	htm := `
<html>
<body id="b">
<p class="c"></p>
</body>
</html>
	`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select("#b .c", body, false, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	t.Logf("res=%v %T", e, e)
	if e.Data != "p" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect3(t *testing.T) {
	htm := `
<html>
<body id="b">
<p class="c"></p>
</body>
</html>
	`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select("p", body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	t.Logf("res=%v %T", e, e)
	if e.Data != "p" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect4(t *testing.T) {
	htm := `
<html>
<body id="b">
<p class="c"></p>
<input type="submit">
</body>
</html>
	`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select("p.c", body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	t.Logf("res=%v %T", e, e)
	if e.Data != "p" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect5(t *testing.T) {
	htm := `<html><body><p></p></body></html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select("body > p", body, false, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	t.Logf("res=%v %T", e, e)
	if e.Data != "p" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect6(t *testing.T) {
	htm := `<html><body><p></p></body></html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select(":scope > p", body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	t.Logf("res=%v %T", e, e)
	if e.Data != "p" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect7(t *testing.T) {
	htm := `<html><body><p id=a1-b__c></p></body></html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select(":scope > #a1-b__c", body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	t.Logf("res=%v %T", e, e)
	if e.Data != "p" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect8(t *testing.T) {
	htm := `<html><body><input type="submit"></body></html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select(`input[type="submit"]`, body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	t.Logf("res=%v %T", e, e)
	if e.Data != "input" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect9(t *testing.T) {
	htm := `<html><body><input type="submit"><p></p><br></body></html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select(`*`, body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 3 {
		t.Fail()
	}
}

func TestSelect30(t *testing.T) {
	htm := `<html><body>
		<li id="l0">
			<a></a>
		</li>
		<li id="l1">
			<a href="#a"></a>
		</li>
	</body></html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select(`:scope > li:has(a[href])`, body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	if e.Data != "li" || attr(*e, "id") != "l1" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect31(t *testing.T) {
	htm := `<html><body>
		<div>
			<p id="p0">
				<a></a>
			</p>
			<br id="b0">
			<p id="p1">
				<a href="#a"></a>
			</p>
		</div>
	</body></html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select(`div p:nth-child(3)`, body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	if e.Data != "p" || attr(*e, "id") != "p1" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect32(t *testing.T) {
	htm := `<html><body>
		<div>
			<p id="p0">
				<a></a>
			</p>
			<br id="b0">
			<p id="p1">
				<a href="#a"></a>
			</p>
		</div>
	</body></html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select(`div p:first-child`, body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	if e.Data != "p" || attr(*e, "id") != "p0" {
		t.Fatalf("%v", render(e))
	}
}

func TestSelect33(t *testing.T) {
	htm := `<html><body>
		<div>
			<h1>info</h1>
			<p>step 1</p>
			<p>step 2</p>
		</div>
	</body></html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select(`div > :not(p)`, body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if l := len(es); l != 1 {
		for _, e := range es {
			t.Logf("data=%v %v", e.Data, render(e))
		}
		t.Fatalf("l=%v", l)
	}
	e := es[0]
	if e.Data != "h1" {
		t.Logf("data=%v %v", e.Data, render(e))
		t.Fatalf("parent data=%v %v", e.Parent.Data, render(e.Parent))
	}
}

func TestSelect34(t *testing.T) {
	htm := `<html>
		<body>
			<p>
				<b>bold stuff</b>
				<i>italic stuff</i>
				<a>link</a>
			</p>
		</body>
	</html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	es, err := Select(`HTML > :nth-child(2) > :nth-child(1) > :nth-child(2)`, d, false, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if l := len(es); l != 1 {
		for _, e := range es {
			t.Logf("data=%v %v", e.Data, render(e))
		}
		t.Fatalf("l=%v", l)
	}
	e := es[0]
	if e.Data != "i" {
		t.Logf("data=%v %v", e.Data, render(e))
		t.Fatalf("parent data=%v %v", e.Parent.Data, render(e.Parent))
	}
}

func TestSelect35(t *testing.T) {
	htm := `<html>
	    <body>
	        <header>
	            <div>
	            </div>
	            <div>
	                <ul>
	                    <li>
	                        <a>1st</a>
	                    </li>
	                    <li>
	                        <a>2nd</a>
	                    </li>
	                    <li>
	                        <a>3rd</a>
	                    </li>
	                    <li>
	                        <a>4th</a>
	                    </li>
	                </ul>
	                <a>≡</a>
	            </div>
	            <div>
	                <ul>
	                </ul>
	                <a>x</a>
	            </div>
	        </header>
	    </body>
	</html>`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	es, err := Select(`header:nth-child(1) > div:nth-child(3) > a:nth-child(2)`, d, false, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if l := len(es); l != 1 {
		for _, e := range es {
			t.Logf("data=%v %v", e.Data, render(e))
		}
		t.Fatalf("l=%v", l)
	}
	e := es[0]
	if e.Data != "a" {
		t.Logf("data=%v %v", e.Data, render(e))
		t.Fatalf("parent data=%v %v", e.Parent.Data, render(e.Parent))
	}
}

func TestSelect36(t *testing.T) {
	htm := `
<html>
<body>
<p id="a.b"></p>
</body>
</html>
	`
	d, err := html.Parse(strings.NewReader(htm))
	if err != nil {
		t.Fatalf("%v", err)
	}
	body := grep(d, "body")
	es, err := Select(`#a\\.b`, body, true, false)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(es) != 1 {
		t.Fail()
	}
	e := es[0]
	t.Logf("res=%v %T", e, e)
	if e.Data != "p" {
		t.Fail()
	}
}

func TestSplitBlock(t *testing.T) {
	tt := map[string][]string{
		"a":                  []string{"a"},
		".c":                 []string{".c"},
		".foo":               []string{".foo"},
		"a.c":                []string{"a", ".c"},
		"a.c#d":              []string{"a", ".c", "#d"},
		`a\\.c#d`:            []string{`a\\.c`, "#d"},
		"[selected]":         []string{"[selected]"},
		"[type=submit]":      []string{"[type=submit]"},
		"input[type=submit]": []string{"input", "[type=submit]"},
		"li:has(a[href])":    []string{"li", ":has(a[href])"},
	}
	for sb, exp := range tt {
		t.Logf("test %v", sb)
		act, err := splitBlock(sb)
		if err != nil {
			t.Fatalf("%v", err)
		}
		t.Logf("act=%+v", act)
		if len(act) != len(exp) {
			t.Fatalf("%+v", len(act))
		}
		for i, x := range act {
			if x != exp[i] {
				t.Fatalf("%v", sb)
			}
		}
	}
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

func render(n *html.Node) string {
	buf := bytes.NewBufferString("")
	if err := html.Render(buf, n); err != nil {
		log.Errorf("render: %v", err)
		return ""
	}
	return buf.String()
}

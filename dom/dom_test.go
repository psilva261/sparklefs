package dom

import (
	"fmt"
	"github.com/psilva261/sparkle/eventloop"
	"github.com/psilva261/sparkle/js"
	"github.com/psilva261/sparklefs/logger"
	"os"
	"strings"
	"testing"
	"time"
)

func init() {
	log.Debug = true
}

const htm = `
<!DOCTYPE html>
<html>
  <head>
  <title>Demo</title>
  </head>
<body>

<h2>Finding HTML Elements Using document.title</h2>

<p id="demo" class="bar" style="font-weight: bold;">the paragraph.</p>

<script>
document.getElementById("demo").innerHTML =
"The title of this document is: " + document.title;
</script>

</body>
</html>
`

func TestNodeName(t *testing.T) {
	vm := js.New()
	_, err := Init(vm, htm, "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := vm.RunString(`
document.nodeName.toLowerCase();
	`)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if v := res.Export(); v != "#document" {
		t.Fatalf("%v", v)
	}
}

func TestIsNode(t *testing.T) {
	vm := js.New()
	_, err := Init(vm, htm, "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := vm.RunString(`
document.body instanceof window.Node;
	`)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if v := res.Export(); v != true {
		t.Fatalf("%v", v)
	}
}

func TestVars(t *testing.T) {
	vm := js.New()
	j := `
document.getElementById('demo').info = 123;
window.document.info = 456;
df = document.createDocumentFragment('p');
df.info = 789;
	`
	d, err := Init(vm, htm, j)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := vm.RunString(`
document.getElementById('demo').info === 123 && document.info === 456;
	`)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if v := res.Export(); v != true {
		t.Fatalf("%v", v)
	}
	if v := d.vars["info"].Export(); v != int64(456) {
		t.Fatalf("%v %T", v, v)
	}
	if l := len(dfObjRefs); l != 1 {
		t.Fatalf("%v", l)
	}
	var df *DocumentFragment
	for f := range dfObjRefs {
		df = f
	}
	if df == nil {
		t.Fatalf("%v", df)
	}
	if v := df.vars["info"].Export(); v != int64(789) {
		t.Fail()
	}
}

// https://www.w3schools.com/js/js_dom_examples.asp
func TestInit(t *testing.T) {
	j := `
document.getElementById("demo").innerHTML =
"The title of this document is: " + document.title;
	`
	vm := js.New()
	d, err := Init(vm, htm, j)
	if err != nil {
		t.Fatalf("%v", err)
	}
	res, err := vm.RunString(`document.getElementById("demo").innerHTML`)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("res=%v", res)
	h := render(d.doc)
	t.Logf("h=%v", h)
	if !strings.Contains(h, `<p id="demo" class="bar" style="font-weight: bold;">The title of this document is: Demo</p>`) {
		t.Fail()
	}
}

func TestJQuery(t *testing.T) {
	bs, err := os.ReadFile("jquery-1.8.2.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	j := `
window = {};
window = this;
global = {};
global.document = document;
` + string(bs) + `
document.getElementById("demo").innerHTML =
"The title of this document is: " + document.title;
	`
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	loop.RunOnLoop(func(vm *js.Runtime) {
		d, err := Init(vm, htm, j)
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		res, err := vm.RunString(`document.getElementById("demo").innerHTML`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res=%v", res)
		h := render(d.doc)
		t.Logf("h=%v", h)
		if !strings.Contains(h, `<p id="demo" class="bar" style="font-weight: bold;">The title of this document is: Demo</p>`) {
			errCh <- fmt.Errorf("fail")
			return
		}
		res, err = vm.RunString(`
			var isReady = false;
			$(document).ready(function() {
				isReady = true;
			});
			isReady;
		`)
		if err != nil {
			errCh <- err
			return
		}
		if v := fmt.Sprintf("%v", res); v == "true" {
			errCh <- fmt.Errorf("%v", v)
			return
		}
		if err = d.Close(); err != nil {
			errCh <- err
			return
		}
		res, err = vm.RunString("isReady")
		if err != nil {
			errCh <- err
			return
		}
		if v := fmt.Sprintf("%v", res); v != "true" {
			errCh <- fmt.Errorf("%v", v)
			return
		}
		res, err = vm.RunString(`$("#demo").html()`)
		if v := fmt.Sprintf("%v", res); v != "The title of this document is: Demo" {
			errCh <- fmt.Errorf("%v", v)
			return
		}
		fmt.Printf(`noow $("p").html():\n`)
		res, err = vm.RunString(`$("p").html()`)
		if v := fmt.Sprintf("%v", res); v != "The title of this document is: Demo" {
			errCh <- fmt.Errorf("%v", v)
			return
		}
		res, err = vm.RunString(`$("p").hasClass('bar')`)
		if v := fmt.Sprintf("%v", res); v != "true" {
			errCh <- fmt.Errorf("%v", v)
			return
		}
		res, err = vm.RunString(`$("p").toggleClass('bar'); $("p").hasClass('bar')`)
		if v := fmt.Sprintf("%v", res); v != "false" {
			errCh <- fmt.Errorf("hasClass: %v", v)
			return
		}
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
}

func TestJQueryPrependTo(t *testing.T) {
	bs, err := os.ReadFile("jqueryui/jquery-3.6.0.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	j := string(bs)
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	loop.RunOnLoop(func(vm *js.Runtime) {
		d, err := Init(vm, htm, j)
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		if err = d.Close(); err != nil {
			errCh <- err
			return
		}
		res, err := vm.RunString(`
var item = $("<h3>1st</h3><h3>2nd</h3>");
item
		`)
		if err != nil {
			errCh <- err
			return
		}
		res, err = vm.RunString(`
item.html();
		`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res=%v", res)
		if res.String() != "1st" {
			errCh <- fmt.Errorf("expected '1st' like in safari: %v", res)
			return
		}
		res, err = vm.RunString(`
icon = $( "<span>" );
icon.prependTo( item );
item.html();
		`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res'=%v", res)
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
}

func TestDOMChange(t *testing.T) {
	j := `
window = {};
window = this;
global = {};
global.document = document;
var p = document.createElement('p');
p.innerHTML = 'yolo';
document.body.appendChild(p);
	`
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	loop.RunOnLoop(func(vm *js.Runtime) {
		document, err := Init(vm, htm, j)
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		if err = document.Close(); err != nil {
			errCh <- err
			return
		}
		res, err := vm.RunString(`document.body.innerHTML`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res=%v", res)
		if !strings.Contains(res.String(), "<p>yolo</p>") {
			errCh <- err
			return
		}
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
}

func TestJQueryDOMChange(t *testing.T) {
	bs, err := os.ReadFile("jquery-1.8.2.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	j := `
window = {};
window = this;
global = {};
global.document = document;
` + string(bs) + `
$('body').append('<p>Test</p>');
	`
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	loop.RunOnLoop(func(vm *js.Runtime) {
		document, err := Init(vm, htm, j)
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		if err = document.Close(); err != nil {
			errCh <- err
			return
		}
		t.Logf("now check innerHTML:")
		res, err := vm.RunString(`$('body').html()`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res=%v", res)
		if !strings.Contains(res.String(), "<p>Test</p>") {
			errCh <- err
			return
		}
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
}

func TestJQueryClick(t *testing.T) {
	bs, err := os.ReadFile("jquery-1.8.2.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	j := `
window = {};
window = this;
global = {};
global.document = document;
` + string(bs) + `
var clicked = false;
$('h2').click(function() {
	clicked = true;
});

	`
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	loop.RunOnLoop(func(vm *js.Runtime) {
		document, err := Init(vm, htm, j)
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		if err = document.Close(); err != nil {
			errCh <- err
			return
		}
		t.Logf("now check clicked:")
		_, err = vm.RunString(`$('h2').trigger('click');`)
		if err != nil {
			errCh <- err
			return
		}
		res, err := vm.RunString(`clicked`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res=%v", res)
		if !strings.Contains(res.String(), "true") {
			errCh <- err
			return
		}
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
}

func TestJQueryClick2(t *testing.T) {
	ResetCalls()
	defer PrintCalls()
	htm := `
<!DOCTYPE html>
<html>
  <head>
  <title>Demo</title>
  </head>
<body>
	<div id=d1 class=av>
		<h2 id=b1 class=b>b1</h2>
	</div>
	<div id=d2 class=a>
		<h2 id=b2 class=b>b2</h2>
	</div>
</body>
</html>
`
	bs, err := os.ReadFile("jquery-1.8.2.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	j := string(bs) + `
var clicked = 0;
var cont = 0;
function reg(el) {
	$('.b', el).click(function() {
		clicked++;
    if ($(this).closest('.a, .av')[0] != el) {
      // Only trigger the closest toggle header.
      return;
    }
    cont++;

    if ($(el).is('.a')) {
      $(el)
        .addClass('av')
        .removeClass('a');
    } else {
      $(el)
        .addClass('a')
        .removeClass('av');
    }
	});
}

$('.a').each(function(i, el) {
  reg(el);
});
$('.av').each(function(i, el) {
  reg(el);
});
	`
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	loop.RunOnLoop(func(vm *js.Runtime) {
		document, err := Init(vm, htm, j)
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		if err = document.Close(); err != nil {
			errCh <- err
			return
		}
		_, err = vm.RunString(`$('#b1').click();`)
		if err != nil {
			errCh <- fmt.Errorf("click: %w", err)
			return
		}
		res, err := vm.RunString(`clicked`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res=%v", res)
		if !strings.Contains(res.String(), "1") {
			errCh <- err
			return
		}
		res, err = vm.RunString(`cont`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res=%v", res)
		if !strings.Contains(res.String(), "1") {
			errCh <- err
			return
		}

		t.Logf("h=%v", document.Element().OuterHTML())
		res, err = vm.RunString(`document.getElementById('d1').className`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res'=%v", res)
		if res.String() != "a" {
			errCh <- err
			return
		}

		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
}

func TestJQueryUITabs(t *testing.T) {
	ResetCalls()
	defer PrintCalls()
	files := make(map[string][]byte)
	var err error
	for _, fn := range []string{"jquery-3.6.0.js", "jquery-ui.js", "tabs.html", "jquery-ui.css", "style.css"} {
		files[fn], err = os.ReadFile("jqueryui/" + fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	j := string(files["jquery-3.6.0.js"]) + string(files["jquery-ui.js"]) + `
$( function() {
	$( "#tabs" ).tabs();
} );
	`
	htm := string(files["tabs.html"])
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	var d *Document
	loop.RunOnLoop(func(vm *js.Runtime) {
		var err error
		d, err = Init(vm, htm, "")
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		_, err = vm.RunString(j)
		if err != nil {
			errCh <- err
			return
		}
		if err = d.Close(); err != nil {
			errCh <- err
			return
		}
		h := d.QuerySelector("body").OuterHTML()
		t.Logf("h=%+v", h)
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	<-time.After(time.Second)
	resCh := make(chan string, 1)
	loop.RunOnLoop(func(vm *js.Runtime) {
		yes := d.getEl(d.QuerySelector("#ui-id-1").n.Parent).GetAttribute("aria-selected")
		if yes != "true" {
			errCh <- fmt.Errorf("expected true but got %v", yes)
			return
		}
		h := d.QuerySelector("body").OuterHTML()
		resCh <- h
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	t.Logf("h=%+v", <-resCh)
}

func TestJQueryUIAccordion(t *testing.T) {
	ResetCalls()
	defer PrintCalls()
	files := make(map[string][]byte)
	var err error
	for _, fn := range []string{"jquery-3.6.0.js", "jquery-ui.js", "accordion.html", "jquery-ui.css", "style.css"} {
		files[fn], err = os.ReadFile("jqueryui/" + fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	j := string(files["jquery-3.6.0.js"]) + string(files["jquery-ui.js"]) + `
$( function() {
	$( "#accordion" ).accordion();
} );
	`
	htm := string(files["accordion.html"])
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	var d *Document
	loop.RunOnLoop(func(vm *js.Runtime) {
		var err error
		d, err = Init(vm, htm, "")
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		_, err = vm.RunString(j)
		if err != nil {
			errCh <- err
			return
		}
		if err = d.Close(); err != nil {
			errCh <- err
			return
		}
		h := d.QuerySelector("body").OuterHTML()
		t.Logf("h=%+v", h)
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	<-time.After(time.Second)
	resCh := make(chan string, 1)
	loop.RunOnLoop(func(vm *js.Runtime) {
		ids := map[string]string{
			"#ui-id-1": "ui-id-2",
			"#ui-id-3": "ui-id-4",
			"#ui-id-5": "ui-id-6",
			"#ui-id-7": "ui-id-8",
		}
		for id, aId := range ids {
			a := d.getEl(d.QuerySelector(id).n).GetAttribute("aria-controls")
			if a != aId {
				errCh <- fmt.Errorf("expected %v but got %v", aId, a)
				return
			}
		}
		t.Logf("now do the click!!!!!")
		ev := &MouseEvent{
			Event: Event{
				Type: "click",
			},
		}
		consumed := d.getEl(d.QuerySelector("#ui-id-7").n).DispatchEvent(ev)
		if !consumed {
			errCh <- fmt.Errorf("ev not consumed")
			return
		}
		h := d.QuerySelector("body").OuterHTML()
		resCh <- h
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	t.Logf("h=%+v", <-resCh)
}

func TestJQueryUIMenu(t *testing.T) {
	ResetCalls()
	defer PrintCalls()
	files := make(map[string][]byte)
	var err error
	for _, fn := range []string{"jquery-3.6.0.js", "jquery-ui.js", "menu.html", "jquery-ui.css", "style.css"} {
		files[fn], err = os.ReadFile("jqueryui/" + fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	j := string(files["jquery-3.6.0.js"]) + string(files["jquery-ui.js"]) + `
$( function() {
	$( "#menu" ).menu();
} );
	`
	htm := string(files["menu.html"])
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	var d *Document
	loop.RunOnLoop(func(vm *js.Runtime) {
		var err error
		d, err = Init(vm, htm, "")
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		_, err = vm.RunString(j)
		if err != nil {
			errCh <- err
			return
		}
		if err = d.Close(); err != nil {
			errCh <- err
			return
		}
		h := d.QuerySelector("body").OuterHTML()
		t.Logf("h=%+v", h)
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	<-time.After(time.Second)
	electrQu := `#menu > li:nth-child(4) > ul`
	resCh := make(chan string, 1)
	loop.RunOnLoop(func(vm *js.Runtime) {
		disp, err := vm.RunString(`$('` + electrQu + `')[0].style.display`)
		if err != nil {
			errCh <- fmt.Errorf("check before click: %v", err)
			return
		}
		if s := disp.String(); s != "none" {
			errCh <- fmt.Errorf("expected electronics to be hidden: %v", s)
			return
		}
		_, err = vm.RunString(`document.getElementById('ui-id-4').parentElement.click();`)
		if err != nil {
			errCh <- fmt.Errorf("click: %v", err)
			return
		}
		h := d.QuerySelector("body").OuterHTML()
		resCh <- h
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	t.Logf("h=%+v", <-resCh)
	<-time.After(time.Second)
	resCh = make(chan string, 1)
	loop.RunOnLoop(func(vm *js.Runtime) {
		disp, err := vm.RunString(`$('` + electrQu + `')[0].style.display`)
		if err != nil {
			errCh <- fmt.Errorf("check before click: %v", err)
			return
		}
		disp, err = vm.RunString(`$('` + electrQu + `')[0].style.display`)
		if err != nil {
			errCh <- err
			return
		}
		if disp.String() != "" {
			errCh <- fmt.Errorf("expected electronics to be visible")
			return
		}
		h := d.QuerySelector("body").OuterHTML()
		resCh <- h
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	t.Logf("h=%+v", <-resCh)
}

func TestJQueryUIDatepicker(t *testing.T) {
	ResetCalls()
	defer PrintCalls()
	Geom = func(string) (string, error) { return "1,1,700,70", nil }
	defer func() { Geom = nil }()
	files := make(map[string][]byte)
	var err error
	for _, fn := range []string{"jquery-3.6.0.js", "jquery-ui.js", "datepicker.html", "jquery-ui.css", "style.css"} {
		files[fn], err = os.ReadFile("jqueryui/" + fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
	j := string(files["jquery-3.6.0.js"]) + string(files["jquery-ui.js"]) + `
$( function() {
	$( "#datepicker" ).datepicker();
} );
	`
	htm := string(files["datepicker.html"])
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	var d *Document
	loop.RunOnLoop(func(vm *js.Runtime) {
		var err error
		d, err = Init(vm, htm, "")
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		_, err = vm.RunString(j)
		if err != nil {
			errCh <- err
			return
		}
		if err = d.Close(); err != nil {
			errCh <- err
			return
		}
		h := d.QuerySelector("body").OuterHTML()
		t.Logf("h=%+v", h)
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	<-time.After(time.Second)
	resCh := make(chan string, 1)
	loop.RunOnLoop(func(vm *js.Runtime) {
		e := &Event{
			Type: "focus",
		}
		consumed := d.QuerySelector("#datepicker").DispatchEvent(e)
		if !consumed {
			errCh <- fmt.Errorf("expected click to be consumed")
			return
		}
		if pos := d.QuerySelector("#ui-datepicker-div").Style().Get("position").String(); pos != "absolute" {
			errCh <- fmt.Errorf("expected pos absolute")
			return
		}
		/*if d := d.QuerySelector("#ui-datepicker-div").Style().Get("display").String(); d != "block" {
			errCh <- fmt.Errorf("expected display block but got %v", d)
			return
		}*/
		h := d.QuerySelector("body").OuterHTML()
		resCh <- h
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	t.Logf("h=%+v", <-resCh)
}

func TestOnClick(t *testing.T) {
	htm := `
<!DOCTYPE html>
<html>
  <head>
  <title>Demo</title>
  </head>
<body>

<h2>Finding HTML Elements Using document.title</h2>

<p id="demo" class="bar" style="font-weight: bold;" onclick="clicked++;">the paragraph.</p>

<script>
document.getElementById("demo").innerHTML =
"The title of this document is: " + document.title;
</script>

</body>
</html>
`
	bs, err := os.ReadFile("jquery-1.8.2.js")
	if err != nil {
		t.Fatalf("%v", err)
	}

	j := `
window = {};
window = this;
global = {};
global.document = document;
` + string(bs) + `
var clicked = 0;

	`
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	loop.RunOnLoop(func(vm *js.Runtime) {
		document, err := Init(vm, htm, j)
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		if err = document.Close(); err != nil {
			errCh <- err
			return
		}
		t.Logf("now check clicked:")
		_, err = vm.RunString(`document.getElementsByTagName('p')[0].click();`)
		if err != nil {
			errCh <- err
			return
		}
		res, err := vm.RunString(`clicked`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res=%v", res)
		if !strings.Contains(res.String(), "1") {
			errCh <- err
			return
		}
		t.Logf("now check clicked':")
		_, err = vm.RunString(`document.getElementsByTagName('p')[0].dispatchEvent(new MouseEvent('click'));`)
		if err != nil {
			errCh <- err
			return
		}
		res, err = vm.RunString(`clicked`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res=%v", res)
		if !strings.Contains(res.String(), "2") {
			errCh <- err
			return
		}
		t.Logf("now check clicked'':")
		_, err = vm.RunString(`$('p')[0].dispatchEvent(new MouseEvent('click'));`)
		if err != nil {
			errCh <- err
			return
		}
		res, err = vm.RunString(`clicked`)
		if err != nil {
			errCh <- err
			return
		}
		t.Logf("res=%v", res)
		if !strings.Contains(res.String(), "3") {
			errCh <- err
			return
		}
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("timeout")
	}
}

// Both $('input[type="submit"]').dispatchEvent(new MouseEvent('click')) and
// $('input[type="submit"]').click() trigger form's onsubmit
func TestJQueryOnSubmit(t *testing.T) {
}

func TestReact(t *testing.T) {
	ResetCalls()
	htm, err := os.ReadFile("react/main.html")
	if err != nil {
		t.Fatalf("%v", err)
	}
	dev, err := os.ReadFile("react/react.development.es5.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	domDev, err := os.ReadFile("react/react-dom.development.es5.js")
	if err != nil {
		t.Fatalf("%v", err)
	}
	j := `
function error() {}
window.error = error;
console.debug = console.log;
console.info = console.log;
console.warn = console.log;
console.error = console.log;
window = {};
window = this;
global = {};
global.document = document;
` + string(dev) + `
` + string(domDev)
	loop := eventloop.NewEventLoop()
	errCh := make(chan error, 1)
	done := make(chan int, 1)
	loop.Start()
	defer loop.Stop()
	var d *Document
	loop.RunOnLoop(func(vm *js.Runtime) {
		var err error
		d, err = Init(vm, string(htm), j)
		if err != nil {
			errCh <- fmt.Errorf("main: %w", err)
			return
		}
		if err = d.Close(); err != nil {
			errCh <- err
			return
		}
		_, err = vm.RunString(`
      const container = document.getElementById('root');
      const root = ReactDOM.createRoot(container);
			const el = React.createElement('h1', null, 'Hello, world!')
      root.render(el);
      //ReactDOM.hydrate(el, root)
      true;
		`)
		if err != nil {
			errCh <- fmt.Errorf("run script: %w", err)
			return
		}
		h := render(d.doc)
		t.Logf("h=%v", h)
		d.GetElementById("root").DispatchEvent(&Event{Type: "load"})
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	<-time.After(time.Second)
	loop.RunOnLoop(func(vm *js.Runtime) {
		t.Logf("h=%v", render(d.doc))
		res, err := vm.RunString(`document.getElementById("root").innerHTML`)
		if err != nil {
			errCh <- fmt.Errorf("check: %v", err)
			return
		}
		if h := res.String(); h != "<h1>Hello, world!</h1>" {
			errCh <- fmt.Errorf("unexpected %v", h)
			return
		}
		done <- 1
	})
	select {
	case err := <-errCh:
		t.Fatalf("%v", err)
	case <-done:
	}
	PrintCalls()
}

func TestBubbling(t *testing.T) {
	ht := `
<body id=b>
the body
<p id=p>the parahraph</p>
<script src="1.js"></script>
</body>
	`
	script := `
var l = [];
function print() {
	const x = l[l.length-1];
}
function name(el) {
	if (el && el.id) {
		return el.id
	}
	if (el === document) {
		return 'document';
	}
	if (el) {
		return el.toString();
	}
	return undefined;
}
function f(e) {
	const r = {
		thisId: name(this),
		targetId: name(e.target),
		curId: name(e.currentTarget),
	};
	l.push(r);
	print();
}
var b = document.getElementById('b');
var p = document.getElementById('p');
b.addEventListener('click', f);
p.addEventListener('click', f);
document.addEventListener('click', f);
	`
	run := func(t *testing.T, fn func(t *testing.T, vm *js.Runtime, d *Document) error) (err error) {
		loop := eventloop.NewEventLoop()
		errCh := make(chan error, 1)
		done := make(chan int, 1)
		loop.Start()
		defer loop.Stop()
		loop.RunOnLoop(func(vm *js.Runtime) {
			document, err := Init(vm, ht, script)
			if err != nil {
				errCh <- fmt.Errorf("main: %w", err)
				return
			}
			if err = document.Close(); err != nil {
				errCh <- err
				return
			}
			if err := fn(t, vm, document); err != nil {
				errCh <- fmt.Errorf("fn: %w", err)
				return
			}
			done <- 1
		})
		select {
		case err = <-errCh:
		case <-done:
		}
		return
	}
	t.Run("click", func(t *testing.T) {
		err := run(t, func(t *testing.T, vm *js.Runtime, d *Document) (err error) {
			if _, err = vm.RunString("p.click()"); err != nil {
				return
			}
			res, err := vm.RunString("l")
			if err != nil {
				return
			}
			<-time.After(time.Second)
			l := res.Export().([]any)
			t.Logf("res=%+v", l)
			if len(l) < 3 {
				return fmt.Errorf("expect 3 but got %v", len(l))
			}
			for i, r := range l {
				t.Logf("i=%v", i)
				m := r.(map[string]any)
				switch i {
				case 0: // <p>
					if id := m["thisId"]; id != "p" {
						return fmt.Errorf("thisId: expected %v but got %v", "p", id)
					}
					if id := m["targetId"]; id != "p" {
						return fmt.Errorf("targetId: expected %v but got %v", "p", id)
					}
					if id := m["curId"]; id != "p" {
						return fmt.Errorf("curId: expected %v but got %v", "p", id)
					}
				case 1: // <body>
					if id := m["thisId"]; id != "b" {
						return fmt.Errorf("thisId: expected %v but got %v", "b", id)
					}
					if id := m["targetId"]; id != "p" {
						return fmt.Errorf("targetId: expected %v but got %v", "p", id)
					}
					if id := m["curId"]; id != "b" {
						return fmt.Errorf("curId: expected %v but got %v", "b", id)
					}
				case 2: // document
					if id := m["thisId"]; id != "document" {
						return fmt.Errorf("thisId: expected %v but got %v", "document", id)
					}
					if id := m["targetId"]; id != "p" {
						return fmt.Errorf("targetId: expected %v but got %v", "p", id)
					}
				}
			}
			return
		})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})
	t.Run("dispatchEvent", func(t *testing.T) {
	})
}

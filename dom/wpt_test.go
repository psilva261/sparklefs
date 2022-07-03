package dom

import (
	"testing"
)

func TestWptNodes(t *testing.T) {
	fns := []string{
		"Document-createElement.html",
		"Document-createEvent.https.html",
		"Document-createTextNode.html",
		"Document-doctype.html",
		"Document-getElementById.html",
		"Document-getElementsByClassName.html",
		"Document-getElementsByTagName.html",
		"DocumentFragment-constructor.html",
		"DocumentFragment-getElementById.html",
		"DocumentFragment-querySelectorAll-after-modification.html",
		"Element-children.html",
		"Element-matches.html",
		"Element-remove.html",
		"Node-childNodes.html",
		//"Node-cloneNode.html",
		"Node-isEqualNode.html",
		"Node-isSameNode.html",
		"Node-parentNode.html",
		"NodeList-Iterable.html",
		"ParentNode-querySelector-All-content.html",
		"ParentNode-querySelector-All.html",
		"ParentNode-querySelector-scope.html",
		"getElementsByClassName-01.htm",
		"getElementsByClassName-02.htm",
		"getElementsByClassName-03.htm",
		"getElementsByClassName-04.htm",
		"getElementsByClassName-05.htm",
		"getElementsByClassName-06.htm",
		"getElementsByClassName-07.htm",
		"getElementsByClassName-08.htm",
		"getElementsByClassName-09.htm",
		"getElementsByClassName-12.htm",
		"getElementsByClassName-13.htm",
		"getElementsByClassName-15.htm",
		"getElementsByClassName-16.htm",
		"getElementsByClassName-17.htm",
	}
	for _, fn := range fns {
		t.Run(fn, func(t *testing.T) {
			t.Logf("========= %v =======", fn)
			err := Main("test/wpt/dom/nodes", fn)
			if err != nil {
				t.Fatalf("%v", err)
			}
		})
	}
}

func TestWptEvents(t *testing.T) {
	fns := []string{
		"Event-initEvent.html",
		"Event-defaultPrevented.html",
		"Event-dispatch-click.html",
		"Event-dispatch-bubbles-true.html",
		"Event-dispatch-bubbles-false.html",
		"Event-dispatch-order.html",
		"Event-propagation.html",
		"EventTarget-this-of-listener.html",
	}
	for _, fn := range fns {
		t.Logf("========= %v =======", fn)
		err := Main("test/wpt/dom/events", fn)
		if err != nil {
			t.Fatalf("%v", err)
		}
	}
}

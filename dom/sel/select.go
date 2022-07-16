package sel

import (
	"fmt"
	"github.com/psilva261/sparklefs/logger"
	"golang.org/x/net/html"
	"strconv"
	"strings"
)

func Select(sel string, el *html.Node, ignoreRoot, rootMustMatchFirst bool) (es []*html.Node, err error) {
	sels := strings.Split(sel, ",")
	for _, s := range sels {
		res, err := SelectSingle(s, el, ignoreRoot, rootMustMatchFirst)
		if err != nil {
			return nil, fmt.Errorf("select single %v: %w", s, err)
		}
		for _, e := range res {
			if e.Type == html.ElementNode {
				es = append(es, e)
			}
		}
	}
	return
}

func explode(sel string) (rest string, nthChild int, err error) {
	nthChild = -1
	l, err := splitBlock(sel)
	if err != nil {
		return
	}
	tmp := make([]string, 0, len(l))
	for _, s := range l {
		if strings.Contains(s, "nth-child") {
			s = s[11 : len(s)-1]
			nthChild, err = strconv.Atoi(s)
			if err != nil {
				return "", 0, fmt.Errorf("atoi %v: %v", s, err)
			}
		} else if strings.Contains(s, "first-child") {
			nthChild = 1
		} else {
			tmp = append(tmp, s)
		}
	}
	rest = strings.Join(tmp, "")
	return
}

func trimSpaces(sel string) string {
	sel = strings.ReplaceAll(sel, "\n", " ")
	sel = strings.ReplaceAll(sel, "\t", " ")
	sel = strings.ReplaceAll(sel, "\f", " ")
	sel = strings.ReplaceAll(sel, "\r", " ")
	for strings.Contains(sel, "  ") {
		strings.ReplaceAll(sel, "  ", " ")
	}
	sel = strings.TrimSpace(sel)
	return sel
}

type chain []string

func (c chain) shift() (s string, cc chain, ok bool) {
	if len(c) == 0 {
		return
	}
	return c[0], c[1:], true
}

// peekSel the next (non combinator) item
func (c chain) peekSel() (s string, ok bool) {
	for _, x := range c {
		if x != ">" {
			return x, true
		}
	}
	return
}

// SelectSingle selects by one item from a comma separated selector group
func SelectSingle(sel string, el *html.Node, ignoreRoot, rootMustMatchFirst bool) (es []*html.Node, err error) {
	if el.Type == html.TextNode || el.Type == html.CommentNode {
		return
	}
	sel = trimSpaces(sel)
	sels := strings.Split(sel, " ")

	var nthChild int = -1
	s, nthChild, err := explode(sels[0])
	if err != nil {
		return nil, fmt.Errorf("explode %v: %w", sels[0], err)
	}
	switch s {
	case ">":
		sel = strings.Join(sels[1:], " ")
		es, err = SelectSingle(sel, el, ignoreRoot, true)
	default:
		if ignoreRoot && s != ":scope" {
			for c := el.FirstChild; c != nil; c = c.NextSibling {
				res, err := SelectSingle(sel, c, false, false)
				if err != nil {
					return nil, fmt.Errorf("select single %v: %w", s, err)
				}
				es = append(es, res...)
			}
			break
		}

		if ElementMatchesSingle(s, el, rootMustMatchFirst, nthChild) || s == ":scope" {
			if len(sels) > 1 {
				sel = strings.Join(sels[1:], " ")
				for c := el.FirstChild; c != nil; c = c.NextSibling {
					res, err := SelectSingle(sel, c, false, false)
					if err != nil {
						return nil, fmt.Errorf("select single %v: %w", s, err)
					}
					es = append(es, res...)
				}
			} else {
				es = append(es, el)
			}
		} else if !rootMustMatchFirst {
			for c := el.FirstChild; c != nil; c = c.NextSibling {
				res, err := SelectSingle(sel, c, false, false)
				if err != nil {
					return nil, fmt.Errorf("select single %v: %w", s, err)
				}
				es = append(es, res...)
			}
		}
	}
	if nthChild >= 0 && nthChild-1 < len(es) {
		es = []*html.Node{es[nthChild-1]}
	}
	return
}

// ElementMatchesSingle selects 1 cascade at a specific element
func ElementMatchesSingle(sel string, el *html.Node, rootMustMatchFirst bool, nthChild int) bool {
	if nthChild > 0 {
		kth := 1
		for sib := el.PrevSibling; sib != nil; sib = sib.PrevSibling {
			if sib.Type == html.ElementNode {
				kth++
			}
		}
		if kth != nthChild {
			return false
		}
	}
	if sel == "*" {
		return true
	}

	matchesAll := true
	l, err := splitBlock(sel)
	if err != nil {
		log.Fatalf("split block %v: %v", sel, err)
	}
	for _, sel := range l {
		found := false
		if strings.HasPrefix(sel, "#") || strings.HasPrefix(sel, ".") || strings.HasPrefix(sel, "[") || strings.HasPrefix(sel, ":") {
			q := sel
			if strings.HasPrefix(sel, "#") && attr(*el, "id") == strings.ReplaceAll(q[1:], `\\`, ``) {
				found = true
			} else if strings.HasPrefix(sel, ".") && matchesClasses(el, []string{q[1:]}) {
				found = true
			} else if strings.HasPrefix(sel, "[") {
				sel = strings.TrimPrefix(sel, "[")
				sel = strings.TrimSuffix(sel, "]")
				i := strings.Index(sel, "=")
				if i < 0 {
					if hasAttr(*el, sel) {
						found = true
					}
				} else {
					k := sel[:i]
					v := sel[i+1:]
					v = strings.ReplaceAll(v, `"`, ``)
					v = strings.ReplaceAll(v, `'`, ``)
					if attr(*el, k) == v {
						found = true
					}
				}
			} else if strings.HasPrefix(sel, ":") && matchesPseudo(el, q, rootMustMatchFirst) {
				found = true
			}
		} else {
			if strings.ToLower(el.Data) == strings.ToLower(sel) {
				found = true
			}
		}
		if !found {
			matchesAll = false
		}
	}
	return matchesAll
}

func splitBlock(sb string) (l []string, err error) {
	if strings.Contains(sb, " ") {
		return nil, fmt.Errorf("no spaces")
	}
	paranth := false
	tmp := ""
	flush := func() {
		if tmp != "" && !paranth {
			l = append(l, tmp)
			tmp = ""
		}
	}
	for i, ch := range sb {
		switch ch {
		case ']':
			tmp += string([]byte{byte(ch)})
			flush()
		case '(':
			paranth = true
			tmp += string([]byte{byte(ch)})
		case ')':
			paranth = false
			tmp += string([]byte{byte(ch)})
		case '.', '#', '[', ':':
			if i < 2 || sb[i-1] != '\\' || sb[i-2] != '\\' {
				flush()
			}
			fallthrough
		default:
			tmp += string([]byte{byte(ch)})
		}
	}
	flush()
	return
}

func matchesPseudo(n *html.Node, q string, rootMustMatchFirst bool) (matches bool) {
	if strings.HasPrefix(q, ":has") {
		q = q[5 : len(q)-1]
		es, err := SelectSingle(q, n, true, false)
		if err != nil {
			log.Errorf("match pseudo %v: %v", q, err)
			return false
		}
		matches = len(es) > 0
	} else if strings.HasPrefix(q, ":not") {
		q = q[5 : len(q)-1]
		es, err := SelectSingle(q, n, false, rootMustMatchFirst)
		if err != nil {
			log.Errorf("match pseudo %v: %v", q, err)
			return false
		}
		matches = len(es) == 0
	} else if strings.HasPrefix(q, ":nth-child") {
		// handled in SelectSingle (explode)
	} else if q == ":scope" {
		// handled in SelectSingle
	} else {
		log.Errorf("unknown pseudo selector %v", q)
	}
	return
}

func attr(n html.Node, key string) (val string) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return
}

func hasAttr(n html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func matchesClasses(n *html.Node, qs []string) bool {
	s := attr(*n, "class")
	cls := classes(s)
	matchesAll := true
	for _, q := range qs {
		found := false
		for _, cl := range cls {
			if cl == q {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return matchesAll
}

func classes(cls string) (res []string) {
	cls = strings.ReplaceAll(cls, "\n", " ")
	cls = strings.ReplaceAll(cls, "\t", " ")
	cls = strings.ReplaceAll(cls, "\f", " ")
	cls = strings.ReplaceAll(cls, "\r", " ")
	tmp := strings.Split(cls, " ")
	res = make([]string, 0, len(tmp))
	for _, c := range tmp {
		if c != "" {
			res = append(res, c)
		}
	}
	return
}

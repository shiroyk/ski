package gq

import (
	"testing"
)

func TestBuildInFuncText(t *testing.T) {
	t.Parallel()

	assertValue(t, `#main #n1 -> text`, "1")

	assertValue(t, `#main #n1`, "1")
}

func TestBuildInFuncAttr(t *testing.T) {
	t.Parallel()
	assertError(t, `#main #n1 -> text -> attr`, "attr(name) must has name")

	assertError(t, `#main -> attr()`, "attr(name) must has name")

	assertValue(t, `#main #n1 -> attr(class)`, "one even row")

	assertValue(t, `#main #n1 -> attr(empty, default)`, "default")
}

func TestBuildInFuncHref(t *testing.T) {
	t.Parallel()
	assertError(t, `.body ul #a4 -> text -> href`, "unexpected content type string")

	assertValue(t, `.body ul #a4 a -> href(https://localhost)`, "https://localhost/home")

	assertValue(t, `.body ul #a4 a -> href(https://localhost/path/)`, "https://localhost/path/home")
}

func TestBuildInFuncHtml(t *testing.T) {
	t.Parallel()
	assertError(t, `.body -> html(test)`, "html(outer) `outer` must bool type value: true/false")

	assertValue(t, `.body ul a -> html`, []string{"Google", "Github", "Golang", "Home"})

	assertValue(t, `.body ul a -> slice(0,2) -> html(true)`,
		[]string{
			"<a href=\"https://google.com\" title=\"Google page\">Google</a>",
			"<a href=\"https://github.com\" title=\"Github page\">Github</a>"})
}

func TestBuildInFuncPrev(t *testing.T) {
	t.Parallel()
	assertError(t, `#foot #nf3 -> text -> prev`, "unexpected content type string")

	assertValue(t, `#foot #nf3 -> prev`, "f2")

	assertValue(t, `#foot #nf3 -> prev(#nf1)`, "f2")
}

func TestBuildInFuncNext(t *testing.T) {
	t.Parallel()
	assertError(t, `#foot #nf2 -> text -> next`, "unexpected type string")

	assertValue(t, `#foot #nf2 -> next`, "f3")

	assertValue(t, `#foot #nf2 -> next(#nf4)`, "f3")
}

func TestBuildInFuncSlice(t *testing.T) {
	t.Parallel()
	assertError(t, `#main -> slice`, "slice(start, end) must have at least one int argument")

	assertError(t, `#main div -> text -> slice(0)`, "slice: unexpected type []string")

	assertValue(t, `#main div -> slice(0)`, "1")

	assertValue(t, `#main div -> slice(-1)`, "6")

	assertValue(t, `#main div -> slice(0, 3)`, []string{"1", "2", "3"})

	assertValue(t, `#main div -> slice(0, -2)`, []string{"1", "2", "3", "4"})
}

func TestBuildInFuncChild(t *testing.T) {
	t.Parallel()
	assertError(t, `.body ul -> text -> child`, "unexpected type string")

	assertValue(t, `.body ul li -> child(a)`, []string{"Google", "Github", "Golang", "Home"})

	assertValue(t, `.body ul li -> child`, []string{"Google", "Github", "Golang", "Home"})
}

func TestBuildInFuncParent(t *testing.T) {
	t.Parallel()
	assertError(t, `.body ul -> text -> parent`, "unexpected type string")

	assertValue(t, `.body ul a -> parent(#a1) -> attr(id)`, "a1")

	assertValue(t, `.body ul a -> parent -> attr(id)`, []string{"a1", "a2", "a3", "a4"})
}

func TestBuildInFuncParents(t *testing.T) {
	t.Parallel()
	assertError(t, `.body ul -> text -> parents`, "unexpected type string")

	assertError(t, `.body ul .selected -> parents(div, test)`, "parents(selector, until) `until` must bool type value: true/false")

	assertValue(t, `.body ul .selected -> parents(div, true) -> attr(id)`, "url")

	assertValue(t, `.body ul .selected -> parents -> slice(0) -> attr(id)`, "url")
}

func TestBuildInFuncPrefix(t *testing.T) {
	t.Parallel()

	assertValue(t, `#main #n1 -> text -> prefix(A)`, "A1")

	assertValue(t, `#main #n1 -> prefix(B)`, "B1")
}

func TestBuildInFuncSuffix(t *testing.T) {
	t.Parallel()

	assertValue(t, `#main #n1 -> text -> suffix(A)`, "1A")

	assertValue(t, `#main #n1 -> suffix(B)`, "1B")
}

func TestBuildInZip(t *testing.T) {
	t.Parallel()

	assertElements(t, `-> zip('#main div', '#foot div')`, []string{
		`<div id="n1" class="one even row">1</div><div id="nf1" class="one even row">f1</div>`,
		`<div id="n2" class="two odd row">2</div><div id="nf2" class="two odd row">f2</div>`,
		`<div id="n3" class="three even row">3</div><div id="nf3" class="three even row">f3</div>`,
		`<div id="n4" class="four odd row">4</div><div id="nf4" class="four odd row">f4</div>`,
		`<div id="n5" class="five even row">5</div><div id="nf5" class="five even row odder">f5</div>`,
		`<div id="n6" class="six odd row">6</div><div id="nf6" class="six odd row">f6</div>`,
	})
}

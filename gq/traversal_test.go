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

	assertValue(t, `#foot #nf3 -> prev`, "f2")

	assertValue(t, `#foot #nf3 -> prev(#nf1)`, "f2")
}

func TestBuildInFuncNext(t *testing.T) {
	t.Parallel()

	assertValue(t, `#foot #nf2 -> next`, "f3")

	assertValue(t, `#foot #nf2 -> next(#nf4)`, "f3")
}

func TestBuildInFuncSlice(t *testing.T) {
	t.Parallel()
	assertError(t, `#main -> slice`, "slice(start, end) must have at least one int argument")

	assertValue(t, `#main div -> slice(0)`, "1")

	assertValue(t, `#main div -> slice(-1)`, "6")

	assertValue(t, `#main div -> slice(0, 3)`, []string{"1", "2", "3"})

	assertValue(t, `#main div -> slice(0, -2)`, []string{"1", "2", "3", "4"})
}

func TestBuildInFuncChunk(t *testing.T) {
	t.Parallel()

	assertValue(t, `.body ul li -> chunk(2)`, []string{"GoogleGithub", "GolangHome"})

	assertValue(t, `.body ul -> chunk(li, 2)`, []string{"GoogleGithub", "GolangHome"})

	assertValue(t, `.body ul li -> chunk(3)`, []string{"GoogleGithubGolang", "Home"})
}

func TestBuildInFuncChild(t *testing.T) {
	t.Parallel()

	assertValue(t, `.body ul li -> child(a)`, []string{"Google", "Github", "Golang", "Home"})

	assertValue(t, `.body ul li -> child`, []string{"Google", "Github", "Golang", "Home"})
}

func TestBuildInFuncParent(t *testing.T) {
	t.Parallel()

	assertValue(t, `.body ul a -> parent(#a1) -> attr(id)`, "a1")

	assertValue(t, `.body ul a -> parent -> attr(id)`, []string{"a1", "a2", "a3", "a4"})
}

func TestBuildInFuncParents(t *testing.T) {
	t.Parallel()

	assertValue(t, `.body ul .selected -> parents -> slice(0) -> attr(id)`, "url")
}

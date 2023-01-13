package gq

import "testing"

func TestBuildInFuncGet(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `-> get`); err == nil {
		t.Fatal("Unexpected function error")
	}

	if _, err := gq.GetString(ctx, content, `.body #a1 -> set(v1)`); err != nil {
		t.Fatal(err)
	}

	assertGetString(t, `-> get(v1) -> child`, func(str string) bool {
		return str == "Google"
	})
}

func TestBuildInFuncSet(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `-> set`); err == nil {
		t.Fatal("Unexpected function error")
	}

	if _, err := gq.GetString(ctx, content, `-> set(v1, '<i>v1</i>')`); err != nil {
		t.Fatal(err)
	}

	if _, err := gq.GetString(ctx, content, `.body #a1 -> text -> set(v1)`); err != nil {
		t.Fatal(err)
	}
}

func TestBuildInFuncText(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `#main #n1 -> text -> text`); err == nil {
		t.Fatal("Unexpected function error")
	}

	assertGetString(t, `#main #n1 -> text`, func(str string) bool {
		return str == "1"
	})

	assertGetString(t, `#main #n1`, func(str string) bool {
		return str == "1"
	})
}

func TestBuildInFuncAttr(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `#main #n1 -> text -> attr`); err == nil {
		t.Fatal("Unexpected function error")
	}

	if _, err := gq.GetString(ctx, content, `-> attr()`); err == nil {
		t.Fatal("Unexpected null argument")
	}

	assertGetString(t, `#main #n1 -> attr(class)`, func(str string) bool {
		return str == "one even row"
	})

	assertGetString(t, `#main #n1 -> attr(empty, default)`, func(str string) bool {
		return str == "default"
	})
}

func TestBuildInFuncJoin(t *testing.T) {
	assertGetString(t, `#main div -> join(' < ')`, func(str string) bool {
		return str == "1 < 2 < 3 < 4 < 5 < 6"
	})

	assertGetString(t, `#main div -> join("")`, func(str string) bool {
		return str == "123456"
	})

	assertGetString(t, `#main div -> join('')`, func(str string) bool {
		return str == "123456"
	})
}

func TestBuildInFuncHref(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `.body ul #a4 -> text -> href`); err == nil {
		t.Fatal("Unexpected function error")
	}

	assertGetString(t, `.body ul #a4 a -> href`, func(str string) bool {
		return str == "https://localhost/home"
	})
}

func TestBuildInFuncHtml(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `-> html(test)`); err == nil {
		t.Fatal("Unexpected function error")
	}

	assertGetString(t, `.body ul a -> html`, func(str string) bool {
		return str == "Google, Github, Golang, Home"
	})

	assertGetString(t, `.body ul a -> slice(0) -> html(true)`, func(str string) bool {
		return str == `<a href="https://google.com" title="Google page">Google</a>`
	})
}

func TestBuildInFuncPrev(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `#foot #nf3 -> text -> prev`); err == nil {
		t.Fatal("Unexpected function error")
	}

	assertGetString(t, `#foot #nf3 -> prev`, func(str string) bool {
		return str == "f2"
	})

	assertGetString(t, `#foot #nf3 -> prev(#nf1)`, func(str string) bool {
		return str == "f2"
	})
}

func TestBuildInFuncNext(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `#foot #nf2 -> text -> next`); err == nil {
		t.Fatal("Unexpected function error")
	}

	assertGetString(t, `#foot #nf2 -> next`, func(str string) bool {
		return str == "f3"
	})

	assertGetString(t, `#foot #nf2 -> next(#nf4)`, func(str string) bool {
		return str == "f3"
	})
}

func TestBuildInFuncSlice(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `-> slice`); err == nil {
		t.Fatal("Unexpected function error")
	}

	if _, err := gq.GetString(ctx, content, `#main div -> text -> slice(0)`); err == nil {
		t.Fatal("Unexpected function error")
	}

	assertGetString(t, `#main div -> slice(0)`, func(str string) bool {
		return str == "1"
	})

	assertGetString(t, `#main div -> slice(-1)`, func(str string) bool {
		return str == "6"
	})

	assertGetString(t, `#main div -> slice(0, 3)`, func(str string) bool {
		return str == "1, 2, 3"
	})

	assertGetString(t, `#main div -> slice(0, -2)`, func(str string) bool {
		return str == "1, 2, 3, 4"
	})
}

func TestBuildInFuncChild(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `.body ul -> text -> child`); err == nil {
		t.Fatal("Unexpected function error")
	}

	assertGetString(t, `.body ul li -> child(a)`, func(str string) bool {
		return str == "Google, Github, Golang, Home"
	})

	assertGetString(t, `.body ul li -> child`, func(str string) bool {
		return str == "Google, Github, Golang, Home"
	})
}

func TestBuildInFuncParent(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `.body ul -> text -> parent`); err == nil {
		t.Fatal("Unexpected function error")
	}

	assertGetString(t, `.body ul a -> parent(#a1) -> attr(id)`, func(str string) bool {
		return str == "a1"
	})

	assertGetString(t, `.body ul a -> parent -> attr(id)`, func(str string) bool {
		return str == "a1, a2, a3, a4"
	})
}

func TestBuildInFuncParents(t *testing.T) {
	if _, err := gq.GetString(ctx, content, `.body ul -> text -> parents`); err == nil {
		t.Fatal("Unexpected type")
	}

	if _, err := gq.GetString(ctx, content, `.body ul .selected -> parents(div, test)`); err == nil {
		t.Fatal("Unexpected argument")
	}

	assertGetString(t, `.body ul .selected -> parents(div, true) -> attr(id)`, func(str string) bool {
		return str == "url"
	})

	assertGetString(t, `.body ul .selected -> parents -> slice(0) -> attr(id)`, func(str string) bool {
		return str == "url"
	})
}

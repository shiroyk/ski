package js

import (
	"flag"
	"os"
	"testing"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/cache/memory"
	"github.com/shiroyk/cloudcat/di"
	"github.com/shiroyk/cloudcat/fetcher"
	p "github.com/shiroyk/cloudcat/parser"
)

var (
	testVM *goja.Runtime
	ctx    *p.Context
)

const TESTLIB = `
function $ERROR(message) {
	throw new Error(message);
}

function Test262Error() {
}

function assert(mustBeTrue, message) {
    if (mustBeTrue === true) {
        return;
    }

    if (message === undefined) {
        message = 'Expected true but got ' + String(mustBeTrue);
    }
    $ERROR(message);
}

assert._isequal = function (a, b) {
    if (a === b) {
        // Handle +/-0 vs. -/+0
        return a !== 0 || 1 / a === 1 / b;
    }

    // Handle NaN vs. NaN
    return a !== a && b !== b;
};

assert.equal = function (actual, expected, message) {
    if (assert._isequal(actual, expected)) {
        return;
    }

    if (message === undefined) {
        message = '';
    } else {
        message += ' ';
    }

    message += 'Expected equal(' + String(actual) + ', ' + String(expected) + ') to be true';

    $ERROR(message);
};

assert.throws = function (expectedErrorConstructor, func, message) {
  if (typeof func !== "function") {
    $ERROR('assert.throws requires two arguments: the error constructor ' +
      'and a function to run');
    return;
  }
  if (message === undefined) {
    message = '';
  } else {
    message += ' ';
  }

  try {
    func();
  } catch (thrown) {
    if (typeof thrown !== 'object' || thrown === null) {
      message += 'Thrown value was not an object!';
      $ERROR(message);
    } else if (thrown.constructor !== expectedErrorConstructor) {
      message += 'Expected a ' + expectedErrorConstructor.name + ' but got a ' + thrown.constructor.name;
      $ERROR(message);
    }
    return;
  }

  message += 'Expected a ' + expectedErrorConstructor.name + ' to be thrown but no exception was thrown at all';
  $ERROR(message);
};

function compareArray(a, b) {
  if (b.length !== a.length) {
    return false;
  }

  for (var i = 0; i < a.length; i++) {
    if (b[i] !== a[i]) {
      return false;
    }
  }
  return true;
}
`

func TestMain(m *testing.M) {
	flag.Parse()
	di.ProvideNamed("cache", memory.NewCache())
	di.ProvideNamed("cookie", memory.NewCookie())
	di.ProvideNamed("shortener", memory.NewShortener())
	di.Provide(fetcher.NewFetcher(&fetcher.Options{}))
	ctx = p.NewContext(&p.Options{
		Url: "http://localhost/home",
	})
	testVM = CreateVMWithContext(ctx, `1919810`)
	_, _ = testVM.RunProgram(goja.MustCompile("testlib.js", TESTLIB, false))
	code := m.Run()
	os.Exit(code)
}

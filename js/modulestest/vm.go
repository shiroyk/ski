package modulestest

import (
	"context"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/js/common"
	"github.com/shiroyk/cloudcat/js/modules"
)

// TestLib js assertion utilities
const TestLib = `
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

// VM test vm
type VM struct {
	vm *goja.Runtime
}

// RunString the js string
func (vm *VM) RunString(_ context.Context, script string) (goja.Value, error) {
	return vm.vm.RunString(script)
}

// Run the js program
func (vm *VM) Run(_ context.Context, program common.Program) (goja.Value, error) {
	return vm.vm.RunString(program.Code)
}

// New returns a test VM instance
func New() *VM {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
	modules.EnableRequire(vm)
	modules.InitGlobalModule(vm)
	_, _ = vm.RunProgram(goja.MustCompile("testlib.js", TestLib, false))
	return &VM{vm}
}

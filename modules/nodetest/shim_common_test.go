package nodetest

// commonShim is the `test/common` module of the upstream node test suite, it
// implements the helpers the tests use, see test/common/index.js.
const commonShim = `'use strict';

const assert = require('assert');
const util = require('util');

const mustCallChecks = [];

function inspect(value) {
  return util.inspect(value);
}

function runCallChecks() {
  const failed = mustCallChecks.filter((context) => {
    const expected = 'minimum' in context ? context.minimum : context.exact;
    return 'minimum' in context ? context.actual < expected : context.actual !== expected;
  });
  if (failed.length === 0) return;
  const detail = failed.map((context) => {
    const expected = 'minimum' in context
      ? 'at least ' + context.minimum
      : 'exactly ' + context.exact;
    return context.name + ' expected ' + expected + ' calls, got ' + context.actual;
  }).join('\n');
  throw new Error('Mismatched function calls:\n' + detail);
}

globalThis.__nodetest_runExitChecks__ = runCallChecks;

function _mustCallInner(fn, criteria = 1, field) {
  if (typeof fn === 'number') {
    criteria = fn;
    fn = undefined;
  } else if (fn === undefined) {
    fn = function() {};
  }
  if (typeof criteria !== 'number') throw new TypeError('Invalid ' + field + ' value: ' + criteria);

  const context = {
    [field]: criteria,
    actual: 0,
    name: fn.name || '<anonymous>',
    stack: new Error().stack,
  };
  mustCallChecks.push(context);

  const wrapped = function(...args) {
    context.actual++;
    return fn.apply(this, args);
  };
  Object.defineProperty(wrapped, 'name', { value: context.name, configurable: true });
  return wrapped;
}

function mustCall(fn, exact) {
  return _mustCallInner(fn, exact, 'exact');
}

function mustCallAtLeast(fn, minimum) {
  return _mustCallInner(fn, minimum, 'minimum');
}

function mustSucceed(fn, exact) {
  return mustCall(function(error, ...args) {
    assert.ifError(error);
    if (typeof fn === 'function') return fn.apply(this, args);
  }, exact);
}

function mustNotCall(msg) {
  return function mustNotCall(...args) {
    const argsInfo = args.length > 0 ? '\ncalled with arguments: ' + args.map(inspect).join(', ') : '';
    assert.fail((msg || 'function should not have been called') + argsInfo);
  };
}

function expectsError(validator, exact) {
  return mustCall((...args) => {
    if (args.length !== 1) assert.fail('Expected one argument, got ' + inspect(args));
    const error = args.pop();
    assert.strictEqual(Object.prototype.propertyIsEnumerable.call(error, 'message'), false);
    assert.throws(() => { throw error; }, validator);
    return true;
  }, exact);
}

let catchWarning;

function expectWarning(nameOrMap, expected, code) {
  if (catchWarning === undefined) {
    catchWarning = {};
    process.on('warning', (warning) => {
      const handler = catchWarning[warning.name];
      if (!handler) {
        throw new TypeError('"' + warning.name + '" was triggered without being expected.');
      }
      handler(warning);
    });
  }

  const expectOne = (name, expected, code) => {
    if (typeof expected === 'string') expected = [[expected, code]];
    else if (!Array.isArray(expected)) expected = Object.entries(expected).map(([a, b]) => [b, a]);
    else if (expected.length !== 0 && !Array.isArray(expected[0])) expected = [[expected[0], expected[1]]];
    if (name === 'DeprecationWarning') {
      expected.forEach(([, code]) => assert(code, 'Missing deprecation code: ' + expected));
    }
    return mustCall((warning) => {
      const properties = expected.shift();
      if (!properties) assert.fail('Unexpected extra warning received: ' + inspect(warning));
      const [message, code] = properties;
      assert.strictEqual(warning.name, name);
      if (typeof message === 'string') assert.strictEqual(warning.message, message);
      else assert.match(warning.message, message);
      assert.strictEqual(warning.code, code);
    }, expected.length);
  };

  if (typeof nameOrMap === 'string') catchWarning[nameOrMap] = expectOne(nameOrMap, expected, code);
  else Object.keys(nameOrMap).forEach((name) => { catchWarning[name] = expectOne(name, nameOrMap[name]); });
}

function printSkipMessage(msg) {
  console.log('1..0 # Skipped: ' + msg);
}

function skip(msg) {
  printSkipMessage(msg);
  globalThis.__nodetest_skipped__ = String(msg);
  const error = new Error('skipped: ' + msg);
  error.__nodetest_skip__ = true;
  throw error;
}

function skipIf(condition, msg) {
  if (condition) skip(msg);
}

function skipIf32Bits() {
  skip('The tested feature is not available in 32bit builds');
}

// A helper to simplify checking for ERR_INVALID_ARG_TYPE output.
function invalidArgTypeHelper(input) {
  if (input == null) return ' Received ' + input;
  if (typeof input === 'function') return ' Received function ' + input.name;
  if (typeof input === 'object') {
    if (input.constructor && input.constructor.name) {
      return ' Received an instance of ' + input.constructor.name;
    }
    return ' Received ' + inspect(input);
  }
  let inspected = inspect(input);
  if (inspected.length > 28) inspected = inspected.slice(0, 25) + '...';
  return ' Received type ' + typeof input + ' (' + inspected + ')';
}

module.exports = {
  mustCall,
  mustCallAtLeast,
  mustSucceed,
  mustNotCall,
  expectsError,
  expectWarning,
  printSkipMessage,
  skip,
  skipIf,
  skipIf32Bits,
  invalidArgTypeHelper,
  inspect,
  isDebug: false,
  hasCrypto: false,
  hasIntl: true,
  hasInspector: false,
  enoughTestMem: true,
  isWindows: false,
  isAIX: false,
  isFreeBSD: false,
  isLinux: false,
  isMacOS: true,
  isMainThread: true,
  platform: 'darwin',
  bits: 64,
};
`

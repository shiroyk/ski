package modulestest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/shiroyk/ski/modules/buffer"
)

func TestAssert(t *testing.T) {
	t.Parallel()
	vm := New()
	ctx := context.Background()

	passing := []string{
		`assert(1)`,
		`assert.strictEqual(1, 1)`,
		// strictEqual uses the Object.is semantics
		`assert.strictEqual(NaN, NaN)`,
		`assert.notStrictEqual(0, -0)`,
		`assert.deepStrictEqual([1, [2]], [1, [2]])`,
		`assert.deepStrictEqual({ a: { b: 1 } }, { a: { b: 1 } })`,
		`assert.deepStrictEqual(Buffer.from("abc"), Buffer.from("abc"))`,
		`assert.deepStrictEqual(new Uint8Array([1]), new Uint8Array([1]))`,
		`assert.notDeepStrictEqual(Buffer.from("abc"), new Uint8Array([97, 98, 99]))`,
		`assert.deepEqual("1", 1)`,
		`assert.throws(() => { throw new TypeError("boom"); }, TypeError)`,
		`assert.throws(() => { throw new Error("boom"); }, /boom/)`,
		`assert.throws(() => { throw new Error("boom"); }, (err) => err.message === "boom")`,
		`assert.throws(() => { const err = new Error("boom"); err.code = "ERR_BOOM"; throw err; }, { code: "ERR_BOOM" })`,
		`assert.throws(() => assert.strictEqual(1, 2), assert.AssertionError)`,
		`assert.throws(() => assert.fail("nope"), assert.AssertionError)`,
		`assert.doesNotThrow(() => 1)`,
		`assert.match("foobar", /oba/)`,
		`assert.ifError(null)`,
	}
	for _, source := range passing {
		t.Run(source, func(t *testing.T) {
			_, err := vm.RunString(ctx, source)
			require.NoError(t, err)
		})
	}

	failing := []string{
		`assert(0)`,
		`assert.strictEqual(1, "1")`,
		`assert.strictEqual(0, -0)`,
		`assert.deepStrictEqual({ a: 1 }, { a: 2 })`,
		`assert.deepStrictEqual(Buffer.from("abc"), new Uint8Array([97, 98, 99]))`,
		`assert.deepStrictEqual(new Uint8Array([1]), new Int8Array([1]))`,
		`assert.throws(() => 1)`,
		`assert.throws(() => { throw new Error("boom"); }, TypeError)`,
		`assert.throws(() => { throw new Error("boom"); }, /bang/)`,
		`assert.doesNotThrow(() => { throw new Error("boom"); })`,
		`assert.match("foobar", /baz/)`,
		`assert.ifError(new Error("boom"))`,
	}
	for _, source := range failing {
		t.Run(source, func(t *testing.T) {
			value, err := vm.RunString(ctx, source)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "AssertionError", value)
		})
	}

	t.Run("assertion error", func(t *testing.T) {
		value, err := vm.RunString(ctx, `
			const err = assert.throws(() => assert.strictEqual(1, 2), assert.AssertionError);
			[err instanceof assert.AssertionError, err instanceof Error, err.code, err.name].join(",");
		`)
		require.NoError(t, err)
		assert.Equal(t, "true,true,ERR_ASSERTION,AssertionError", value.String())
	})
}

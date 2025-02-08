// Package js the JavaScript implementation.
//
// Run ESM/CJS modules:
//
//	func main() {
//		module, err := js.CompileModule("", "module.exports = () => 'some value'")
//		if err != nil {
//			panic(err)
//		}
//
//		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
//		defer cancel()
//
//		value, err := js.RunModule(ctx, module)
//		if err != nil {
//			panic(err)
//		}
//
//		fmt.Println(value.Export())
//	}
package js

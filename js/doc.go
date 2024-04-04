// Package js the JavaScript implementation.
//
// Run ESM/CJS modules:
//
//	func main() {
//		module, err := js.GetScheduler().Loader().CompileModule("", "module.exports = () => 'some value'")
//		if err != nil {
//			fmt.Println(err)
//			return
//		}
//
//		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
//		defer cancel()
//
//		value, err := js.RunModule(ctx, module)
//		if err != nil {
//			fmt.Println(err)
//			return
//		}
//
//		fmt.Println(value.Export())
//	}
//
// Configure JS Scheduler:
//
//	func init() {
//		js.SetScheduler(js.NewScheduler(js.SchedulerOptions{
//			MaxVMs: 8,
//			Loader: js.NewModuleLoader(),
//		}))
//	}
package js

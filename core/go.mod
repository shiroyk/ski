module github.com/shiroyk/cloudcat/core

go 1.20

require (
	github.com/dop251/goja v0.0.0-20230605162241-28ee0ee714f3
	github.com/shiroyk/cloudcat/plugin v0.2.0
	github.com/spf13/cast v1.5.1
	github.com/stretchr/testify v1.8.4
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/pprof v0.0.0-20230602150820-91b7bce49751 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.9.0 // indirect
)

replace github.com/shiroyk/cloudcat/plugin => ../plugin

module github.com/shiroyk/cloudcat/core

go 1.20

require (
	github.com/dop251/goja v0.0.0-20230402114112-623f9dda9079
	github.com/shiroyk/cloudcat/plugin v0.2.0
	github.com/spf13/cast v1.5.0
	github.com/stretchr/testify v1.8.2
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.7.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.3.8 // indirect
)

replace (
	github.com/shiroyk/cloudcat/plugin => ../plugin
)
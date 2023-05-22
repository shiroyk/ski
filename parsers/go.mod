module github.com/shiroyk/cloudcat/parsers

go 1.20

require (
	github.com/PuerkitoBio/goquery v1.8.1
	github.com/antchfx/htmlquery v1.3.0
	github.com/dlclark/regexp2 v1.10.0
	github.com/ohler55/ojg v1.18.5
	github.com/shiroyk/cloudcat/core v0.2.0
	github.com/shiroyk/cloudcat/plugin v0.2.0
	github.com/spf13/cast v1.5.1
	github.com/stretchr/testify v1.8.3
	golang.org/x/net v0.10.0
)

require (
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/antchfx/xpath v1.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dop251/goja v0.0.0-20230402114112-623f9dda9079 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29 // indirect
	golang.org/x/text v0.9.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/shiroyk/cloudcat/core => ../core
	github.com/shiroyk/cloudcat/plugin => ../plugin
)

module github.com/shiroyk/cloudcat

go 1.20

require (
	github.com/PuerkitoBio/goquery v1.8.1
	github.com/antchfx/htmlquery v1.3.0
	github.com/dlclark/regexp2 v1.10.0
	github.com/dop251/goja v0.0.0-20230919151941-fc55792775de
	github.com/ohler55/ojg v1.19.4
	github.com/shiroyk/cloudcat/plugin v0.4.0
	github.com/spf13/cast v1.5.1
	github.com/stretchr/testify v1.8.4
	golang.org/x/crypto v0.14.0
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9
	golang.org/x/net v0.16.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/antchfx/xpath v1.2.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4-0.20211119122758-180fcef48034+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/pprof v0.0.0-20230926050212-f7f687d19a98 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.13.0 // indirect
)

replace github.com/shiroyk/cloudcat/plugin => ./plugin

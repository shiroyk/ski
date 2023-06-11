module github.com/shiroyk/cloudcat

go 1.20

require (
	github.com/shiroyk/cloudcat/core v0.3.0
	github.com/shiroyk/cloudcat/plugin v0.3.0
)

require (
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/shiroyk/cloudcat/core => ./core
	github.com/shiroyk/cloudcat/plugin => ./plugin
)

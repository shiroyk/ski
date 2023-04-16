module github.com/shiroyk/cloudcat

go 1.20

require (
	github.com/shiroyk/cloudcat/core v0.0.0-00010101000000-000000000000
	github.com/shiroyk/cloudcat/plugin v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/shiroyk/cloudcat/core => ./core
	github.com/shiroyk/cloudcat/plugin => ./plugin
)

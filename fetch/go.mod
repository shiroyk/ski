module github.com/shiroyk/cloudcat/fetch

go 1.20

require (
	github.com/andybalholm/brotli v1.0.5
	github.com/refraction-networking/utls v1.3.2
	github.com/shiroyk/cloudcat/core v0.3.0
	github.com/stretchr/testify v1.8.4
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1
	golang.org/x/net v0.11.0
	golang.org/x/text v0.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gaukas/godicttls v0.0.3 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shiroyk/cloudcat/plugin v0.3.0 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	golang.org/x/crypto v0.10.0 // indirect
	golang.org/x/sys v0.9.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/shiroyk/cloudcat/core => ../core
	github.com/shiroyk/cloudcat/plugin => ../plugin
)

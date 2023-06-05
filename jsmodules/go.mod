module github.com/shiroyk/cloudcat/jsmodules

go 1.20

require (
	github.com/dop251/goja v0.0.0-20230402114112-623f9dda9079
	github.com/shiroyk/cloudcat/core v0.2.0
	github.com/shiroyk/cloudcat/fetch v0.2.0
	github.com/shiroyk/cloudcat/plugin v0.2.0
	github.com/spf13/cast v1.5.1
	github.com/stretchr/testify v1.8.4
	golang.org/x/crypto v0.9.0
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.8.1 // indirect
	github.com/gaukas/godicttls v0.0.3 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4-0.20211119122758-180fcef48034+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/refraction-networking/utls v1.3.2 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/shiroyk/cloudcat/core => ../core
	github.com/shiroyk/cloudcat/fetch => ../fetch
	github.com/shiroyk/cloudcat/plugin => ../plugin
)

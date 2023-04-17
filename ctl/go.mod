module github.com/shiroyk/cloudcat/ctl

go 1.20

require (
	github.com/go-chi/chi/v5 v5.0.8
	github.com/shiroyk/cloudcat/core v0.1.0-beta
	github.com/shiroyk/cloudcat/fetch v0.1.0-beta
	github.com/shiroyk/cloudcat/jsmodules v0.1.0-beta
	github.com/shiroyk/cloudcat/parsers v0.1.0-beta
	github.com/shiroyk/cloudcat/plugin v0.1.0-beta
	github.com/spf13/cobra v1.7.0
	github.com/stretchr/testify v1.8.2
	go.etcd.io/bbolt v1.3.7
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/PuerkitoBio/goquery v1.8.1 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/antchfx/htmlquery v1.3.0 // indirect
	github.com/antchfx/xpath v1.2.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.9.0 // indirect
	github.com/dop251/goja v0.0.0-20230402114112-623f9dda9079 // indirect
	github.com/gaukas/godicttls v0.0.3 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4-0.20211119122758-180fcef48034+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/ohler55/ojg v1.18.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/refraction-networking/utls v1.3.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/crypto v0.8.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
)

replace (
	github.com/shiroyk/cloudcat/core => ../core
	github.com/shiroyk/cloudcat/fetch => ../fetch
	github.com/shiroyk/cloudcat/jsmodules => ../jsmodules
	github.com/shiroyk/cloudcat/parsers => ../parsers
	github.com/shiroyk/cloudcat/plugin => ../plugin
)

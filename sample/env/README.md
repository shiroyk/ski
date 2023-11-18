# Env Plugin
A cloudcat js plugin for reading environment variables.
### Build the plugin
```shell
go build -buildmode=plugin -o env.so
```
### Plugin usage
```shell
export FOO=BAR
cat << EOF | cloudcat -p $(pwd) -d -s -
require("cloudcat/env").get("FOO")
EOF
# "BAR"
```
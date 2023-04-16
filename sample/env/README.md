# Env Plugin
A cloudcat js plugin for reading environment variables.
### Build the plugin
```shell
go build -buildmode=plugin -o env.so
```
### Plugin path configuration
```shell
cat << EOF | > ./config.yaml
plugin:
    path: ./
EOF
```
### Plugin usage
```shell
export FOO=BAR
cat << EOF | cloudcat --config ./config.yaml run -s -
require("cloudcat/env").get("FOO")
EOF
# "BAR"
```
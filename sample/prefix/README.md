# Prefix Plugin
A cloudcat parser plugin for adding string prefix.
### Build the plugin
```shell
go build -buildmode=plugin -o prefix.so
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
cat << EOF | cloudcat --config ./config.yaml run -s -
cat.getString("prefix", "...", "test");
EOF
# "...test"
```
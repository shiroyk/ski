# Prefix Plugin
A cloudcat parser plugin for adding string prefix.
### Build the plugin
```shell
go build -buildmode=plugin -o prefix.so
```
### Plugin usage
```shell
cat << EOF | cloudcat --plugin $(pwd) run -s -
cat.getString("prefix", "...", "test");
EOF
# "...test"
```
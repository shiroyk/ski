// Package common provides the Web and Node.js primitives shared by the
// built-in modules: the DOMException global, the Node.js style error codes and
// the WebIDL dictionary conversion used by the module APIs.
package common

import "github.com/shiroyk/ski/modules"

func init() {
	modules.Register("common", modules.Global{
		"DOMException": new(DOMException),
	})
}

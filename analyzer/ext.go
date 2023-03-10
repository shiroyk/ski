package analyzer

import (
	_ "github.com/shiroyk/cloudcat/js/modules/cache" // init the extensions
	_ "github.com/shiroyk/cloudcat/js/modules/cookie"
	_ "github.com/shiroyk/cloudcat/js/modules/crypto"
	_ "github.com/shiroyk/cloudcat/js/modules/encoding"
	_ "github.com/shiroyk/cloudcat/js/modules/http"
	_ "github.com/shiroyk/cloudcat/parser/parsers/gq"
	_ "github.com/shiroyk/cloudcat/parser/parsers/js"
	_ "github.com/shiroyk/cloudcat/parser/parsers/json"
	_ "github.com/shiroyk/cloudcat/parser/parsers/regex"
	_ "github.com/shiroyk/cloudcat/parser/parsers/xpath"
)

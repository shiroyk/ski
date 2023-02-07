package analyzer

import (
	"testing"

	"github.com/shiroyk/cloudcat/parser"
)

func BenchmarkAnalyzer(b *testing.B) {
	ctx := parser.NewContext(parser.Options{Url: "https://localhost"})
	b.StartTimer()
	analyzer := NewAnalyzer()
	for i := 0; i < b.N; i++ {
		analyzer.ExecuteSchema(ctx, schema, content)
	}
	b.StopTimer()
}

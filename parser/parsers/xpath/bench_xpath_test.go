package xpath

import (
	"testing"
)

func BenchmarkParser(b *testing.B) {
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := xpath.GetString(ctx, ``, `//div[@class="body"]/ul//a/@title`)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
}

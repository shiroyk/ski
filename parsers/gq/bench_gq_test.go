package gq

import (
	"testing"
)

func BenchmarkParser(b *testing.B) {
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := gq.GetString(ctx, content, `.body ul a -> parent(li) -> slice(0) -> next(.selected) -> join(-)`)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
}

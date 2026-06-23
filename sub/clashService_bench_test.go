package sub

import "testing"

// BenchmarkConvertToClashMeta measures the Clash/YAML rendering hot path for a
// 6-protocol outbound set (TLS/reality/utls/transport variety). Used to decide
// whether Clash generation warrants optimization before touching it.
func BenchmarkConvertToClashMeta(b *testing.B) {
	svc := &ClashService{}
	outbounds := diverseClashOutbounds()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.ConvertToClashMeta(&outbounds, basicClashConfig); err != nil {
			b.Fatal(err)
		}
	}
}

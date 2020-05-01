package mph

import (
	"bufio"
	"bytes"
	"hash/maphash"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	sampleData = map[uint64]uint64{
		1:  2,
		3:  4,
		5:  6,
		7:  8,
		9:  10,
		11: 12,
		13: 14,
	}
)

var (
	words []uint64
)

func init() {
	f, err := os.Open("/usr/share/dict/words")
	if err != nil {
		panic(err)
	}
	r := bufio.NewReader(f)
	seen := map[uint64]struct{}{}
	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		var h maphash.Hash
		h.Write(line)
		v := h.Sum64()
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		words = append(words, v)
	}
}

func TestCHDBuilder(t *testing.T) {
	b := Builder()
	for k, v := range sampleData {
		b.Add(k, v)
	}
	c, err := b.Build()
	assert.NoError(t, err)
	assert.Equal(t, 7, len(c.keys))
	for k, v := range sampleData {
		v2, ok := c.Get(k)
		assert.True(t, ok)
		assert.Equal(t, v, v2)
	}
	_, ok := c.Get(123)
	assert.False(t, ok)
}

func TestCHDSerialization(t *testing.T) {
	cb := Builder()
	for _, v := range words {
		cb.Add(v, v)
	}
	m, err := cb.Build()
	assert.NoError(t, err)
	w := &bytes.Buffer{}
	err = m.Write(w)
	assert.NoError(t, err)

	n, err := Mmap(w.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, n.r, m.r)
	assert.Equal(t, n.indices, m.indices)
	assert.Equal(t, n.keys, m.keys)
	assert.Equal(t, n.values, m.values)
	for _, v := range words {
		v2, ok := n.Get(v)
		assert.True(t, ok)
		assert.Equal(t, v, v2)
	}
}

func TestCHDSerialization_empty(t *testing.T) {
	cb := Builder()
	m, err := cb.Build()
	assert.NoError(t, err)
	w := &bytes.Buffer{}
	err = m.Write(w)
	assert.NoError(t, err)

	n, err := Mmap(w.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, n.r, m.r)
	assert.Equal(t, n.indices, m.indices)
	assert.Equal(t, n.keys, m.keys)
	assert.Equal(t, n.values, m.values)
}

func TestCHDSerialization_one(t *testing.T) {
	cb := Builder()
	cb.Add(123, 456)
	m, err := cb.Build()
	assert.NoError(t, err)
	w := &bytes.Buffer{}
	err = m.Write(w)
	assert.NoError(t, err)

	n, err := Mmap(w.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, n.r, m.r)
	assert.Equal(t, n.indices, m.indices)
	assert.Equal(t, n.keys, m.keys)
	assert.Equal(t, n.values, m.values)
}

func BenchmarkBuiltinMap(b *testing.B) {
	d := make(map[uint64]uint64, len(words))
	for _, k := range words {
		d[k] = k
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d[words[i%len(words)]]
	}
}

func BenchmarkBuiltinMapBuild(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d := make(map[uint64]uint64, len(words))
		for _, k := range words {
			d[k] = k
		}
		_ = d[words[i%len(words)]]
	}
}

func BenchmarkCHD(b *testing.B) {
	mph := Builder()
	for _, k := range words {
		mph.Add(k, k)
	}
	h, _ := mph.Build()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = h.Get(words[i%len(words)])
	}
}

func BenchmarkCHDBuild(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mph := Builder()
		for _, k := range words {
			mph.Add(k, k)
		}
		h, _ := mph.Build()
		_, _ = h.Get(words[i%len(words)])
	}
}

// TestMemoryUsage is for just outputting memory usage.
//
//   go test -run TestMemoryUsage -v
func TestMemoryUsage(t *testing.T) {
	mph := Builder()
	for _, k := range words {
		mph.Add(k, k)
	}
	h, _ := mph.Build()

	su64 := func(r []uint64) int {
		// 3 is for slice ptr, len, cap
		return 3*8 + len(r)*8
	}
	su16 := func(r []uint16) int {
		// 3 is for slice ptr, len, cap
		return 3*8 + len(r)*2
	}
	t.Logf("CHD of size %d uses %d bytes", len(words), su64(h.r)+su16(h.indices)+su64(h.keys)+su64(h.values))

	var mbefore, mafter runtime.MemStats
	runtime.ReadMemStats(&mbefore)
	d := make(map[uint64]uint64, len(words))
	for _, k := range words {
		d[k] = k
	}
	runtime.ReadMemStats(&mafter)
	t.Logf("Map of size %d uses %d bytes", len(words), mafter.TotalAlloc-mbefore.TotalAlloc)
}

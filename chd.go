// Package mph is a Go implementation of the compress, hash and displace (CHD)
// minimal perfect hash algorithm.
//
// See http://cmph.sourceforge.net/papers/esa09.pdf for details.
//
// To create and serialize a hash table:
//
//		b := mph.Builder()
// 		for k, v := range data {
// 			b.Add(k, v)
// 		}
// 		h, _ := b.Build()
// 		w, _ := os.Create("data.idx")
// 		b, _ := h.Write(w)
//
// To read from the hash table:
//
//		r, _ := os.Open("data.idx")
//		h, _ := h.Read(r)
//
//		v := h.Get([]byte("some key"))
//		if v == nil {
//		    // Key not found
//		}
//
// MMAP is also indirectly supported, by deserializing from a byte
// slice and slicing the keys and values.
//
// See https://github.com/alecthomas/mph for source.
package mph

import (
	"encoding/binary"
	"io"
	"io/ioutil"
)

// CHD hash table lookup.
type CHD struct {
	// Random hash function table.
	r []uint64
	// Array of indices into hash function table r. We assume there aren't
	// more than 2^16 hash functions O_o
	indices []uint16
	// Final table of values.
	keys   []uint64
	values []uint64
}

func hasher(data uint64) uint64 {
	return data
}

// Read a serialized CHD.
func Read(r io.Reader) (*CHD, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Mmap(b)
}

// Mmap creates a new CHD aliasing the CHD structure over an existing byte region (typically mmapped).
func Mmap(b []byte) (*CHD, error) {
	c := &CHD{}

	bi := &sliceReader{b: b}

	// Read vector of hash functions.
	rl := bi.ReadInt()
	c.r = bi.ReadUint64Array(rl)

	// Read hash function indices.
	il := bi.ReadInt()
	c.indices = bi.ReadUint16Array(il)

	el := bi.ReadInt()
	if el == 0 {
		c.keys = []uint64{}
		c.values = []uint64{}
		return c, nil
	}

	c.keys = bi.ReadUint64Array(el)
	c.values = bi.ReadUint64Array(el)

	return c, nil
}

// Get an entry from the hash table.
func (c *CHD) Get(key uint64) (uint64, bool) {
	r0 := c.r[0]
	h := hasher(key) ^ r0
	i := h % uint64(len(c.indices))
	ri := c.indices[i]
	// This can occur if there were unassigned slots in the hash table.
	if ri >= uint16(len(c.r)) {
		return 0, false
	}
	r := c.r[ri]
	ti := (h ^ r) % uint64(len(c.keys))
	// fmt.Printf("r[0]=%d, h=%d, i=%d, ri=%d, r=%d, ti=%d\n", c.r[0], h, i, ri, r, ti)
	k := c.keys[ti]
	if k != key {
		return 0, false
	}
	v := c.values[ti]
	return v, true
}

func (c *CHD) Len() int {
	return len(c.keys)
}

// Iterate over entries in the hash table.
func (c *CHD) Iterate() *Iterator {
	if len(c.keys) == 0 {
		return nil
	}
	return &Iterator{c: c}
}

// Serialize the CHD. The serialized form is conducive to mmapped access. See
// the Mmap function for details.
func (c *CHD) Write(w io.Writer) error {
	write := func(nd ...interface{}) error {
		for _, d := range nd {
			if err := binary.Write(w, binary.LittleEndian, d); err != nil {
				return err
			}
		}
		return nil
	}

	data := []interface{}{
		uint32(len(c.r)), c.r,
		uint32(len(c.indices)), c.indices,
		uint32(len(c.keys)), c.keys, c.values,
	}

	if err := write(data...); err != nil {
		return err
	}

	return nil
}

type Iterator struct {
	i int
	c *CHD
}

func (c *Iterator) Get() (key uint64, value uint64) {
	return c.c.keys[c.i], c.c.values[c.i]
}

func (c *Iterator) Next() *Iterator {
	c.i++
	if c.i >= len(c.c.keys) {
		return nil
	}
	return c
}

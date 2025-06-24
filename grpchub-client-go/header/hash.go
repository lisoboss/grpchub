package header

import (
	"hash/fnv"
	"sort"
)

type HashHeader Header

func (m HashHeader) Hash() uint64 {
	h := fnv.New64a()
	for k, vs := range m {
		h.Write([]byte(k))
		for _, v := range vs {
			h.Write([]byte{0})
			h.Write([]byte(v))
		}
	}

	return h.Sum64()
}

// Shash sort and hash
func (m HashHeader) Shash() uint64 {
	h := fnv.New64a()
	// 为了稳定性，先对 key 排序
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		h.Write([]byte(k))
		vals := m[k]
		// 可选：sort vals，如果你不要求顺序敏感
		for _, v := range vals {
			h.Write([]byte{0}) // 分隔符，防止 "ab" + "c" 和 "a" + "bc" 混淆
			h.Write([]byte(v))
		}
	}
	return h.Sum64()
}

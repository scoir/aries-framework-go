package indy

import (
	"sort"
)

type SortedMap struct {
	Keys []string
}

func NewSortedMap(m map[string]interface{}) *SortedMap {
	k := make([]string, len(m))
	i := 0
	for key, _ := range m {
		k[i] = key
		i++
	}
	sort.Strings(k)
	return &SortedMap{Keys: k}
}

package storage

import (
	"sort"
)

// SortByCreatedAt returns the list of storage objects sorted by an
// object's created at timestamp (in seconds).
func SortByCreatedAt(objs []*Object) {
	sort.SliceStable(objs, func(i, j int) bool {
		ti := objs[i].GetCreatedAt().GetSeconds()
		tj := objs[j].GetCreatedAt().GetSeconds()
		return ti < tj
	})
}

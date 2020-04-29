package kube

import (
	"sort"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

type ByName []runtime.Object

func (a ByName) Len() int      { return len(a) }
func (a ByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool {
	o1 := a[i]
	o2 := a[j]
	a1, _ := meta.Accessor(o1)
	a2, _ := meta.Accessor(o2)
	if a1 == nil || a2 == nil {
		return false
	}
	return a1.GetName() < a2.GetName()
}

func SortRuntimeObjectsByName(objects []runtime.Object) {
	sort.Sort(ByName(objects))
}

func SortListWatchByName(listWatch *cache.ListWatch) {
	oldFn := listWatch.ListFunc
	listWatch.ListFunc = func(options metav1.ListOptions) (runtime.Object, error) {
		result, err := oldFn(options)
		if err == nil {
			initialItems, err := meta.ExtractList(result)
			if err == nil {
				SortRuntimeObjectsByName(initialItems)
				meta.SetList(result, initialItems) //nolint:errcheck
			}
		}
		return result, err
	}
}

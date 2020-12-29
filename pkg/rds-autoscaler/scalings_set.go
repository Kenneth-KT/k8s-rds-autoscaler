package autoscaler

import (
	"encoding/json"
	"fmt"
	"sort"
)

type ScalingsSet struct {
	sortedScalings []ScalingCapacity
	mapByScale     map[string]ScalingCapacity
}

func ParseScalingsSet(jsonString string) *ScalingsSet {
	var scalings []ScalingCapacity
	err := json.Unmarshal([]byte(jsonString), &scalings)
	if err != nil {
		panic(err)
	}

	return NewScalingsSet(scalings)
}

func NewScalingsSet(scalings []ScalingCapacity) *ScalingsSet {
	sortedScalings := immutableSort(scalings, func(ls []ScalingCapacity, i, j int) bool {
		return ls[i].ConnectionLimit < ls[j].ConnectionLimit
	})

	mapByScale := make(map[string]ScalingCapacity)
	for _, element := range scalings {
		mapByScale[element.Scale] = element
	}

	return &ScalingsSet{
		sortedScalings: sortedScalings,
		mapByScale:     mapByScale,
	}
}

func (set *ScalingsSet) FitScale(connectionCount int) *ScalingCapacity {
	list := set.sortedScalings
	index := sort.Search(len(list), func(i int) bool { return list[i].ConnectionLimit >= connectionCount })
	if index < len(list) {
		return &list[index]
	} else {
		// connection count exceeded connection limits in all available scalings
		return nil
	}
}

func (set *ScalingsSet) GetConnectionLimit(scale string) int {
	return set.mapByScale[scale].ConnectionLimit
}

func (set *ScalingsSet) Describe() string {
	jsonString, _ := json.Marshal(set.sortedScalings)
	return fmt.Sprintf("(%d elements) %s", len(set.sortedScalings), string(jsonString))
}

func immutableSort(input []ScalingCapacity, less func(list []ScalingCapacity, i, j int) bool) []ScalingCapacity {
	sorted := make([]ScalingCapacity, len(input))
	copy(sorted, input)
	sort.SliceStable(sorted, func(i, j int) bool {
		return less(sorted, i, j)
	})
	return sorted
}

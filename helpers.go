package main

import "sort"

func getSortedHabits(m map[string]HabitObject) []HabitObject {
	h := []HabitObject{}
	for _, v := range m {
		h = append(h, v)
	}

	sort.Slice(h, func(i, j int) bool {
		return h[i].Habit.Order < h[j].Habit.Order // Sort by Age
	})

	return h
}

func getHabitHistoryMap(hh []History) map[int64]map[string]History {
	dtMap := make(map[int64]map[string]History)
	for i := range hh {
		dt := hh[i].Date.Unix()
		_, ok := dtMap[dt]
		if !ok {
			dtMap[dt] = make(map[string]History)
		}

		_, ok = dtMap[dt][hh[i].HabitID]
		if !ok {
			dtMap[dt][hh[i].HabitID] = hh[i]
		}
	}

	return dtMap
}

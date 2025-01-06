package main

import (
	"bytes"
	_ "embed"
	"log"
	"text/template"
	"time"

	"github.com/charles-m-knox/go-uuid"
	"gorm.io/gorm"
)

var habitTemplate *template.Template

type Habit struct {
	ID   string `gorm:"type:uuid;primarykey"`
	Name string
	Done bool `gorm:"-"` // UI state only
	// If False, the habit isn't shown on the UI and isn't considered for
	// completion.
	Active bool
	Order  int
}

type History struct {
	ID      string `gorm:"type:uuid;primarykey"`
	HabitID string `gorm:"type:uuid;index"`
	Date    time.Time
	Done    bool
	// If False, the habit isn't shown on the UI and isn't considered for
	// completion.
	Active bool
	Name   string
	Order  int
}

// TODO: remove this later on once I have a proper UI for creating new habits
var habitDefinitions = []Habit{
	{Order: 0, ID: "4d208dd2-9d8f-4ebd-a043-b968da4abd60", Done: false, Active: true, Name: "Eat well"},
	{Order: 1, ID: "d2f1b1c4-2acc-4fcc-af8c-8f75ce03e3c1", Done: false, Active: true, Name: "Exercise"},
	{Order: 2, ID: "4d208dd2-9d8f-4ebd-a043-b968da4abd62", Done: false, Active: true, Name: "Floss"},
	{Order: 3, ID: "4d208dd2-9d8f-4ebd-a043-b968da4abd64", Done: false, Active: true, Name: "Stretch"},
	{Order: 4, ID: "4d208dd2-9d8f-4ebd-a043-b968da4abd68", Done: false, Active: false, Name: "Old habit that I completed"},
}

// Gets passed to the habit.ui template, rendered, then presented to the user in
// the UI
type HabitObject struct {
	Habit                  Habit
	ToggleID               string
	HabitID                string
	ProgressID             string
	TwoWeekCompletionCount int
}

func setupHabitTemplate() {
	habitTemplate = template.Must(template.New("habit").Parse(habitXML))
}

func getHabitObject(habit HabitObject) string {
	var buf bytes.Buffer
	err := habitTemplate.Execute(&buf, habit)
	if err != nil {
		log.Fatalf("failed to getHabitObject: %v", err.Error())
	}

	return buf.String()
}

func getHabitDefinitions(db *gorm.DB) (map[string]Habit, error) {
	q := []Habit{}
	m := make(map[string]Habit)

	r := db.Where(&Habit{
		Active: true,
	}).Find(&q)
	if r.Error != nil {
		return m, r.Error
	}

	for _, h := range q {
		m[h.ID] = h
	}

	return m, nil
}

func setHabits(db *gorm.DB, habits []Habit) error {
	for i := range habits {
		log.Printf("saving habit %v...", habits[i].ID)

		r := db.Save(habits[i])

		if r.Error != nil {
			return r.Error
		}
	}

	return nil
}

func saveHabitHistory(db *gorm.DB, habitHistory History) error {
	log.Printf("saving habit %v...", habitHistory.HabitID)

	h := History{}
	s := db.Where("date = ? AND habit_id = ?", habitHistory.Date, habitHistory.HabitID).Find(&h)
	if s.Error != nil {
		return s.Error
	}

	if h.ID != "" {
		habitHistory.ID = h.ID
	} else {
		habitHistory.ID = uuid.New()
	}

	r := db.Save(habitHistory)

	if r.Error != nil {
		return r.Error
	}

	return nil
}

func getHabitsForDate(db *gorm.DB, dt time.Time) ([]History, error) {
	q := []History{}
	// h := []Habit{}

	// each time value stored in the DB is UTC at 00:00 and must be parsed as
	// UTC, but then converted to e.g. 2024-10-03 in the user's current timezone

	// query back 2 weeks to get the completion history
	daysAgo := dt.AddDate(0, 0, -14)

	// r := db.Where("date >= ? AND date <= ? AND active = ?", daysAgo, dt, true).Find(&q)
	r := db.Where("date >= ? AND date <= ?", daysAgo, dt).Find(&q)
	if r.Error != nil {
		return q, r.Error
	}

	// dayOf := dt.AddDate(0, 0, 1)
	// s := db.Where("date >= ? AND date <= ? AND active = ?", dt, dayOf, true).Find(&h)
	// if s.Error != nil {
	// 	return q, s.Error
	// }

	return q, nil
}

func getHabitHistoryEntryForDate(habitID string, habitHistoryMap map[int64]map[string]History, dt int64) (History, bool) {
	result := History{}

	_, ok := habitHistoryMap[dt]
	if !ok {
		return result, false
	}

	result, ok = habitHistoryMap[dt][habitID]
	return result, ok
}

func countHabits(hist []History, habitMap map[string]Habit) map[string]HabitObject {
	totals := make(map[string]int)
	for i := range hist {
		_, ok := totals[hist[i].HabitID]
		if !ok {
			totals[hist[i].HabitID] = 0
		}

		if hist[i].Done {
			totals[hist[i].HabitID] += 1
		}
	}

	// populate the habit map for faster lookups later
	// habitMap := make(map[string]Habit)
	// for i := range all {
	// 	_, ok := habitMap[all[i].ID]
	// 	if !ok {
	// 		habitMap[all[i].ID] = all[i]
	// 	}
	// }

	results := make(map[string]HabitObject)
	for k, v := range habitMap {
		results[k] = HabitObject{
			Habit:                  v,
			TwoWeekCompletionCount: totals[k],
		}
	}
	for i := range hist {
		habit, ok := habitMap[hist[i].HabitID]
		if !ok {
			log.Printf("found a habit that doesn't exist or is inactive: %v", hist[i].HabitID)
			continue
		}

		if totals[hist[i].HabitID] > 14 {
			totals[hist[i].HabitID] = 14
		}

		results[hist[i].HabitID] = HabitObject{
			Habit:                  habit,
			TwoWeekCompletionCount: totals[hist[i].HabitID],
		}
	}

	return results
}

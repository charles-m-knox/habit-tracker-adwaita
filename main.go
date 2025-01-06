package main

import (
	"fmt"
	"log"
	"os"
	"time"

	_ "embed"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"gorm.io/gorm"
)

//go:embed ui/habits.ui
var uiXML string

//go:embed ui/habit.ui
var habitXML string

//go:embed ui/titlebar.ui
var titlebarXML string

type AppState struct {
	DB *gorm.DB
}

var App AppState

func main() {
	App.DB = getDB()

	err := setHabits(App.DB, habitDefinitions)
	if err != nil {
		log.Fatalf("failed to set habits in main(): %v", err.Error())
	}

	app := adw.NewApplication("com.charlesmknox.habits", 0)

	setupHabitTemplate()

	app.ConnectActivate(func() { activate(app) })

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate(app *adw.Application) {
	window := gtk.NewApplicationWindow(&app.Application)
	// window := adw.NewApplicationWindow(&app.Application)

	builder := gtk.NewBuilderFromString(uiXML)
	err := builder.AddFromString(titlebarXML)
	if err != nil {
		log.Fatalf("failed to load titlebarXML: %v", err.Error())
	}

	root := builder.GetObject("root").Cast().(*gtk.Box)
	clamp := builder.GetObject("clamp").Cast().(*adw.Clamp)
	// flowbox := builder.GetObject("flowbox").Cast().(*gtk.FlowBox)
	listbox := builder.GetObject("listbox").Cast().(*gtk.ListBox)
	todayBtn := builder.GetObject("todayBtn").Cast().(*adw.PreferencesRow)
	calendar := builder.GetObject("calendar").Cast().(*gtk.Calendar)
	titlebar := builder.GetObject("titlebar").Cast().(*adw.HeaderBar)
	about := getAboutDialog()

	todayBtn.Connect("activated", func() {
		now := time.Now()
		calendar.SetDay(now.Day())
		calendar.SetMonth(int(now.Month()) - 1)
		calendar.SetYear(now.Year())
	})

	// actionGroup := gio.NewSimpleActionGroup()
	aboutAction := gio.NewSimpleAction("about", nil)
	aboutAction.Connect("activate", func() {
		about.Present(window)
	})

	app.AddAction(aboutAction)

	// flowbox.Connect("child-activated", func(self any, item any) {
	// 	log.Println(self, item)
	// 	log.Println("item activated")
	// })

	calendar.SetShowDayNames(true)
	calendar.SetShowHeading(true)
	calendar.SetShowWeekNumbers(true)

	now := time.Now()
	nowYear, nowMonth, nowDay := now.Date()
	nowDate := time.Date(nowYear, nowMonth, nowDay, 0, 0, 0, 0, time.UTC)
	calendar.SetDay(nowDate.Day())
	calendar.SetMonth(int(nowDate.Month()) - 1)
	calendar.SetYear(nowDate.Year())

	definedHabits, err := getHabitDefinitions(App.DB)
	if err != nil {
		log.Fatalf("failed to get habit definitions: %v", err.Error())
	}

	var updateHabitView func(time.Time)

	updateHabitView = func(dt time.Time) {
		dtUnix := dt.Unix()

		habitHistory, err := getHabitsForDate(App.DB, dt)
		if err != nil {
			log.Fatalf("failed to get habits for date: %v", err.Error())
		}
		habitMap := countHabits(habitHistory, definedHabits)
		habitHistoryMap := getHabitHistoryMap(habitHistory)

		calendar.ClearMarks()
		for k := range habitHistoryMap {
			tv := time.Unix(k, 0)
			tvd := time.Date(tv.Year(), tv.Month(), tv.Day(), 0, 0, 0, 0, time.UTC)
			if tvd.Month() == time.Month(calendar.Month()+1) {
				calendar.MarkDay(uint(tvd.Day() + 1))
			}
		}

		listbox.RemoveAll()
		listbox.Append(todayBtn)

		// ids := getSortedKeys(habits)
		habits := getSortedHabits(habitMap)

		for i, habit := range habits {
			// fbChild := adw.NewBin()
			// fbChild.SetSizeRequest(100, 100)

			// binChild := gtk.NewLabel(habit.name)
			// binChild.SetHExpand(true)
			// binChild.SetVExpand(true)

			// fbChild.SetChild(binChild)
			// flowbox.Append(fbChild)

			toggleID := fmt.Sprintf("habit_toggle_%v", i)
			habitID := fmt.Sprintf("habit_row_%v", i)
			progressID := fmt.Sprintf("habit_progress_%v", i)

			habit.ToggleID = toggleID
			habit.HabitID = habitID
			habit.ProgressID = progressID

			err := builder.AddObjectsFromString(getHabitObject(habit), []string{habitID, progressID})
			if err != nil {
				log.Fatalf("failed to add habit %v: %v", i, err.Error())
			}

			habitRow := builder.GetObject(habitID).Cast().(*adw.ActionRow)
			habitToggle := builder.GetObject(toggleID).Cast().(*gtk.ToggleButton)
			habitProgress := builder.GetObject(progressID).Cast().(*gtk.LevelBar)

			habitRow.SetActivatableWidget(habitToggle)

			habitHistoryEntry, ok := getHabitHistoryEntryForDate(habit.Habit.ID, habitHistoryMap, dtUnix)

			// habitToggle.SetCSSClasses([]string{"error"})

			if (ok && habitHistoryEntry.Done) || habit.Habit.Done {
				habitToggle.SetCSSClasses([]string{"success"})
				habitToggle.SetIconName("checkbox-checked-symbolic")
				habitToggle.SetActive(true)
			} else {
				habitToggle.SetCSSClasses([]string{"error"})
				habitToggle.SetIconName("process-stop-symbolic")
				habitToggle.SetActive(false)
			}

			connectToggled := func() {
				state := false
				newProgress := habit.TwoWeekCompletionCount
				if habitToggle.Active() {
					state = true
					newProgress += 1
					habitToggle.SetCSSClasses([]string{"success"})
					habitToggle.SetIconName("checkbox-checked-symbolic")
				} else {
					newProgress -= 1
					habitToggle.SetCSSClasses([]string{"error"})
					habitToggle.SetIconName("process-stop-symbolic")
				}

				t := time.Unix(calendar.Date().ToUnix(), 0)
				if err != nil {
					log.Printf("failed to parse time: %v", err.Error())
					return
				}

				year, month, day := t.Date()
				err = saveHabitHistory(App.DB, History{
					// ID:      uuid.New(),
					HabitID: habit.Habit.ID,
					Date:    time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
					Done:    state,
					Active:  true,
					Name:    habit.Habit.Name,
					Order:   habit.Habit.Order,
				})
				if err != nil {
					log.Printf("failed to save habit %v: %v", habit.Habit.ID, err.Error())
				}

				if newProgress > 14 {
					newProgress = 14
				} else if newProgress < 0 {
					newProgress = 0
				}

				habitProgress.SetValue(float64(newProgress))
				updateHabitView(dt)
			}

			habitProgress.SetValue(float64(habit.TwoWeekCompletionCount))

			habitToggle.ConnectToggled(connectToggled)
			habitRow.SetSubtitle(fmt.Sprintf("%v-%02d-%02d %v/14", dt.Year(), int(dt.Month()), dt.Day(), habit.TwoWeekCompletionCount))
			// habitRow.Connect("activate", connectToggled)

			listbox.Append(habitRow)
			listbox.Append(habitProgress)
		}
	}

	updateHabitView(nowDate)

	getNormalDate := func(timestamp int64) time.Time {
		y, m, d := time.Unix(timestamp, 0).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}

	calendar.Connect("notify::day", func() {
		updateHabitView(getNormalDate(calendar.Date().ToUnix()))
	})
	calendar.Connect("notify::month", func() {
		updateHabitView(getNormalDate(calendar.Date().ToUnix()))
	})
	calendar.Connect("notify::year", func() {
		updateHabitView(getNormalDate(calendar.Date().ToUnix()))
	})

	clamp.SetObjectProperty("maximum-size", 600)

	// window.SetContent(root) // for adw window
	window.SetChild(root) // for gtk window

	window.SetTitle("Habits")
	window.SetTitlebar(titlebar)
	window.SetDefaultSize(500, 600)
	window.Present()
}

func getAboutDialog() *adw.AboutDialog {
	a := adw.NewAboutDialog()
	a.SetApplicationIcon("application-x-executable")
	a.SetApplicationName("Habit Tracker")
	a.SetDeveloperName("Charles M. Knox")
	a.SetVersion("0.0.1")
	a.SetComments("A simple habit tracker.")
	a.SetWebsite("https://github.com/charles-m-knox/habit-tracker-adwaita")
	a.SetIssueURL("https://github.com/charles-m-knox/habit-tracker-adwaita")
	a.SetSupportURL("https://github.com/charles-m-knox/habit-tracker-adwaita")
	a.SetCopyright("© 2024 Charles M. Knox")
	a.SetLicenseType(gtk.LicenseAGPL30_Only)
	a.SetDevelopers([]string{"Charles M. Knox <habit-tracker-adwaita-contact@charlesmknox.com>"})
	a.SetArtists([]string{})
	a.SetTranslatorCredits("N/A")

	a.AddLink("Documentation", "https://github.com/charles-m-knox/habit-tracker-adwaita")

	return a
}

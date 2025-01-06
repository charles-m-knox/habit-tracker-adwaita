# Habit Tracker

A simple habit tracking application that leverages `libadwaita` and Gtk4.

Features a rolling two-week window into the past that shows your habit history at a glance.

## Screenshot

![Habit tracker in default view](./screenshot.png)

## Installation

```bash
sudo pacman -S libadwaita gtk4 go sqlite git
git clone https://github.com/charles-m-knox/habit-tracker-adwaita.git
go get -v && go build -v
./habit-tracker-adwaita
```

## Notes

- Currently lacks a built-in way to add/delete habits
  - you have to edit `habitDefinitions` in `habits.go`, rebuild, and rerun the app (until I get around to implementing something better later)
- This is not a polished or finished application, it was more of a proof of concept/example for me to use when writing libadwaita/gtk4 apps
- Use at your own risk, probably not stable
- libadwaita apps seem RAM-hungry (130MB consumed by this app)
- Dark mode may or may not work
- requires CGO due to sqlite dependency

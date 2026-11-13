package models

import (
	"fmt"
	"time"
	_ "time/tzdata"
)

// Embedded time-zone data also supports minimal Linux/Raspberry Pi installations.
var workshopTimeZone = func() *time.Location {
	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(err)
	}
	return location
}()

type Season struct {
	Kind        string    `json:"kind"`
	Label       string    `json:"label"`
	MountedType string    `json:"mounted_type"`
	StoredType  string    `json:"stored_type"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
}

func CurrentSeason(now time.Time) Season {
	now = now.In(workshopTimeZone)
	year := now.Year()
	if now.Month() >= time.April && now.Month() < time.October {
		return Season{Kind: "summer", Label: fmt.Sprintf("Sommer %d", year), MountedType: "Sommer", StoredType: "Winter", Start: time.Date(year, time.April, 1, 0, 0, 0, 0, workshopTimeZone), End: time.Date(year, time.October, 1, 0, 0, 0, 0, workshopTimeZone)}
	}
	if now.Month() < time.April {
		year--
	}
	return Season{Kind: "winter", Label: fmt.Sprintf("Winter %d/%02d", year, (year+1)%100), MountedType: "Winter", StoredType: "Sommer", Start: time.Date(year, time.October, 1, 0, 0, 0, 0, workshopTimeZone), End: time.Date(year+1, time.April, 1, 0, 0, 0, 0, workshopTimeZone)}
}

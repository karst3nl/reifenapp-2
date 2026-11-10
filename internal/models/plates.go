package models

import (
	_ "embed"
	"regexp"
	"strings"
)

//go:embed plate_districts.txt
var plateDistricts string

var platePattern = regexp.MustCompile(`^([A-ZÄÖÜ]{1,3})-([A-Z]{1,2}) ([1-9][0-9]{0,3}[EH]?)$`)

func validPlate(plate string) bool {
	match := platePattern.FindStringSubmatch(plate)
	if match == nil {
		return false
	}
	for _, district := range strings.Fields(plateDistricts) {
		if match[1] == district {
			return true
		}
	}
	return false
}

package utilities

import (
	"fmt"
	"time"
)

const layout = "2006-01-02"

func Year() int {
	now := time.Now()
	return now.Year()
}

func DateTime() string {
	currentTime := time.Now()
	return fmt.Sprintf("%d-%d-%d %d:%d:%d\n",
		currentTime.Year(),
		currentTime.Month(),
		currentTime.Day(),
		currentTime.Hour(),
		currentTime.Hour(),
		currentTime.Second())
}

func YearMonthDay(dateString string) (int, int, int) {
	date, err := time.Parse("2006-01-02", dateString)

	if err != nil {
		return 0, 0, 0
	}

	month := date.Month()
	year := date.Year()
	day := date.Day()

	return int(year), int(month), int(day)
}

func DateNow() string {
	currentTime := time.Now()

	return currentTime.Format(layout)
}

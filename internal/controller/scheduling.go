package kronosapp

import (
	"fmt"
	"github.com/KronosOrg/kronos-core/api/v1alpha1"
	"sort"
	"strconv"
	"strings"
	"time"
)

type SleepSchedule struct {
	now        time.Time
	StartSleep time.Time
	EndSleep   time.Time
	Weekdays   []time.Weekday
	Timezone   *time.Location
	Holidays   map[string][]time.Time
}

func getTime(literal string, location *time.Location) (time.Time, error) {
	now := time.Now().In(location)
	targetTime, err := time.Parse("15:04", literal)
	if err != nil {
		return time.Time{}, err
	}
	time := time.Date(now.Year(), now.Month(), now.Day(), targetTime.Hour(), targetTime.Minute(), 0, 0, location)
	return time, nil
}

func extractDatesFromHoliday(combinedDates string, location *time.Location) ([]time.Time, error) {
	var dateList []time.Time
	combinedDatesParts := strings.Split(combinedDates, "-")
	base := combinedDatesParts[0] + "-" + combinedDatesParts[1]
	days := strings.Split(combinedDatesParts[2], "/")
	for _, day := range days {
		formattedDate := base + "-" + day
		date, err := time.ParseInLocation("2006-01-02", formattedDate, location)
		if err != nil {
			return nil, err
		}
		dateList = append(dateList, date)
	}
	return dateList, nil
}

func extractHolidays(holidays []v1alpha1.Holiday, location *time.Location) (map[string][]time.Time, error) {
	var holidaysMap = make(map[string][]time.Time)
	var err error
	for _, holiday := range holidays {
		holidaysMap[holiday.Name], err = extractDatesFromHoliday(holiday.Date, location)
		if err != nil {
			return nil, err
		}
	}
	return holidaysMap, nil
}

func NewSleepSchedule(startSleep, endSleep string, weekDays string, timezone string, holidays []v1alpha1.Holiday) (*SleepSchedule, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	holidaysMap, err := extractHolidays(holidays, loc)
	if err != nil {
		return nil, err
	}
	weekdaySet := extractWeekdays(weekDays)
	weekdays := mapWeekdays(weekdaySet)
	start, err := getTime(startSleep, loc)
	if err != nil {
		return nil, err
	}
	end, err := getTime(endSleep, loc)
	if err != nil {
		return nil, err
	}
	now := time.Now().In(loc)
	if end.Before(start) && now.After(end) {
		end = end.Add(24 * time.Hour)
	}

	if end.Before(start) && now.Before(end) {
		start = start.Add(-24 * time.Hour)
	}

	return &SleepSchedule{
		now:        now,
		StartSleep: start,
		EndSleep:   end,
		Weekdays:   weekdays,
		Timezone:   loc,
		Holidays:   holidaysMap,
	}, nil
}

type Set map[string]struct{}

func (s Set) Add(element string) {
	s[element] = struct{}{}
}
func (s Set) Contains(element string) bool {
	_, exists := s[element]
	return exists
}

func (s Set) getAllDays() Set {
	for i := 1; i <= 7; i++ {
		s[fmt.Sprintf("%d", i)] = struct{}{}
	}
	return s
}

func extractWeekdays(weekdays string) Set {
	weekdaysSet := make(Set)
	if weekdays == "*" {
		return weekdaysSet.getAllDays()
	}
	weekdaysList := strings.Split(weekdays, ",")
	for _, item := range weekdaysList {
		if strings.Contains(item, "-") {
			// If item is a range (e.g., "1-5"), extract start and end values
			parts := strings.Split(item, "-")
			start, _ := strconv.Atoi(parts[0])
			end, _ := strconv.Atoi(parts[1])

			// Add all numbers in the range to the set
			for i := start; i <= end; i++ {
				weekdaysSet.Add(strconv.Itoa(i))
			}
		} else {
			// If item is a single number, add it to the set
			weekdaysSet.Add(item)
		}
	}
	return weekdaysSet
}

func mapWeekdays(weekdaysSet Set) []time.Weekday {
	weekdayMap := map[string]time.Weekday{
		"1": time.Monday,
		"2": time.Tuesday,
		"3": time.Wednesday,
		"4": time.Thursday,
		"5": time.Friday,
		"6": time.Saturday,
		"7": time.Sunday,
	}

	var convertedWeekdays []time.Weekday
	for wd := range weekdaysSet {
		if weekday, ok := weekdayMap[wd]; ok {
			convertedWeekdays = append(convertedWeekdays, weekday)
		}
	}
	return convertedWeekdays
}

func removeRedundancy(sortedList []time.Time) []time.Time {
	if len(sortedList) == 0 {
		return sortedList
	}
	result := []time.Time{sortedList[0]}
	lastElement := sortedList[0]
	for _, t := range sortedList[1:] {
		if t.Year() != lastElement.Year() || t.Month() != lastElement.Month() || t.Day() != lastElement.Day() {
			result = append(result, t)
			lastElement = t
		}
	}
	return result
}

func getAllHolidaysDates(schedule SleepSchedule) []time.Time {
	var dateList []time.Time
	for _, holidays := range schedule.Holidays {
		for _, date := range holidays {
			dateList = append(dateList, date)
		}
	}
	sort.Slice(dateList, func(i, j int) bool {
		return dateList[i].Before(dateList[j])
	})
	filteredDateList := removeRedundancy(dateList)
	return filteredDateList
}

func isItSameDay(date1 time.Time, date2 time.Time) bool {
	if date1.Year() == date2.Year() && date1.Month() == date2.Month() && date1.Day() == date2.Day() {
		return true
	}
	return false
}

func checkConsecutiveDates(schedule SleepSchedule, dateList []time.Time, targetDate time.Time) time.Duration {
	var requeueTime, durationOffset time.Duration
	requeueTime = 0
	durationOffset = 0

	for index, date := range dateList {
		if isItSameDay(targetDate, date) {
			durationOffset = date.AddDate(0, 0, 1).Sub(schedule.now)
		} else if date.After(targetDate) {
			diff := date.Sub(targetDate)
			if diff == 24*time.Hour {
				requeueTime = checkConsecutiveDates(schedule, dateList[index+1:len(dateList)], date) + durationOffset
				return requeueTime + 24*time.Hour
			}
		}
	}

	return requeueTime + durationOffset
}

func IsItHoliday(schedule SleepSchedule) (bool, time.Duration) {
	isHoliday := false
	var holidayDuration time.Duration
	for _, holidays := range schedule.Holidays {
		for _, date := range holidays {
			if isItSameDay(schedule.now, date) {
				isHoliday = true
				allHolidays := getAllHolidaysDates(schedule)
				holidayDuration = checkConsecutiveDates(schedule, allHolidays, date)
				break
			}
		}
		if isHoliday {
			break
		}
	}
	return isHoliday, holidayDuration
}

func IsTimeToSleep(schedule SleepSchedule, kronosapp *v1alpha1.KronosApp) (bool, bool, time.Duration, error) {
	ok, holidayDuration := IsItHoliday(schedule)
	if ok {
		return true, true, holidayDuration, nil
	}
	if kronosapp.Spec.ForceSleep == true {
		return false, true, 0, nil
	}
	if kronosapp.Spec.ForceWake == true {
		return false, false, 0, nil
	}
	// Check if today is one of the weekdays specified
	isWeekdayIncluded := false
	for _, weekday := range schedule.Weekdays {
		if schedule.now.Weekday() == weekday {
			isWeekdayIncluded = true
			// Check if the current time is between start and end sleep times
			if schedule.now.After(schedule.StartSleep) && schedule.now.Before(schedule.EndSleep) {
				return false, true, 0, nil
			}
		}
	}
	if isWeekdayIncluded == false {
		return false, true, 0, nil
	}
	return false, false, 0, nil
}

func getRequeueTime(schedule SleepSchedule) time.Duration {
	var nextRequeue time.Time
	if schedule.now.Before(schedule.StartSleep) {
		nextRequeue = schedule.StartSleep
	} else if schedule.now.After(schedule.StartSleep) && schedule.now.Before(schedule.EndSleep) {
		nextRequeue = schedule.EndSleep
	} else {
		nextRequeue = schedule.StartSleep.Add(24 * time.Hour)
	}
	nextRequeueDiff := nextRequeue.Sub(schedule.now)
	return nextRequeueDiff
}

func formatDuration(d time.Duration) string {
	hours := d / time.Hour
	minutes := (d % time.Hour) / time.Minute
	return fmt.Sprintf("%dh%02dm", hours, minutes)
}

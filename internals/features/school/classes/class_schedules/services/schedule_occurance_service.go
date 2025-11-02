// file: internals/features/school/sessions/schedules/service/schedule_occurrence.go
package service

// import (
// 	"math"
// 	"time"

// 	rmodel "schoolku_backend/internals/features/school/sessions/schedules/model"
// )

// // Map Go's time.Weekday() (0=Sunday) → 1..7 (Mon..Sun)
// func dowTo1Mon7Sun(t time.Time) int {
// 	wd := int(t.Weekday()) // 0=Sunday..6=Saturday
// 	if wd == 0 {
// 		return 7 // Sunday → 7
// 	}
// 	return wd // Mon..Sat = 1..6
// }

// // Letakkan ke Senin dari minggu yg sama (basis weekly calc)
// func toMonday(t time.Time) time.Time {
// 	// in: any date 00:00
// 	dow := int(t.Weekday()) // 0=Sun..6=Sat
// 	// we want Monday(1). For Sunday(0), back 6d; for Monday(1), back 0d ...
// 	// shift = (dow + 6) % 7  → Monday=0, Tue=1,..., Sun=6
// 	shift := (dow + 6) % 7
// 	return time.Date(t.Year(), t.Month(), t.Day()-shift, 0, 0, 0, 0, t.Location())
// }

// // weekIndex: 0 untuk minggu basis, 1 untuk minggu berikutnya, dst.
// func weeksBetween(baseMonday, dMonday time.Time) int {
// 	diffDays := int(dMonday.Sub(baseMonday).Hours() / 24)
// 	return int(math.Floor(float64(diffDays) / 7.0))
// }

// // 1..5 posisi minggu dalam bulan (by ISO-ish simple rule)
// func weekOfMonth(d time.Time) int {
// 	// minggu ke-1: hari tanggal 1..7 yg mengandung d
// 	// Cara mudah: ambil tanggal 1 pada bulan tsb → cari Mondaynya → offset/7 + 1
// 	first := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
// 	firstMon := toMonday(first)
// 	dMon := toMonday(d)
// 	w := weeksBetween(firstMon, dMon) + 1 // 1-based
// 	if w < 1 {
// 		w = 1
// 	}
// 	if w > 5 {
// 		w = 5
// 	}
// 	return w
// }

// func isLastWeekOfMonth(d time.Time) bool {
// 	// kalau 7 hari ke depan sudah beda bulan → ini minggu terakhir
// 	return d.AddDate(0, 0, 7).Month() != d.Month()
// }

// // Cek apakah tanggal d (00:00) match sebuah rule
// func dateMatchesRule(d, scheduleStartDate time.Time, r rmodel.ClassScheduleRuleModel) bool {
// 	// Day of week
// 	if dowTo1Mon7Sun(d) != r.ClassScheduleRuleDayOfWeek {
// 		return false
// 	}

// 	// Basis per minggu dihitung dari Monday pada scheduleStartDate
// 	baseMonday := toMonday(scheduleStartDate)
// 	dMonday := toMonday(d)

// 	wk := weeksBetween(baseMonday, dMonday)

// 	// Start offset weeks
// 	wk = wk - r.ClassScheduleRuleStartOffsetWeeks
// 	if wk < 0 {
// 		return false
// 	}

// 	// Interval weeks
// 	if r.ClassScheduleRuleIntervalWeeks <= 0 {
// 		if wk != 0 {
// 			return false
// 		}
// 	} else {
// 		if wk%r.ClassScheduleRuleIntervalWeeks != 0 {
// 			return false
// 		}
// 	}

// 	// Parity: odd/even/all — wk=0 dianggap minggu ke-1 (odd)
// 	switch r.ClassScheduleRuleWeekParity {
// 	case rmodel.WeekParityOdd:
// 		if (wk % 2) != 0 { // 0,2,4 → odd; 1,3 → even
// 			return false
// 		}
// 	case rmodel.WeekParityEven:
// 		if (wk % 2) == 0 {
// 			return false
// 		}
// 	}

// 	// Weeks of month
// 	if len(r.ClassScheduleRuleWeeksOfMonth) > 0 {
// 		wom := int64(weekOfMonth(d))
// 		ok := false
// 		for _, v := range r.ClassScheduleRuleWeeksOfMonth {
// 			if v == wom {
// 				ok = true
// 				break
// 			}
// 		}
// 		if !ok {
// 			return false
// 		}
// 	}

// 	// Last week of month
// 	if r.ClassScheduleRuleLastWeekOfMonth && !isLastWeekOfMonth(d) {
// 		return false
// 	}

// 	return true
// }

// // Gabungkan "tanggal" (d) + "jam" (t-of-day) → time pointer
// func combineDateAndTOD(d, tod time.Time) *time.Time {
// 	if tod.IsZero() {
// 		return nil
// 	}
// 	out := time.Date(d.Year(), d.Month(), d.Day(), tod.Hour(), tod.Minute(), tod.Second(), 0, d.Location())
// 	return &out
// }

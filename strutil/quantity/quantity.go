// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package quantity

import (
	"fmt"
	"github.com/snapcore/snapd/i18n"
	"math"
	"sort"
	"strconv"
)

// these are taken from github.com/chipaca/quantity with permission :-)

func FormatAmount(amount uint64, width int) string {
	if width < 0 {
		width = 5
	}
	max := uint64(5000)
	maxFloat := 999.5

	if width < 4 {
		width = 3
		max = 999
		maxFloat = 99.5
	}

	if amount <= max {
		pad := ""
		if width > 5 {
			pad = " "
		}
		return fmt.Sprintf("%*d%s", width-len(pad), amount, pad)
	}
	var prefix rune
	r := float64(amount)
	// zetta and yotta are me being pedantic: maxuint64 is ~18EB
	for _, prefix = range "kMGTPEZY" {
		r /= 1000
		if r < maxFloat {
			break
		}
	}

	width--
	digits := 3
	if r < 99.5 {
		digits--
		if r < 9.5 {
			digits--
			if r < .95 {
				digits--
			}
		}
	}
	precision := 0
	if (width - digits) > 1 {
		precision = width - digits - 1
	}

	s := fmt.Sprintf("%*.*f%c", width, precision, r, prefix)
	if r < .95 {
		return s[1:]
	}
	return s
}

func FormatBPS(n, sec float64, width int) string {
	if sec < 0 {
		sec = -sec
	}
	return FormatAmount(uint64(n/sec), width-2) + "B/s"
}

const (
	period = 365.25 // julian years (c.f. the actual orbital period, 365.256363004d)
)

func divmod(a, b float64) (q, r float64) {
	q = math.Floor(a / b)
	return q, a - q*b
}

var (
	// TRANSLATORS: this needs to be a single rune that is understood to mean "seconds" in e.g. 1m30s
	//    (I fully expect this to always be "s", given it's a SI unit)
	secs = i18n.G("s")
	// TRANSLATORS: this needs to be a single rune that is understood to mean "minutes" in e.g. 1m30s
	mins = i18n.G("m")
	// TRANSLATORS: this needs to be a single rune that is understood to mean "hours" in e.g. 1h30m
	hours = i18n.G("h")
	// TRANSLATORS: this needs to be a single rune that is understood to mean "days" in e.g. 1d20h
	days = i18n.G("d")
	// TRANSLATORS: this needs to be a single rune that is understood to mean "years" in e.g. 1y45d
	years = i18n.G("y")
)

// dt is seconds (as in the output of time.Now().Seconds())
func FormatDuration(dt float64) string {
	if dt < 60 {
		if dt >= 9.995 {
			return fmt.Sprintf("%.1f%s", dt, secs)
		} else if dt >= .9995 {
			return fmt.Sprintf("%.2f%s", dt, secs)
		}

		var prefix rune
		for _, prefix = range "mµn" {
			dt *= 1000
			if dt >= .9995 {
				break
			}
		}

		if dt > 9.5 {
			return fmt.Sprintf("%3.f%c%s", dt, prefix, secs)
		}

		return fmt.Sprintf("%.1f%c%s", dt, prefix, secs)
	}

	if dt < 600 {
		m, s := divmod(dt, 60)
		return fmt.Sprintf("%.f%s%02.f%s", m, mins, s, secs)
	}

	dt /= 60 // dt now minutes

	if dt < 99.95 {
		return fmt.Sprintf("%3.1f%s", dt, mins)
	}

	if dt < 10*60 {
		h, m := divmod(dt, 60)
		return fmt.Sprintf("%.f%s%02.f%s", h, hours, m, mins)
	}

	if dt < 24*60 {
		if h, m := divmod(dt, 60); m < 10 {
			return fmt.Sprintf("%.f%s%1.f%s", h, hours, m, mins)
		}

		return fmt.Sprintf("%3.1f%s", dt/60, hours)
	}

	dt /= 60 // dt now hours

	if dt < 10*24 {
		d, h := divmod(dt, 24)
		return fmt.Sprintf("%.f%s%02.f%s", d, days, h, hours)
	}

	if dt < 99.95*24 {
		if d, h := divmod(dt, 24); h < 10 {
			return fmt.Sprintf("%.f%s%.f%s", d, days, h, hours)
		}
		return fmt.Sprintf("%4.1f%s", dt/24, days)
	}

	dt /= 24 // dt now days

	if dt < 2*period {
		return fmt.Sprintf("%4.0f%s", dt, days)
	}

	dt /= period // dt now years

	if dt < 9.995 {
		return fmt.Sprintf("%4.2f%s", dt, years)
	}

	if dt < 99.95 {
		return fmt.Sprintf("%4.1f%s", dt, years)
	}

	if dt < 999.5 {
		return fmt.Sprintf("%4.f%s", dt, years)
	}

	if dt > math.MaxUint64 || uint64(dt) == 0 {
		// TODO: figure out exactly what overflow causes the ==0
		return "ages!"
	}

	return FormatAmount(uint64(dt), 4) + years
}

// Duration is represented in nano seconds.
type Duration uint64

const (
	NSec Duration = 1
	USec Duration = 1000
	MSec Duration = 1000000
	Sec  Duration = 1000000000
	Min  Duration = (60 * Sec)
	Hour Duration = (60 * Min)
	Day  Duration = (24 * Hour)
	Year Duration = (1461 * Day / 4)
)

// Maximum number of places (units of time) to render starting with years.
// In the Compact case, only the first two places will be rendered,
// discarding the rest. This allows for a more predictable width (at the
// expense of accuracy).
type Places uint32

const (
	ShowAll  Places = 8
	ShowComp Places = 2
)

// Space delimit options
type SpaceMode bool

const (
	SpaceOff SpaceMode = false
	SpaceOn  SpaceMode = true
)

// The minimum Duration parameter allows the rendering to discard the units
// of time less than the minimum. However, depending on what we are
// rendering, we either need to peform a ceiling, floor or rounding
// operation with the remainder below the minimum.
type RenderMode uint32

const (
	TimeLeft    RenderMode = 0
	TimePassed  RenderMode = 1
	TimeRounded RenderMode = 2
)

// FormatDurationGeneric formats time duration similar to systemd
// format_timespan, but with options allowing for a more compact
// arrangement, including shortened suffixes. The function is provided
// with dt in seconds (as in the output of time.Now().Seconds()). The 'min'
// and 'max' Duration allows limiting the unit range rendered. Place values
// less than 'min' will be processed according to RenderMode. RenderMode
// allows rounding, flooring or ceiling to be applied to place value below
// 'min'. Durations above the 'max' place value will result in "ages!".
func FormatDurationGeneric(dt float64, min Duration, max Duration, count Places, mode RenderMode, space SpaceMode) string {
	var units = map[Duration]string{
		Year: "y",
		Day:  "d",
		Hour: "h",
		Min:  "m",
		Sec:  "s",
		MSec: "ms",
		USec: "µs",
		NSec: "ns",
	}

	// We need to access the map in order using an ordered array
	var keys []Duration
	for k := range units {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })

	var render string

	// Invalid case: min > max
	if min > max {
		return "inv!"
	}

	// Special case: zero duration
	if dt <= 0 {
		return "0" + units[min]
	}

	// Special case: render as "ages!"
	if dt >= float64(math.MaxUint64) {
		return "ages!"
	}

	// Fractional seconds to nanoseconds
	delta := uint64(dt * float64(Sec))

	// If we only render a subset of place values, we need to apply
	// the render mode to the last place value. We do this indirectly
	// by moving up the 'min' accuracy.
	if count == ShowComp {
		for i := 0; i < len(keys); i++ {
			// Only render places as indicated by count
			if delta >= uint64(keys[i]) && keys[i] >= Min {
				index := i + int(count) - 1
				if keys[index] > min {
					min = keys[index]
				}
				break
			}

			// Only one place as we are s, ms, us or ns
			if delta >= uint64(keys[i]) && keys[i] < Min {
				if keys[i] > min {
					min = keys[i]
				}
				break
			}
		}
	}

	// Apply the rendermode to the least significant place value
	mod := uint64(delta % uint64(min))
	switch {
	case mode == TimeLeft && mod > 0:
		// Ceiling
		delta = delta + uint64(min) - mod
	case mode == TimePassed:
		// Floor
		delta = delta - mod
	case mode == TimeRounded && mod >= (uint64(min)/2):
		// Rounded Up
		delta = delta + uint64(min) - mod
	case mode == TimeRounded && mod < (uint64(min)/2):
		// Rounded Down
		delta = delta - mod
	}

	// Special case: less than minimum accuracy
	if delta < uint64(min) {
		return "0" + units[min]
	}

	// In ShowComp mode only 2-digit days are possible
	if count == ShowComp && delta >= (100*uint64(Day)) {
		return "ages!"
	}

	// Iterate through units of time in order from
	// highest to lowest (years, days, ...)
	for i := 0; i < len(keys); i++ {
		done := false

		// No place value left to render
		if delta <= 0 {
			break
		}

		// No place value left greater than required
		// accuracy to render.
		if delta < uint64(min) && len(render) != 0 {
			break
		}

		// Remainder is less than current place value,
		// skip to next.
		if delta < uint64(keys[i]) {
			continue
		}

		// If the duration exceeds the maximum place value
		// (unit if time) we will not render it, just return
		// "ages!".
		if uint64(keys[i]) > uint64(max) {
			return "ages!"
		}

		unit := uint64(delta / uint64(keys[i]))
		remainder := uint64(delta % uint64(keys[i]))

		// If the accuracy is less than 1s, we support
		// rendering a fractional second part.
		if delta < uint64(Min) && remainder > 0 && count == ShowAll {
			digits := 0

			// Number of digits required to render
			// nano second fractunal accuracy.
			for cc := uint64(keys[i]); cc > 1; cc /= 10 {
				digits++
			}

			// Number of digits to render specified
			// fractunal accuracy.
			for cc := uint64(min); cc > 1; cc /= 10 {
				digits--
			}

			frac := float64(delta) / float64(uint64(keys[i]))

			// Should we generate a fractional second, else
			// use the generic rendering function.
			if digits > 0 {
				if len(render) > 0 && space == SpaceOn {
					render += " "
				}
				render += strconv.FormatFloat(frac, 'f', digits, 64)
				render += units[keys[i]]

				delta = 0
				done = true
			}
		}

		// Generic place value rendering
		if done == false {
			// Insert spaced between place values if enabled.
			if len(render) > 0 && space == SpaceOn {
				render += " "
			}

			render += fmt.Sprintf("%v", unit)
			render += units[keys[i]]

			delta = remainder
		}
	}
	return render
}

// ProgressBarTimeLeft presents duration in a layout suitable for progress
// indicators such as ANSIMeter. The rendered duration (time left) range is
// between (and includes) hours and seconds. The width is guaranteed not to
// exceed 6 runes by rendering only the two most significant units of time.
// The ceiling() operation is performed on the unrendered least significant
// place values (+1 on the least significant rendered unit).
func ProgressBarTimeLeft(dt float64) string {
	return FormatDurationGeneric(dt, Sec, Hour, ShowComp, TimeLeft, SpaceOff)
}

// ProgressBarTimePassed presents duration in a layout suitable for progress
// indicators such as ANSIMeter. The rendered duration (time elapse) range is
// between (and includes) hours and seconds. The width is guaranteed not to
// exceed 6 runes by rendering only the two most significant units of time.
// The floor() operation is performed on the unrendered least significant
// place values (discarded).
func ProgressBarTimePassed(dt float64) string {
	return FormatDurationGeneric(dt, Sec, Hour, ShowComp, TimePassed, SpaceOff)
}

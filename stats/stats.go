package stats

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/olekukonko/tablewriter"

	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/sort"
)

var (
	averages = make(map[string]int)
	asets    = make(map[string][]float32)

	frequencies = make(map[string]float32)
	fsets       = make(map[string][]float32)

	matches = make(map[string]int)

	images = make(map[string]int)

	statsq = make(chan func(), 1024)
)

func init() {
	go func() {
		for fn := range statsq {
			fn()
		}
	}()
}

func Average(stat string, maxv float32) {
	if math.IsInf(float64(maxv), 1) {
		maxv = 1
	}

	statsq <- func() {
		asets[stat] = append(asets[stat], maxv)

		sum := float32(0)
		for _, n := range asets[stat] {
			sum += n
		}

		avg := int((sum / float32(len(asets[stat]))) * 100)
		if avg > 0 {
			averages[stat] = avg
		}
	}
}

func Clear() {
	notify.Feed(rgba.White, "Clearing matched image template statistics")
	statsq <- func() {
		for stat := range matches {
			delete(matches, stat)
			delete(averages, stat)
			delete(asets, stat)
			delete(frequencies, stat)
			delete(fsets, stat)
		}
	}
}

func Count(stat string) {
	statsq <- func() {
		matches[stat]++
	}
}

func Data() {
	statsq <- func() {
		if len(matches) == 0 {
			notify.Feed(rgba.White, "No matched image template statistics to display...")
			return
		}

		buf := &bytes.Buffer{}
		table := tablewriter.NewWriter(buf)
		table.SetCenterSeparator("")
		table.SetAutoFormatHeaders(false)
		table.SetColumnSeparator("|")
		table.SetRowSeparator("")
		table.SetColMinWidth(0, 10)
		table.SetColMinWidth(1, 7)
		table.SetColMinWidth(3, 7)
		table.SetColumnAlignment(
			[]int{
				tablewriter.ALIGN_LEFT,
				tablewriter.ALIGN_LEFT,
				tablewriter.ALIGN_LEFT,
				tablewriter.ALIGN_LEFT,
			},
		)
		table.SetBorder(false)

		table.Append([]string{"Matches", "\tAvg\t", "\tFreq\t", "\tFile"})

		sorted := sort.Stats{}
		for n := range matches {
			sorted = sorted.Append(n, matches[n], averages[n], frequencies[n])
		}
		sorted.Sort()

		colors := []color.RGBA{color.RGBA(rgba.White)}

		for _, s := range sorted {
			c := color.RGBA(rgba.ForestGreen)

			switch {
			case s.Average == 0:
				if s.Matches == 0 {
					c = color.RGBA(rgba.SlateGray)
				} else {
					c = color.RGBA(rgba.DarkRed)
				}
			case s.Average < 50:
				c = rgba.Orange
			case s.Average >= 50 && s.Average < 70:
				c = color.RGBA(rgba.DarkYellow)
			case s.Average >= 70 && s.Average < 80:
				c = color.RGBA(rgba.ForestGreen)
			case s.Average >= 80:
				c = color.RGBA(rgba.Green)
			}

			colors = append(colors, c)

			table.Append(
				[]string{
					fmt.Sprintf("\t%8d\t", s.Matches),
					fmt.Sprintf("\t%8d%s\t", s.Average, "%%"),
					fmt.Sprintf("\t%8.1f%s\t", s.Frequency, "%%"),
					fmt.Sprintf("\t%10s", s.Name),
				},
			)

		}

		if len(sorted) > 0 {
			table.Render()
		}

		// Empty lines.
		colors = append(colors, rgba.White)
		colors = append(colors, rgba.Black)

		notify.Feed(rgba.White, "Matched image template statistics")

		lines := strings.Split(buf.String(), "\n")
		for i := range lines {
			if lines[i] == "" {
				continue
			}

			notify.Append(colors[i], lines[i])
		}
	}
}

func Frequency(stat string, freq float32) {
	if math.IsInf(float64(freq), 1) {
		freq = 1
	}

	statsq <- func() {
		fsets[stat] = append(fsets[stat], freq)

		sum := float32(0)
		for _, n := range fsets[stat] {
			sum += n
		}

		freq := (sum / float32(len(fsets[stat]))) * 100
		if freq > 0 {
			frequencies[stat] = freq
		}
	}
}

func Images() {

}

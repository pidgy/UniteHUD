package stats

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/guptarohit/asciigraph"
	"github.com/olekukonko/tablewriter"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/global"
)

const maxX = 10

var (
	averages = make(map[string]int)
	asets    = make(map[string][]float32)

	frequencies = make(map[string]float32)
	fsets       = make(map[string][]float32)

	matches = make(map[string]int)

	cpus = []float64{0}
	rams = []float64{0}

	statsq = make(chan func(), 1024)
)

func init() {
	go func() {
		for fn := range statsq {
			fn()
		}
	}()

	statsq <- func() {
		clear()
	}
}

func Clear() {
	notify.System("[Stats] Clearing matched image template statistics")
	statsq <- func() {
		clear()
	}
}

func Collect(stat string, maxv float32) {
	if config.Current.Advanced.Stats.Disabled {
		return
	}

	if math.IsInf(float64(maxv), 1) {
		maxv = 1
	}

	stat = sanitize(stat)

	statsq <- func() {
		// Average
		asets[stat] = append(asets[stat], maxv)

		sum := float32(0)
		for _, n := range asets[stat] {
			sum += n
		}

		avg := int((sum / float32(len(asets[stat]))) * 100)
		if avg > 0 {
			averages[stat] = avg
		}

		// Count.
		matches[stat]++

		// Frequency.
		fsets[stat] = append(fsets[stat], maxv)

		fsum := float32(0)
		for _, n := range fsets[stat] {
			fsum += n
		}

		freq := (fsum / float32(len(fsets[stat]))) * 100
		if freq > 0 {
			frequencies[stat] = freq
		}
	}
}

func Counts() map[string]int {
	fq := make(chan map[string]int)

	counts := make(map[string]int)

	for f := range config.Current.TemplateMatchMap() {
		counts[sanitize(f)] = 0
	}

	statsq <- func() {
		defer close(fq)

		for f, c := range matches {
			counts[f] = c
		}

		fq <- counts
	}

	return <-fq
}

func CPU(v float64) {
	statsq <- func() {
		if len(cpus) == maxX {
			cpus = append(cpus[:1], round(v))
		} else {
			cpus = append(cpus, round(v))
		}
	}
}

func CPUGraph() string {
	return asciigraph.Plot(cpus, []asciigraph.Option{
		asciigraph.Height(5),
		asciigraph.Width(20),
		asciigraph.Precision(0),
	}...)
}

func Data() {
	for _, line := range Lines() {
		if line == "" {
			continue
		}

		switch {
		case strings.Contains(line, team.Orange.Name):
			notify.Append(team.Orange.NRGBA, line)
		case strings.Contains(line, team.Purple.Name):
			notify.Append(team.Purple.NRGBA, line)
		case strings.Contains(line, team.First.Name):
			notify.Append(team.First.NRGBA, line)
		case strings.Contains(line, team.Energy.Name):
			notify.Append(nrgba.DarkYellow, line)
		case strings.Contains(line, team.Time.Name):
			notify.Append(nrgba.Slate, line)
		case strings.Contains(line, team.Game.Name):
			notify.Append(nrgba.Gray, line)
		default:
			notify.SystemAppend(line)
		}
	}
}

func Lines() []string {
	lineq := make(chan []string)

	statsq <- func() {
		defer close(lineq)

		if len(averages) == 0 {
			notify.Warn("[Stats] No matched image template statistics to display...")
			return
		}

		buf := &bytes.Buffer{}
		table := tablewriter.NewWriter(buf)
		table.SetCenterSeparator("-")
		table.SetAutoFormatHeaders(false)
		table.SetColumnSeparator("|")
		table.SetRowSeparator("")
		table.SetColMinWidth(0, 6)
		table.SetColMinWidth(1, 5)
		table.SetColMinWidth(2, 4)
		table.SetColMinWidth(3, 7)
		table.SetColumnAlignment(
			[]int{
				tablewriter.ALIGN_LEFT,
				tablewriter.ALIGN_LEFT,
				tablewriter.ALIGN_LEFT,
				tablewriter.ALIGN_LEFT,
				tablewriter.ALIGN_LEFT,
			},
		)
		table.SetBorder(false)
		table.Append(
			[]string{
				"Matches",
				"Tally",
				"Avg %%",
				"Freq %%",
				"File",
			},
		)

		sorted := sortable{}

		// Use frequencies to see all images sent to be matched, or use matches to
		// only see matched images.
		if global.DebugMode {
			for n := range frequencies {
				if frequencies[n] < 1 {
					continue
				}
				sorted.add(n, matches[n], averages[n], frequencies[n])
			}
		} else {
			for n := range matches {
				sorted.add(n, matches[n], averages[n], frequencies[n])
			}
		}

		sorted.Sort()

		for _, s := range sorted {
			table.Append(
				[]string{
					fmt.Sprintf("%d", s.Matches),
					fmt.Sprintf("%d", len(fsets[s.Name])),
					fmt.Sprintf("%d%s", s.Average, "%%"),
					fmt.Sprintf("%.1f%s", s.Frequency, "%%"),
					s.Name,
				},
			)
		}

		if len(sorted) > 0 {
			table.Render()
		}

		notify.System("[Stats] Matched image template statistics")

		lineq <- strings.Split(buf.String(), "\n")
	}

	return <-lineq
}

func RAM(v float64) {
	statsq <- func() {
		if v == rams[len(rams)-1] {
			return
		}

		if len(cpus) == maxX {
			rams = append(rams[:1], round(v))
		} else {
			rams = append(rams, round(v))
		}
	}
}

func RAMGraph() string {
	return asciigraph.Plot(rams, []asciigraph.Option{
		asciigraph.Height(5),
		asciigraph.Width(20),
		asciigraph.Precision(0),
	}...)
}

func clear() {
	averages = make(map[string]int)
	asets = make(map[string][]float32)

	frequencies = make(map[string]float32)
	fsets = make(map[string][]float32)

	matches = make(map[string]int)
}

func round(v float64) float64 {
	if v > 95 {
		return 100
	}

	return float64(int(math.Round(math.Floor(v/5))) * 5)
}

func sanitize(stat string) string {
	stat = strings.ReplaceAll(strings.ReplaceAll(stat, "\\", "/"), "PNG", "png")

	args := strings.Split(stat, "device/")
	if len(args) > 1 {
		stat = args[1]
	}

	return stat
}

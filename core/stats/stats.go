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
	"github.com/pidgy/unitehud/exe"
)

// maxX is the maximum number of samples retained per series.
const maxX = 100000

var (
	// averages holds per-stat average match percentages.
	averages = make(map[string]int)
	// asets stores per-stat samples used to compute averages.
	asets = make(map[string][]float32)

	// frequencies holds per-stat average frequency percentages.
	frequencies = make(map[string]float32)
	// fsets stores per-stat samples used to compute frequencies.
	fsets = make(map[string][]float32)

	// matches tracks the number of matches per stat.
	matches = make(map[string]int)

	// cpus holds sampled CPU usage percentages.
	cpus = []float64{0}
	// rams holds sampled RAM usage values.
	rams = []float64{0}
	// threads holds sampled thread counts.
	threads = []float64{0}

	// statsq serializes stats mutations to avoid data races.
	statsq = make(chan func(), 1024)
)

// init starts the stats worker and clears any existing data.
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

// CPUGraph renders an ASCII graph of recent CPU samples.
func CPUGraph() string {
	return asciigraph.Plot(cpus,
		asciigraph.LowerBound(0),
		asciigraph.UpperBound(100),
		asciigraph.Height(5),
		asciigraph.Width(20),
		asciigraph.Precision(0),
	)
}

// Clear resets all collected template match statistics.
func Clear() {
	notify.System("[Stats] Clearing matched image template statistics")
	statsq <- func() {
		clear()
	}
}

// Collect records a template match sample for aggregation.
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

// Counts returns the current per-template match counts.
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

// Data emits formatted stats lines to the notifier with colors.
func Data() {
	for _, line := range Lines() {
		if line == "" {
			continue
		}

		switch {
		case strings.Contains(line, team.Orange.Name):
			notify.Append(team.Orange.NRGBA, "%s", line)
		case strings.Contains(line, team.Purple.Name):
			notify.Append(team.Purple.NRGBA, "%s", line)
		case strings.Contains(line, team.First.Name):
			notify.Append(team.First.NRGBA, "%s", line)
		case strings.Contains(line, team.Energy.Name):
			notify.Append(nrgba.DarkYellow, "%s", line)
		case strings.Contains(line, team.Time.Name):
			notify.Append(nrgba.Slate, "%s", line)
		case strings.Contains(line, team.Game.Name):
			notify.Append(nrgba.Gray, "%s", line)
		default:
			notify.SystemAppend(line)
		}
	}
}

// Lines builds the formatted stats table lines.
func Lines() []string {
	lineq := make(chan []string)

	statsq <- func() {
		defer close(lineq)

		if len(averages) == 0 {
			notify.Warn("[Stats] No image template statistics to display...")
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
		if exe.Debug {
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

		notify.System("[Stats] Fetched image template statistics")

		lineq <- strings.Split(buf.String(), "\n")
	}

	return <-lineq
}

// Procs records process resource samples for graphing.
func Procs(cpu, ram, thread float64) {
	statsq <- func() {
		if ram > 0 && ram != rams[len(rams)-1] {
			if len(rams) == maxX {
				rams = append(rams[:1], round(ram))
			} else {
				rams = append(rams, round(ram))
			}
		}

		if cpu > 0 {
			if len(cpus) == maxX {
				cpus = append(cpus[:1], round(cpu))
			} else {
				cpus = append(cpus, round(cpu))
			}
		}

		if len(threads) == maxX {
			threads = append(threads[:1], round(thread))
		} else {
			threads = append(threads, round(thread))
		}
	}
}

// RAMGraph renders an ASCII graph of recent RAM samples.
func RAMGraph() string {
	return asciigraph.Plot(rams,
		asciigraph.LowerBound(0),
		asciigraph.UpperBound(1000),
		asciigraph.Height(5),
		asciigraph.Width(20),
		asciigraph.Precision(0),
	)
}

// ThreadsGraph renders an ASCII graph of recent thread samples.
func ThreadsGraph() string {
	return asciigraph.Plot(threads,
		asciigraph.LowerBound(0),
		asciigraph.UpperBound(100),
		asciigraph.Height(5),
		asciigraph.Width(20),
		asciigraph.Precision(0),
	)
}

// clear resets in-memory aggregates.
func clear() {
	averages = make(map[string]int)
	asets = make(map[string][]float32)

	frequencies = make(map[string]float32)
	fsets = make(map[string][]float32)

	matches = make(map[string]int)
}

// round buckets a value into 5-unit steps with a 100% cap.
func round(v float64) float64 {
	if v > 95 {
		return 100
	}

	return float64(int(math.Round(math.Floor(v/5))) * 5)
}

// sanitize normalizes template names for consistent keys.
func sanitize(stat string) string {
	stat = strings.ReplaceAll(strings.ReplaceAll(stat, "\\", "/"), "PNG", "png")

	args := strings.Split(stat, "device/")
	if len(args) > 1 {
		stat = args[1]
	}

	return stat
}

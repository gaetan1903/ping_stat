package main

import (
	"fmt"
	"os"
	"os/signal"

	tslc "github.com/NimbleMarkets/ntcharts/linechart/streamlinechart"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	probing "github.com/prometheus-community/pro-bing"
)

type model struct {
	chart       tslc.Model
	zoneManager *zone.Manager
}

type SuccessMsg struct {
	Success bool
}

var pingSet = []float64{0, 0}

var prog *tea.Program

func (m model) Init() tea.Cmd {
	go initPing()
	return nil
}

func initPing() {
	pinger, err := probing.NewPinger("8.8.8.8")
	if err != nil {
		panic(err)
	}
	// Listen for Ctrl-C.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			pinger.Stop()
		}
	}()
	pinger.OnRecv = func(pkt *probing.Packet) {
		if len(pingSet) < 3 {
			pingSet = pingSet[:0]
			for i := 0; i < 20; i++ {
				pingSet = append(pingSet, float64(pkt.Rtt.Milliseconds()))
			}
		} else {
			for i := 0; i < len(pingSet)-1; i++ {
				pingSet[i] = pingSet[i+1]
			}
			pingSet[len(pingSet)-1] = float64(pkt.Rtt.Milliseconds())
		}
		prog.Send(SuccessMsg{Success: true})
	}
	pinger.OnFinish = func(stats *probing.Statistics) {}
	err = pinger.Run()
	if err != nil {
		panic(err)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	for _, v := range pingSet {
		m.chart.Push(v)
	}

	m.chart, _ = m.chart.Update(msg)
	// add label to above the chart

	// draw the chart
	m.chart.DrawAll()
	return m, nil
}

func (m model) View() string {
	// call bubblezone Manager.Scan() at root model
	return m.zoneManager.Scan(
		lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#fff000")).
			Render(fmt.Sprintf("Ping: %.2f ms\n\n", pingSet[len(pingSet)-1]) + m.chart.View()),
	)
}

func main() {

	width, height := GetSize()

	chart := tslc.New(width-5, height-5)

	for _, v := range pingSet {
		chart.Push(v)
	}

	// mouse support is enabled with BubbleZone
	zoneManager := zone.New()
	chart.SetZoneManager(zoneManager)
	chart.Focus() // set focus to process keyboard and mouse messages

	// start new Bubble Tea program with mouse support enabled
	m := model{chart, zoneManager}
	prog = tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := prog.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

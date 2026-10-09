package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
)

type Result struct {
	nodes       int
	packetLoss  float64
	successRate float64
	hops        float64
	probes      float64
}

var linePattern = regexp.MustCompile(
	`nodes=(\d+) .*packet_loss=([\d.]+).*success_rate=([\d.]+).*average_hops=([\d.]+).*average_probes=([\d.]+)`,
)

func main() {
	data, err := os.ReadFile("experiment.log")
	if err != nil {
		panic(err)
	}

	var results []Result
	for _, match := range linePattern.FindAllStringSubmatch(string(data), -1) {
		nodes, _ := strconv.Atoi(match[1])
		packetLoss, _ := strconv.ParseFloat(match[2], 64)
		successRate, _ := strconv.ParseFloat(match[3], 64)
		hops, _ := strconv.ParseFloat(match[4], 64)
		probes, _ := strconv.ParseFloat(match[5], 64)

		results = append(results, Result{
			nodes:       nodes,
			packetLoss:  packetLoss,
			successRate: successRate,
			hops:        hops,
			probes:      probes,
		})
	}

	writeScalability(results)
	writeReliability(results)
}

func writeScalability(results []Result) {
	byNodes := make(map[int]Result)

	for _, result := range results {
		if result.packetLoss == 0 {
			byNodes[result.nodes] = result
		}
	}

	var values []Result
	for _, result := range byNodes {
		values = append(values, result)
	}

	sort.Slice(values, func(i, j int) bool {
		return values[i].nodes < values[j].nodes
	})

	var hops, probes, expected []Point
	for _, result := range values {
		hops = append(hops, Point{
			x: float64(result.nodes),
			y: result.hops,
		})
		probes = append(probes, Point{
			x: float64(result.nodes),
			y: result.probes,
		})
		expected = append(expected, Point{
			x: float64(result.nodes),
			y: math.Log2(float64(result.nodes)),
		})
	}

	writeSVG(
		"scalability-hops.svg",
		"Lookup scalability: hops",
		"Network size N",
		"Average hops",
		hops,
		expected,
	)

	writeSVG(
		"scalability-probes.svg",
		"Lookup scalability: probes",
		"Network size N",
		"Average probes",
		probes,
		nil,
	)
}

func writeReliability(results []Result) {
	var values []Result

	for _, result := range results {
		if result.nodes == 100 && result.packetLoss > 0 {
			values = append(values, result)
		}
	}

	sort.Slice(values, func(i, j int) bool {
		return values[i].packetLoss < values[j].packetLoss
	})

	var measured []Point
	for _, result := range values {
		measured = append(measured, Point{
			x: result.packetLoss,
			y: result.successRate,
		})
	}

	writeSVG(
		"reliability.svg",
		"Lookup reliability",
		"Packet loss probability",
		"Success rate",
		measured,
		nil,
	)
}

type Point struct {
	x float64
	y float64
}

func writeSVG(
	filename string,
	title string,
	xLabel string,
	yLabel string,
	first []Point,
	second []Point,
) {
	const width = 800
	const height = 500
	const left = 80
	const right = 30
	const top = 60
	const bottom = 70

	xMax := 1.0
	yMax := 1.0

	for _, point := range first {
		if point.x > xMax {
			xMax = point.x
		}
		if point.y > yMax {
			yMax = point.y
		}
	}

	for _, point := range second {
		if point.x > xMax {
			xMax = point.x
		}
		if point.y > yMax {
			yMax = point.y
		}
	}

	yMax *= 1.1

	x := func(value float64) float64 {
		return left + value/xMax*float64(width-left-right)
	}

	y := func(value float64) float64 {
		return height - bottom - value/yMax*float64(height-top-bottom)
	}

	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fmt.Fprintf(file, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`, width, height)
	fmt.Fprintf(file, `<rect width="100%%" height="100%%" fill="white"/>`)
	fmt.Fprintf(file, `<text x="%d" y="30" font-size="20">%s</text>`, left, title)

	fmt.Fprintf(file, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="black"/>`,
		left, height-bottom, width-right, height-bottom)
	fmt.Fprintf(file, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="black"/>`,
		left, top, left, height-bottom)

	fmt.Fprintf(file, `<text x="%d" y="%d" text-anchor="middle">%s</text>`,
		width/2, height-20, xLabel)
	fmt.Fprintf(file, `<text x="20" y="%d" transform="rotate(-90 20,%d)" text-anchor="middle">%s</text>`,
		height/2, height/2, yLabel)

	const tickCount = 5
	plotWidth := float64(width - left - right)
	plotHeight := float64(height - top - bottom)
	formatTick := func(value float64) string {
		if math.Abs(value-math.Round(value)) < 0.0001 {
			return fmt.Sprintf("%.0f", value)
		}
		return fmt.Sprintf("%.2f", value)
	}

	for index := 0; index <= tickCount; index++ {
		fraction := float64(index) / tickCount
		xValue := fraction * xMax
		xPosition := float64(left) + fraction*plotWidth
		yValue := fraction * yMax
		yPosition := float64(height-bottom) - fraction*plotHeight

		fmt.Fprintf(file, `<line x1="%.1f" y1="%d" x2="%.1f" y2="%d" stroke="#dddddd"/>`,
			xPosition, height-bottom, xPosition, height-bottom+5)
		fmt.Fprintf(file, `<text x="%.1f" y="%d" text-anchor="middle" font-size="12">%s</text>`,
			xPosition, height-bottom+20, formatTick(xValue))
		fmt.Fprintf(file, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="#dddddd"/>`,
			left-5, yPosition, left, yPosition)
		fmt.Fprintf(file, `<text x="%d" y="%.1f" text-anchor="end" dominant-baseline="middle" font-size="12">%s</text>`,
			left-10, yPosition, formatTick(yValue))
	}

	drawLine := func(points []Point, color string, name string, legendY int) {
		if len(points) == 0 {
			return
		}

		fmt.Fprintf(file, `<polyline fill="none" stroke="%s" stroke-width="3" points="`, color)
		for _, point := range points {
			fmt.Fprintf(file, "%.1f,%.1f ", x(point.x), y(point.y))
		}
		fmt.Fprint(file, `"/>`)

		for _, point := range points {
			fmt.Fprintf(file, `<circle cx="%.1f" cy="%.1f" r="4" fill="%s"/>`,
				x(point.x), y(point.y), color)
		}

		legendX := width - right - 150
		fmt.Fprintf(file, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="3"/>`,
			legendX, legendY, legendX+25, legendY, color)
		fmt.Fprintf(file, `<text x="%d" y="%d" dominant-baseline="middle" font-size="12">%s</text>`,
			legendX+32, legendY, name)
	}

	drawLine(first, "blue", "Measured", 42)
	drawLine(second, "red", "Expected log2(N)", 58)

	fmt.Fprint(file, `</svg>`)
}

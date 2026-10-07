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
	probes      float64
}

var linePattern = regexp.MustCompile(
	`config nodes=(\d+) packet_loss=([\d.]+).*success_rate=([\d.]+).*average_probes=([\d.]+)`,
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
		probes, _ := strconv.ParseFloat(match[4], 64)

		results = append(results, Result{
			nodes:       nodes,
			packetLoss:  packetLoss,
			successRate: successRate,
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

	var measured, expected []Point
	for _, result := range values {
		measured = append(measured, Point{
			x: float64(result.nodes),
			y: result.probes,
		})
		expected = append(expected, Point{
			x: float64(result.nodes),
			y: math.Log2(float64(result.nodes)),
		})
	}

	writeSVG(
		"scalability.svg",
		"Lookup scalability",
		"Network size N",
		"Average probes",
		measured,
		expected,
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

	drawLine := func(points []Point, color string) {
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
	}

	drawLine(first, "blue")
	drawLine(second, "red")

	fmt.Fprint(file, `</svg>`)
}

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
)

func main() {
	start := flag.Float64("start", 1, "Start value for linear or exponential buckets.")
	factor := flag.Float64("factor", 5, "Factor used when computing exponential buckets.")
	width := flag.Float64("width", 1, "Width of linear buckets")
	count := flag.Int("count", 10, "Number of linear or exponential buckets")
	mode := flag.String("mode", "linear", "Linear or exponential.")
	columnWidth := flag.Int("column-width", 30, "Width of the largest bin")
	explicitBounds := flag.String("buckets", "", "Explicit buckets: comma separated bucket boundaries.")

	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)

	var bounds []float64
	var err error

	if *explicitBounds != "" {
		bounds, err = parseBucketBoundaries(*explicitBounds)
	} else if *mode == "linear" || *mode == "lin" {
		bounds, err = linearBuckets(*start, *width, *count)
	} else if *mode == "exponential" || *mode == "exp" {
		bounds, err = exponentialBuckets(*start, *factor, *count)
	}

	if err != nil {
		printlnAndExit("Failed to create buckets:", err)
	}

	buckets, sum, samples, min, max := parseValues(scanner, bounds)

	printHistogram(os.Stdout, buckets, samples, float64(*columnWidth), true)
	printSummary(os.Stdout, buckets, sum, samples, min, max)
}

func parseBucketBoundaries(inp string) ([]float64, error) {
	s := strings.Split(inp, ",")

	result := make([]float64, 0, len(s))
	for _, b := range s {
		v, err := strconv.ParseFloat(b, 64)
		if err != nil {
			return nil, fmt.Errorf("non-numeric input: %q", b)
		}
		result = append(result, v)
	}

	sort.Float64s(result)
	return result, nil
}

// Returns sum of values for each bucket, total sum and total number of samples. One extra bucket for values larger
// than latest bucket is created. Input buckets must be sorted.
func parseValues(scanner *bufio.Scanner, buckets []float64) (result []promBucket, sum, count, min, max float64) {
	result = make([]promBucket, len(buckets)+1)
	for ix := 0; ix < len(buckets); ix++ {
		result[ix].upperBound = buckets[ix]
	}
	result[len(buckets)].upperBound = math.Inf(1)

	first := true

	for scanner.Scan() {
		v := strings.TrimSpace(scanner.Text())
		sample, err := strconv.ParseFloat(v, 64)
		if err != nil {
			printlnAndExit("found non-numerical input:", v)
		}

		if first {
			min = sample
			max = sample
			first = false
		}

		if sample < min {
			min = sample
		}
		if sample > max {
			max = sample
		}

		// Increment all buckets where sample is <= upperBound.
		for ix := sort.SearchFloat64s(buckets, sample); ix < len(result); ix++ {
			result[ix].count++
		}
		sum += sample
		count++
	}

	return
}

func linearBuckets(start, width float64, count int) ([]float64, error) {
	if count < 1 {
		return nil, fmt.Errorf("--linear-buckets needs a positive count")
	}
	buckets := make([]float64, count)
	for i := range buckets {
		buckets[i] = start
		start += width
	}
	return buckets, nil
}

func exponentialBuckets(start, factor float64, count int) ([]float64, error) {
	if count < 1 {
		return nil, fmt.Errorf("exponential buckets need a positive count")
	}
	if start <= 0 {
		return nil, fmt.Errorf("exponential buckets need a positive start value")
	}
	if factor <= 1 {
		return nil, fmt.Errorf("exponential buckets need a factor greater than 1")
	}
	buckets := make([]float64, count)
	for i := range buckets {
		buckets[i] = start
		start *= factor
	}
	return buckets, nil
}

func printlnAndExit(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
	os.Exit(1)
}

// printHistogram displays a histogram. The bar width determines the width of
// the widest bar. Labels can optionally be right justified.
func printHistogram(out io.Writer, buckets []promBucket, samples float64, barWidth float64, justify bool) {
	var labels []string
	for i := 0; i < len(buckets); i++ {
		switch {
		case i == 0:
			labels = append(labels, fmt.Sprintf("(-∞ .. %0.6g]", buckets[i].upperBound))
		case i == len(buckets)-1:
			labels = append(labels, fmt.Sprintf("(%.6g .. +∞)", buckets[i-1].upperBound))
		default:
			labels = append(labels, fmt.Sprintf("(%.6g .. %.6g]", buckets[i-1].upperBound, buckets[i].upperBound))
		}
	}

	var (
		maxFreq    = maxFrequency(buckets)
		labelWidth = maxStringWidth(labels)
	)

	prev := float64(0)
	for ix := range buckets {
		bucketSamples := buckets[ix].count - prev
		normalizedWidth := bucketSamples / maxFreq
		prev = buckets[ix].count

		width := normalizedWidth * barWidth
		prefix := paddedString(labels[ix], labelWidth, justify)

		fmt.Fprintf(out, "%s %s %.0f (%0.1f %%)\n", prefix, column(width), bucketSamples, 100*bucketSamples/samples)
	}
}

func printSummary(out io.Writer, bucketVals []promBucket, sum, samples, min, max float64) {
	stats := []string{
		fmt.Sprintf("%s=%.0f", "count", samples),
		fmt.Sprintf("%s=%g", "p50", bucketQuantile(0.5, bucketVals)),
		fmt.Sprintf("%s=%g", "p90", bucketQuantile(0.9, bucketVals)),
		fmt.Sprintf("%s=%g", "p95", bucketQuantile(0.95, bucketVals)),
		fmt.Sprintf("%s=%g", "p99", bucketQuantile(0.99, bucketVals)),
		fmt.Sprintf("%s=%g", "avg", sum/samples),
		fmt.Sprintf("%s=%g", "min", min),
		fmt.Sprintf("%s=%g", "max", max),
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "summary:")
	fmt.Fprintln(out, " "+strings.Join(stats, ", "))
}

// paddedString returns the string justified in a string of given width.
func paddedString(str string, width int, justify bool) string {
	if justify {
		return just(str, width)
	}

	return fill(str, width)
}

func fill(s string, w int) string {
	return s + strings.Repeat(" ", w-runewidth.StringWidth(s))
}

func just(s string, w int) string {
	return strings.Repeat(" ", w-runewidth.StringWidth(s)) + s
}

var boxes = []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}

// columns returns a horizontal bar of a given size.
func column(size float64) string {
	fraction := size - math.Floor(size)
	index := int(fraction * float64(len(boxes)))

	return strings.Repeat(boxes[len(boxes)-1], int(size)) + boxes[index]
}

// maxStringWidth returns the width of the widest string in a string slice. It
// supports CJK through the go-runewidth package.
func maxStringWidth(strs []string) int {
	var max int

	for _, str := range strs {
		w := runewidth.StringWidth(str)
		if w > max {
			max = w
		}
	}

	return max
}

func maxFrequency(buckets []promBucket) float64 {
	var max = buckets[0].count

	for ix := 1; ix < len(buckets); ix++ {
		d := buckets[ix].count - buckets[ix-1].count
		if d > max {
			max = d
		}
	}

	return max
}

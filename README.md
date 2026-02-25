# promfreq

Command-line utility for displaying histograms from standard input using Prometheus-style buckets.

Based on [freq](https://github.com/marcusolsson/freq), but uses Prometheus cumulative buckets and doesn't keep all values in memory.

## Install

```bash
go install github.com/pstibrany/promfreq@latest
```

Or build from source:

```bash
git clone https://github.com/pstibrany/promfreq.git
cd promfreq
go build
```

## Usage

Pipe numeric values (one per line) into `promfreq`:

```bash
echo -e "1\n2\n3\n5\n8\n10\n12\n15\n20\n25\n30\n50\n100" | promfreq --mode linear --start 0 --width 10 --count 10
```

```
 (-∞ .. 0] ▏ 0 (0.0 %)
 (0 .. 10] ██████████████████████████████▏ 6 (46.2 %)
(10 .. 20] ███████████████▏ 3 (23.1 %)
(20 .. 30] ██████████▏ 2 (15.4 %)
(30 .. 40] ▏ 0 (0.0 %)
(40 .. 50] █████▏ 1 (7.7 %)
(50 .. 60] ▏ 0 (0.0 %)
(60 .. 70] ▏ 0 (0.0 %)
(70 .. 80] ▏ 0 (0.0 %)
(80 .. 90] ▏ 0 (0.0 %)
(90 .. +∞) █████▏ 1 (7.7 %)

summary:
 count=13, p50=11.666666666666666, p90=47, p95=90, p99=90, avg=21.615384615384617, min=1, max=100
```

### Explicit buckets

```bash
echo -e "1\n2\n3\n5\n8\n10\n12\n15\n20\n25\n30\n50\n100" | promfreq --buckets "1,5,10,25,50,100"
```

```
  (-∞ .. 1] ███████▋ 1 (7.7 %)
   (1 .. 5] ██████████████████████▋ 3 (23.1 %)
  (5 .. 10] ███████████████▏ 2 (15.4 %)
 (10 .. 25] ██████████████████████████████▏ 4 (30.8 %)
 (25 .. 50] ███████████████▏ 2 (15.4 %)
(50 .. 100] ███████▋ 1 (7.7 %)
(100 .. +∞) ▏ 0 (0.0 %)
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--mode` | `linear` | Bucket mode: `linear` (or `lin`) / `exponential` (or `exp`) |
| `--start` | `1` | Start value for linear or exponential buckets |
| `--width` | `1` | Width of each linear bucket |
| `--factor` | `5` | Multiplication factor for exponential buckets |
| `--count` | `10` | Number of buckets to generate |
| `--buckets` | | Explicit comma-separated bucket boundaries (overrides `--mode`) |
| `--column-width` | `30` | Character width of the largest histogram bar |

## How it works

Values are read from stdin and counted into cumulative Prometheus-style buckets. Each bucket stores the count of all values less than or equal to its upper bound. This means the tool uses constant memory proportional to the number of buckets, regardless of input size.

The summary line reports count, p50, p90, p95, p99, average, min, and max. Quantiles are estimated from bucket boundaries using linear interpolation (the same method Prometheus uses).


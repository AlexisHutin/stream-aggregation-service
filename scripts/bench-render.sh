#!/usr/bin/env bash

###############################################################################
# NOTE:
# This script was created and iterated with AI assistance.
# Human review and validation remain required before production use.
###############################################################################

set -e

INPUT="./build/bench.txt"
CSV="./build/bench.csv"
CSV_PERCENTILE="./build/bench-percentile.csv"
CSV_COMPUTE="./build/bench-compute.csv"
MD="./build/bench.md"
PNG_PERCENTILE="./assets/bench-percentile.png"
PNG_COMPUTE="./assets/bench-compute.png"

echo "==> Parsing bench.txt -> CSV"

# Extract clean benchmark lines
grep -E '^Benchmark' "$INPUT" | awk '
function isnum(v) {
    return (v ~ /^[0-9]+([.][0-9]+)?$/)
}
{
    name=$1
    ns=$3
    bytes=$5
    allocs=$7

    # Strip benchmark units
    gsub("ns/op", "", ns)
    gsub("B/op", "", bytes)
    gsub("allocs/op", "", allocs)

    # Ignore incomplete or invalid rows
    if (name == "" || ns == "" || bytes == "" || allocs == "") {
        next
    }
    if (!isnum(ns) || !isnum(bytes) || !isnum(allocs)) {
        next
    }

    print name "," ns "," bytes "," allocs
}
' | awk 'NF > 0' > "$CSV"

# Add CSV header
sed -i '1i name,ns_per_op,bytes_per_op,allocs' "$CSV"

# If no valid data is available, exit gracefully
if [ "$(wc -l < "$CSV")" -le 1 ]; then
    echo "No valid benchmark data found in $INPUT"
    exit 0
fi

awk -F',' 'NR == 1 || $1 ~ /^BenchmarkPercentile/ { print }' "$CSV" > "$CSV_PERCENTILE"
awk -F',' 'NR == 1 || $1 ~ /^BenchmarkComputeAnalysisStats/ { print }' "$CSV" > "$CSV_COMPUTE"

echo "==> Generating Markdown table"

write_table() {
    local title="$1"
    local data_csv="$2"
    local image_file="$3"

    echo "### ${title}" >> "$MD"
    echo >> "$MD"

    if [ "$(wc -l < "$data_csv")" -le 1 ]; then
        echo "_No benchmark data available._" >> "$MD"
        echo >> "$MD"
        return
    fi

    echo "| Benchmark | ns/op | B/op | allocs |" >> "$MD"
    echo "|----------|------:|-----:|-------:|" >> "$MD"

    tail -n +2 "$data_csv" | while IFS=',' read -r name ns bytes allocs
    do
        printf "| %s | %s | %s | %s |\n" "$name" "$ns" "$bytes" "$allocs" >> "$MD"
    done

    echo >> "$MD"
    echo "![${title}](${image_file})" >> "$MD"
    echo >> "$MD"
}

: > "$MD"
write_table "Percentile benchmarks" "$CSV_PERCENTILE" "$PNG_PERCENTILE"
write_table "ComputeAnalysisStats benchmarks" "$CSV_COMPUTE" "$PNG_COMPUTE"

README="README.md"
TMP="./build/README.tmp"

awk '
BEGIN {in_block=0}
{
    if ($0 ~ /<!-- BENCHMARKS START -->/) {
        print
        system("cat ./build/bench.md")
        in_block=1
        next
    }
    if ($0 ~ /<!-- BENCHMARKS END -->/) {
        in_block=0
    }
    if (!in_block) print
}
' "$README" > "$TMP"

mv "$TMP" "$README"

echo "==> Generating graph"

render_graph() {
    local data_csv="$1"
    local output_png="$2"
    local title="$3"

    if [ "$(wc -l < "$data_csv")" -le 1 ]; then
        return
    fi

gnuplot <<EOF
set terminal png size 1200,600
set output "$output_png"
set datafile separator ","
set title "$title"
set xlabel "Benchmark"
set ylabel "ns/op"
set xtics rotate by -45
set style data linespoints
plot "$data_csv" skip 1 using 2:xtic(1) with linespoints title "ns/op"
EOF
}

render_graph "$CSV_PERCENTILE" "$PNG_PERCENTILE" "Percentile Benchmark Scaling"
render_graph "$CSV_COMPUTE" "$PNG_COMPUTE" "ComputeAnalysisStats Benchmark Scaling"

echo "==> Done"
echo "Generated:"
echo "- $CSV"
echo "- $CSV_PERCENTILE"
echo "- $CSV_COMPUTE"
echo "- $MD"
echo "- $PNG_PERCENTILE"
echo "- $PNG_COMPUTE"
#!/usr/bin/env bash
set -euo pipefail

EXP_DIR=""
SOURCE_FILE=""
OUT_DIR=""
MIN_RATIO="0.90"
MIN_SCENARIOS=""

usage() {
	cat <<'EOF'
Usage:
  scripts/unknown-register-apply-noise-filter.sh [options]

Options:
  --exp <dir>            Apply-survey directory (default: latest experiments/apply-survey-*)
  --source <file>        Source TSV (default: <exp>/summary/_unknown_source.tsv)
  --out <dir>            Output dir (default: <exp>/summary/apply-noise-filter-<timestamp>)
  --min-ratio <float>    Treat keys present in at least this ratio of scenarios as noise
                         (default: 0.90)
  --min-scenarios <n>    Absolute threshold for noise keys; overrides --min-ratio
  -h, --help             Show help

Input format:
  <scenario>\t<category>\t<bank>\t<register>\t<baseline_hex>\t<seen_hex>

What it does:
  1. Counts how many distinct scenarios each unknown register key appears in.
  2. Builds a "common apply noise" set from keys present in most scenarios.
  3. Filters those keys out.
  4. Produces per-scenario, aggregate-by-register, and aggregate-by-category reports.
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--exp)
			EXP_DIR="$2"
			shift 2
			;;
		--source)
			SOURCE_FILE="$2"
			shift 2
			;;
		--out)
			OUT_DIR="$2"
			shift 2
			;;
		--min-ratio)
			MIN_RATIO="$2"
			shift 2
			;;
		--min-scenarios)
			MIN_SCENARIOS="$2"
			shift 2
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			echo "Unknown argument: $1" >&2
			usage >&2
			exit 1
			;;
	esac
done

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"

if [[ -z "$EXP_DIR" ]]; then
	latest_exp="$(ls -1dt "$ROOT_DIR"/experiments/apply-survey-* 2>/dev/null | head -n 1 || true)"
	if [[ -z "$latest_exp" ]]; then
		echo "No apply-survey directories found under $ROOT_DIR/experiments" >&2
		exit 1
	fi
	EXP_DIR="$latest_exp"
fi

if [[ -z "$SOURCE_FILE" ]]; then
	SOURCE_FILE="$EXP_DIR/summary/_unknown_source.tsv"
fi

if [[ ! -f "$SOURCE_FILE" ]]; then
	echo "Source file not found: $SOURCE_FILE" >&2
	exit 1
fi

if [[ -z "$OUT_DIR" ]]; then
	OUT_DIR="$EXP_DIR/summary/apply-noise-filter-$(date +%Y%m%d-%H%M%S)"
fi

mkdir -p "$OUT_DIR/per-scenario"

TOTAL_SCENARIOS="$(awk -F '\t' '!seen[$1]++ { count++ } END { print count + 0 }' "$SOURCE_FILE")"
if [[ "$TOTAL_SCENARIOS" == "0" ]]; then
	echo "No scenarios found in $SOURCE_FILE" >&2
	exit 1
fi

if [[ -z "$MIN_SCENARIOS" ]]; then
	MIN_SCENARIOS="$(awk -v total="$TOTAL_SCENARIOS" -v ratio="$MIN_RATIO" '
		BEGIN {
			x = total * ratio
			if (x == int(x)) {
				print int(x)
			} else {
				print int(x) + 1
			}
		}
	')"
fi

if ! [[ "$MIN_SCENARIOS" =~ ^[0-9]+$ ]] || (( MIN_SCENARIOS < 1 )); then
	echo "--min-scenarios must be an integer >= 1" >&2
	exit 1
fi

if (( MIN_SCENARIOS > TOTAL_SCENARIOS )); then
	echo "--min-scenarios ($MIN_SCENARIOS) exceeds total scenarios ($TOTAL_SCENARIOS)" >&2
	exit 1
fi

NOISE_KEYS_FILE="$OUT_DIR/noise_keys.tsv"
FILTERED_FILE="$OUT_DIR/filtered_source.tsv"
AGG_TSV="$OUT_DIR/aggregate.tsv"
AGG_TXT="$OUT_DIR/aggregate.txt"
CAT_TSV="$OUT_DIR/by-category.tsv"
CAT_TXT="$OUT_DIR/by-category.txt"

awk -F '\t' -v total="$TOTAL_SCENARIOS" -v min_scenarios="$MIN_SCENARIOS" '
	{
		key = $3 ":" $4
		scenario = $1
		if (!(key SUBSEP scenario in seen_scenario)) {
			seen_scenario[key SUBSEP scenario] = 1
			count[key]++
			if (scenarios[key] == "") {
				scenarios[key] = scenario
			} else {
				scenarios[key] = scenarios[key] "," scenario
			}
		}
		if (!(key SUBSEP $6 in seen_value)) {
			seen_value[key SUBSEP $6] = 1
			if (values[key] == "") {
				values[key] = $6
			} else {
				values[key] = values[key] "," $6
			}
		}
		base[key] = $5
	}
	END {
		for (key in count) {
			if (count[key] < min_scenarios) {
				continue
			}
			split(key, p, ":")
			printf "%s\t%s\t%s\t%d\t%d\t%.4f\t%s\t%s\t%s\n",
				p[1], p[2], key, count[key], total, count[key] / total, base[key], values[key], scenarios[key]
		}
	}
' "$SOURCE_FILE" | sort -t $'\t' -k4,4nr -k1,1n -k2,2n >"$NOISE_KEYS_FILE"

awk -F '\t' '
	FNR == NR {
		noise[$3] = 1
		next
	}
	{
		key = $3 ":" $4
		if (!(key in noise)) {
			print
		}
	}
' "$NOISE_KEYS_FILE" "$SOURCE_FILE" >"$FILTERED_FILE"

if [[ -s "$FILTERED_FILE" ]]; then
	awk -F '\t' -v out="$OUT_DIR/per-scenario" '
		{
			print > (out "/" $1 ".tsv")
		}
	' "$FILTERED_FILE"

	for tsv_file in "$OUT_DIR"/per-scenario/*.tsv; do
		step="$(basename "$tsv_file" .tsv)"
		txt_file="$OUT_DIR/per-scenario/$step.txt"
		{
			echo "[$step] unknown registers after common-apply-noise filtering"
			awk -F '\t' '
				{
					printf "  B%s reg %3d (0x%02X): baseline=0x%s seen=0x%s [%s]\n", $3, $4, $4, $5, $6, $2
				}
			' "$tsv_file"
		} >"$txt_file"
	done

	awk -F '\t' '
		{
			key = $3 ":" $4
			scenario = $1
			category = $2
			if (!(key SUBSEP scenario in seen_scenario)) {
				seen_scenario[key SUBSEP scenario] = 1
				count[key]++
				if (scenarios[key] == "") {
					scenarios[key] = scenario
				} else {
					scenarios[key] = scenarios[key] "," scenario
				}
			}
			if (!(key SUBSEP category in seen_category)) {
				seen_category[key SUBSEP category] = 1
				if (categories[key] == "") {
					categories[key] = category
				} else {
					categories[key] = categories[key] "," category
				}
			}
			if (!(key SUBSEP $6 in seen_value)) {
				seen_value[key SUBSEP $6] = 1
				if (values[key] == "") {
					values[key] = $6
				} else {
					values[key] = values[key] "," $6
				}
			}
			base[key] = $5
		}
		END {
			for (key in count) {
				split(key, p, ":")
				printf "%s\t%s\t%s\t%d\t%s\t%s\t%s\n", p[1], p[2], base[key], count[key], values[key], categories[key], scenarios[key]
			}
		}
	' "$FILTERED_FILE" | sort -n -k1,1 -k2,2 >"$AGG_TSV"

	{
		echo "Unknown register activity after common-apply-noise filtering:"
		awk -F '\t' '
			{
				printf "  B%s reg %3d (0x%02X): baseline=0x%s seen=0x%s in %d scenario(s), categories=%s\n", $1, $2, $2, $3, $5, $4, $6
			}
		' "$AGG_TSV"
	} >"$AGG_TXT"

	awk -F '\t' '
		{
			category = $2
			key = $3 ":" $4
			if (!(category SUBSEP key in seen_key)) {
				seen_key[category SUBSEP key] = 1
				count[category]++
			}
		}
		END {
			for (category in count) {
				printf "%s\t%d\n", category, count[category]
			}
		}
	' "$FILTERED_FILE" | sort >"$CAT_TSV"

	{
		echo "Remaining unknown-register counts by category:"
		awk -F '\t' '
			{
				printf "  %s: %d register(s)\n", $1, $2
			}
		' "$CAT_TSV"
	} >"$CAT_TXT"
else
	cat >"$AGG_TXT" <<'EOF'
No unknown registers remain after common-apply-noise filtering.
EOF
	: >"$AGG_TSV"
	cat >"$CAT_TXT" <<'EOF'
No unknown registers remain after common-apply-noise filtering.
EOF
	: >"$CAT_TSV"
fi

{
	echo "Common apply-noise keys:"
	if [[ ! -s "$NOISE_KEYS_FILE" ]]; then
		echo "  None"
	else
		awk -F '\t' '
			{
				printf "  B%s reg %3d (0x%02X): %d/%d scenarios (ratio=%.2f), baseline=0x%s seen=0x%s\n", $1, $2, $2, $4, $5, $6, $7, $8
			}
		' "$NOISE_KEYS_FILE"
	fi
} >"$OUT_DIR/noise_keys.txt"

echo "Done."
echo "Experiment: $EXP_DIR"
echo "Source: $SOURCE_FILE"
echo "Total scenarios: $TOTAL_SCENARIOS"
echo "Noise threshold: $MIN_SCENARIOS scenario(s)"
echo "Output: $OUT_DIR"
echo
echo "Noise summary:"
cat "$OUT_DIR/noise_keys.txt"
echo
echo "Filtered summary:"
cat "$AGG_TXT"

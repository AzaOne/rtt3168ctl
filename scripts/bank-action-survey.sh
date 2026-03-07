#!/usr/bin/env bash
set -euo pipefail

SAMPLES=8
DURATION=6
BANK_SPEC="0,1,2,7,128,133,147,255"
OUT_DIR=""
BIN_PATH=""

usage() {
	cat <<'EOF'
Usage:
  scripts/bank-action-survey.sh [options]

Options:
  --banks <spec>       Bank spec for -dump-banks
                       (default: 0,1,2,7,128,133,147,255)
  --samples <n>        Dumps per step (default: 8)
  --duration <sec>     Step duration in seconds (default: 6)
  --out <dir>          Output directory
                       (default: experiments/bank-action-<timestamp>)
  --bin <path>         Path to rtt3168ctl binary (default: auto)
  -h, --help           Show help

What this script does:
  1. Captures a baseline dump for selected banks.
  2. Guides runtime actions (move/click/scroll/buttons).
  3. Reports changed registers per step relative to baseline.
  4. Builds an idle-filtered summary to isolate action-specific keys.
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--banks)
			BANK_SPEC="$2"
			shift 2
			;;
		--samples)
			SAMPLES="$2"
			shift 2
			;;
		--duration)
			DURATION="$2"
			shift 2
			;;
		--out)
			OUT_DIR="$2"
			shift 2
			;;
		--bin)
			BIN_PATH="$2"
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

if ! [[ "$SAMPLES" =~ ^[0-9]+$ ]] || (( SAMPLES < 1 )); then
	echo "--samples must be an integer >= 1" >&2
	exit 1
fi

if ! [[ "$DURATION" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
	echo "--duration must be numeric" >&2
	exit 1
fi

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"

if [[ -z "$OUT_DIR" ]]; then
	OUT_DIR="$ROOT_DIR/experiments/bank-action-$(date +%Y%m%d-%H%M%S)"
fi

if [[ -n "$BIN_PATH" ]]; then
	if [[ ! -x "$BIN_PATH" ]]; then
		echo "Binary is not executable: $BIN_PATH" >&2
		exit 1
	fi
elif [[ -x "$ROOT_DIR/build/rtt3168ctl" ]]; then
	BIN_PATH="$ROOT_DIR/build/rtt3168ctl"
fi

if (( SAMPLES == 1 )); then
	STEP_INTERVAL="0"
else
	STEP_INTERVAL="$(awk -v d="$DURATION" -v s="$SAMPLES" 'BEGIN { printf "%.3f", d / (s - 1) }')"
fi

mkdir -p "$OUT_DIR/raw" "$OUT_DIR/parsed" "$OUT_DIR/summary"

run_dump() {
	local out_file="$1"
	if [[ -n "$BIN_PATH" ]]; then
		"$BIN_PATH" -mode dump -dump-banks "$BANK_SPEC" >"$out_file"
		return
	fi

	(
		cd "$ROOT_DIR"
		go run ./cmd/rtt3168ctl -mode dump -dump-banks "$BANK_SPEC"
	) >"$out_file"
}

parse_dump() {
	local raw_file="$1"
	local parsed_file="$2"
	awk '
		match($0, /^Memory Dump \(Bank ([0-9]+), registers 0\.\.(127|255)\)$/, m) {
			bank = m[1] + 0
			next
		}
		match($0, /^([0-9]+) \(0x[0-9A-F]{2}\): 0x([0-9A-F]{2})$/, m) {
			printf "%d\t%d\t%s\n", bank, m[1] + 0, m[2]
		}
	' "$raw_file" >"$parsed_file"
}

capture_step_samples() {
	local step_id="$1"
	local step_title="$2"
	local step_instruction="$3"
	local step_raw_dir="$OUT_DIR/raw/$step_id"
	local step_parsed_dir="$OUT_DIR/parsed/$step_id"

	mkdir -p "$step_raw_dir" "$step_parsed_dir"

	echo
	echo "[$step_id] $step_title"
	echo "Instruction: $step_instruction"
	echo "Keep doing it for ${DURATION}s while samples are captured."
	read -r -p "Press Enter to start this step..."

	local i
	for (( i=1; i<=SAMPLES; i++ )); do
		local raw_file="$step_raw_dir/sample-$(printf "%02d" "$i").txt"
		local parsed_file="$step_parsed_dir/sample-$(printf "%02d" "$i").tsv"
		run_dump "$raw_file"
		parse_dump "$raw_file" "$parsed_file"
		printf "  captured sample %d/%d\r" "$i" "$SAMPLES"
		if (( i < SAMPLES )) && [[ "$STEP_INTERVAL" != "0" ]]; then
			sleep "$STEP_INTERVAL"
		fi
	done
	printf "\n"
}

detect_changes() {
	local baseline_file="$1"
	shift
	awk '
		FNR == NR {
			base[$1 ":" $2] = $3
			next
		}
		{
			key = $1 ":" $2
			if (!(key in base)) {
				next
			}
			if ($3 != base[key]) {
				changed[key] = 1
				if (vals[key] == "") {
					vals[key] = $3
				} else if (index("," vals[key] ",", "," $3 ",") == 0) {
					vals[key] = vals[key] "," $3
				}
			}
		}
		END {
			for (k in changed) {
				split(k, p, ":")
				printf "%s\t%s\t%s\t%s\n", p[1], p[2], base[k], vals[k]
			}
		}
	' "$baseline_file" "$@" | sort -n -k1,1 -k2,2
}

write_step_summary() {
	local step_id="$1"
	local step_title="$2"
	local baseline_file="$3"
	shift 3
	local files=("$@")
	local tsv_file="$OUT_DIR/summary/$step_id.tsv"
	local txt_file="$OUT_DIR/summary/$step_id.txt"

	detect_changes "$baseline_file" "${files[@]}" >"$tsv_file"

	{
		echo "[$step_id] $step_title"
		if [[ ! -s "$tsv_file" ]]; then
			echo "  No register changes vs baseline."
		else
			awk -F '\t' '
				{
					vals = $4
					gsub(",", ",0x", vals)
					printf "  B%s reg %3d (0x%02X): baseline=0x%s seen=0x%s\n", $1, $2, $2, $3, vals
				}
			' "$tsv_file"
		fi
	} >"$txt_file"

	cat "$txt_file"
}

build_aggregate_summary() {
	local aggregate_source="$OUT_DIR/summary/_aggregate_source.tsv"
	local aggregate_tsv="$OUT_DIR/summary/aggregate.tsv"
	local aggregate_txt="$OUT_DIR/summary/aggregate.txt"

	: >"$aggregate_source"

	local step_tsv
	for step_tsv in "$OUT_DIR"/summary/S*.tsv; do
		if [[ -s "$step_tsv" ]]; then
			local step_name
			step_name="$(basename "$step_tsv" .tsv)"
			awk -F '\t' -v step="$step_name" '{ printf "%s\t%s\t%s\t%s\t%s\n", step, $1, $2, $3, $4 }' "$step_tsv" >>"$aggregate_source"
		fi
	done

	if [[ ! -s "$aggregate_source" ]]; then
		cat >"$aggregate_txt" <<'EOF'
No registers changed in any action step.
EOF
		return
	fi

	awk -F '\t' '
		{
			key = $2 ":" $3
			step = $1
			if (!(key SUBSEP step in seen_step)) {
				seen_step[key SUBSEP step] = 1
				count[key]++
				if (steps[key] == "") {
					steps[key] = step
				} else {
					steps[key] = steps[key] "," step
				}
			}
		}
		END {
			for (k in count) {
				split(k, p, ":")
				printf "%s\t%s\t%d\t%s\n", p[1], p[2], count[k], steps[k]
			}
		}
	' "$aggregate_source" | sort -n -k1,1 -k2,2 >"$aggregate_tsv"

	{
		echo "Register activity by action coverage:"
		awk -F '\t' '
			{
				printf "  B%s reg %3d (0x%02X): changed in %d step(s): %s\n", $1, $2, $2, $3, $4
			}
		' "$aggregate_tsv"
	} >"$aggregate_txt"
}

build_idle_filtered_summary() {
	local aggregate_source="$OUT_DIR/summary/_aggregate_source.tsv"
	local idle_step="S01-idle-control"
	local noise_keys="$OUT_DIR/summary/idle-noise-keys.tsv"
	local filtered_source="$OUT_DIR/summary/action-specific.tsv"
	local filtered_txt="$OUT_DIR/summary/action-specific.txt"

	if [[ ! -s "$aggregate_source" ]]; then
		return
	fi

	awk -F '\t' -v idle="$idle_step" '
		$1 == idle {
			key = $2 ":" $3
			if (!(key in seen)) {
				seen[key] = 1
				printf "%s\t%s\t%s\n", $2, $3, key
			}
		}
	' "$aggregate_source" | sort -n -k1,1 -k2,2 >"$noise_keys"

	awk -F '\t' '
		FNR == NR {
			noise[$3] = 1
			next
		}
		{
			key = $2 ":" $3
			if ($1 == "S01-idle-control") {
				next
			}
			if (!(key in noise)) {
				print
			}
		}
	' "$noise_keys" "$aggregate_source" >"$filtered_source"

	{
		echo "Action-specific register activity after idle filtering:"
		if [[ ! -s "$filtered_source" ]]; then
			echo "  None."
		else
			awk -F '\t' '
				{
					vals = $5
					gsub(",", ",0x", vals)
					printf "  [%s] B%s reg %3d (0x%02X): baseline=0x%s seen=0x%s\n", $1, $2, $3, $3, $4, vals
				}
			' "$filtered_source"
		fi
	} >"$filtered_txt"
}

echo "Bank action survey"
echo "Output directory: $OUT_DIR"
echo "Bank spec: $BANK_SPEC"
if [[ -n "$BIN_PATH" ]]; then
	echo "CLI binary: $BIN_PATH"
else
	echo "CLI binary: go run ./cmd/rtt3168ctl (fallback)"
fi
echo
echo "Preparation:"
echo "  1) Close vendor software."
echo "  2) Keep mouse on the same surface."
echo "  3) Do not change profile settings during this run."
echo "  4) Ensure device access permissions are configured."
echo

BASELINE_ID="B00-baseline"
BASELINE_TITLE="Baseline (idle)"
BASELINE_INSTRUCTION="Do not touch the mouse."
capture_step_samples "$BASELINE_ID" "$BASELINE_TITLE" "$BASELINE_INSTRUCTION"

BASELINE_FILE="$OUT_DIR/parsed/$BASELINE_ID/sample-01.tsv"

STEP_IDS=(
	"S01-idle-control"
	"S02-move"
	"S03-left-click"
	"S04-right-click"
	"S05-middle-click"
	"S06-scroll"
	"S07-side-buttons"
	"S08-cpi-button"
)

STEP_TITLES=(
	"Idle control"
	"Move tracking"
	"Left button"
	"Right button"
	"Middle button"
	"Wheel scroll"
	"Side buttons"
	"CPI/DPI button"
)

STEP_INSTRUCTIONS=(
	"Do not move or click."
	"Move mouse continuously in circles and straight lines."
	"Click left button repeatedly."
	"Click right button repeatedly."
	"Click wheel button repeatedly."
	"Scroll wheel up and down repeatedly."
	"Press side back/forward buttons repeatedly."
	"Press CPI/DPI button repeatedly."
)

total_steps="${#STEP_IDS[@]}"
for (( idx=0; idx<total_steps; idx++ )); do
	step_id="${STEP_IDS[$idx]}"
	step_title="${STEP_TITLES[$idx]}"
	step_instruction="${STEP_INSTRUCTIONS[$idx]}"

	capture_step_samples "$step_id" "$step_title" "$step_instruction"

	step_files=( "$OUT_DIR/parsed/$step_id"/sample-*.tsv )
	write_step_summary "$step_id" "$step_title" "$BASELINE_FILE" "${step_files[@]}"
done

build_aggregate_summary
build_idle_filtered_summary

echo
echo "Final aggregate summary:"
cat "$OUT_DIR/summary/aggregate.txt"
echo
echo "Idle-filtered summary:"
cat "$OUT_DIR/summary/action-specific.txt"
echo
echo "Done. Review artifacts:"
echo "  Raw dumps:    $OUT_DIR/raw"
echo "  Parsed dumps: $OUT_DIR/parsed"
echo "  Summaries:    $OUT_DIR/summary"

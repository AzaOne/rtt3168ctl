#!/usr/bin/env bash
set -euo pipefail

EXP_DIR=""
SOURCE_FILE=""
OUT_DIR=""
IDLE_STEP="S01-idle-control"

usage() {
	cat <<'EOF'
Usage:
  scripts/unknown-register-action-specific.sh [options]

Options:
  --exp <dir>         Experiment directory (default: latest experiments/unknown-reg-*)
  --source <file>     Source TSV (default: <exp>/summary/_aggregate_source.tsv)
  --out <dir>         Output dir (default: <exp>/summary/action-specific-<timestamp>)
  --idle-step <id>    Step used as noise profile (default: S01-idle-control)
  -h, --help          Show help

Input format:
  <step>\t<bank>\t<register>\t<baseline_hex>\t<seen_hex_csv>

What it does:
  1) Builds a noise set from --idle-step.
  2) Removes any register keys (bank:register) present in the noise set.
  3) Produces per-step and aggregate action-specific reports.
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
		--idle-step)
			IDLE_STEP="$2"
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
	latest_exp="$(ls -1dt "$ROOT_DIR"/experiments/unknown-reg-* 2>/dev/null | head -n 1 || true)"
	if [[ -z "$latest_exp" ]]; then
		echo "No experiment directories found under $ROOT_DIR/experiments" >&2
		exit 1
	fi
	EXP_DIR="$latest_exp"
fi

if [[ -z "$SOURCE_FILE" ]]; then
	SOURCE_FILE="$EXP_DIR/summary/_aggregate_source.tsv"
fi

if [[ ! -f "$SOURCE_FILE" ]]; then
	echo "Source file not found: $SOURCE_FILE" >&2
	exit 1
fi

if [[ -z "$OUT_DIR" ]]; then
	OUT_DIR="$EXP_DIR/summary/action-specific-$(date +%Y%m%d-%H%M%S)"
fi

mkdir -p "$OUT_DIR/per-step"

FILTERED_FILE="$OUT_DIR/filtered_source.tsv"
NOISE_KEYS_FILE="$OUT_DIR/noise_keys.tsv"

awk -F '\t' -v idle="$IDLE_STEP" '
	$1 == idle {
		key = $2 ":" $3
		if (!(key in seen)) {
			seen[key] = 1
			printf "%s\t%s\t%s\n", $2, $3, key
		}
	}
' "$SOURCE_FILE" | sort -n -k1,1 -k2,2 >"$NOISE_KEYS_FILE"

if [[ ! -s "$NOISE_KEYS_FILE" ]]; then
	echo "No entries found for idle step \"$IDLE_STEP\" in $SOURCE_FILE" >&2
	exit 1
fi

awk -F '\t' -v idle="$IDLE_STEP" '
	FNR == NR {
		noise[$3] = 1
		next
	}
	{
		key = $2 ":" $3
		if ($1 == idle) {
			next
		}
		if (!(key in noise)) {
			print
		}
	}
' "$NOISE_KEYS_FILE" "$SOURCE_FILE" >"$FILTERED_FILE"

if [[ ! -s "$FILTERED_FILE" ]]; then
	cat >"$OUT_DIR/aggregate.txt" <<'EOF'
No action-specific unknown registers after idle filtering.
EOF
	echo "No action-specific unknown registers after idle filtering."
	echo "Output directory: $OUT_DIR"
	exit 0
fi

awk -F '\t' -v out="$OUT_DIR/per-step" '
	{
		file = out "/" $1 ".tsv"
		print > file
	}
' "$FILTERED_FILE"

for tsv_file in "$OUT_DIR"/per-step/*.tsv; do
	step="$(basename "$tsv_file" .tsv)"
	txt_file="$OUT_DIR/per-step/$step.txt"
	{
		echo "[$step] action-specific unknown registers (idle-filtered)"
		awk -F '\t' '
			{
				vals = $5
				gsub(",", ",0x", vals)
				printf "  B%s reg %3d (0x%02X): baseline=0x%s seen=0x%s\n", $2, $3, $3, $4, vals
			}
		' "$tsv_file"
	} >"$txt_file"
done

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
' "$FILTERED_FILE" | sort -n -k1,1 -k2,2 >"$OUT_DIR/aggregate.tsv"

{
	echo "Action-specific unknown register activity after idle filtering:"
	awk -F '\t' '
		{
			printf "  B%s reg %3d (0x%02X): changed in %d step(s): %s\n", $1, $2, $2, $3, $4
		}
	' "$OUT_DIR/aggregate.tsv"
} >"$OUT_DIR/aggregate.txt"

echo "Done."
echo "Experiment: $EXP_DIR"
echo "Source: $SOURCE_FILE"
echo "Idle step: $IDLE_STEP"
echo "Output: $OUT_DIR"
echo
echo "Summary:"
cat "$OUT_DIR/aggregate.txt"

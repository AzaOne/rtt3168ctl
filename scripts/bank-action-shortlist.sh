#!/usr/bin/env bash
set -euo pipefail

EXP_DIR=""
SOURCE_FILE=""
OUT_DIR=""
MIN_BANKS="4"

usage() {
	cat <<'EOF'
Usage:
  scripts/bank-action-shortlist.sh [options]

Options:
  --exp <dir>          Bank-action experiment directory
                       (default: latest experiments/bank-action-*)
  --source <file>      Source TSV
                       (default: <exp>/summary/action-specific.tsv)
  --out <dir>          Output dir
                       (default: <exp>/summary/shortlist-<timestamp>)
  --min-banks <n>      Minimum distinct banks for core shortlist
                       (default: 4)
  -h, --help           Show help

Input format:
  <step>\t<bank>\t<register>\t<baseline_hex>\t<seen_hex_csv>

What this script does:
  1. Maps action steps to categories: move/buttons/scroll/cpi.
  2. Produces a per-bank shortlist view.
  3. Produces a cross-bank core shortlist for registers shared by many banks.
  4. Produces a specialists view for useful single-bank registers.
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
		--min-banks)
			MIN_BANKS="$2"
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

if ! [[ "$MIN_BANKS" =~ ^[0-9]+$ ]] || (( MIN_BANKS < 1 )); then
	echo "--min-banks must be an integer >= 1" >&2
	exit 1
fi

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"

if [[ -z "$EXP_DIR" ]]; then
	latest_exp="$(ls -1dt "$ROOT_DIR"/experiments/bank-action-* 2>/dev/null | head -n 1 || true)"
	if [[ -z "$latest_exp" ]]; then
		echo "No bank-action directories found under $ROOT_DIR/experiments" >&2
		exit 1
	fi
	EXP_DIR="$latest_exp"
fi

if [[ -z "$SOURCE_FILE" ]]; then
	SOURCE_FILE="$EXP_DIR/summary/action-specific.tsv"
fi

if [[ ! -f "$SOURCE_FILE" ]]; then
	echo "Source file not found: $SOURCE_FILE" >&2
	exit 1
fi

if [[ -z "$OUT_DIR" ]]; then
	OUT_DIR="$EXP_DIR/summary/shortlist-$(date +%Y%m%d-%H%M%S)"
fi

mkdir -p "$OUT_DIR"

BY_BANK_TSV="$OUT_DIR/by-bank.tsv"
BY_BANK_TXT="$OUT_DIR/by-bank.txt"
CORE_TSV="$OUT_DIR/core-shortlist.tsv"
CORE_TXT="$OUT_DIR/core-shortlist.txt"
SPECIALISTS_TSV="$OUT_DIR/specialists.tsv"
SPECIALISTS_TXT="$OUT_DIR/specialists.txt"

awk '
	function category_for_step(step) {
		if (step == "S02-move") {
			return "move"
		}
		if (step == "S03-left-click" || step == "S04-right-click" || step == "S05-middle-click" || step == "S07-side-buttons") {
			return "buttons"
		}
		if (step == "S06-scroll") {
			return "scroll"
		}
		if (step == "S08-cpi-button") {
			return "cpi"
		}
		return "other"
	}
	function append_unique(map_key, value, target,    haystack) {
		haystack = "," target[map_key] ","
		if (index(haystack, "," value ",") != 0) {
			return target[map_key]
		}
		if (target[map_key] == "") {
			target[map_key] = value
		} else {
			target[map_key] = target[map_key] "," value
		}
		return target[map_key]
	}
	{
		step = $1
		bank = $2 + 0
		reg = $3 + 0
		base = $4
		seen = $5
		category = category_for_step(step)
		key = bank ":" reg
		reg_key = category ":" reg

		baseline[key] = base
		append_unique(key, category, bank_categories)
		append_unique(key, step, bank_steps)
		append_unique(key, seen, bank_seen)
		if (!(key SUBSEP category in seen_bank_category)) {
			seen_bank_category[key SUBSEP category] = 1
			bank_category_count[key]++
		}
		if (!(key SUBSEP step in seen_bank_step)) {
			seen_bank_step[key SUBSEP step] = 1
			bank_step_count[key]++
		}

		append_unique(reg_key, bank, reg_banks)
		append_unique(reg_key, step, reg_steps)
		if (!(reg_key SUBSEP bank in seen_reg_bank)) {
			seen_reg_bank[reg_key SUBSEP bank] = 1
			reg_bank_count[reg_key]++
		}
		if (!(reg_key SUBSEP step in seen_reg_step)) {
			seen_reg_step[reg_key SUBSEP step] = 1
			reg_step_count[reg_key]++
		}
	}
	END {
		for (key in bank_step_count) {
			split(key, p, ":")
			printf "BANK\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\n",
				p[1], p[2],
				bank_category_count[key] + 0,
				bank_step_count[key] + 0,
				bank_categories[key],
				bank_steps[key],
				baseline[key],
				bank_seen[key]
		}

		for (reg_key in reg_bank_count) {
			split(reg_key, p, ":")
			printf "REG\t%s\t%d\t%d\t%d\t%s\t%s\n",
				p[1], p[2],
				reg_bank_count[reg_key] + 0,
				reg_step_count[reg_key] + 0,
				reg_banks[reg_key],
				reg_steps[reg_key]
		}
	}
' "$SOURCE_FILE" | sort -t $'\t' -k1,1 -k2,2 -k3,3n >"$OUT_DIR/_combined.tsv"

awk -F '\t' '$1 == "BANK" { printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", $2, $3, $4, $5, $6, $7, $8, $9 }' \
	"$OUT_DIR/_combined.tsv" >"$BY_BANK_TSV"

{
	echo "Per-bank shortlist:"
	current_bank=""
	awk -F '\t' '
		{
			bank = $1
			if (bank != current_bank) {
				if (current_bank != "") {
					print ""
				}
				printf "[B%s]\n", bank
				current_bank = bank
			}
			printf "  reg %3d (0x%02X): cats=%s steps=%s baseline=0x%s seen=%s\n", $2, $2, $5, $6, $7, $8
		}
	' "$BY_BANK_TSV"
} >"$BY_BANK_TXT"

awk -F '\t' -v min_banks="$MIN_BANKS" '
	$1 == "REG" && ($4 + 0) >= min_banks {
		printf "%s\t%s\t%s\t%s\t%s\t%s\n", $2, $3, $4, $5, $6, $7
	}
' "$OUT_DIR/_combined.tsv" | sort -t $'\t' -k1,1 -k3,3nr -k2,2n >"$CORE_TSV"

{
	echo "Core shortlist (shared by many banks):"
	if [[ ! -s "$CORE_TSV" ]]; then
		echo "  None."
	else
		awk -F '\t' '
			{
				printf "  [%s] reg %3d (0x%02X): banks=%s steps=%s bank_ids=%s step_ids=%s\n", $1, $2, $2, $3, $4, $5, $6
			}
		' "$CORE_TSV"
	fi
} >"$CORE_TXT"

awk -F '\t' '
	$1 == "REG" && ($4 + 0) == 1 {
		printf "%s\t%s\t%s\t%s\t%s\t%s\n", $2, $3, $4, $5, $6, $7
	}
' "$OUT_DIR/_combined.tsv" | sort -t $'\t' -k1,1 -k4,4nr -k2,2n >"$SPECIALISTS_TSV"

{
	echo "Single-bank specialists:"
	if [[ ! -s "$SPECIALISTS_TSV" ]]; then
		echo "  None."
	else
		awk -F '\t' '
			{
				printf "  [%s] reg %3d (0x%02X): bank=%s steps=%s step_ids=%s\n", $1, $2, $2, $5, $4, $6
			}
		' "$SPECIALISTS_TSV"
	fi
} >"$SPECIALISTS_TXT"

rm -f "$OUT_DIR/_combined.tsv"

echo "Done."
echo "Experiment: $EXP_DIR"
echo "Source: $SOURCE_FILE"
echo "Output: $OUT_DIR"
echo
cat "$CORE_TXT"
echo
cat "$SPECIALISTS_TXT"

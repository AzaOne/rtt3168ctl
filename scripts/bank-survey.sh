#!/usr/bin/env bash
set -euo pipefail

SAMPLES=5
INTERVAL_SEC="0.20"
BANK_SPEC="0-7"
OUT_DIR=""
BIN_PATH=""

usage() {
	cat <<'EOF'
Usage:
  scripts/bank-survey.sh [options]

Options:
  --banks <spec>       Bank spec for -dump-banks (default: 0-7)
  --samples <n>        Number of dumps to capture (default: 5)
  --interval <sec>     Delay between dumps in seconds (default: 0.20)
  --out <dir>          Output directory (default: experiments/bank-survey-<timestamp>)
  --bin <path>         Path to rtt3168ctl binary (default: auto)
  -h, --help           Show help

What this script does:
  1. Captures repeated dumps for selected banks.
  2. Detects volatile registers across samples.
  3. Reports stable non-trivial values (excluding 0x00 and 0xFF).
  4. Groups banks with identical sample-1 snapshots.

Typical use:
  scripts/bank-survey.sh --banks 0-15
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
		--interval)
			INTERVAL_SEC="$2"
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

if ! [[ "$INTERVAL_SEC" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
	echo "--interval must be numeric and >= 0" >&2
	exit 1
fi

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"

if [[ -z "$OUT_DIR" ]]; then
	OUT_DIR="$ROOT_DIR/experiments/bank-survey-$(date +%Y%m%d-%H%M%S)"
fi

if [[ -n "$BIN_PATH" ]]; then
	if [[ ! -x "$BIN_PATH" ]]; then
		echo "Binary is not executable: $BIN_PATH" >&2
		exit 1
	fi
elif [[ -x "$ROOT_DIR/build/rtt3168ctl" ]]; then
	BIN_PATH="$ROOT_DIR/build/rtt3168ctl"
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
		match($0, /^Memory Dump \(Bank ([0-9]+), registers 0\.\.255\)$/, m) {
			bank = m[1] + 0
			next
		}
		match($0, /^([0-9]+) \(0x[0-9A-F]{2}\): 0x([0-9A-F]{2})$/, m) {
			printf "%d\t%d\t%s\n", bank, m[1] + 0, m[2]
		}
	' "$raw_file" >"$parsed_file"
}

capture_samples() {
	local i
	for (( i=1; i<=SAMPLES; i++ )); do
		local raw_file="$OUT_DIR/raw/sample-$(printf "%02d" "$i").txt"
		local parsed_file="$OUT_DIR/parsed/sample-$(printf "%02d" "$i").tsv"
		run_dump "$raw_file"
		parse_dump "$raw_file" "$parsed_file"
		printf "Captured sample %d/%d\r" "$i" "$SAMPLES"
		if (( i < SAMPLES )); then
			sleep "$INTERVAL_SEC"
		fi
	done
	printf "\n"
}

build_volatile_summary() {
	local out_tsv="$OUT_DIR/summary/volatile.tsv"
	local out_txt="$OUT_DIR/summary/volatile.txt"
	: >"$out_tsv"

	awk '
		{
			key = $1 ":" $2
			bank[key] = $1
			reg[key] = $2
			if (!(key in first)) {
				first[key] = $3
			}
			if (vals[key] == "") {
				vals[key] = $3
			} else if (index("," vals[key] ",", "," $3 ",") == 0) {
				vals[key] = vals[key] "," $3
			}
		}
		END {
			for (key in vals) {
				if (index(vals[key], ",") == 0) {
					continue
				}
				printf "%s\t%s\t%s\t%s\n", bank[key], reg[key], first[key], vals[key]
			}
		}
	' "$OUT_DIR"/parsed/*.tsv | sort -n -k1,1 -k2,2 >"$out_tsv"

	{
		echo "Volatile registers across $SAMPLES sample(s):"
		if [[ ! -s "$out_tsv" ]]; then
			echo "  None."
		else
			awk -F '\t' '
				{
					vals = $4
					gsub(",", ",0x", vals)
					printf "  B%s reg %3d (0x%02X): first=0x%s seen=0x%s\n", $1, $2, $2, $3, vals
				}
			' "$out_tsv"
		fi
	} >"$out_txt"
}

build_stable_interesting_summary() {
	local first_sample="$OUT_DIR/parsed/sample-01.tsv"
	local volatile_tsv="$OUT_DIR/summary/volatile.tsv"
	local out_tsv="$OUT_DIR/summary/stable-interesting.tsv"
	local out_txt="$OUT_DIR/summary/stable-interesting.txt"

	awk -F '\t' '
		ARGIND == 1 {
			vol[$1 ":" $2] = 1
			next
		}
		{
			key = $1 ":" $2
			if (key in vol) {
				next
			}
			if ($3 == "00" || $3 == "FF") {
				next
			}
			print
		}
	' "$volatile_tsv" "$first_sample" | sort -n -k1,1 -k2,2 >"$out_tsv"

	{
		echo "Stable non-trivial registers from sample 1 (excluding 0x00/0xFF):"
		if [[ ! -s "$out_tsv" ]]; then
			echo "  None."
		else
			awk -F '\t' '
				{
					printf "  B%s reg %3d (0x%02X): 0x%s\n", $1, $2, $2, $3
				}
			' "$out_tsv"
		fi
	} >"$out_txt"
}

build_bank_overview() {
	local first_sample="$OUT_DIR/parsed/sample-01.tsv"
	local volatile_tsv="$OUT_DIR/summary/volatile.tsv"
	local out_tsv="$OUT_DIR/summary/bank-overview.tsv"
	local out_txt="$OUT_DIR/summary/bank-overview.txt"

	awk -F '\t' '
		ARGIND == 1 {
			volatile[$1 ":" $2] = 1
			volatile_count[$1]++
			next
		}
		{
			bank = $1
			val = $3
			total[bank]++
			if (val == "00") {
				zero[bank]++
			} else if (val == "FF") {
				ff[bank]++
			} else {
				other[bank]++
			}

			uniq_key = bank ":" val
			if (!(uniq_key in seen_val)) {
				seen_val[uniq_key] = 1
				unique_vals[bank]++
			}
		}
		END {
			for (bank in total) {
				printf "%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
					bank,
					total[bank] + 0,
					zero[bank] + 0,
					ff[bank] + 0,
					other[bank] + 0,
					unique_vals[bank] + 0,
					volatile_count[bank] + 0
			}
		}
	' "$volatile_tsv" "$first_sample" | sort -n -k1,1 >"$out_tsv"

	{
		echo "Bank overview (sample 1 + volatility over $SAMPLES sample(s)):"
		awk -F '\t' '
			{
				printf "  B%s: total=%d zero=%d ff=%d other=%d unique_values=%d volatile=%d\n",
					$1, $2, $3, $4, $5, $6, $7
			}
		' "$out_tsv"
	} >"$out_txt"
}

build_identical_banks_summary() {
	local first_sample="$OUT_DIR/parsed/sample-01.tsv"
	local out_txt="$OUT_DIR/summary/identical-banks.txt"

	awk -F '\t' '
		{
			sig[$1] = sig[$1] $3
		}
		END {
			for (bank in sig) {
				if (groups[sig[bank]] == "") {
					groups[sig[bank]] = bank
				} else {
					groups[sig[bank]] = groups[sig[bank]] "," bank
				}
			}

			found = 0
			for (signature in groups) {
				if (index(groups[signature], ",") == 0) {
					continue
				}
				n = split(groups[signature], raw, ",")
				delete banks
				for (i = 1; i <= n; i++) {
					banks[i] = raw[i] + 0
				}
				for (i = 1; i <= n; i++) {
					for (j = i + 1; j <= n; j++) {
						if (banks[i] > banks[j]) {
							tmp = banks[i]
							banks[i] = banks[j]
							banks[j] = tmp
						}
					}
				}

				line = ""
				for (i = 1; i <= n; i++) {
					if (i > 1) {
						line = line ","
					}
					line = line "B" banks[i]
				}
				print line
				found = 1
			}

			if (!found) {
				print "None."
			}
		}
	' "$first_sample" >"$out_txt"
}

print_summary() {
	echo
	echo "Done."
	echo "Output: $OUT_DIR"
	echo
	cat "$OUT_DIR/summary/bank-overview.txt"
	echo
	echo "Identical banks in sample 1:"
	cat "$OUT_DIR/summary/identical-banks.txt"
	echo
	cat "$OUT_DIR/summary/volatile.txt"
	echo
	cat "$OUT_DIR/summary/stable-interesting.txt"
}

echo "Capturing $SAMPLES dump sample(s) for banks: $BANK_SPEC"
capture_samples
build_volatile_summary
build_stable_interesting_summary
build_bank_overview
build_identical_banks_summary
print_summary

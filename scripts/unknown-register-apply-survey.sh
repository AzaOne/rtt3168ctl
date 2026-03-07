#!/usr/bin/env bash
set -euo pipefail

OUT_DIR=""
BIN_PATH=""

RATE_VALUES="125,250,500,1000"
RGB_MODES="off,on,breath,breath_segment,cycle6,cycle12,cycle768"
SPEED_VALUES="0,40,128,255"
ACTIVE_SLOTS="1,2,3,4"
DPI_VALUES="200,800,1600,3200"
COLOR_VALUES="0,7,15"
CPI_ACTIONS="backward,forward,ctrl,win,browser,double_click,sniper,rgb_switch,dpi_cycle,play_pause,mute,next_track,prev_track,stop,vol_up,vol_down,win_d,copy,paste,prev_page,next_page,my_computer,calculator,ctrl_w"
WITH_RATES=0
RUN_RETRIES=6
RUN_RETRY_DELAY_SEC="0.30"
SETTLE_DELAY_SEC="0.20"

# Registers already documented in SPEC.md.
KNOWN_BANK0="2"
KNOWN_BANK1="1,2,3,4,5,6,7,9,10,11,14,29"

BASELINE_READY=0
RESTORE_ON_EXIT=1

declare -A BASE_HEX=()
declare -a BASE_SLOT_DPI=()
declare -a BASE_SLOT_COLOR=()

usage() {
	cat <<'EOF'
Usage:
  scripts/unknown-register-apply-survey.sh [options]

Options:
  --out <dir>            Output directory
                         (default: experiments/apply-survey-<timestamp>)
  --bin <path>           Path to rtt3168ctl binary (default: auto)
  --with-rates           Include rate scenarios (disabled by default)
  --rates <csv>          Rate scenarios when --with-rates is set
                         (default: 125,250,500,1000)
  --rgb-modes <csv>      RGB mode scenarios
                         (default: off,on,breath,breath_segment,cycle6,cycle12,cycle768)
  --speed-values <csv>   RGB speed scenarios (default: 0,40,128,255)
  --active-slots <csv>   Active slot scenarios (default: 1,2,3,4)
  --dpi-values <csv>     Candidate DPI values for per-slot tests
                         (default: 200,800,1600,3200)
  --color-values <csv>   Candidate colors for per-slot tests
                         (default: 0,7,15)
  --cpi-actions <csv>    CPI action scenarios
                         (default: all known actions)
  --retries <n>          Retries for transient USB/libusb errors (default: 6)
  --retry-delay <sec>    Initial retry delay in seconds (default: 0.30)
  --settle-delay <sec>   Delay after successful device command (default: 0.20)
  -h, --help             Show help

What this script does:
  1. Captures a baseline dump and baseline readout.
  2. Applies controlled configuration changes with `-mode apply`.
  3. Captures a unique-register dump after each change.
  4. Reports diffs against baseline, with focus on unknown registers.
  5. Restores the original bank1 configuration between scenarios and on exit.

Notes:
  - This script mutates device settings during the survey.
  - Restoration uses exact raw baseline values for documented bank1 config registers.
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--out)
			OUT_DIR="$2"
			shift 2
			;;
		--bin)
			BIN_PATH="$2"
			shift 2
			;;
		--rates)
			RATE_VALUES="$2"
			shift 2
			;;
		--with-rates)
			WITH_RATES=1
			shift
			;;
		--rgb-modes)
			RGB_MODES="$2"
			shift 2
			;;
		--speed-values)
			SPEED_VALUES="$2"
			shift 2
			;;
		--active-slots)
			ACTIVE_SLOTS="$2"
			shift 2
			;;
		--dpi-values)
			DPI_VALUES="$2"
			shift 2
			;;
		--color-values)
			COLOR_VALUES="$2"
			shift 2
			;;
		--cpi-actions)
			CPI_ACTIONS="$2"
			shift 2
			;;
		--retries)
			RUN_RETRIES="$2"
			shift 2
			;;
		--retry-delay)
			RUN_RETRY_DELAY_SEC="$2"
			shift 2
			;;
		--settle-delay)
			SETTLE_DELAY_SEC="$2"
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

if [[ -z "$OUT_DIR" ]]; then
	OUT_DIR="$ROOT_DIR/experiments/apply-survey-$(date +%Y%m%d-%H%M%S)"
fi

if [[ -n "$BIN_PATH" ]]; then
	if [[ ! -x "$BIN_PATH" ]]; then
		echo "Binary is not executable: $BIN_PATH" >&2
		exit 1
	fi
elif [[ -x "$ROOT_DIR/build/rtt3168ctl" ]]; then
	BIN_PATH="$ROOT_DIR/build/rtt3168ctl"
fi

mkdir -p \
	"$OUT_DIR/raw" \
	"$OUT_DIR/parsed" \
	"$OUT_DIR/status" \
	"$OUT_DIR/logs" \
	"$OUT_DIR/summary"

cleanup() {
	local status=$?
	if [[ "$BASELINE_READY" == "1" && "$RESTORE_ON_EXIT" == "1" ]]; then
		echo "Restoring baseline configuration..." >&2
		restore_baseline >>"$OUT_DIR/logs/restore-on-exit.log" 2>&1 || \
			echo "Baseline restore failed; inspect $OUT_DIR/logs/restore-on-exit.log" >&2
	fi
	return "$status"
}

trap cleanup EXIT

run_tool() {
	local attempt=0
	local current_delay="$RUN_RETRY_DELAY_SEC"
	local out_tmp
	local err_tmp

	out_tmp="$(mktemp)"
	err_tmp="$(mktemp)"

	while true; do
		if [[ -n "$BIN_PATH" ]]; then
			if "$BIN_PATH" "$@" >"$out_tmp" 2>"$err_tmp"; then
				cat "$out_tmp"
				rm -f "$out_tmp" "$err_tmp"
				sleep "$SETTLE_DELAY_SEC"
				return 0
			fi
		else
			if (
				cd "$ROOT_DIR"
				go run ./cmd/rtt3168ctl "$@"
			) >"$out_tmp" 2>"$err_tmp"; then
				cat "$out_tmp"
				rm -f "$out_tmp" "$err_tmp"
				sleep "$SETTLE_DELAY_SEC"
				return 0
			fi
		fi

		if (( attempt >= RUN_RETRIES )) || ! is_transient_usb_error "$err_tmp"; then
			cat "$out_tmp"
			cat "$err_tmp" >&2
			rm -f "$out_tmp" "$err_tmp"
			return 1
		fi

		sleep "$current_delay"
		current_delay="$(awk -v d="$current_delay" 'BEGIN { printf "%.2f", d * 1.7 }')"
		attempt=$((attempt + 1))
	done
}

is_transient_usb_error() {
	local err_file="$1"
	awk '
		BEGIN { IGNORECASE = 1 }
		/libusb: i\/o error/ ||
		/libusb: pipe error/ ||
		/usb device not found/ ||
		/no such device/ {
			found = 1
		}
		END { exit(found ? 0 : 1) }
	' "$err_file"
}

run_tool_no_retry() {
	if [[ -n "$BIN_PATH" ]]; then
		"$BIN_PATH" "$@"
		return
	fi

	(
		cd "$ROOT_DIR"
		go run ./cmd/rtt3168ctl "$@"
	)
}

run_dump() {
	local out_file="$1"
	run_tool -mode dump >"$out_file"
}

run_read_json() {
	local out_file="$1"
	run_tool -mode read -json >"$out_file"
}

parse_dump() {
	local raw_file="$1"
	local parsed_file="$2"
	awk '
		/^Memory Dump \(Bank 0, registers 0\.\.(127|255)\)$/ { bank=0; next }
		/^Memory Dump \(Bank 1, registers 0\.\.(127|255)\)$/ { bank=1; next }
		match($0, /^([0-9]+) \(0x[0-9A-F]{2}\): 0x([0-9A-F]{2})$/, m) {
			printf "%d\t%d\t%s\n", bank, m[1] + 0, m[2]
		}
	' "$raw_file" >"$parsed_file"
}

load_baseline_map() {
	local parsed_file="$1"
	while IFS=$'\t' read -r bank reg hex; do
		BASE_HEX["$bank:$reg"]="$hex"
	done <"$parsed_file"
}

hex_to_dec() {
	printf '%d\n' "$((16#$1))"
}

decode_rate_raw() {
	case "$1" in
		C2) echo "125" ;;
		82) echo "250" ;;
		42) echo "500" ;;
		02) echo "1000" ;;
		*) return 1 ;;
	esac
}

decode_rgb_mode_raw() {
	local raw_dec
	raw_dec="$((16#$1))"
	case "$((raw_dec & 0xF0))" in
		0) echo "on" ;;
		32) echo "breath" ;;
		64) echo "breath_segment" ;;
		96) echo "cycle6" ;;
		128) echo "cycle12" ;;
		160) echo "cycle768" ;;
		224) echo "off" ;;
		*) return 1 ;;
	esac
}

decode_active_slot_raw() {
	local raw_dec
	raw_dec="$((16#$1))"
	if (( raw_dec >= 0 && raw_dec <= 3 )); then
		echo "$((raw_dec + 1))"
		return
	fi

	case "$raw_dec" in
		32) echo "2" ;;
		64) echo "3" ;;
		96) echo "4" ;;
		*) echo "$(((raw_dec & 0x03) + 1))" ;;
	esac
}

decode_cpi_action_raw() {
	case "$1" in
		E0) echo "backward" ;;
		E1) echo "forward" ;;
		E2) echo "ctrl" ;;
		E3) echo "win" ;;
		E4) echo "browser" ;;
		E5) echo "double_click" ;;
		E6) echo "sniper" ;;
		E7) echo "rgb_switch" ;;
		E8) echo "dpi_cycle" ;;
		EC) echo "play_pause" ;;
		ED) echo "mute" ;;
		EE) echo "next_track" ;;
		EF) echo "prev_track" ;;
		F0) echo "stop" ;;
		F2) echo "vol_up" ;;
		F3) echo "vol_down" ;;
		F5) echo "win_d" ;;
		F6) echo "copy" ;;
		F7) echo "paste" ;;
		F8) echo "prev_page" ;;
		F9) echo "next_page" ;;
		FA) echo "my_computer" ;;
		FB) echo "calculator" ;;
		FC) echo "ctrl_w" ;;
		*) return 1 ;;
	esac
}

decode_dpi_slot_raw() {
	local raw_dec dpi color
	raw_dec="$((16#$1))"
	dpi="$((200 + ((raw_dec & 0x0F) * 200)))"
	color="$(((raw_dec >> 4) & 0x0F))"
	printf '%s\t%s\n' "$dpi" "$color"
}

choose_csv_alternative() {
	local current="$1"
	local csv="$2"
	local value
	IFS=',' read -r -a _choices <<<"$csv"
	for value in "${_choices[@]}"; do
		if [[ -z "$value" ]]; then
			continue
		fi
		if [[ "$value" != "$current" ]]; then
			echo "$value"
			return 0
		fi
	done
	return 1
}

detect_changes() {
	local baseline_file="$1"
	local current_file="$2"
	awk -F '\t' '
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
				printf "%s\t%s\t%s\t%s\n", $1, $2, base[key], $3
			}
		}
	' "$baseline_file" "$current_file" | sort -n -k1,1 -k2,2
}

filter_unknown_changes() {
	local source_file="$1"
	awk -F '\t' -v known0="$KNOWN_BANK0" -v known1="$KNOWN_BANK1" '
		BEGIN {
			split(known0, b0, ",")
			for (i in b0) {
				if (b0[i] != "") {
					known["0:" b0[i]] = 1
				}
			}

			split(known1, b1, ",")
			for (i in b1) {
				if (b1[i] != "") {
					known["1:" b1[i]] = 1
				}
			}
		}
		{
			key = $1 ":" $2
			if (!(key in known)) {
				print
			}
		}
	' "$source_file"
}

write_diff_summary() {
	local title="$1"
	local diff_file="$2"
	local out_file="$3"

	{
		echo "$title"
		if [[ ! -s "$diff_file" ]]; then
			echo "  No unknown register changes vs baseline."
		else
			awk -F '\t' '
				{
					printf "  B%s reg %3d (0x%02X): baseline=0x%s seen=0x%s\n", $1, $2, $2, $3, $4
				}
			' "$diff_file"
		fi
	} >"$out_file"
}

restore_baseline() {
	local reg hex value
	for reg in 1 2 3 4 5 6 7 9 10 11 14; do
		hex="${BASE_HEX["1:$reg"]:-}"
		if [[ -z "$hex" ]]; then
			continue
		fi
		value="$(hex_to_dec "$hex")"
		run_tool -mode write -reg "$reg" -regval "$value" >/dev/null
	done
}

SCENARIO_SEQ=0
SCENARIO_META="$OUT_DIR/summary/scenarios.tsv"
UNKNOWN_SOURCE="$OUT_DIR/summary/_unknown_source.tsv"
ALL_SOURCE="$OUT_DIR/summary/_all_source.tsv"

printf "scenario_id\tcategory\ttitle\tapply_args\tapply_ok\tcapture_ok\tall_changes\tunknown_changes\n" >"$SCENARIO_META"
: >"$UNKNOWN_SOURCE"
: >"$ALL_SOURCE"

run_scenario() {
	local category="$1"
	local title="$2"
	shift 2

	local apply_args=("$@")
	local id
	local apply_log
	local status_json
	local raw_file
	local parsed_file
	local all_diff_file
	local unknown_diff_file
	local summary_file
	local apply_ok="no"
	local capture_ok="no"
	local all_count="0"
	local unknown_count="0"
	local args_pretty=""
	local read_log
	local dump_log

	SCENARIO_SEQ=$((SCENARIO_SEQ + 1))
	id="$(printf "S%02d-%s" "$SCENARIO_SEQ" "$category")"
	apply_log="$OUT_DIR/logs/$id-apply.txt"
	read_log="$OUT_DIR/logs/$id-read.txt"
	dump_log="$OUT_DIR/logs/$id-dump.txt"
	status_json="$OUT_DIR/status/$id.json"
	raw_file="$OUT_DIR/raw/$id.txt"
	parsed_file="$OUT_DIR/parsed/$id.tsv"
	all_diff_file="$OUT_DIR/summary/$id-all.tsv"
	unknown_diff_file="$OUT_DIR/summary/$id-unknown.tsv"
	summary_file="$OUT_DIR/summary/$id.txt"

	printf -v args_pretty '%q ' "${apply_args[@]}"
	args_pretty="${args_pretty% }"

	echo "[$id] $title"

	restore_baseline >/dev/null

	if run_tool -mode apply "${apply_args[@]}" >"$apply_log" 2>&1; then
		apply_ok="yes"
	else
		echo "  apply failed; see $apply_log"
		printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n" \
			"$id" "$category" "$title" "$args_pretty" "$apply_ok" "$capture_ok" "$all_count" "$unknown_count" \
			>>"$SCENARIO_META"
		return 0
	fi

	if run_read_json "$status_json" 2>"$read_log" && run_dump "$raw_file" 2>"$dump_log"; then
		capture_ok="yes"
	else
		echo "  post-apply capture failed; see $read_log and $dump_log"
		printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n" \
			"$id" "$category" "$title" "$args_pretty" "$apply_ok" "$capture_ok" "$all_count" "$unknown_count" \
			>>"$SCENARIO_META"
		return 0
	fi

	parse_dump "$raw_file" "$parsed_file"

	detect_changes "$OUT_DIR/parsed/baseline.tsv" "$parsed_file" >"$all_diff_file"
	filter_unknown_changes "$all_diff_file" >"$unknown_diff_file"
	write_diff_summary "[$id] $title" "$unknown_diff_file" "$summary_file"

	if [[ -s "$all_diff_file" ]]; then
		all_count="$(wc -l <"$all_diff_file" | tr -d ' ')"
		awk -F '\t' -v scenario="$id" -v category="$category" \
			'{ printf "%s\t%s\t%s\t%s\t%s\t%s\n", scenario, category, $1, $2, $3, $4 }' \
			"$all_diff_file" >>"$ALL_SOURCE"
	fi

	if [[ -s "$unknown_diff_file" ]]; then
		unknown_count="$(wc -l <"$unknown_diff_file" | tr -d ' ')"
		awk -F '\t' -v scenario="$id" -v category="$category" \
			'{ printf "%s\t%s\t%s\t%s\t%s\t%s\n", scenario, category, $1, $2, $3, $4 }' \
			"$unknown_diff_file" >>"$UNKNOWN_SOURCE"
	fi

	printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n" \
		"$id" "$category" "$title" "$args_pretty" "$apply_ok" "$capture_ok" "$all_count" "$unknown_count" \
		>>"$SCENARIO_META"

	cat "$summary_file"
}

build_register_aggregate() {
	local source_file="$1"
	local tsv_file="$2"
	local txt_file="$3"
	local heading="$4"

	if [[ ! -s "$source_file" ]]; then
		cat >"$txt_file" <<EOF
$heading
No matching register changes captured.
EOF
		: >"$tsv_file"
		return
	fi

	awk -F '\t' '
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
				split(key, p, ":")
				printf "%s\t%s\t%s\t%d\t%s\t%s\n", p[1], p[2], base[key], count[key], values[key], scenarios[key]
			}
		}
	' "$source_file" | sort -n -k1,1 -k2,2 >"$tsv_file"

	{
		echo "$heading"
		awk -F '\t' '
			{
				printf "  B%s reg %3d (0x%02X): baseline=0x%s seen=0x%s in %d scenario(s): %s\n", $1, $2, $2, $3, $5, $4, $6
			}
		' "$tsv_file"
	} >"$txt_file"
}

build_category_aggregate() {
	local source_file="$1"
	local tsv_file="$2"
	local txt_file="$3"
	local heading="$4"

	if [[ ! -s "$source_file" ]]; then
		cat >"$txt_file" <<EOF
$heading
No matching register changes captured.
EOF
		: >"$tsv_file"
		return
	fi

	awk -F '\t' '
		{
			category = $2
			key = $3 ":" $4
			if (!(category SUBSEP key in seen_key)) {
				seen_key[category SUBSEP key] = 1
				count[category]++
			}
			if (!(category SUBSEP $1 in seen_scenario)) {
				seen_scenario[category SUBSEP $1] = 1
				scenarios[category]++
			}
		}
		END {
			for (category in count) {
				printf "%s\t%d\t%d\n", category, count[category], scenarios[category]
			}
		}
	' "$source_file" | sort >"$tsv_file"

	{
		echo "$heading"
		awk -F '\t' '
			{
				printf "  %s: %d unique register(s) across %d scenario(s)\n", $1, $2, $3
			}
		' "$tsv_file"
	} >"$txt_file"
}

echo "Capturing baseline state..."
run_read_json "$OUT_DIR/status/baseline.json"
run_dump "$OUT_DIR/raw/baseline.txt"
parse_dump "$OUT_DIR/raw/baseline.txt" "$OUT_DIR/parsed/baseline.tsv"
load_baseline_map "$OUT_DIR/parsed/baseline.tsv"
BASELINE_READY=1

BASE_RGB_SPEED="$(hex_to_dec "${BASE_HEX["1:1"]}")"
BASE_RATE_HZ=""
if [[ -n "${BASE_HEX["1:14"]:-}" ]]; then
	BASE_RATE_HZ="$(decode_rate_raw "${BASE_HEX["1:14"]}" || true)"
fi

BASE_RGB_MODE=""
if [[ -n "${BASE_HEX["1:10"]:-}" ]]; then
	BASE_RGB_MODE="$(decode_rgb_mode_raw "${BASE_HEX["1:10"]}" || true)"
fi

BASE_ACTIVE_SLOT=""
if [[ -n "${BASE_HEX["1:9"]:-}" ]]; then
	BASE_ACTIVE_SLOT="$(decode_active_slot_raw "${BASE_HEX["1:9"]}")"
fi

BASE_CPI_ACTION=""
if [[ -n "${BASE_HEX["1:11"]:-}" ]]; then
	BASE_CPI_ACTION="$(decode_cpi_action_raw "${BASE_HEX["1:11"]}" || true)"
fi

for slot in 1 2 3 4; do
	reg="$((slot + 1))"
	if [[ -z "${BASE_HEX["1:$reg"]:-}" ]]; then
		echo "Missing baseline dump value for bank1 register $reg" >&2
		exit 1
	fi
	IFS=$'\t' read -r BASE_SLOT_DPI["$slot"] BASE_SLOT_COLOR["$slot"] < <(decode_dpi_slot_raw "${BASE_HEX["1:$reg"]}")
done

echo
echo "Running RGB mode scenarios..."
IFS=',' read -r -a RGB_MODE_LIST <<<"$RGB_MODES"
for mode in "${RGB_MODE_LIST[@]}"; do
	[[ -z "$mode" ]] && continue
	if [[ -n "$BASE_RGB_MODE" && "$mode" == "$BASE_RGB_MODE" ]]; then
		continue
	fi
	run_scenario "rgb-mode-$mode" "RGB mode -> $mode (speed=$BASE_RGB_SPEED)" -rgb-mode "$mode" -speed "$BASE_RGB_SPEED"
done

echo
echo "Running RGB speed scenarios..."
IFS=',' read -r -a SPEED_LIST <<<"$SPEED_VALUES"
if [[ -z "$BASE_RGB_MODE" ]]; then
	echo "Skipping RGB speed scenarios: baseline RGB mode is unknown." | tee "$OUT_DIR/logs/skip-rgb-speed.log"
else
	for speed in "${SPEED_LIST[@]}"; do
		[[ -z "$speed" ]] && continue
		if [[ "$speed" == "$BASE_RGB_SPEED" ]]; then
			continue
		fi
		run_scenario "rgb-speed-$speed" "RGB speed -> $speed (mode=$BASE_RGB_MODE)" -rgb-mode "$BASE_RGB_MODE" -speed "$speed"
	done
fi

echo
echo "Running CPI action scenarios..."
IFS=',' read -r -a CPI_ACTION_LIST <<<"$CPI_ACTIONS"
for action in "${CPI_ACTION_LIST[@]}"; do
	[[ -z "$action" ]] && continue
	if [[ -n "$BASE_CPI_ACTION" && "$action" == "$BASE_CPI_ACTION" ]]; then
		continue
	fi
	run_scenario "cpi-$action" "CPI action -> $action" -cpi-action "$action"
done

echo
echo "Running active-slot scenarios..."
IFS=',' read -r -a ACTIVE_SLOT_LIST <<<"$ACTIVE_SLOTS"
for slot in "${ACTIVE_SLOT_LIST[@]}"; do
	[[ -z "$slot" ]] && continue
	if [[ -n "$BASE_ACTIVE_SLOT" && "$slot" == "$BASE_ACTIVE_SLOT" ]]; then
		continue
	fi
	run_scenario "active-slot-$slot" "Active slot -> $slot" -active-slot "$slot"
done

echo
echo "Running per-slot DPI scenarios..."
for slot in 1 2 3 4; do
	alt_dpi="$(choose_csv_alternative "${BASE_SLOT_DPI["$slot"]}" "$DPI_VALUES" || true)"
	if [[ -z "$alt_dpi" ]]; then
		alt_dpi="$((BASE_SLOT_DPI["$slot"] + 200))"
		if (( alt_dpi > 3200 )); then
			alt_dpi=200
		fi
	fi

	alt_color="$(choose_csv_alternative "${BASE_SLOT_COLOR["$slot"]}" "$COLOR_VALUES" || true)"
	if [[ -z "$alt_color" ]]; then
		alt_color="$(((BASE_SLOT_COLOR["$slot"] + 1) % 16))"
	fi

	run_scenario \
		"dpi${slot}-value-$alt_dpi" \
		"Slot $slot DPI -> $alt_dpi (color=${BASE_SLOT_COLOR["$slot"]})" \
		"-dpi${slot}" "${alt_dpi}:${BASE_SLOT_COLOR["$slot"]}"

	run_scenario \
		"dpi${slot}-color-$alt_color" \
		"Slot $slot color -> $alt_color (dpi=${BASE_SLOT_DPI["$slot"]})" \
		"-dpi${slot}" "${BASE_SLOT_DPI["$slot"]}:${alt_color}"
done

if [[ "$WITH_RATES" == "1" ]]; then
	echo
	echo "Running rate scenarios..."
	IFS=',' read -r -a RATE_LIST <<<"$RATE_VALUES"
	for rate in "${RATE_LIST[@]}"; do
		[[ -z "$rate" ]] && continue
		if [[ -n "$BASE_RATE_HZ" && "$rate" == "$BASE_RATE_HZ" ]]; then
			continue
		fi
		run_scenario "rate-$rate" "Rate -> ${rate}Hz" -rate "$rate"
	done
fi

build_register_aggregate \
	"$UNKNOWN_SOURCE" \
	"$OUT_DIR/summary/unknown-aggregate.tsv" \
	"$OUT_DIR/summary/unknown-aggregate.txt" \
	"Unknown register changes aggregated by register:"

build_register_aggregate \
	"$ALL_SOURCE" \
	"$OUT_DIR/summary/all-aggregate.tsv" \
	"$OUT_DIR/summary/all-aggregate.txt" \
	"All register changes aggregated by register:"

build_category_aggregate \
	"$UNKNOWN_SOURCE" \
	"$OUT_DIR/summary/unknown-by-category.tsv" \
	"$OUT_DIR/summary/unknown-by-category.txt" \
	"Unknown register changes aggregated by category:"

echo
echo "Restoring baseline configuration..."
restore_baseline
RESTORE_ON_EXIT=0

echo "Done."
echo "Output: $OUT_DIR"
echo
echo "Unknown aggregate:"
cat "$OUT_DIR/summary/unknown-aggregate.txt"

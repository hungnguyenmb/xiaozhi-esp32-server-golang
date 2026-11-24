#!/usr/bin/env bash
set -euo pipefail
# v0.1, including its built-in model. No training/model download at runtime.
revision=cdf196b1e9de2f8ff1003328ebf9a4316477429d
destination="${1:?usage: build_rnnoise.sh OUTPUT_DIRECTORY}"
mkdir -p "$destination"
destination="$(cd "$destination" && pwd)"
source_dir="$(mktemp -d "${TMPDIR:-/tmp}/xiaozhi-rnnoise.XXXXXX")"
# Keep source/license for reproducibility; do not remove user files.
git -C "$source_dir" init -q
git -C "$source_dir" fetch -q --depth 1 https://github.com/xiph/rnnoise.git "$revision"
git -C "$source_dir" checkout -q --detach FETCH_HEAD
test "$(git -C "$source_dir" rev-parse HEAD)" = "$revision"
flags=(-shared -fPIC)
output=libxiaozhi_rnnoise_v01.so
if [[ "$(uname -s)" == Darwin ]]; then
  flags=(-dynamiclib -fPIC)
  output=libxiaozhi_rnnoise_v01.dylib
fi
sources=(denoise kiss_fft pitch celt_lpc rnn rnn_data)
inputs=()
for name in "${sources[@]}"; do inputs+=("$source_dir/src/$name.c"); done
"${CC:-cc}" -O3 -fvisibility=hidden "${flags[@]}" -DRNNOISE_BUILD -I"$source_dir/include" "${inputs[@]}" -lm -o "$destination/$output"
cp "$source_dir/COPYING" "$destination/RNNOISE-COPYING"
printf '%s\n' "$revision" > "$destination/RNNOISE-REVISION"
printf 'Built %s; source retained at %s\n' "$destination/$output" "$source_dir"

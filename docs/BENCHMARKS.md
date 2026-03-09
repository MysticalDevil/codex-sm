# Benchmark Baselines

This document records lightweight benchmark baselines for `codexsm`.

## Baseline Snapshot

- Date: `2026-03-09`
- Commit: `637a4a4`
- Go: `go1.26.1 linux/amd64`
- CPU: `AMD Ryzen 7 4800H with Radeon Graphics`
- Runs: `go test -run '^$' -bench ... -benchmem -count=3`
- Interpretation: use these numbers as a first local baseline, not as hard gates.

## Commands

```bash
go test -run '^$' -bench 'Benchmark(ScanSessions|FilterSessions)' ./session -benchmem -count=3
go test -run '^$' -bench 'Benchmark(RenderTable|RenderJSON|DoctorRiskJSON)' ./cli -benchmem -count=3
go test -run '^$' -bench 'Benchmark(SortTUISessions_3k|SortTUISessions_10k|BuildPreviewLines|PreviewIndex)' ./tui -benchmem -count=3
```

## Session

| Benchmark | Median ns/op | Median B/op | Median allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkScanSessions` | `1,007,494` | `197,501` | `682` |
| `BenchmarkFilterSessions/all` | `3,710` | `5,080` | `4` |
| `BenchmarkFilterSessions/host_head_health` | `5,364` | `5,080` | `4` |
| `BenchmarkFilterSessions/older_than` | `2,826` | `4,888` | `2` |
| `BenchmarkScanSessionsLimited_3k` | `98,246,819` | `17,477,820` | `60,275` |
| `BenchmarkScanSessions_AllVsLimited_3k/all` | `88,038,863` | `19,257,079` | `60,371` |
| `BenchmarkScanSessions_AllVsLimited_3k/limited_100` | `91,810,551` | `17,519,064` | `60,305` |
| `BenchmarkScanSessions_ExtremeMix` | `35,036,053` | `8,200,257` | `24,248` |

Observation:

- `ScanSessionsLimited_3k` reduces retained-result memory versus the full-scan path, but current scan cost still dominates because all files are parsed.

## CLI

| Benchmark | Median ns/op | Median B/op | Median allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkDoctorRiskJSON` | `20,815,548` | `4,924,777` | `17,077` |
| `BenchmarkRenderTable` | `3,876,639` | `2,161,013` | `15,674` |
| `BenchmarkRenderTable_LargeColumns` | `6,744,459` | `4,184,035` | `18,081` |
| `BenchmarkRenderJSON` | `2,147,482` | `1,188,728` | `9` |

Observation:

- `doctor risk --format json` is the heaviest CLI benchmark in this set and is the best early candidate for later optimization work.

## TUI

| Benchmark | Median ns/op | Median B/op | Median allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkSortTUISessions_3k` | `2,553,720` | `409,601` | `1` |
| `BenchmarkSortTUISessions_10k` | `9,159,531` | `1,368,064` | `1` |
| `BenchmarkBuildPreviewLines_LargeSession` | `39,065,986` | `910,256` | `13,221` |
| `BenchmarkBuildPreviewLines_OversizeUser` | `150,375,845` | `3,245,960` | `18,076` |
| `BenchmarkBuildPreviewLines_OversizeAssistant` | `230,387,602` | `4,950,398` | `30,124` |
| `BenchmarkBuildPreviewLines_UnicodeWide` | `159,790,651` | `7,423,374` | `15,135` |
| `BenchmarkPreviewIndexLoad_1k` | `2,253,474` | `731,142` | `6,048` |
| `BenchmarkPreviewIndexUpsert_1k` | `6,860,861` | `1,231,998` | `9,132` |
| `BenchmarkPreviewIndexUpsert_Trimmed` | `50,964,512` | `29,871,661` | `6,509` |

Observation:

- Oversize preview inputs and byte-budget trimming are the most expensive TUI paths in the current lightweight benchmark set.

## Next Use

- Re-run this file's command set after meaningful scanner, preview, or CLI rendering changes.
- Only add new CI thresholds after at least one more baseline pass on a second machine or runner class.

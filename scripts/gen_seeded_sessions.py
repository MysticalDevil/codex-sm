#!/usr/bin/env python3
"""Generate deterministic Codex session fixture datasets.

Usage:
  python3 scripts/gen_seeded_sessions.py --seed 20260308 --count 50 \
    --time-range-start 2026-03-01T00:00:00Z \
    --time-range-end 2026-03-31T23:59:59Z \
    --output-root ./testdata/_generated/sessions

Output:
  Writes session files under <output-root>/YYYY/MM/DD/*.jsonl with session_meta and
  response_item lines. Can also emit intentionally risky/extreme fixtures such as
  missing-meta, corrupted, oversize-meta, oversize user/assistant messages, large
  files, no-final-newline files, and Unicode-heavy samples.
"""

from __future__ import annotations

import argparse
import json
import random
from datetime import UTC, datetime, timedelta
from pathlib import Path


HOSTS = [
    "/workspace/proj-alpha",
    "/workspace/proj-beta",
    "/workspace/proj-gamma",
    "/srv/codex/mono-repo",
]

USER_PROMPTS = [
    "请帮我实现会话恢复功能",
    "please add retry logic for scanner",
    "por favor corrige el analisis de sesiones",
    "salve quaeso sessiones refice",
    "セッション一覧の表示を最適化してください",
    "세션 스캐너 성능을 개선해 주세요",
    "يرجى تحسين فحص الجلسات",
    "请修复 list command gracias 日本語も対応 مرحبا",
    "fix flaky test 😄🔥 in scanner",
    "add pagination support for the list command",
    "请增加 --head-width 的边界测试",
    "optimize ScanSessions for large directories",
    "por favor agrega pruebas de mezcla emoji 😅🚀",
    "日本語とEnglishが混在するヘッド抽出を直して",
    "세션 필터에서 대소문자 무시 검색을 추가해 주세요",
    "يرجى إضافة وضع dry-run أكثر وضوحا",
    "latin quaestio: quomodo session id ex filename legitur",
    "need help with sorting by updated_at desc",
    "请检查 corrupted 文件的健康状态判定",
    "añade soporte para exportar csv con columnas personalizadas",
    "テストカバレッジを 80% 以上にしたい",
    "unicode normalization issue in matcher",
    "请把多语言关键字检索做成组合过滤",
    "can we benchmark filter performance with 100k sessions",
    "세션 삭제 전 미리보기 출력 형식을 개선해 주세요",
    "هل يمكن دعم استعادة حسب batch_id",
    "introduce deterministic fixtures for integration tests",
    "请把错误提示文案更具体一些",
    "por favor revisa el flujo de confirmacion interactiva",
    "emoji-only prompt 😄😄😄 should still be searchable",
    "日本語ヘッドの省略表示が崩れます",
    "mixed text: 修复 restore bug por favor 🙏",
    "need robust parsing when json line has extra spaces",
]

ASSISTANT_PROMPTS = [
    "I will inspect scanner and selector code paths first.",
    "我会先补充单元测试再验证覆盖率。",
    "Podemos agregar pruebas para mezcla multilingue y emoji.",
    "了解しました。まず失敗ケースを再現します。",
    "네, 대화 길이가 긴 케이스도 추가하겠습니다.",
    "سأضيف سيناريوهات خاصة ثم اشغل الاختبارات.",
    "I can add deterministic fixtures so results are reproducible across runs.",
    "我会先写一个最小失败用例，再做针对性修复。",
    "Primero validare el comportamiento actual y despues ajustamos la logica.",
    "了解です。境界値ケースから順に確認します。",
    "좋습니다. 성능 회귀가 없는지도 함께 확인하겠습니다.",
    "سأتحقق من التوافق مع تنسيق JSONL الحالي.",
    "I suggest splitting parsing and scoring tests for clearer failure signals.",
    "我会把多语言场景拆成独立子测试，便于定位问题。",
    "Podemos medir impacto con benchmark antes y despues del cambio.",
    "この変更は既存のCLIフラグ互換性を維持します。",
    "필요하면 selector 매칭 로직에 추가 테스트를 넣겠습니다.",
    "سأضيف رسائل خطأ أوضح في حالات الادخال غير الصحيح.",
    "I will keep the patch minimal and avoid touching unrelated files.",
    "我会优先保证 dry-run 和 confirm 的安全语义不变。",
    "Podemos incluir casos con arabe, coreano y japones en la misma conversacion.",
    "日本語ヘッドの切り詰め表示も合わせて確認します。",
    "네, emoji 혼합 문자열 검색도 회귀 테스트에 포함하겠습니다.",
    "سأشغل اختبار الوحدة الاصغر ذي الصلة بعد التعديل.",
    "I can also wire this generator into justfile if you want a shortcut target.",
    "我会确保相同 seed 下输出完全一致。",
    "Podemos extender los prompts sin cambiar el formato de salida.",
    "必要なら start-time を固定して时系列数据を再现します。",
    "성능이 걱정되면 생성 턴 수 범위를 조정할 수 있습니다.",
    "يمكننا اضافة سيناريوهات فساد متعمد للملفات لاحقا.",
    "I will provide command examples for quick local smoke checks.",
]

CODE_HEAVY_SNIPPETS = [
    "func restoreSession(id string) error { return nil }",
    "if err := scanner.Scan(); err != nil { return err }",
    "previewCache[key] = append([]string(nil), lines...)",
    "for _, item := range items { total += item.SizeBytes }",
]

LOG_HEAVY_SNIPPETS = [
    "2026-03-09T09:00:00Z level=info component=scanner msg=\"scan started\"",
    "2026-03-09T09:00:01Z level=warn component=preview msg=\"oversize token discarded\"",
    "2026-03-09T09:00:02Z level=error component=doctor msg=\"integrity mismatch\"",
    "2026-03-09T09:00:03Z level=info component=tui msg=\"preview request queued\"",
]

UNICODE_WIDE_SNIPPETS = [
    "请处理这个超长宽字符会话",
    "全角ＡＢＣ１２３",
    "emoji family 👨‍👩‍👧‍👦",
    "rocket 🚀",
    "Arabic مرحبا بالعالم",
    "Hebrew שלום",
    "Japanese セッション復元",
    "Korean 한글 테스트",
]

PAYLOAD_SHAPES = {"text-only", "content-only", "mixed", "code-heavy", "log-heavy"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate deterministic random Codex session files.")
    parser.add_argument("--output-root", default="testdata/_generated/sessions", help="output root directory")
    parser.add_argument("--seed", type=int, required=True, help="RNG seed for deterministic output")
    parser.add_argument("--count", type=int, default=3000, help="number of normal session files to generate")
    parser.add_argument("--min-turns", type=int, default=12, help="minimum conversation turns per session")
    parser.add_argument("--max-turns", type=int, default=48, help="maximum conversation turns per session")
    parser.add_argument("--risk-missing-meta-count", type=int, default=3, help="number of missing-meta risky files")
    parser.add_argument("--risk-corrupted-count", type=int, default=3, help="number of corrupted risky files")
    parser.add_argument("--large-file-count", type=int, default=0, help="number of large valid session files")
    parser.add_argument("--oversize-meta-count", type=int, default=0, help="number of oversize session_meta files")
    parser.add_argument("--oversize-user-count", type=int, default=0, help="number of oversize single-user-message files")
    parser.add_argument("--oversize-assistant-count", type=int, default=0, help="number of oversize single-assistant-message files")
    parser.add_argument("--no-newline-count", type=int, default=0, help="number of files written without trailing newline")
    parser.add_argument("--mixed-corrupt-huge-count", type=int, default=0, help="number of corrupted-plus-huge files")
    parser.add_argument("--unicode-wide-count", type=int, default=0, help="number of Unicode-heavy files")
    parser.add_argument("--long-message-bytes", type=int, default=128 * 1024, help="target size for oversize message payloads")
    parser.add_argument("--meta-line-bytes", type=int, default=96 * 1024, help="target size for oversize meta cwd payloads")
    parser.add_argument("--large-file-target-bytes", type=int, default=1024 * 1024, help="target size for large generated files")
    parser.add_argument(
        "--payload-shape",
        default="mixed",
        help="message payload shape: text-only|content-only|mixed|code-heavy|log-heavy",
    )
    parser.add_argument(
        "--omit-final-newline",
        action="store_true",
        help="omit trailing newline on a subset of generated normal and extreme files",
    )
    parser.add_argument(
        "--start-time",
        default="2026-03-01T00:00:00Z",
        help="legacy base RFC3339 timestamp in UTC; used when --time-range-start is omitted",
    )
    parser.add_argument(
        "--time-range-start",
        default="",
        help="RFC3339 UTC start timestamp for randomized created_at range",
    )
    parser.add_argument(
        "--time-range-end",
        default="",
        help="RFC3339 UTC end timestamp for randomized created_at range",
    )
    return parser.parse_args()


def parse_start_time(raw: str) -> datetime:
    value = raw.strip()
    if value.endswith("Z"):
        value = value[:-1] + "+00:00"
    dt = datetime.fromisoformat(value)
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=UTC)
    return dt.astimezone(UTC)


def make_session_id(rng: random.Random) -> str:
    h = f"{rng.getrandbits(128):032x}"
    return f"{h[:8]}-{h[8:12]}-{h[12:16]}-{h[16:20]}-{h[20:]}"


def random_time_in_range(rng: random.Random, start: datetime, end: datetime) -> datetime:
    if end < start:
        raise ValueError("time range end must be >= start")
    start_ts = int(start.timestamp())
    end_ts = int(end.timestamp())
    picked = rng.randint(start_ts, end_ts)
    return datetime.fromtimestamp(picked, tz=UTC)


def encode_json_line(payload: dict) -> str:
    return json.dumps(payload, ensure_ascii=False, separators=(",", ":"))


def write_json_line(fp, payload: dict, *, newline: bool = True) -> None:
    fp.write(encode_json_line(payload))
    if newline:
        fp.write("\n")


def output_day_dir(out_root: Path, created: datetime) -> Path:
    day_dir = out_root / created.strftime("%Y") / created.strftime("%m") / created.strftime("%d")
    day_dir.mkdir(parents=True, exist_ok=True)
    return day_dir


def shaped_message_payload(role: str, text: str, payload_shape: str) -> dict:
    payload = {"type": "message", "role": role}
    if payload_shape == "text-only":
        payload["text"] = text
        return payload
    if payload_shape == "content-only":
        payload["content"] = [{"type": "input_text", "text": text}]
        return payload
    payload["text"] = text if payload_shape in {"mixed", "log-heavy"} else ""
    payload["content"] = [{"type": "input_text", "text": text}]
    return payload


def text_fragments(payload_shape: str, role: str) -> list[str]:
    if payload_shape == "code-heavy":
        return CODE_HEAVY_SNIPPETS + USER_PROMPTS[:6]
    if payload_shape == "log-heavy":
        return LOG_HEAVY_SNIPPETS + ASSISTANT_PROMPTS[:6]
    if role == "assistant":
        return ASSISTANT_PROMPTS
    return USER_PROMPTS


def build_repeated_text(label: str, target_bytes: int, fragments: list[str]) -> str:
    parts: list[str] = [label]
    size = len(label.encode("utf-8"))
    idx = 0
    while size < target_bytes:
        chunk = f" [{idx:04d}] {fragments[idx % len(fragments)]}"
        parts.append(chunk)
        size += len(chunk.encode("utf-8"))
        idx += 1
    return "".join(parts)


def maybe_drop_final_newline(path: Path, enabled: bool) -> None:
    if not enabled:
        return
    data = path.read_bytes()
    if data.endswith(b"\n"):
        path.write_bytes(data[:-1])


def generate_one_session(
    out_root: Path,
    rng: random.Random,
    range_start: datetime,
    range_end: datetime,
    min_turns: int,
    max_turns: int,
    payload_shape: str,
    omit_final_newline: bool,
) -> Path:
    session_id = make_session_id(rng)
    created = random_time_in_range(rng, range_start, range_end)
    day_dir = output_day_dir(out_root, created)

    file_name = f"rollout-{created.strftime('%Y-%m-%dT%H-%M-%S')}-{session_id}.jsonl"
    file_path = day_dir / file_name
    turns = rng.randint(min_turns, max_turns)

    with file_path.open("w", encoding="utf-8", newline="\n") as fp:
        write_json_line(
            fp,
            {
                "timestamp": created.isoformat().replace("+00:00", "Z"),
                "type": "session_meta",
                "payload": {
                    "id": session_id,
                    "timestamp": created.isoformat().replace("+00:00", "Z"),
                    "cwd": rng.choice(HOSTS),
                },
            },
        )

        for turn in range(turns):
            role = "user" if turn % 2 == 0 else "assistant"
            fragments = text_fragments(payload_shape, role)
            text = rng.choice(fragments)
            if rng.random() < 0.25:
                text = f"{text} #{turn:03d}"
            newline = not omit_final_newline or turn != turns - 1
            write_json_line(
                fp,
                {
                    "timestamp": (created + timedelta(seconds=turn * 9)).isoformat().replace("+00:00", "Z"),
                    "type": "response_item",
                    "payload": shaped_message_payload(role, text, payload_shape),
                },
                newline=newline,
            )

    return file_path


def generate_risky_missing_meta(out_root: Path, rng: random.Random, range_start: datetime, range_end: datetime, idx: int) -> Path:
    created = random_time_in_range(rng, range_start, range_end)
    day_dir = output_day_dir(out_root, created)
    session_id = make_session_id(rng)
    file_name = f"risk-missing-meta-{idx:03d}-{created.strftime('%Y-%m-%dT%H-%M-%S')}-{session_id}.jsonl"
    file_path = day_dir / file_name
    with file_path.open("w", encoding="utf-8", newline="\n") as fp:
        write_json_line(
            fp,
            {
                "timestamp": created.isoformat().replace("+00:00", "Z"),
                "type": "response_item",
                "payload": shaped_message_payload("user", "risk fixture missing session_meta", "content-only"),
            },
        )
    return file_path


def generate_risky_corrupted(out_root: Path, rng: random.Random, range_start: datetime, range_end: datetime, idx: int) -> Path:
    created = random_time_in_range(rng, range_start, range_end)
    day_dir = output_day_dir(out_root, created)
    session_id = make_session_id(rng)
    file_name = f"risk-corrupted-{idx:03d}-{created.strftime('%Y-%m-%dT%H-%M-%S')}-{session_id}.jsonl"
    file_path = day_dir / file_name
    with file_path.open("w", encoding="utf-8", newline="\n") as fp:
        fp.write('{"type":"session_meta","payload":{"id":"broken"\n')
        fp.write("not-json-line\n")
    return file_path


def generate_oversize_meta(
    out_root: Path, rng: random.Random, range_start: datetime, range_end: datetime, idx: int, meta_line_bytes: int
) -> Path:
    created = random_time_in_range(rng, range_start, range_end)
    day_dir = output_day_dir(out_root, created)
    session_id = make_session_id(rng)
    file_name = f"oversize-meta-{idx:03d}-{created.strftime('%Y-%m-%dT%H-%M-%S')}-{session_id}.jsonl"
    file_path = day_dir / file_name
    cwd = build_repeated_text("/workspace/extreme-meta", meta_line_bytes, [f"segment-{i:03d}" for i in range(64)])

    with file_path.open("w", encoding="utf-8", newline="\n") as fp:
        write_json_line(
            fp,
            {
                "timestamp": created.isoformat().replace("+00:00", "Z"),
                "type": "session_meta",
                "payload": {
                    "id": session_id,
                    "timestamp": created.isoformat().replace("+00:00", "Z"),
                    "cwd": cwd,
                },
            },
        )
        write_json_line(
            fp,
            {
                "timestamp": (created + timedelta(seconds=9)).isoformat().replace("+00:00", "Z"),
                "type": "response_item",
                "payload": shaped_message_payload("user", "oversize meta fixture follow-up", "content-only"),
            },
        )
    return file_path


def generate_oversize_message(
    out_root: Path,
    rng: random.Random,
    range_start: datetime,
    range_end: datetime,
    idx: int,
    role: str,
    payload_shape: str,
    target_bytes: int,
) -> Path:
    created = random_time_in_range(rng, range_start, range_end)
    day_dir = output_day_dir(out_root, created)
    session_id = make_session_id(rng)
    file_name = f"oversize-{role}-{idx:03d}-{created.strftime('%Y-%m-%dT%H-%M-%S')}-{session_id}.jsonl"
    file_path = day_dir / file_name
    label = f"{role.upper()}-LONG-START"
    text = build_repeated_text(label, target_bytes, text_fragments(payload_shape, role))

    with file_path.open("w", encoding="utf-8", newline="\n") as fp:
        write_json_line(
            fp,
            {
                "timestamp": created.isoformat().replace("+00:00", "Z"),
                "type": "session_meta",
                "payload": {
                    "id": session_id,
                    "timestamp": created.isoformat().replace("+00:00", "Z"),
                    "cwd": f"/workspace/extreme-{role}",
                },
            },
        )
        if role == "assistant":
            write_json_line(
                fp,
                {
                    "timestamp": (created + timedelta(seconds=5)).isoformat().replace("+00:00", "Z"),
                    "type": "response_item",
                    "payload": shaped_message_payload("user", "priming prompt before long assistant payload", "content-only"),
                },
            )
        write_json_line(
            fp,
            {
                "timestamp": (created + timedelta(seconds=9)).isoformat().replace("+00:00", "Z"),
                "type": "response_item",
                "payload": shaped_message_payload(role, text, payload_shape),
            },
        )
    return file_path


def generate_large_file(
    out_root: Path,
    rng: random.Random,
    range_start: datetime,
    range_end: datetime,
    idx: int,
    payload_shape: str,
    target_bytes: int,
) -> Path:
    created = random_time_in_range(rng, range_start, range_end)
    day_dir = output_day_dir(out_root, created)
    session_id = make_session_id(rng)
    file_name = f"large-session-{idx:03d}-{created.strftime('%Y-%m-%dT%H-%M-%S')}-{session_id}.jsonl"
    file_path = day_dir / file_name

    with file_path.open("w", encoding="utf-8", newline="\n") as fp:
        write_json_line(
            fp,
            {
                "timestamp": created.isoformat().replace("+00:00", "Z"),
                "type": "session_meta",
                "payload": {
                    "id": session_id,
                    "timestamp": created.isoformat().replace("+00:00", "Z"),
                    "cwd": "/workspace/extreme-large",
                },
            },
        )
        turn = 0
        while file_path.stat().st_size < target_bytes:
            role = "user" if turn % 2 == 0 else "assistant"
            text = build_repeated_text(
                f"LARGE-{role}-{turn:04d}",
                min(8192, max(2048, target_bytes // 16)),
                text_fragments(payload_shape, role),
            )
            write_json_line(
                fp,
                {
                    "timestamp": (created + timedelta(seconds=turn * 3)).isoformat().replace("+00:00", "Z"),
                    "type": "response_item",
                    "payload": shaped_message_payload(role, text, payload_shape),
                },
            )
            turn += 1
    return file_path


def generate_no_newline(
    out_root: Path, rng: random.Random, range_start: datetime, range_end: datetime, idx: int, target_bytes: int
) -> Path:
    created = random_time_in_range(rng, range_start, range_end)
    day_dir = output_day_dir(out_root, created)
    session_id = make_session_id(rng)
    file_name = f"no-newline-{idx:03d}-{created.strftime('%Y-%m-%dT%H-%M-%S')}-{session_id}.jsonl"
    file_path = day_dir / file_name
    text = build_repeated_text("NO-NEWLINE", max(512, target_bytes // 8), USER_PROMPTS[:8])
    with file_path.open("w", encoding="utf-8", newline="\n") as fp:
        write_json_line(
            fp,
            {
                "timestamp": created.isoformat().replace("+00:00", "Z"),
                "type": "response_item",
                "payload": shaped_message_payload("user", text, "content-only"),
            },
            newline=False,
        )
    return file_path


def generate_mixed_corrupt_huge(
    out_root: Path, rng: random.Random, range_start: datetime, range_end: datetime, idx: int, target_bytes: int
) -> Path:
    created = random_time_in_range(rng, range_start, range_end)
    day_dir = output_day_dir(out_root, created)
    session_id = make_session_id(rng)
    file_name = f"mixed-corrupt-huge-{idx:03d}-{created.strftime('%Y-%m-%dT%H-%M-%S')}-{session_id}.jsonl"
    file_path = day_dir / file_name
    broken = build_repeated_text('{"type":"session_meta","payload":{"id":"broken","cwd":"', max(1024, target_bytes // 4), LOG_HEAVY_SNIPPETS)
    with file_path.open("w", encoding="utf-8", newline="\n") as fp:
        fp.write(broken)
        fp.write("\n")
        write_json_line(
            fp,
            {
                "timestamp": (created + timedelta(seconds=9)).isoformat().replace("+00:00", "Z"),
                "type": "response_item",
                "payload": shaped_message_payload("user", "tail line after corruption", "content-only"),
            },
        )
    return file_path


def generate_unicode_wide(
    out_root: Path, rng: random.Random, range_start: datetime, range_end: datetime, idx: int, target_bytes: int
) -> Path:
    created = random_time_in_range(rng, range_start, range_end)
    day_dir = output_day_dir(out_root, created)
    session_id = make_session_id(rng)
    file_name = f"unicode-wide-{idx:03d}-{created.strftime('%Y-%m-%dT%H-%M-%S')}-{session_id}.jsonl"
    file_path = day_dir / file_name
    text = build_repeated_text("UNICODE-WIDE", target_bytes, UNICODE_WIDE_SNIPPETS)

    with file_path.open("w", encoding="utf-8", newline="\n") as fp:
        write_json_line(
            fp,
            {
                "timestamp": created.isoformat().replace("+00:00", "Z"),
                "type": "session_meta",
                "payload": {
                    "id": session_id,
                    "timestamp": created.isoformat().replace("+00:00", "Z"),
                    "cwd": "/workspace/extreme-unicode",
                },
            },
        )
        write_json_line(
            fp,
            {
                "timestamp": (created + timedelta(seconds=9)).isoformat().replace("+00:00", "Z"),
                "type": "response_item",
                "payload": shaped_message_payload("user", text, "content-only"),
            },
        )
    return file_path


def main() -> int:
    args = parse_args()
    if args.count < 0:
        raise SystemExit("--count must be >= 0")
    if args.min_turns <= 0 or args.max_turns <= 0:
        raise SystemExit("--min-turns and --max-turns must be > 0")
    if args.min_turns > args.max_turns:
        raise SystemExit("--min-turns cannot be greater than --max-turns")
    if args.payload_shape not in PAYLOAD_SHAPES:
        raise SystemExit(f"--payload-shape must be one of: {', '.join(sorted(PAYLOAD_SHAPES))}")

    count_args = [
        args.risk_missing_meta_count,
        args.risk_corrupted_count,
        args.large_file_count,
        args.oversize_meta_count,
        args.oversize_user_count,
        args.oversize_assistant_count,
        args.no_newline_count,
        args.mixed_corrupt_huge_count,
        args.unicode_wide_count,
    ]
    if any(v < 0 for v in count_args):
        raise SystemExit("count-style arguments must be >= 0")
    if args.long_message_bytes <= 0 or args.meta_line_bytes <= 0 or args.large_file_target_bytes <= 0:
        raise SystemExit("byte-size arguments must be > 0")

    rng = random.Random(args.seed)
    range_start = parse_start_time(args.time_range_start) if args.time_range_start.strip() else parse_start_time(args.start_time)
    range_end = parse_start_time(args.time_range_end) if args.time_range_end.strip() else range_start + timedelta(days=30)
    if range_end < range_start:
        raise SystemExit("--time-range-end cannot be earlier than --time-range-start")

    out_root = Path(args.output_root).expanduser().resolve()
    out_root.mkdir(parents=True, exist_ok=True)

    generated: dict[str, list[Path]] = {
        "normal": [],
        "risk_missing_meta": [],
        "risk_corrupted": [],
        "large_file": [],
        "oversize_meta": [],
        "oversize_user": [],
        "oversize_assistant": [],
        "no_newline": [],
        "mixed_corrupt_huge": [],
        "unicode_wide": [],
    }

    for _ in range(args.count):
        generated["normal"].append(
            generate_one_session(
                out_root,
                rng,
                range_start,
                range_end,
                args.min_turns,
                args.max_turns,
                args.payload_shape,
                args.omit_final_newline and rng.random() < 0.1,
            )
        )
    for i in range(args.risk_missing_meta_count):
        generated["risk_missing_meta"].append(generate_risky_missing_meta(out_root, rng, range_start, range_end, i))
    for i in range(args.risk_corrupted_count):
        generated["risk_corrupted"].append(generate_risky_corrupted(out_root, rng, range_start, range_end, i))
    for i in range(args.large_file_count):
        p = generate_large_file(out_root, rng, range_start, range_end, i, args.payload_shape, args.large_file_target_bytes)
        maybe_drop_final_newline(p, args.omit_final_newline and rng.random() < 0.2)
        generated["large_file"].append(p)
    for i in range(args.oversize_meta_count):
        p = generate_oversize_meta(out_root, rng, range_start, range_end, i, args.meta_line_bytes)
        maybe_drop_final_newline(p, args.omit_final_newline and rng.random() < 0.2)
        generated["oversize_meta"].append(p)
    for i in range(args.oversize_user_count):
        p = generate_oversize_message(out_root, rng, range_start, range_end, i, "user", args.payload_shape, args.long_message_bytes)
        maybe_drop_final_newline(p, args.omit_final_newline and rng.random() < 0.2)
        generated["oversize_user"].append(p)
    for i in range(args.oversize_assistant_count):
        p = generate_oversize_message(out_root, rng, range_start, range_end, i, "assistant", args.payload_shape, args.long_message_bytes)
        maybe_drop_final_newline(p, args.omit_final_newline and rng.random() < 0.2)
        generated["oversize_assistant"].append(p)
    for i in range(args.no_newline_count):
        generated["no_newline"].append(generate_no_newline(out_root, rng, range_start, range_end, i, args.long_message_bytes))
    for i in range(args.mixed_corrupt_huge_count):
        generated["mixed_corrupt_huge"].append(generate_mixed_corrupt_huge(out_root, rng, range_start, range_end, i, args.long_message_bytes))
    for i in range(args.unicode_wide_count):
        p = generate_unicode_wide(out_root, rng, range_start, range_end, i, max(4096, args.long_message_bytes // 4))
        maybe_drop_final_newline(p, args.omit_final_newline and rng.random() < 0.2)
        generated["unicode_wide"].append(p)

    summary = " ".join(f"{k}={len(v)}" for k, v in generated.items())
    print(
        f"generated {summary} seed={args.seed} payload_shape={args.payload_shape} "
        f"time_range={range_start.isoformat().replace('+00:00', 'Z')}..{range_end.isoformat().replace('+00:00', 'Z')} "
        f"root={out_root}"
    )
    samples = [p for paths in generated.values() for p in paths][:8]
    for path in samples:
        print(f"sample={path}")
    if sum(len(paths) for paths in generated.values()) > len(samples):
        print("sample=...")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
"""reflect_extract.py — Stage 3 code extractor for /reflect.

Pure code, no model calls. Reads Claude Code .jsonl transcripts, applies the
is_human_turn primitive, runs the 4 code-tier signal scorers, and emits a
per-session signal table + per-project rollup. Every signal carries record
evidence so a finding can cite a uuid (feedback-loop rule).

Usage: reflect_extract.py [window_days] [--scope substr,substr] [--json out.json]
Defaults: 7 days, product-work scope (harness self-dev + leva + tmp excluded).
"""
import os, json, re, sys, time, glob
from collections import defaultdict

ROOT = os.path.expanduser("~/.claude/projects")
DAYS = 7
SCOPE = None
JSON_OUT = None
av = sys.argv[1:]
i = 0
while i < len(av):
    a = av[i]
    if a == "--scope": SCOPE = av[i+1].split(","); i += 2
    elif a == "--json": JSON_OUT = av[i+1]; i += 2
    elif a.endswith("d") and a[:-1].isdigit(): DAYS = int(a[:-1]); i += 1
    elif a.isdigit(): DAYS = int(a); i += 1
    else: i += 1
CUTOFF = time.time() - DAYS*86400

DEFAULT_INCLUDE = [
    "effective-sales-vpo", "effective-litellm", "effective-card-balance",
    "MeetilyNative", "Projects-meetily", "effective-rubrick-controller",
    "effective-notes", "effective-slides", "effective-yandex-edu-bridge",
    "effective-mokka-ai-course", "effective-sales-presale", "Projects-vpn",
    "effective-effective-dev-site",
]
EXCLUDE = ["effective-harness", "leva-install-ipa", "Projects-sample", "private-tmp"]

def included(d):
    if any(x in d for x in EXCLUDE): return False
    return any(x in d for x in (SCOPE or DEFAULT_INCLUDE))

# --- signal scorers over user pushback (proxy) + assistant length (slop) ---
SIG = {
 "redo":      [r"переделай", r"переделать", r"не так", r"опять", r"снова", r"заново",
               r"верни как было", r"откати", r"\bredo\b", r"do it again"],
 "slop":      [r"слоп", r"\bslop\b", r"вода\b", r"водянист", r"сократи", r"короче пиши",
               r"много текста", r"не отдал текст"],
 "factcheck": [r"не провер", r"перепровер", r"проверь факт", r"выдума", r"наврал",
               r"галлюцин", r"это ложь", r"неверн", r"ты уверен", r"сам гоняю", r"\bя сам\b"],
 "skipskill": [r"забыл", r"не запустил", r"не прогнал", r"не вызвал скилл", r"надо было прогнать"],
 "notest":    [r"не протест", r"не потест", r"симулятор", r"simulator", r"сломал",
               r"не работает", r"\bбаг\b", r"build green", r"не запустил прилож"],
 "ugly_ui":   [r"отвратительн", r"ужасн", r"некрасив", r"уродлив", r"обрезан", r"вылез",
               r"налез", r"костыл", r"некликабельн", r"страшн"],
 "anger":     [r"\bбля", r"нахуй", r"\bхуй", r"пиздец", r"задолбал", r"заебал", r"\bблин\b"],
}
PAT = {k: re.compile("|".join(v)) for k, v in SIG.items()}
ANGER_W = lambda n: 2.0 if n >= 3 else (1.5 if n >= 1 else 1.0)

REMINDER = re.compile(r"<system-reminder.*?</system-reminder>", re.S)

def is_human_turn(obj):
    if obj.get("type") != "user": return None
    c = obj.get("message", {}).get("content")
    if not isinstance(c, str): return None          # tool_result / array -> not human
    txt = REMINDER.sub(" ", c)
    if txt.lstrip().startswith("<command-"): return None
    if not txt.strip(): return None
    return txt

def assistant_text_len(obj):
    if obj.get("type") != "assistant": return 0
    c = obj.get("message", {}).get("content")
    n = 0
    if isinstance(c, list):
        for b in c:
            if isinstance(b, dict) and b.get("type") == "text":
                n += len(b.get("text", ""))
    return n

def scan(path):
    humans, long_ans, uuids = [], 0, []
    try:
        with open(path, errors="ignore") as f:
            for line in f:
                line = line.strip()
                if not line: continue
                try: o = json.loads(line)
                except: continue
                t = is_human_turn(o)
                if t is not None:
                    humans.append((o.get("uuid", ""), t))
                elif assistant_text_len(o) > 4000:
                    long_ans += 1
                    uuids.append(o.get("uuid", ""))
    except: pass
    return humans, long_ans, uuids

sessions, proj = [], defaultdict(lambda: defaultdict(float))
for d in glob.glob(os.path.join(ROOT, "*")):
    base = os.path.basename(d)
    if not included(base): continue
    for path in glob.glob(os.path.join(d, "*.jsonl")):
        if os.path.getmtime(path) < CUTOFF: continue
        humans, long_ans, long_uuids = scan(path)
        if not humans: continue
        joined = "\n".join(t for _, t in humans).lower()
        hits, ev = {}, {}
        for k, p in PAT.items():
            ms = p.findall(joined)
            if ms:
                hits[k] = len(ms)
                ev[k] = [u for u, t in humans if p.search(t.lower())][:3]
        if long_ans:                                # assistant-side slop heuristic
            hits["slop"] = hits.get("slop", 0) + long_ans
            ev.setdefault("slop", []).extend(long_uuids[:3])
        if not hits: continue
        sev = ANGER_W(hits.get("anger", 0))
        weighted = sum(v for k, v in hits.items() if k != "anger") * sev
        sessions.append({"session": os.path.basename(path), "project": base,
                         "n_human": len(humans), "hits": hits, "evidence": ev,
                         "anger_weight": sev, "weighted": round(weighted, 1)})
        for k, v in hits.items(): proj[base][k] += v

sessions.sort(key=lambda s: -s["weighted"])
print(f"=== /reflect extract | {DAYS}d | sessions with signals: {len(sessions)} ===\n")
print("--- per-project signal totals ---")
for p, h in sorted(proj.items(), key=lambda kv: -sum(kv[1].values())):
    det = " ".join(f"{k}:{int(v)}" for k, v in sorted(h.items(), key=lambda x: -x[1]) if v)
    print(f"{int(sum(h.values())):4d}  {p[-50:]:50s} {det}")
print("\n--- top 25 sessions by anger-weighted signal ---")
for s in sessions[:25]:
    det = ",".join(f"{k}:{int(v)}" for k, v in sorted(s['hits'].items(), key=lambda x: -x[1]) if v)
    print(f"{s['weighted']:6.1f} [{s['n_human']:3d}h] {s['project'][-34:]:34s} {s['session'][:12]} {det}")

if JSON_OUT:
    with open(JSON_OUT, "w") as f: json.dump({"days": DAYS, "sessions": sessions, "projects": proj}, f, indent=2, default=str)
    print(f"\nwrote {JSON_OUT}")

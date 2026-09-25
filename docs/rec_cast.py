#!/usr/bin/env python3
"""Drive herdr-palette in a pty and record an asciicast v2 file.

Usage: rec_cast.py <binary> <cast-out>
Deterministic scripted demo: open, filter, navigate, re-filter, quit.
"""
import json
import os
import pty
import sys
import time

WIDTH, HEIGHT = 100, 28

DOWN = "\x1b[B"
ESC = "\x1b"
BS = "\x7f"

# Canned answers to the queries Bubble Tea / termenv send on startup.
# Without a real terminal emulator behind the pty these would block startup.
# OSC replies use BEL termination so Bubble Tea consumes them as responses
# instead of leaking bytes into the input line.
QUERIES = [
    (b"\x1b]11;?\x1b\\", b"\x1b]11;rgb:0000/0000/0000\x07"),  # bg: black
    (b"\x1b]11;?\x07", b"\x1b]11;rgb:0000/0000/0000\x07"),
    (b"\x1b[6n", b"\x1b[1;1R"),  # cursor report: home (never_HEIGHT; Bubble Tea
    # derives renderer geometry from this and a wrong row drops top lines)
    (b"\x1b[c", b"\x1b[?62c"),  # primary DA: VT220
    (b"\x1b[>c", b"\x1b[>0;0;0c"),  # secondary DA
    (b"\x1b[?u", b"\x1b[?0u"),  # kitty keyboard: unsupported
    (b"\x1b[0u", b"\x1b[?0u"),
]


def respond(fd: int, pending: bytearray) -> bytearray:
    """Answer any complete terminal queries sitting in pending output."""
    for want, reply in QUERIES:
        while True:
            i = pending.find(want)
            if i < 0:
                break
            try:
                os.write(fd, reply)
            except OSError:
                pass
            del pending[: i + len(want)]
    # keep the tail bounded (may hold a split query)
    if len(pending) > 256:
        del pending[: len(pending) - 256]
    return pending

SCRIPT = [
    (1.5, "resu"),
    (2.0, DOWN + DOWN),
    (1.5, BS * 4),
    (1.0, "split"),
    (2.0, ESC),
    (1.0, None),  # let it exit, then EOF
]


def main() -> int:
    binary, out = sys.argv[1], sys.argv[2]
    pid, fd = pty.fork()
    if pid == 0:
        os.execvpe(binary, [binary], {**os.environ, "TERM": "xterm-256color"})
        os._exit(1)

    import fcntl
    import struct
    import termios

    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", HEIGHT, WIDTH, 0, 0))
    # No local echo: our query replies travel the input channel (as on a real
    # terminal) and must not be echoed back into the stream a second time.
    attrs = termios.tcgetattr(fd)
    attrs[3] &= ~termios.ECHO
    termios.tcsetattr(fd, termios.TCSANOW, attrs)

    events: list[tuple[float, str, str]] = []
    start = time.time()
    pending = bytearray()

    import selectors

    sel = selectors.DefaultSelector()
    sel.register(fd, selectors.EVENT_READ)
    buf = b""
    deadline = start + sum(t for t, _ in SCRIPT) + 3.0
    step_idx = 0
    next_at = start + SCRIPT[0][0]
    alive = True
    while time.time() < deadline:
        timeout = max(0.0, next_at - time.time()) if step_idx < len(SCRIPT) else 0.2
        for key, _ in sel.select(timeout):
            try:
                chunk = os.read(fd, 65536)
            except OSError:
                chunk = b""
            if not chunk:
                alive = False
                break
            pending += chunk
            respond(fd, pending)
            events.append((time.time() - start, "o", chunk.decode("utf-8", "replace")))
        if not alive:
            break
        if step_idx < len(SCRIPT) and time.time() >= next_at:
            _, keys = SCRIPT[step_idx]
            if keys:
                os.write(fd, keys.encode())
            step_idx += 1
            if step_idx < len(SCRIPT):
                next_at = time.time() + SCRIPT[step_idx][0]
        if step_idx >= len(SCRIPT):
            # drain briefly, then check liveness
            time.sleep(0.2)
            try:
                _, status = os.waitpid(pid, os.WNOHANG)
                if status != 0 or not os.kill(pid, 0):
                    pass
            except (ChildProcessError, ProcessLookupError, OSError):
                break

    try:
        os.waitpid(pid, 0)
    except (ChildProcessError, OSError):
        pass
    try:
        os.close(fd)
    except OSError:
        pass

    header = {
        "version": 2,
        "width": WIDTH,
        "height": HEIGHT,
        "timestamp": int(start),
        "env": {"TERM": "xterm-256color", "SHELL": "/bin/zsh"},
    }
    with open(out, "w") as f:
        f.write(json.dumps(header) + "\n")
        for t, typ, data in events:
            f.write(json.dumps([round(t, 6), typ, data]) + "\n")
    print(f"wrote {out}: {len(events)} events, {time.time()-start:.1f}s")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

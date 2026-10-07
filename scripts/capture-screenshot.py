#!/usr/bin/env python3
"""Capture the actual TUI in a PTY, using disposable example projects only."""

import argparse
import codecs
import fcntl
import json
import os
from pathlib import Path
import pty
import select
import shutil
import struct
import subprocess
import termios
import time

import pyte
from PIL import Image, ImageDraw, ImageFont

REPO = Path(__file__).resolve().parents[1]
COLS, ROWS = 124, 20


def font_path(override):
    candidates = [override] if override else [
        '/System/Library/Fonts/Menlo.ttc',
        '/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf',
        '/usr/share/fonts/truetype/liberation2/LiberationMono-Regular.ttf',
    ]
    for candidate in candidates:
        if candidate and Path(candidate).is_file():
            return candidate
    raise SystemExit('No monospaced font found; provide --font /path/to/font.ttf')


def create_fixture(root):
    examples = [
        (".codex/sessions/rollout-01.jsonl", "atlas", "Review the workspace navigation", "codex-example-01"),
        (".claude/projects/atlas/02.jsonl", "atlas", "Add coverage for reconnect behavior", "claude-example-02"),
        (".cursor/projects/frontend/agent-transcripts/03.jsonl", "frontend", "Polish the documentation layout", "cursor-example-03"),
        (".codex/sessions/rollout-04.jsonl", "payments-api", "Fix idempotency in the retry queue", "codex-example-04"),
        (".pi/agent/sessions/05.jsonl", "data-pipeline", "Investigate missing event timestamps", "pi-example-05"),
        (".claude/projects/frontend/06.jsonl", "frontend", "Improve keyboard navigation", "claude-example-06"),
        (".codex/archived_sessions/rollout-07.jsonl", "atlas", "Compare release packaging options", "codex-example-07"),
    ]
    for index, (relative, project, title, identity) in enumerate(examples):
        cwd = root / "projects" / project
        cwd.mkdir(parents=True, exist_ok=True)
        path = root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text("\n".join(json.dumps(v) for v in [
            {"type": "session_meta", "payload": {"id": identity, "cwd": str(cwd), "model": "example-model"}},
            {"type": "response_item", "payload": {"role": "user", "content": [{"text": title}]}},
        ]) + "\n")
        timestamp = time.time() - (index + 1) * 3600
        os.utime(path, (timestamp, timestamp))
    return root


def capture(binary, config):
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', ROWS, COLS, 0, 0))
    env = dict(os.environ, TERM='xterm-256color', COLORTERM='truecolor', CLICOLOR_FORCE='1')
    env.pop('NO_COLOR', None)
    env['OPENCLAW_STATE_DIR'] = str(config / '.openclaw')
    process = None
    screen = pyte.Screen(COLS, ROWS)
    stream = pyte.Stream(screen)
    decoder = codecs.getincrementaldecoder('utf-8')(errors='replace')

    def pump(seconds):
        end = time.monotonic() + seconds
        while time.monotonic() < end:
            readable, _, _ = select.select([master], [], [], min(0.1, max(0, end - time.monotonic())))
            if readable:
                try:
                    data = os.read(master, 65536)
                except OSError:
                    break
                if not data:
                    break
                stream.feed(decoder.decode(data))

    try:
        process = subprocess.Popen([str(binary), '--home', str(config)],
                                   stdin=slave, stdout=slave, stderr=slave, env=env)
        os.close(slave)
        slave = None
        deadline = time.monotonic() + 15
        while not any('Review the workspace navigation' in line for line in screen.display):
            pump(0.15)
            if process.poll() is not None or time.monotonic() > deadline:
                raise RuntimeError('TUI did not finish the example scan:\n' + '\n'.join(screen.display))
        return screen
    finally:
        if process is not None and process.poll() is None:
            os.write(master, b'q')
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.terminate()
                process.wait(timeout=3)
        os.close(master)
        if slave is not None:
            os.close(slave)


def render(screen, font, output):
    cell_width, cell_height = 12, 24
    padding, top, bottom = 30, 86, 42
    image = Image.new('RGB', (COLS * cell_width + padding * 2,
                             ROWS * cell_height + top + bottom), '#151821')
    draw = ImageDraw.Draw(image)
    face = ImageFont.truetype(font, 20)
    title_face = ImageFont.truetype(font, 16)
    draw.rectangle((0, 0, image.width, 54), fill='#1f2330')
    for x in (28, 50, 72):
        draw.ellipse((x, 23, x + 10, 33), fill='#647083')
    title = 'Agent Sessions / generated sample histories'
    draw.text(((image.width - draw.textlength(title, title_face)) / 2, 17),
              title, font=title_face, fill='#acb5c4')
    palette = {'black': '#161923', 'red': '#e06c75', 'green': '#98c379',
               'brown': '#e5c07b', 'blue': '#61afef', 'magenta': '#c678dd',
               'cyan': '#56b6c2', 'white': '#d9e0ec'}

    def color(value, default):
        if value == 'default':
            return default
        if value in palette:
            return palette[value]
        if len(value) == 6:
            return '#' + value
        return default

    for y in range(ROWS):
        for x in range(COLS):
            cell = screen.buffer[y][x]
            fg = color(cell.fg, '#d9e0ec')
            bg = color(cell.bg, '#151821')
            if cell.reverse:
                fg, bg = bg, fg
            left, upper = padding + x * cell_width, top + y * cell_height
            if bg != '#151821':
                draw.rectangle((left, upper, left + cell_width - 1, upper + cell_height - 1), fill=bg)
            if cell.data.strip():
                draw.text((left, upper), cell.data, font=face, fill=fg)
    output.parent.mkdir(parents=True, exist_ok=True)
    image.save(output, optimize=True)
    print(f'Saved {output} ({image.width} × {image.height})')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--font', help='Path to a monospaced TTF or TTC font')
    parser.add_argument('--output', type=Path, default=REPO / 'docs/assets/screenshot.png')
    args = parser.parse_args()
    font = font_path(args.font)
    binary = REPO / 'bin/agent-sessions-tui'
    if not binary.is_file():
        parser.error('Build first: go build -o bin/agent-sessions-tui ./cmd/agent-sessions-tui')
    # Stable, private, disposable path keeps example paths readable in the image.
    root = Path('/tmp/agent-sessions-demo')
    try:
        root.mkdir(mode=0o700)
    except FileExistsError:
        parser.error(f'{root} already exists; refusing to modify it')
    try:
        config = create_fixture(root)
        render(capture(binary, config), font, args.output)
    finally:
        shutil.rmtree(root)


if __name__ == '__main__':
    main()

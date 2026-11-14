#!/usr/bin/env python3
"""生成 MiniPost 应用图标的全部版本（需要 macOS + Xcode 自带的 Icon Composer / ictool）。

设计概念：Mini + Post = 一枚“迷你邮票”，票面上是一架正在发出的纸飞机（发送请求）。
  - 浅色：靛紫到电蓝的渐变底，白色锯齿孔邮票；
  - 深色：近黑底，邮票变成深灰蓝，纸飞机保持鲜亮；
  - 着色 / 透明：由系统按图层亮度生成。
  邮票背后露出半枚邮戳（圆环 + 波浪消印线），增加层次但不抢戏。
  图层自上而下：纸飞机（带速度线）→ 邮票（玻璃质感）→ 邮戳。

产物（在仓库根目录运行）：
  AppIcon.icon/              Icon Composer 图层图标：浅色 / 深色 / 着色 / 透明 由系统自动切换
  resources/icon.png         浅色版 1024 位图（Windows / Linux / 旧系统的回退）
  internal/gui/icons/logo*.png  应用内（空状态、设置-关于）使用的 256px 浅色 / 深色版本
  docs/icon/*.png            各版本 1024 预览：light / dark / tinted / clear + 对比图

用法：python3 scripts/icon/gen_icon.py
"""
import json
import os
import shutil
import subprocess
import sys

ICTOOL = "/Applications/Xcode.app/Contents/Applications/Icon Composer.app/Contents/Executables/ictool"

# ---------- 几何（1024 画布，Icon Composer 的画布就是整个图标） ----------
CX = CY = 512
STAMP = 600                    # 邮票边长
HOLES, HOLE_R = 12, 15         # 每边的锯齿孔数量与半径
TILT = -8                      # 邮票与飞机整体倾斜角度
PLANE_SCALE = 4.5              # 纸飞机：100 单位坐标 → 像素


def svg(body, defs=""):
    d = f"<defs>\n{defs}</defs>\n" if defs else ""
    return ('<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">\n'
            + d + body + "</svg>\n")


def stamp_svg():
    """白色邮票：圆角方块，四边各有一排向内凹的半圆缺口（锯齿孔），整体倾斜。
    沿顺时针描边，缺口用 sweep=0 的圆弧向内凹，这样缺口只会削掉票面，不会在票外多出东西。"""
    x0 = y0 = CX - STAMP / 2
    x1 = y1 = CX + STAMP / 2
    r, rr = HOLE_R, 22
    step = STAMP / HOLES
    notch = lambda ex, ey: f"A{r} {r} 0 0 0 {ex:.2f} {ey:.2f}"
    d = [f"M{x0 + rr} {y0}"]
    for i in range(1, HOLES):  # 上边：左→右
        x = x0 + i * step
        d.append(f"L{x - r:.2f} {y0} {notch(x + r, y0)}")
    d.append(f"L{x1 - rr} {y0} Q{x1} {y0} {x1} {y0 + rr}")
    for i in range(1, HOLES):  # 右边：上→下
        y = y0 + i * step
        d.append(f"L{x1} {y - r:.2f} {notch(x1, y + r)}")
    d.append(f"L{x1} {y1 - rr} Q{x1} {y1} {x1 - rr} {y1}")
    for i in range(1, HOLES):  # 下边：右→左
        x = x1 - i * step
        d.append(f"L{x + r:.2f} {y1} {notch(x - r, y1)}")
    d.append(f"L{x0 + rr} {y1} Q{x0} {y1} {x0} {y1 - rr}")
    for i in range(1, HOLES):  # 左边：下→上
        y = y1 - i * step
        d.append(f"L{x0} {y + r:.2f} {notch(x0, y - r)}")
    d.append(f"L{x0} {y0 + rr} Q{x0} {y0} {x0 + rr} {y0} Z")
    return svg(f'  <g transform="rotate({TILT} {CX} {CY})">\n'
               f'    <path fill="#FFFFFF" d="{" ".join(d)}"/>\n  </g>\n')


def plane_svg():
    """票面内的细框、速度线与双色纸飞机。"""
    s = PLANE_SCALE
    ox, oy = CX - 50 * s + 6, CY - 50 * s + 4

    def pts(points):
        return " ".join(f"{ox + x * s:.1f},{oy + y * s:.1f}" for x, y in points)

    defs = (
        '  <linearGradient id="up" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#9A8CFF"/><stop offset="1" stop-color="#6D6BF5"/></linearGradient>\n'
        '  <linearGradient id="lo" x1="0" y1="0" x2="0.4" y2="1"><stop offset="0" stop-color="#5B54EA"/><stop offset="1" stop-color="#2F62EA"/></linearGradient>\n'
        '  <linearGradient id="fd" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#3D35B5"/><stop offset="1" stop-color="#2A2C99"/></linearGradient>\n'
    )
    inset = 50
    frame = (f'    <rect x="{CX - STAMP / 2 + inset}" y="{CY - STAMP / 2 + inset}" width="{STAMP - 2 * inset}" height="{STAMP - 2 * inset}" '
             f'rx="14" fill="none" stroke="#8A8FB8" stroke-opacity="0.28" stroke-width="5"/>\n')
    jw = 11  # 圆角：同色描边 + round 连接
    face = lambda pid, points: (f'    <polygon points="{pts(points)}" fill="url(#{pid})" stroke="url(#{pid})" '
                                f'stroke-width="{jw}" stroke-linejoin="round"/>\n')
    lines = ""
    for ax, ay, bx, by, wd in [(-8, 71, 18, 61, 6.5), (2, 85, 24, 76, 5.2), (12, 96, 28, 90, 4)]:
        lines += (f'    <line x1="{ox + ax * s:.1f}" y1="{oy + ay * s:.1f}" x2="{ox + bx * s:.1f}" y2="{oy + by * s:.1f}" '
                  f'stroke="#A9A6FF" stroke-opacity="0.85" stroke-width="{wd * s / 2.6:.1f}" stroke-linecap="round"/>\n')
    body = (f'  <g transform="rotate({TILT} {CX} {CY})">\n' + frame + lines
            + face("lo", [(92, 8), (42, 57), (56, 92)])
            + face("up", [(92, 8), (10, 44), (42, 57)])
            + face("fd", [(42, 57), (56, 92), (46, 72)])
            + "  </g>\n")
    return svg(body, defs)


def postmark_svg():
    """邮戳：右上一对同心圆环 + 右下三道波浪消印线（大部分被邮票遮住，露出一角）。"""
    out = ['  <circle cx="780" cy="262" r="124" fill="none" stroke="#FFFFFF" stroke-width="18"/>\n',
           '  <circle cx="780" cy="262" r="80" fill="none" stroke="#FFFFFF" stroke-width="9" stroke-opacity="0.6"/>\n']
    for k in range(3):
        y = 640 + k * 50
        d = f"M600 {y}"
        for i in range(1, 9):
            d += f" Q{600 + i * 40 - 20} {y + (-14 if i % 2 else 14)} {600 + i * 40} {y}"
        out.append(f'  <path d="{d}" fill="none" stroke="#FFFFFF" stroke-width="15" stroke-linecap="round"/>\n')
    return svg("".join(out))


def srgb(h, a=1.0):
    h = h.lstrip("#")
    r, g, b = (int(h[i:i + 2], 16) / 255 for i in (0, 2, 4))
    return f"srgb:{r:.5f},{g:.5f},{b:.5f},{a:.5f}"


def write_bundle(root="AppIcon.icon"):
    shutil.rmtree(root, ignore_errors=True)
    os.makedirs(os.path.join(root, "Assets"))
    for name, content in (("plane.svg", plane_svg()), ("stamp.svg", stamp_svg()), ("postmark.svg", postmark_svg())):
        with open(os.path.join(root, "Assets", name), "w") as f:
            f.write(content)
    doc = {
        # 底色：浅色用靛紫蓝，深色换成近黑；着色/透明外观由系统按图层亮度生成
        "fill": {"automatic-gradient": srgb("#5B5CF2")},
        "fill-specializations": [{"appearance": "dark", "value": {"automatic-gradient": srgb("#16171F")}}],
        "groups": [{
            "layers": [  # 自上而下
                {"name": "plane", "image-name": "plane.svg", "glass": False},
                {"name": "stamp", "image-name": "stamp.svg", "glass": True,
                 "fill-specializations": [{"appearance": "dark", "value": {"solid": srgb("#2C3045")}}]},
                {"name": "postmark", "image-name": "postmark.svg", "glass": False, "opacity": 0.32},
            ],
            "shadow": {"kind": "neutral", "opacity": 0.5},
            "translucency": {"enabled": False, "value": 0.5},
        }],
        "supported-platforms": {"squares": "shared"},
    }
    with open(os.path.join(root, "icon.json"), "w") as f:
        json.dump(doc, f, indent=2, ensure_ascii=False)
        f.write("\n")


def export(rendition, out, size=1024):
    subprocess.run([ICTOOL, "AppIcon.icon", "--export-image", "--output-file", out, "--platform", "macOS",
                    "--rendition", rendition, "--width", str(size), "--height", str(size), "--scale", "1",
                    "--tint-color", "0.60", "--tint-strength", "0.80"],
                   check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def main():
    write_bundle()
    if not os.path.exists(ICTOOL):
        sys.exit("未找到 ictool（需要安装 Xcode 26+），已只生成 AppIcon.icon")
    os.makedirs("docs/icon", exist_ok=True)
    versions = {"light": "Default", "dark": "Dark", "tinted": "TintedDark", "clear": "ClearLight"}
    for name, rendition in versions.items():
        export(rendition, f"docs/icon/AppIcon-{name}.png")
    shutil.copyfile("docs/icon/AppIcon-light.png", "resources/icon.png")
    # 应用内（空状态、设置-关于）用的小图：256px，浅色 / 深色各一张
    export("Default", "internal/gui/icons/logo.png", 256)
    export("Dark", "internal/gui/icons/logo-dark.png", 256)
    try:  # 压缩 PNG + 对比图（需要 Pillow，没有就跳过）
        from PIL import Image
        for f in ("internal/gui/icons/logo.png", "internal/gui/icons/logo-dark.png", "resources/icon.png"):
            Image.open(f).save(f, optimize=True)
        sheet = Image.new("RGBA", (4 * 300 + 20, 300 + 20 + 120), (232, 234, 240, 255))
        for i, name in enumerate(versions):
            im = Image.open(f"docs/icon/AppIcon-{name}.png").convert("RGBA")
            sheet.alpha_composite(im.resize((280, 280), Image.LANCZOS), (20 + i * 300, 10))
            x = 30 + i * 300
            for s in (64, 48, 32, 16):
                sheet.alpha_composite(im.resize((s, s), Image.LANCZOS), (x, 310 + (64 - s)))
                x += s + 12
        sheet.save("docs/icon/preview.png", optimize=True)
    except ImportError:
        pass
    print("已生成 AppIcon.icon、resources/icon.png、docs/icon/*")


if __name__ == "__main__":
    main()

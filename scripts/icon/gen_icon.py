#!/usr/bin/env python3
"""生成 MiniPost 应用图标的全部版本（需要 macOS + Xcode 自带的 Icon Composer / ictool）。

设计概念：把“响应时间”瀑布图做成图标。
  五条粗圆角色块一级一级错开，后一条从前一条的右端开始，合起来正好是一次请求的总耗时：
  准备 → 连接 → TLS → 等待首字节（最长，暖橙色突出）→ 下载。
  只有一个图形、两种颜色（白 + 橙），在 16px 到 1024px 都能一眼认出。

产物（在仓库根目录运行）：
  AppIcon.icon/        Icon Composer 图层图标：浅色 / 深色 / 着色 / 透明 由系统自动切换
  resources/icon.png   浅色版 1024 位图（Windows / Linux / 旧系统的回退）
  docs/icon/*.png      各版本 1024 预览：light / dark / tinted / clear + 对比图

用法：python3 scripts/icon/gen_icon.py
"""
import json
import os
import shutil
import subprocess
import sys

ICTOOL = "/Applications/Xcode.app/Contents/Applications/Icon Composer.app/Contents/Executables/ictool"

# ---------- 几何（1024 画布，Icon Composer 的画布就是整个图标） ----------
BAR_H, BAR_GAP, BAR_R = 118, 24, 34
SPAN_X = 162
# 五个阶段的宽度，之和 = 总宽；第 4 条（等待首字节）最长
PHASES = [110, 122, 142, 240, 86]
SPAN_W = sum(PHASES)
ROWS_H = 5 * BAR_H + 4 * BAR_GAP
TOP = (1024 - ROWS_H) / 2
ACCENT = "#FF8A57"                       # 等待首字节
ALPHAS = [0.55, 0.72, 0.88, 1.0, 1.0]    # 白色色块由浅到深，强调“越来越接近结果”


def svg(body):
    return ('<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">\n'
            + body + "</svg>\n")


def bars_svg():
    """五条顺序相接的色块：后一条的起点 = 前一条的终点。"""
    out, x = [], SPAN_X
    for i, w in enumerate(PHASES):
        y = TOP + i * (BAR_H + BAR_GAP)
        col = ACCENT if i == 3 else "#FFFFFF"
        out.append(f'  <rect x="{x}" y="{y:g}" width="{w}" height="{BAR_H}" rx="{BAR_R}" '
                   f'fill="{col}" fill-opacity="{ALPHAS[i]}"/>\n')
        x += w
    return svg("".join(out))


def srgb(h, a=1.0):
    h = h.lstrip("#")
    r, g, b = (int(h[i:i + 2], 16) / 255 for i in (0, 2, 4))
    return f"srgb:{r:.5f},{g:.5f},{b:.5f},{a:.5f}"


def write_bundle(root="AppIcon.icon"):
    shutil.rmtree(root, ignore_errors=True)
    os.makedirs(os.path.join(root, "Assets"))
    with open(os.path.join(root, "Assets", "waterfall.svg"), "w") as f:
        f.write(bars_svg())
    doc = {
        # 底色：浅色用品牌蓝，深色换成近黑；着色/透明外观由系统按图层亮度生成
        "fill": {"automatic-gradient": srgb("#2F6FF0")},
        "fill-specializations": [{"appearance": "dark", "value": {"automatic-gradient": srgb("#15171E")}}],
        "groups": [{
            "layers": [{"name": "waterfall", "image-name": "waterfall.svg", "glass": True}],
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
    try:  # 对比图（需要 Pillow，没有就跳过）
        from PIL import Image
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

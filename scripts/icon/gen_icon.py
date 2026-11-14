#!/usr/bin/env python3
"""生成 MiniPost 应用图标 resources/icon.png（1024x1024）。

设计概念：Mini + Post = “迷你邮票”。
  - 靛紫到电蓝的渐变圆角方块作底，背后露出半枚邮戳（圆环 + 波浪消印线）；
  - 一枚带锯齿孔的白色邮票微微倾斜，邮票中央是一架双色纸飞机（发送请求）；
  - 纸飞机身后拖着三道圆头短线，象征请求飞出去的速度感。

依赖：Pillow、numpy。用法：python3 scripts/icon/gen_icon.py [输出路径]
内部以 3 倍超采样绘制，最后缩回 1024 以获得平滑边缘。
"""
import math
import sys

import numpy as np
from PIL import Image, ImageChops, ImageDraw, ImageFilter

OUT = sys.argv[1] if len(sys.argv) > 1 else "resources/icon.png"
SIZE = 1024
SS = 3  # 超采样倍数
W = SIZE * SS


def px(v):
    """把 1024 画布上的长度换成超采样画布上的像素。"""
    return v * SS


def hex_rgb(h):
    h = h.lstrip("#")
    return tuple(int(h[i : i + 2], 16) for i in (0, 2, 4))


def gradient(size, c0, c1, angle_deg=90.0):
    """线性渐变图。angle=90 表示从上到下；0 表示从左到右。"""
    w, h = size
    a = math.radians(angle_deg)
    dx, dy = math.cos(a), math.sin(a)
    ys, xs = np.mgrid[0:h, 0:w].astype(np.float32)
    t = ((xs - w / 2) * dx + (ys - h / 2) * dy) / (abs(w * dx) + abs(h * dy)) + 0.5
    t = np.clip(t, 0, 1)[..., None]
    a0 = np.array(hex_rgb(c0) if isinstance(c0, str) else c0, np.float32)
    a1 = np.array(hex_rgb(c1) if isinstance(c1, str) else c1, np.float32)
    img = (a0 * (1 - t) + a1 * t).astype(np.uint8)
    return Image.fromarray(img, "RGB").convert("RGBA")


def squircle_mask(size, margin, n=5.0):
    """苹果风格的超椭圆（squircle）蒙版。"""
    m = Image.new("L", (size, size), 0)
    r = (size - 2 * margin) / 2
    c = size / 2
    pts = []
    for i in range(720):
        th = i / 720 * 2 * math.pi
        ct, st = math.cos(th), math.sin(th)
        x = c + r * math.copysign(abs(ct) ** (2 / n), ct)
        y = c + r * math.copysign(abs(st) ** (2 / n), st)
        pts.append((x, y))
    ImageDraw.Draw(m).polygon(pts, fill=255)
    return m


def rounded_poly_mask(size, pts, radius):
    """带圆角的多边形蒙版：填充 + 顶点画圆 + 边描粗线。"""
    m = Image.new("L", (size, size), 0)
    d = ImageDraw.Draw(m)
    d.polygon(pts, fill=255)
    for i, p in enumerate(pts):
        q = pts[(i + 1) % len(pts)]
        d.line([p, q], fill=255, width=int(radius * 2))
        d.ellipse([p[0] - radius, p[1] - radius, p[0] + radius, p[1] + radius], fill=255)
    return m


def paste_masked(base, layer, mask):
    base.alpha_composite(Image.composite(layer, Image.new("RGBA", layer.size, (0, 0, 0, 0)), mask))


def shadow(mask, blur, offset, color, opacity):
    """由蒙版生成柔和投影图层。"""
    sh = Image.new("RGBA", mask.size, color + (0,))
    a = mask.point(lambda v: int(v * opacity))
    a = ImageChops.offset(a, offset[0], offset[1]).filter(ImageFilter.GaussianBlur(blur))
    sh.putalpha(a)
    return sh


# ---------- 1. 底板：靛紫 → 电蓝渐变 + 柔光 ----------
canvas = Image.new("RGBA", (W, W), (0, 0, 0, 0))
margin = px(100)  # 遵循 macOS 图标网格：824 的内容区，四周留 100
sq = squircle_mask(W, margin)

bg = gradient((W, W), "#8B5CF6", "#2563EB", 62)
glow = Image.new("RGBA", (W, W), (0, 0, 0, 0))
gd = ImageDraw.Draw(glow)
gd.ellipse([px(120), px(40), px(700), px(560)], fill=(255, 255, 255, 70))
glow = glow.filter(ImageFilter.GaussianBlur(px(110)))
bg.alpha_composite(glow)
# 底部压暗，增加层次
dark = gradient((W, W), (0, 0, 0), (0, 0, 0), 90)
dark.putalpha(Image.fromarray((np.linspace(0, 70, W, dtype=np.uint8)[:, None] * np.ones((1, W), np.uint8))))
bg.alpha_composite(dark)

canvas.alpha_composite(shadow(sq, px(22), (0, px(14)), (30, 20, 90), 0.30))
paste_masked(canvas, bg, sq)

# 用于把后续元素裁在底板内的蒙版
clip = sq

# ---------- 2. 背后的邮戳：圆环 + 三道波浪线 ----------
postmark = Image.new("RGBA", (W, W), (0, 0, 0, 0))
pd = ImageDraw.Draw(postmark)
cx, cy, rr = px(742), px(300), px(118)
pd.ellipse([cx - rr, cy - rr, cx + rr, cy + rr], outline=(255, 255, 255, 110), width=int(px(16)))
rr2 = px(78)
pd.ellipse([cx - rr2, cy - rr2, cx + rr2, cy + rr2], outline=(255, 255, 255, 70), width=int(px(8)))
for k in range(3):
    y0 = px(560 + k * 46)
    pts = []
    for i in range(0, 241):
        x = px(560) + px(i) * 1.4
        y = y0 + math.sin(i / 240 * 3.5 * 2 * math.pi + k) * px(11)
        pts.append((x, y))
    pd.line(pts, fill=(255, 255, 255, 95), width=int(px(14)), joint="curve")
    for p in (pts[0], pts[-1]):
        pd.ellipse([p[0] - px(7), p[1] - px(7), p[0] + px(7), p[1] + px(7)], fill=(255, 255, 255, 95))
pm_alpha = ImageChops.multiply(postmark.getchannel("A"), clip)
postmark.putalpha(pm_alpha)
canvas.alpha_composite(postmark)

# ---------- 3. 邮票：白色纸张 + 锯齿孔，微微倾斜 ----------
stamp_size = px(500)
pad = px(60)  # 给旋转留余量
st_w = int(stamp_size + pad * 2)
stamp = Image.new("L", (st_w, st_w), 0)
sd = ImageDraw.Draw(stamp)
x0 = y0 = pad
x1 = y1 = pad + stamp_size
sd.rounded_rectangle([x0, y0, x1, y1], radius=px(18), fill=255)
# 锯齿孔：沿四边等距挖半圆
holes = 11
hr = px(15)
step = stamp_size / holes
for i in range(holes + 1):
    t = i * step
    for (hx, hy) in ((x0 + t, y0), (x0 + t, y1), (x0, y0 + t), (x1, y0 + t)):
        sd.ellipse([hx - hr, hy - hr, hx + hr, hy + hr], fill=0)

paper = gradient((st_w, st_w), "#FFFFFF", "#E8EDFF", 70)
# 邮票内框：一圈细淡色描边，增加“票面”感
inner = Image.new("L", (st_w, st_w), 0)
idr = ImageDraw.Draw(inner)
inset = px(44)
idr.rounded_rectangle([x0 + inset, y0 + inset, x1 - inset, y1 - inset], radius=px(14), outline=255, width=int(px(5)))
tint = Image.new("RGBA", (st_w, st_w), (124, 92, 246, 60))
paste_masked(paper, tint, inner)

# ---------- 4. 纸飞机（画在邮票票面上） ----------
def plane_pts(pts, ox, oy, s):
    return [(ox + x * s, oy + y * s) for x, y in pts]

cxs = (x0 + x1) / 2
cys = (y0 + y1) / 2
s = px(3.3)  # 100 单位坐标 → 像素
ox = cxs - 50 * s - px(10)
oy = cys - 50 * s + px(2)
upper = plane_pts([(92, 8), (10, 44), (42, 57)], ox, oy, s)
lower = plane_pts([(92, 8), (42, 57), (56, 92)], ox, oy, s)
fold = plane_pts([(42, 57), (56, 92), (46, 72)], ox, oy, s)
rad = px(6)

m_upper = rounded_poly_mask(st_w, upper, rad)
m_lower = rounded_poly_mask(st_w, lower, rad)
m_fold = rounded_poly_mask(st_w, fold, rad * 0.8)
paste_masked(paper, gradient((st_w, st_w), "#A78BFA", "#6366F1", 35), m_upper)
paste_masked(paper, gradient((st_w, st_w), "#4F46E5", "#2563EB", 80), m_lower)
paste_masked(paper, gradient((st_w, st_w), "#3730A3", "#312E81", 90), m_fold)

# 速度线：飞机身后三道圆头短线
for (ax, ay, bx, by, wd) in [(-12, 69, 14, 59, 7), (-2, 83, 22, 74, 5.5), (10, 94, 28, 88, 4)]:
    a = (ox + ax * s, oy + ay * s)
    b = (ox + bx * s, oy + by * s)
    lm = Image.new("L", (st_w, st_w), 0)
    ld = ImageDraw.Draw(lm)
    ld.line([a, b], fill=255, width=int(wd * s))
    for p in (a, b):
        r = wd * s / 2
        ld.ellipse([p[0] - r, p[1] - r, p[0] + r, p[1] + r], fill=255)
    paste_masked(paper, gradient((st_w, st_w), "#C4B5FD", "#818CF8", 0), lm)

stamp_rgba = Image.new("RGBA", (st_w, st_w), (0, 0, 0, 0))
paste_masked(stamp_rgba, paper, stamp)
stamp_mask = stamp
angle = -7
stamp_rgba = stamp_rgba.rotate(angle, resample=Image.BICUBIC, center=(st_w / 2, st_w / 2))
stamp_mask = stamp_mask.rotate(angle, resample=Image.BICUBIC, center=(st_w / 2, st_w / 2))

# 放到画布中央偏左下一点，给右上的邮戳让位
pos = (int(W / 2 - st_w / 2 - px(22)), int(W / 2 - st_w / 2 + px(14)))
layer = Image.new("RGBA", (W, W), (0, 0, 0, 0))
layer.paste(stamp_rgba, pos, stamp_rgba)
mask_full = Image.new("L", (W, W), 0)
mask_full.paste(stamp_mask, pos)
canvas.alpha_composite(shadow(mask_full, px(26), (px(6), px(22)), (25, 15, 85), 0.55))
canvas.alpha_composite(layer)

# ---------- 5. 底板高光边 ----------
rim = ImageChops.subtract(sq, sq.filter(ImageFilter.MinFilter(int(px(5)) | 1)))
rim_layer = Image.new("RGBA", (W, W), (255, 255, 255, 0))
top_fade = Image.fromarray((np.linspace(110, 0, W, dtype=np.uint8)[:, None] * np.ones((1, W), np.uint8)))
rim_layer.putalpha(ImageChops.multiply(rim, top_fade))
canvas.alpha_composite(rim_layer)

out = canvas.resize((SIZE, SIZE), Image.LANCZOS)
out.save(OUT, optimize=True)
print("已生成", OUT, out.size)

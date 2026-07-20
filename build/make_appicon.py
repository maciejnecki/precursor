#!/usr/bin/env python3
"""Render the application icon: an orange dancing stick figure on a slate
background. The geometry is a faithful copy of the design SVG (viewBox 0 0 1024
1024, 52px round strokes, head circle at 472,230). The slate fills every pixel
edge to edge with no transparency and no rounded corners, so macOS can mask the
solid square into its own icon shape without a light backing plate showing
through. Uses only the Python standard library. The artwork is drawn at twice
the target resolution and box-downsampled so the limbs read with smooth edges."""

import math
import struct
import zlib

# Target icon size in pixels and the supersampling factor used for antialiasing.
SIZE = 1024
SUPERSAMPLE = 2
CANVAS = SIZE * SUPERSAMPLE

# Palette copied from the design: slate background and the SVG's orange, which is
# hsl(24.6, 100%, 53.1%) resolved to sRGB.
SLATE = (51, 65, 85)
ORANGE = (255, 114, 16)

# Stroke radius: the SVG draws limbs at stroke-width 52, so the round strokes
# have a radius of 26 in 1024 space.
STROKE_RADIUS = 26


def distance_to_segment(point_x, point_y, start_x, start_y, end_x, end_y):
    """Return the shortest distance from a point to a line segment."""
    segment_x = end_x - start_x
    segment_y = end_y - start_y
    length_squared = segment_x * segment_x + segment_y * segment_y
    if length_squared == 0:
        return math.hypot(point_x - start_x, point_y - start_y)
    projection = ((point_x - start_x) * segment_x + (point_y - start_y) * segment_y) / length_squared
    projection = max(0.0, min(1.0, projection))
    closest_x = start_x + projection * segment_x
    closest_y = start_y + projection * segment_y
    return math.hypot(point_x - closest_x, point_y - closest_y)


def fill_capsule(pixels, start, end, radius):
    """Paint a rounded-cap thick line (capsule) between two points."""
    min_x = int(max(0, math.floor(min(start[0], end[0]) - radius)))
    max_x = int(min(CANVAS - 1, math.ceil(max(start[0], end[0]) + radius)))
    min_y = int(max(0, math.floor(min(start[1], end[1]) - radius)))
    max_y = int(min(CANVAS - 1, math.ceil(max(start[1], end[1]) + radius)))
    for pixel_y in range(min_y, max_y + 1):
        for pixel_x in range(min_x, max_x + 1):
            if distance_to_segment(pixel_x, pixel_y, start[0], start[1], end[0], end[1]) <= radius:
                pixels[pixel_y * CANVAS + pixel_x] = ORANGE


def fill_circle(pixels, centre, radius):
    """Paint a filled circle."""
    min_x = int(max(0, math.floor(centre[0] - radius)))
    max_x = int(min(CANVAS - 1, math.ceil(centre[0] + radius)))
    min_y = int(max(0, math.floor(centre[1] - radius)))
    max_y = int(min(CANVAS - 1, math.ceil(centre[1] + radius)))
    for pixel_y in range(min_y, max_y + 1):
        for pixel_x in range(min_x, max_x + 1):
            if math.hypot(pixel_x - centre[0], pixel_y - centre[1]) <= radius:
                pixels[pixel_y * CANVAS + pixel_x] = ORANGE


def scaled(point):
    """Scale a coordinate from 1024 space into the supersampled canvas."""
    return (point[0] * SUPERSAMPLE, point[1] * SUPERSAMPLE)


def stroke(radius):
    """Scale a stroke radius into the supersampled canvas."""
    return radius * SUPERSAMPLE


def draw_polyline(pixels, points, radius):
    """Stroke a polyline as round-capped, round-joined segments, matching the
    SVG's stroke-linecap and stroke-linejoin of round."""
    for index in range(len(points) - 1):
        fill_capsule(pixels, scaled(points[index]), scaled(points[index + 1]), stroke(radius))
    # Circles at the interior joints round off the corners between segments.
    for joint in points[1:-1]:
        fill_circle(pixels, scaled(joint), stroke(radius))


def downsample(pixels):
    """Box-downsample the supersampled canvas to the target size."""
    output = bytearray(SIZE * SIZE * 3)
    block = SUPERSAMPLE * SUPERSAMPLE
    for target_y in range(SIZE):
        for target_x in range(SIZE):
            red = green = blue = 0
            for offset_y in range(SUPERSAMPLE):
                source_y = target_y * SUPERSAMPLE + offset_y
                for offset_x in range(SUPERSAMPLE):
                    source_x = target_x * SUPERSAMPLE + offset_x
                    colour = pixels[source_y * CANVAS + source_x]
                    red += colour[0]
                    green += colour[1]
                    blue += colour[2]
            index = (target_y * SIZE + target_x) * 3
            output[index] = red // block
            output[index + 1] = green // block
            output[index + 2] = blue // block
    return output


def encode_png(rgb_bytes):
    """Encode a flat RGB byte buffer as a PNG image."""
    raw = bytearray()
    for row in range(SIZE):
        raw.append(0)
        start = row * SIZE * 3
        raw.extend(rgb_bytes[start:start + SIZE * 3])

    def chunk(tag, data):
        return (
            struct.pack(">I", len(data))
            + tag
            + data
            + struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF)
        )

    header = struct.pack(">IIBBBBB", SIZE, SIZE, 8, 2, 0, 0, 0)
    return (
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", header)
        + chunk(b"IDAT", zlib.compress(bytes(raw), 9))
        + chunk(b"IEND", b"")
    )


def main():
    # Slate fills the whole canvas so the icon is opaque edge to edge.
    pixels = [SLATE] * (CANVAS * CANVAS)

    # Torso.
    draw_polyline(pixels, [(474, 310), (556, 600)], STROKE_RADIUS)

    # Arms: left raised up, right thrown out — a dancing flourish.
    draw_polyline(pixels, [(480, 412), (322, 322), (250, 196)], STROKE_RADIUS)
    draw_polyline(pixels, [(480, 412), (648, 430), (724, 300)], STROKE_RADIUS)

    # Legs: left kicked out, right bent — mid step.
    draw_polyline(pixels, [(556, 600), (452, 762), (372, 880)], STROKE_RADIUS)
    draw_polyline(pixels, [(556, 600), (684, 742), (640, 902)], STROKE_RADIUS)

    # Head last so it sits cleanly over the torso and arms.
    fill_circle(pixels, scaled((472, 230)), stroke(102))

    with open("build/appicon.png", "wb") as handle:
        handle.write(encode_png(downsample(pixels)))


if __name__ == "__main__":
    main()

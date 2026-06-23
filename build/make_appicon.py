#!/usr/bin/env python3
"""Render the application icon: an orange dancing stick figure on a slate
background. Uses only the Python standard library. The figure is drawn at twice
the target resolution and box-downsampled so the limbs read with smooth edges."""

import math
import struct
import zlib

# Target icon size in pixels and the supersampling factor used for antialiasing.
SIZE = 1024
SUPERSAMPLE = 2
CANVAS = SIZE * SUPERSAMPLE

# Palette: slate background and orange figure.
SLATE = (51, 65, 85)
ORANGE = (249, 115, 22)


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
    pixels = [SLATE] * (CANVAS * CANVAS)

    # Joints of the dancing pose, in 1024 space.
    head_centre = (440, 250)
    head_radius = 86
    neck = (468, 350)
    shoulder = (480, 388)
    hip = (556, 600)
    limb_radius = 30

    # Torso.
    fill_capsule(pixels, scaled(neck), scaled(hip), limb_radius * SUPERSAMPLE)

    # Arms: left raised up, right thrown out — a dancing flourish.
    fill_capsule(pixels, scaled(shoulder), scaled((322, 322)), limb_radius * SUPERSAMPLE)
    fill_capsule(pixels, scaled((322, 322)), scaled((250, 196)), limb_radius * SUPERSAMPLE)
    fill_capsule(pixels, scaled(shoulder), scaled((648, 430)), limb_radius * SUPERSAMPLE)
    fill_capsule(pixels, scaled((648, 430)), scaled((724, 300)), limb_radius * SUPERSAMPLE)

    # Legs: left kicked out, right bent — mid step.
    fill_capsule(pixels, scaled(hip), scaled((452, 762)), limb_radius * SUPERSAMPLE)
    fill_capsule(pixels, scaled((452, 762)), scaled((372, 880)), limb_radius * SUPERSAMPLE)
    fill_capsule(pixels, scaled(hip), scaled((684, 742)), limb_radius * SUPERSAMPLE)
    fill_capsule(pixels, scaled((684, 742)), scaled((640, 902)), limb_radius * SUPERSAMPLE)

    # Smooth the joints so the limbs connect without notches.
    for joint in [shoulder, hip, (322, 322), (648, 430), (452, 762), (684, 742)]:
        fill_circle(pixels, scaled(joint), limb_radius * SUPERSAMPLE)

    # Head last so it sits cleanly over the neck.
    fill_circle(pixels, scaled(head_centre), head_radius * SUPERSAMPLE)

    with open("build/appicon.png", "wb") as handle:
        handle.write(encode_png(downsample(pixels)))


if __name__ == "__main__":
    main()

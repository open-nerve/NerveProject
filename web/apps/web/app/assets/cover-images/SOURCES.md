# Sources of the default project cover

Copyright (c) 2026-present OpenNerve
SPDX-License-Identifier: AGPL-3.0-only

The picture in this directory is an original Nerve drawing: an abstract pattern on a 1600 × 900 canvas, with no
photograph, person, name or product in it. The SVG is the source, and carries this notice in an XML comment; the
WebP file cannot, so it is listed here.

`helpers/cover-image.helper.ts` imports the WebP file: it is the cover a project shows while it has none of its own
(M3 design 3.2). `image_1.webp` is its SVG rendered at the SVG's size, 1920 × 1080, and encoded as lossy WebP at
quality 0.9 (in Chromium, with `canvas.toBlob(callback, "image/webp", 0.9)`). To change it, edit the SVG and render
the WebP again at the same size; the code reads the file by name only. The 28 other preset covers went with the
cover picker (M3/P8b): M5 brings covers back with uploads.

The app crops a cover to a wide band (the project card, the project settings) and writes white text and icons on it,
so the drawing keeps its detail across the middle of the canvas and stays mid to dark.

| Files                         | Drawing                     |
| ----------------------------- | --------------------------- |
| `image_1.svg`, `image_1.webp` | Soft fields of colour, teal |

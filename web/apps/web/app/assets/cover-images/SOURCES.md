# Sources of the preset project covers

Copyright (c) 2026-present OpenNerve
SPDX-License-Identifier: AGPL-3.0-only

Every picture in this directory is an original Nerve drawing: an abstract pattern on a 1600 × 900 canvas, with no
photograph, person, name or product in it. The SVGs are the sources, and carry this notice in an XML comment; the
WebP files cannot, so they are listed here.

`helpers/cover-image.helper.ts` imports the WebP files. When a project or a profile takes a preset cover, the app
uploads a copy of it, and the upload accepts JPEG, PNG and WebP, not SVG. Each `image_<n>.webp` is its SVG
rendered at the SVG's size, 1920 × 1080, and encoded as lossy WebP at quality 0.9 (in Chromium, with
`canvas.toBlob(callback, "image/webp", 0.9)`). To change a cover, edit its SVG and render the WebP again at the
same size; the code reads the files by name only.

The app crops a cover to a wide band (the project card, the project and profile settings) and writes white text and
icons on it, so every drawing keeps its detail across the middle of the canvas and stays mid to dark.

| Files                           | Drawing                         |
| ------------------------------- | ------------------------------- |
| `image_1.svg`, `image_1.webp`   | Soft fields of colour, teal     |
| `image_2.svg`, `image_2.webp`   | Layered waves, indigo           |
| `image_3.svg`, `image_3.webp`   | Concentric discs, sunset        |
| `image_4.svg`, `image_4.webp`   | Stepped diagonal bands, ocean   |
| `image_5.svg`, `image_5.webp`   | Low-poly facets, forest         |
| `image_6.svg`, `image_6.webp`   | Halftone dots, plum             |
| `image_7.svg`, `image_7.webp`   | Contour lines, slate            |
| `image_8.svg`, `image_8.webp`   | Soft fields of colour, rose     |
| `image_9.svg`, `image_9.webp`   | Layered waves, ocean            |
| `image_10.svg`, `image_10.webp` | Concentric discs, emerald       |
| `image_11.svg`, `image_11.webp` | Stepped diagonal bands, amber   |
| `image_12.svg`, `image_12.webp` | Low-poly facets, indigo         |
| `image_13.svg`, `image_13.webp` | Soft fields of colour, night    |
| `image_14.svg`, `image_14.webp` | Layered waves, terracotta       |
| `image_15.svg`, `image_15.webp` | Halftone dots, teal             |
| `image_16.svg`, `image_16.webp` | Contour lines, indigo           |
| `image_17.svg`, `image_17.webp` | Concentric discs, plum          |
| `image_18.svg`, `image_18.webp` | Stepped diagonal bands, cobalt  |
| `image_19.svg`, `image_19.webp` | Low-poly facets, ember          |
| `image_20.svg`, `image_20.webp` | Soft fields of colour, northern |
| `image_21.svg`, `image_21.webp` | Layered waves, forest           |
| `image_22.svg`, `image_22.webp` | Contour lines, emerald          |
| `image_23.svg`, `image_23.webp` | Halftone dots, sunset           |
| `image_24.svg`, `image_24.webp` | Stepped diagonal bands, slate   |
| `image_25.svg`, `image_25.webp` | Concentric discs, ocean         |
| `image_26.svg`, `image_26.webp` | Low-poly facets, teal           |
| `image_27.svg`, `image_27.webp` | Layered waves, plum             |
| `image_28.svg`, `image_28.webp` | Contour lines, amber            |
| `image_29.svg`, `image_29.webp` | Soft fields of colour, cobalt   |

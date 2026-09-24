# Sources of the Nerve brand files

Copyright (c) 2026-present OpenNerve
SPDX-License-Identifier: AGPL-3.0-only

Every file in this directory is an original Nerve drawing. The SVGs carry this notice in an XML comment; the
raster files cannot, so they are listed here.

| File                                                                             | Source                                                                                 |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `mark.svg`                                                                       | The mark, drawn by hand                                                                |
| `lockup.svg`, `lockup-on-dark.svg`                                               | The mark at half size and the wordmark "nerve", drawn by hand; dark and light ink      |
| `favicon-16x16.png`, `favicon-32x32.png`, `icon-180x180.png`, `icon-512x512.png` | `mark.svg`, rendered at that size on a transparent background                          |
| `favicon.ico`                                                                    | `mark.svg`, rendered at 16, 32 and 48 px                                               |
| `og-image.png`                                                                   | `lockup.svg` and the line "Open-source project management" in Inter, 1200×630 on white |

The code reads these files by name only. A new logo replaces the SVGs and renders the raster files again at the
same sizes; no code changes.

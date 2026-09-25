# Sources of the product tour's pictures

Copyright (c) 2026-present OpenNerve
SPDX-License-Identifier: AGPL-3.0-only

The four pictures are screenshots of the app's interface that came with the code, under its notice:

Copyright (c) 2023-present Plane Software, Inc. and contributors
SPDX-License-Identifier: AGPL-3.0-only

Every avatar that showed a photograph (of a person, of a cartoon character, or a count badge laid over a photo) is a
letter avatar now: a flat disc in one of six colours with a white initial (or the count) in Inter, the app's font,
painted over the old disc and clipped so that the discs drawn over it keep their pixels. Real people's names are
neutral ones, written in the old words' place in Inter at the size, weight, colour and baseline fitted to the old
words. Nothing else changed, apart from one more lossy WebP encode, at the quality whose file size is closest to
the original's. How each picture was measured and changed: `docs/v0/M1-frontend-trim/plans/closeout.md`, Task 15.

| File           | Changed                                                                                       |
| -------------- | --------------------------------------------------------------------------------------------- |
| `cycles.webp`  | 11 avatars; the cycle lead's name is now "vera"                                               |
| `issues.webp`  | 5 avatars                                                                                     |
| `views.webp`   | 5 avatars (a 5-px sliver of a dark avatar under the edge of the views panel is left as it is) |
| `modules.webp` | Nothing: it shows no avatar and no name                                                       |

The controls of features this milestone removed were painted out, each with the flat background it sat on (a solid
colour sampled beside it), then one more lossy WebP encode at the size-closest quality, as above. How each was
measured: `docs/v0/M1-frontend-trim/plans/closeout.md`, Task 20.

| File          | Removed                                                                                 |
| ------------- | --------------------------------------------------------------------------------------- |
| `cycles.webp` | the reporting button (left of Add Issue) and the Gantt layout icon in the work-item bar |
| `views.webp`  | the Gantt layout icon in the work-item bar                                              |
| `issues.webp` | the "Public" visibility pill next to the Issues breadcrumb                              |

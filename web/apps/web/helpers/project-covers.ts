/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ProjectCreate } from "@nerve/api-client";
import cover01 from "@/app/assets/project-covers/cover-01.svg?url";
import cover02 from "@/app/assets/project-covers/cover-02.svg?url";
import cover03 from "@/app/assets/project-covers/cover-03.svg?url";
import cover04 from "@/app/assets/project-covers/cover-04.svg?url";
import cover05 from "@/app/assets/project-covers/cover-05.svg?url";
import cover06 from "@/app/assets/project-covers/cover-06.svg?url";
import cover07 from "@/app/assets/project-covers/cover-07.svg?url";
import cover08 from "@/app/assets/project-covers/cover-08.svg?url";
import cover09 from "@/app/assets/project-covers/cover-09.svg?url";
import cover10 from "@/app/assets/project-covers/cover-10.svg?url";
import cover11 from "@/app/assets/project-covers/cover-11.svg?url";
import cover12 from "@/app/assets/project-covers/cover-12.svg?url";
import cover13 from "@/app/assets/project-covers/cover-13.svg?url";
import cover14 from "@/app/assets/project-covers/cover-14.svg?url";
import cover15 from "@/app/assets/project-covers/cover-15.svg?url";
import cover16 from "@/app/assets/project-covers/cover-16.svg?url";
import cover17 from "@/app/assets/project-covers/cover-17.svg?url";
import cover18 from "@/app/assets/project-covers/cover-18.svg?url";
import cover19 from "@/app/assets/project-covers/cover-19.svg?url";
import cover20 from "@/app/assets/project-covers/cover-20.svg?url";
import cover21 from "@/app/assets/project-covers/cover-21.svg?url";
import cover22 from "@/app/assets/project-covers/cover-22.svg?url";
import cover23 from "@/app/assets/project-covers/cover-23.svg?url";
import cover24 from "@/app/assets/project-covers/cover-24.svg?url";
import cover25 from "@/app/assets/project-covers/cover-25.svg?url";
import cover26 from "@/app/assets/project-covers/cover-26.svg?url";
import cover27 from "@/app/assets/project-covers/cover-27.svg?url";
import cover28 from "@/app/assets/project-covers/cover-28.svg?url";
import cover29 from "@/app/assets/project-covers/cover-29.svg?url";

/** Only owner API fields; an absent selection is {}, explicit removal has two nulls. */
export type ProjectCoverSelection = Pick<ProjectCreate, "cover_image_asset_id" | "cover_builtin_id">;
/** IDs are persisted; bundler URLs are used only when rendering. */
export const PROJECT_COVERS = [
  { id: "cover-01", url: cover01 },
  { id: "cover-02", url: cover02 },
  { id: "cover-03", url: cover03 },
  { id: "cover-04", url: cover04 },
  { id: "cover-05", url: cover05 },
  { id: "cover-06", url: cover06 },
  { id: "cover-07", url: cover07 },
  { id: "cover-08", url: cover08 },
  { id: "cover-09", url: cover09 },
  { id: "cover-10", url: cover10 },
  { id: "cover-11", url: cover11 },
  { id: "cover-12", url: cover12 },
  { id: "cover-13", url: cover13 },
  { id: "cover-14", url: cover14 },
  { id: "cover-15", url: cover15 },
  { id: "cover-16", url: cover16 },
  { id: "cover-17", url: cover17 },
  { id: "cover-18", url: cover18 },
  { id: "cover-19", url: cover19 },
  { id: "cover-20", url: cover20 },
  { id: "cover-21", url: cover21 },
  { id: "cover-22", url: cover22 },
  { id: "cover-23", url: cover23 },
  { id: "cover-24", url: cover24 },
  { id: "cover-25", url: cover25 },
  { id: "cover-26", url: cover26 },
  { id: "cover-27", url: cover27 },
  { id: "cover-28", url: cover28 },
  { id: "cover-29", url: cover29 },
] as const;

export function projectCoverURL(id: string | null | undefined): string | undefined {
  return PROJECT_COVERS.find((cover) => cover.id === id)?.url;
}

/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useFormContext } from "react-hook-form";
import type { ComponentProps } from "react";
import { ProjectCoverEditor } from "@/components/assets/project-cover-editor";
import type { ProjectCreationForm } from "./use-create-project";

/** The create form owns its fields; the shared editor never creates or updates a project. */
export function CreateProjectCover(props: Omit<ComponentProps<typeof ProjectCoverEditor>, "value" | "onChange">) {
  const { watch, setValue } = useFormContext<ProjectCreationForm>();
  return (
    <ProjectCoverEditor
      {...props}
      value={{ cover_image_asset_id: watch("cover_image_asset_id"), cover_builtin_id: watch("cover_builtin_id") }}
      onChange={(next) => {
        setValue("cover_image_asset_id", next.cover_image_asset_id, { shouldDirty: true });
        setValue("cover_builtin_id", next.cover_builtin_id, { shouldDirty: true });
      }}
    />
  );
}

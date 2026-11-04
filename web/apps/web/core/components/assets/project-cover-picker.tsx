/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { cn } from "@nerve/utils";
import { PROJECT_COVERS, type ProjectCoverSelection } from "@/helpers/project-covers";
import type { UploadSession } from "@/store/asset/upload-session";

type Props = {
  value: ProjectCoverSelection;
  upload: UploadSession;
  disabled?: boolean;
  onChange(value: ProjectCoverSelection): void;
};
/** Preset actions are explicit replacement choices; the parent owns uploading and saving. */
export const ProjectCoverPicker = observer(function ProjectCoverPicker({ value, upload, disabled, onChange }: Props) {
  const { t } = useTranslation();
  const locked = !!disabled || !upload.current;
  const select = (next: ProjectCoverSelection) => {
    if (locked || !upload.current) return;
    upload.cancel();
    if (upload.current) onChange(next);
  };
  return (
    <fieldset className="space-y-4">
      <legend className="text-13 font-medium">{t("project_cover.presets")}</legend>
      <div className="grid max-h-80 grid-cols-2 gap-3 overflow-y-auto p-1 sm:grid-cols-4">
        {PROJECT_COVERS.map((cover, index) => (
          <button
            key={cover.id}
            type="button"
            aria-label={`${t("project_cover.preset")} ${index + 1}`}
            aria-pressed={value.cover_builtin_id === cover.id}
            disabled={locked}
            className={cn(
              "aspect-video overflow-hidden rounded-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent-strong disabled:cursor-not-allowed disabled:opacity-50",
              value.cover_builtin_id === cover.id && "ring-offset-surface-1 ring-2 ring-accent-strong ring-offset-2"
            )}
            onClick={() => select({ cover_builtin_id: cover.id })}
          >
            <img src={cover.url} alt="" className="h-full w-full object-cover" />
          </button>
        ))}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="secondary"
          disabled={locked}
          onClick={() => {
            if (locked || !upload.current) return;
            const cover = PROJECT_COVERS[Math.floor(Math.random() * PROJECT_COVERS.length)];
            if (cover) select({ cover_builtin_id: cover.id });
          }}
        >
          {t("project_cover.random")}
        </Button>
        <Button
          type="button"
          variant="error-outline"
          disabled={locked}
          onClick={() => select({ cover_builtin_id: null, cover_image_asset_id: null })}
        >
          {t("project_cover.remove")}
        </Button>
      </div>
    </fieldset>
  );
});

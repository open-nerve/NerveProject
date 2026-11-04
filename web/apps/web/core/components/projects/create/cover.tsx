/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useState, type ReactNode } from "react";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { ImagePicker } from "@/components/assets/image-picker";
import { ProjectCoverPicker } from "@/components/assets/project-cover-picker";
import type { ProjectCoverSelection } from "@/helpers/project-covers";
import type { UploadSession } from "@/store/asset/upload-session";
import type { ProjectCreationForm } from "./use-create-project";

type Props = {
  upload: UploadSession;
  busyBinding: boolean;
  onBusyChange(busy: boolean): void;
  onClose(): void;
  currentImage: ReactNode;
};
/** Content inside the existing creation modal; choosing a cover does not create a project. */
export function CreateProjectCover({ upload, busyBinding, onBusyChange, onClose, currentImage }: Props) {
  const { t } = useTranslation();
  const { watch, setValue } = useFormContext<ProjectCreationForm>();
  const [mode, setMode] = useState<"presets" | "upload">("presets");
  const value = { cover_image_asset_id: watch("cover_image_asset_id"), cover_builtin_id: watch("cover_builtin_id") };
  const choose = (next: ProjectCoverSelection) => {
    if (!upload.current || busyBinding) return;
    setValue("cover_image_asset_id", next.cover_image_asset_id, { shouldDirty: true });
    setValue("cover_builtin_id", next.cover_builtin_id, { shouldDirty: true });
  };
  return (
    <section aria-label={t("image_upload.change_cover")} className="space-y-4 rounded-lg border border-subtle p-4">
      <div className="flex gap-2">
        <Button
          type="button"
          variant="secondary"
          aria-pressed={mode === "presets"}
          disabled={busyBinding}
          onClick={() => setMode("presets")}
        >
          {t("project_cover.presets")}
        </Button>
        <Button
          type="button"
          variant="secondary"
          aria-pressed={mode === "upload"}
          disabled={busyBinding}
          onClick={() => setMode("upload")}
        >
          {t("project_cover.upload")}
        </Button>
      </div>
      {mode === "presets" ? (
        <ProjectCoverPicker
          value={value}
          upload={upload}
          disabled={busyBinding}
          onChange={(next) => {
            choose(next);
            onBusyChange(false);
            onClose();
          }}
        />
      ) : (
        <ImagePicker
          upload={upload}
          currentImage={currentImage}
          busyBinding={busyBinding}
          onBusyChange={onBusyChange}
          onSubmit={async (asset) =>
            choose(asset ? { cover_image_asset_id: asset.id } : { cover_image_asset_id: null, cover_builtin_id: null })
          }
          onSaved={onClose}
          onCancel={() => {
            if (value.cover_image_asset_id) choose({});
            onBusyChange(false);
            onClose();
          }}
        />
      )}
    </section>
  );
}

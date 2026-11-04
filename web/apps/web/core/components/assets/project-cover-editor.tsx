/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useState, type ReactNode } from "react";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { ImagePicker } from "@/components/assets/image-picker";
import { ProjectCoverPicker } from "@/components/assets/project-cover-picker";
import type { ProjectCoverSelection } from "@/helpers/project-covers";
import type { UploadSession } from "@/store/asset/upload-session";

type Props = {
  upload: UploadSession;
  value: ProjectCoverSelection;
  onChange(value: ProjectCoverSelection): void;
  busyBinding: boolean;
  onBusyChange(busy: boolean): void;
  onClose(): void;
  currentImage: ReactNode;
};
/** Selection content shared by create and settings; the owner decides when to bind its chosen fields. */
export function ProjectCoverEditor({
  upload,
  value,
  onChange,
  busyBinding,
  onBusyChange,
  onClose,
  currentImage,
}: Props) {
  const { t } = useTranslation();
  const [mode, setMode] = useState<"presets" | "upload">("presets");
  const choose = (next: ProjectCoverSelection) => {
    if (!upload.current || busyBinding) return;
    onChange(next);
  };
  return (
    <section aria-label={t("image_upload.change_cover")} className="space-y-4 rounded-lg border border-subtle p-4">
      <div className="flex gap-2">
        <Button
          type="button"
          variant="secondary"
          aria-pressed={mode === "presets"}
          disabled={busyBinding}
          onClick={() => {
            if (!upload.current || busyBinding) return;
            if (mode === "upload") {
              upload.cancel();
              if (value.cover_image_asset_id) choose({});
              onBusyChange(false);
            }
            setMode("presets");
          }}
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

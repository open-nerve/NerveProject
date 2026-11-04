/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useTranslation } from "@nerve/i18n";
import { EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
import { observer } from "mobx-react";
import type { ReactNode } from "react";
import { ProjectCoverEditor } from "@/components/assets/project-cover-editor";
import type { useCoverDraft } from "./use-cover-draft";

type Props = { draft: ReturnType<typeof useCoverDraft>; busyBinding: boolean; currentImage: ReactNode };
export const CoverDraftModal = observer(function CoverDraftModal({ draft, busyBinding, currentImage }: Props) {
  return (
    <ModalCore
      isOpen={draft.open && !!draft.upload?.current}
      handleClose={draft.busy || busyBinding ? undefined : draft.dismiss}
      width={EModalWidth.XL}
    >
      <div className="space-y-4 p-6">
        <ModalTitle>{useTranslation().t("image_upload.change_cover")}</ModalTitle>
        {draft.upload && (
          <ProjectCoverEditor
            upload={draft.upload}
            value={draft.fields()}
            onChange={draft.choose}
            busyBinding={busyBinding}
            onBusyChange={draft.setBusy}
            onClose={draft.close}
            currentImage={currentImage}
          />
        )}
      </div>
    </ModalCore>
  );
});

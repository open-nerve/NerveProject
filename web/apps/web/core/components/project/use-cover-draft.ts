/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useEffect, useRef, useState } from "react";
import { projectCoverURL, type ProjectCoverSelection } from "@/helpers/project-covers";
import { useProfileUpload } from "@/hooks/use-profile-upload";
import type { UploadSession } from "@/store/asset/upload-session";

/** Unbound cover fields live with the settings form, never in the canonical ProjectStore. */
export function useCoverDraft(projectID: string, editable: boolean) {
  const [active, setActive] = useState(false);
  const [open, setOpen] = useState(false);
  const [working, setWorking] = useState(false);
  const busy = useRef(false);
  const choice = useRef<{ upload: UploadSession; fields: ProjectCoverSelection } | undefined>(undefined);
  const [, redraw] = useState(0);
  const upload = useProfileUpload(
    { purpose: "PROJECT_COVER", scope: { kind: "project", id: projectID } },
    active && editable
  );
  const reset = useCallback(() => {
    choice.current = undefined;
    busy.current = false;
    setWorking(false);
    setOpen(false);
    setActive(false);
  }, []);
  useEffect(() => {
    if (upload && !upload.current) reset();
  }, [upload, upload?.current, reset]);
  const fields = (): ProjectCoverSelection =>
    choice.current && choice.current.upload === upload && upload?.current ? choice.current.fields : {};
  const choose = (next: ProjectCoverSelection) => {
    if (!upload?.current) return;
    choice.current = { upload, fields: { ...next } };
    redraw((value) => value + 1);
  };
  const selected = fields();
  return {
    upload,
    open,
    fields,
    busy: working || !!upload?.busy,
    previewURL:
      projectCoverURL(selected.cover_builtin_id) ??
      (selected.cover_image_asset_id && selected.cover_image_asset_id === upload?.readyAsset?.id
        ? upload.previewURL
        : undefined),
    clearing: selected.cover_image_asset_id === null && selected.cover_builtin_id === null,
    begin: () => {
      if (editable) {
        setActive(true);
        setOpen(true);
      }
    },
    choose,
    close: () => setOpen(false),
    dismiss: () => {
      if (busy.current || upload?.busy) return;
      upload?.cancel();
      if (selected.cover_image_asset_id) choose({});
      setOpen(false);
    },
    setBusy: (value: boolean) => {
      busy.current = value;
      setWorking(value);
    },
    canSubmit: () =>
      !busy.current &&
      (!upload ||
        (upload.current &&
          !["selected", "preparing", "uploading", "confirming", "failed"].includes(upload.phase) &&
          (!fields().cover_image_asset_id || fields().cover_image_asset_id === upload.readyAsset?.id))),
    capture: () => {
      const captured = choice.current;
      return () => choice.current === captured && (!captured || captured.upload.current);
    },
    saved: reset,
  };
}

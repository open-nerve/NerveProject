/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useEffect, useId, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { observer } from "mobx-react";
import type { Asset } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { ImageSelectionError, type UploadSession } from "@/store/asset/upload-session";
import { UploadFailure } from "@/lib/asset-upload";
import { errorMessageKey } from "@/lib/error-messages";

export type ImagePickerProps = {
  upload: UploadSession;
  currentImage?: ReactNode;
  canRemove?: boolean;
  busyBinding?: boolean;
  onSubmit(asset: Asset | null): Promise<void>;
  onSaved?(): void;
  onCancel(): void;
  onBusyChange?(busy: boolean): void;
};
/** The parent owns the session lifetime and the owner write; this content may also live inside a project form. */
export const ImagePicker = observer(function ImagePicker(props: ImagePickerProps) {
  const { upload, busyBinding = false, onSubmit, onSaved, onCancel, onBusyChange } = props;
  const { t } = useTranslation();
  const id = useId();
  const input = useRef<HTMLInputElement>(null);
  const submitButton = useRef<HTMLButtonElement>(null);
  const errorElement = useRef<HTMLParagraphElement>(null);
  const life = useRef({ mounted: false, operation: 0 });
  const pending = useRef(false);
  const [working, setWorking] = useState(false);
  const [binding, setBinding] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [selection, setSelection] = useState(0);
  const busy = working || busyBinding || upload.busy;
  const disabled = busy || !upload.current;
  const error = !upload.current
    ? "errors.asset_scope_changed"
    : (failure ?? (upload.error ? message(upload.error) : undefined));
  useEffect(() => {
    const lifetime = life.current;
    lifetime.mounted = true;
    return () => {
      lifetime.mounted = false;
      ++lifetime.operation;
    };
  }, []);
  useLayoutEffect(() => {
    if (!upload.current) return;
    if (error) errorElement.current?.focus({ preventScroll: true });
    else if (!busy && upload.phase === "selected") submitButton.current?.focus({ preventScroll: true });
    else if (!busy && upload.phase === "cancelled") input.current?.focus({ preventScroll: true });
  }, [error, selection, upload, upload.phase, busy]);

  const rejectSelection = (key: string) => {
    upload.cancel();
    setFailure(key);
    setSelection((value) => value + 1);
  };
  const choose = (files: FileList | readonly File[]) => {
    if (disabled || pending.current || files.length === 0) return;
    if (files.length !== 1) {
      rejectSelection("image_upload.errors.multiple");
      return;
    }
    setFailure(undefined);
    upload.select(files[0]!);
    setSelection((value) => value + 1);
  };
  const perform = async (remove: boolean) => {
    if (disabled || pending.current || (remove && !props.canRemove)) return;
    pending.current = true;
    const revision = ++life.current.operation;
    const alive = () => life.current.mounted && life.current.operation === revision && upload.current;
    setWorking(true);
    setFailure(undefined);
    onBusyChange?.(true);
    try {
      let asset: Asset | null | undefined = remove ? null : upload.readyAsset;
      if (remove) upload.cancel();
      else if (!asset)
        asset = await (upload.recovery === "confirm"
          ? upload.retryComplete()
          : upload.recovery === "restart"
            ? upload.restart()
            : upload.start());
      if (!alive() || asset === undefined || (asset !== null && upload.readyAsset !== asset)) return;
      setBinding(true);
      await onSubmit(asset);
      if (alive() && (asset === null ? upload.readyAsset === undefined : upload.readyAsset === asset)) onSaved?.();
    } catch (reason) {
      if (alive()) setFailure(message(reason));
    } finally {
      if (alive()) {
        pending.current = false;
        setWorking(false);
        setBinding(false);
        onBusyChange?.(false);
      }
    }
  };
  const cancelUpload = () => {
    if (!upload.busy || binding || busyBinding) return;
    ++life.current.operation;
    pending.current = false;
    upload.cancel();
    setWorking(false);
    setBinding(false);
    setFailure(undefined);
    onBusyChange?.(false);
  };
  const submitLabel = upload.readyAsset
    ? "save"
    : upload.recovery === "confirm"
      ? "check_upload"
      : upload.recovery === "restart"
        ? "upload_again"
        : upload.recovery === "prepare"
          ? "retry"
          : "upload_save";
  return (
    <div className="space-y-4" aria-busy={busy}>
      <div
        aria-label={t("image_upload.drop")}
        className="relative grid min-h-52 place-items-center rounded-lg border-2 border-dashed border-subtle p-4"
        onDragOver={(event) => {
          event.preventDefault();
          if (event.dataTransfer) event.dataTransfer.dropEffect = disabled ? "none" : "copy";
        }}
        onDrop={(event) => {
          event.preventDefault();
          if (disabled || pending.current) return;
          const { files, items } = event.dataTransfer;
          if (files.length !== 1 || items.length > 1) {
            rejectSelection(
              files.length > 1 || items.length > 1 ? "image_upload.errors.multiple" : "image_upload.errors.not_file"
            );
            return;
          }
          if ([...items].some((item) => item.kind !== "file")) {
            rejectSelection("image_upload.errors.not_file");
            return;
          }
          if ([...items].some((item) => item.webkitGetAsEntry?.()?.isDirectory)) {
            rejectSelection("image_upload.errors.directory");
            return;
          }
          choose(files);
        }}
      >
        {upload.previewURL ? (
          <img
            src={upload.previewURL}
            alt={t("image_upload.preview")}
            className="max-h-72 max-w-full rounded-md object-contain"
          />
        ) : (
          (props.currentImage ?? <p className="text-13 text-secondary">{t("image_upload.drop")}</p>)
        )}
      </div>
      <div className="space-y-2">
        <label htmlFor={id} className="block text-13 font-medium">
          {t("image_upload.choose")}
        </label>
        <input
          ref={input}
          id={id}
          type="file"
          accept="image/jpeg,image/png,image/webp,image/gif"
          disabled={disabled}
          aria-describedby={`${id}-help${error ? ` ${id}-error` : ""}`}
          aria-invalid={!!error}
          className="block w-full rounded-md text-13 focus-visible:outline-2 focus-visible:outline-accent-strong"
          onChange={(event) => {
            if (event.target.files) choose(event.target.files);
            event.target.value = "";
          }}
        />
        <p id={`${id}-help`} className="text-12 text-secondary">
          {t("image_upload.formats")}
        </p>
      </div>
      <p aria-live="polite" aria-atomic="true" className="text-13 text-secondary">
        {t(`image_upload.phases.${binding || busyBinding ? "binding" : upload.phase}`)}
      </p>
      {error && (
        <p ref={errorElement} id={`${id}-error`} role="alert" tabIndex={-1} className="text-13 text-danger-primary">
          {t(error)}
        </p>
      )}
      <div className="flex flex-wrap items-center justify-between gap-3">
        {props.canRemove && (
          <Button
            type="button"
            variant="error-outline"
            disabled={disabled}
            onClick={() => {
              void perform(true);
            }}
          >
            {t("image_upload.remove")}
          </Button>
        )}
        <div className="ml-auto flex flex-wrap items-center gap-2">
          {upload.busy && (
            <Button type="button" variant="secondary" disabled={binding || busyBinding} onClick={cancelUpload}>
              {t("image_upload.cancel_upload")}
            </Button>
          )}
          <Button
            type="button"
            variant="secondary"
            disabled={busy}
            onClick={() => {
              if (!busy && !pending.current) {
                upload.cancel();
                onCancel();
              }
            }}
          >
            {t("cancel")}
          </Button>
          <Button
            ref={submitButton}
            type="button"
            variant="primary"
            disabled={disabled || (!upload.readyAsset && upload.phase !== "selected" && !upload.recovery)}
            onClick={() => {
              void perform(false);
            }}
          >
            {t(`image_upload.${submitLabel}`)}
          </Button>
        </div>
      </div>
    </div>
  );
});

function message(error: unknown): string {
  if (error instanceof ImageSelectionError) return `image_upload.errors.${error.kind}`;
  if (error instanceof UploadFailure) return `image_upload.errors.${error.kind}`;
  return errorMessageKey(error);
}

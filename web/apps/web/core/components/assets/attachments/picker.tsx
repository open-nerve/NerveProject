/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useEffect, useId, useImperativeHandle, useLayoutEffect, useRef, useState, type Ref } from "react";
import { observer } from "mobx-react";
import type { Asset } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { FileSelectionError, type UploadSession } from "@/store/asset/upload-session";
import { UploadFailure } from "@/lib/asset-upload";
import { errorMessageKey } from "@/lib/error-messages";
export type AttachmentPickerProps = {
  /** The file input, for a parent that returns focus to it. */
  input?: Ref<HTMLInputElement>;
  upload: UploadSession;
  onUploaded(asset: Asset): Promise<void>;
  onBusyChange?(busy: boolean): void;
};
export const AttachmentPicker = observer(function AttachmentPicker({
  input: parentInput,
  upload,
  onUploaded,
  onBusyChange,
}: AttachmentPickerProps) {
  const { t } = useTranslation();
  const id = useId();
  const input = useRef<HTMLInputElement>(null);
  useImperativeHandle(parentInput, () => input.current!);
  const alert = useRef<HTMLParagraphElement>(null);
  const submit = useRef<HTMLButtonElement>(null);
  const life = useRef({ mounted: false, revision: 0 });
  const pending = useRef(false);
  const [working, setWorking] = useState(false);
  const [completed, setCompleted] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [selection, setSelection] = useState<{ name: string; size: number }>();
  const busy = working || upload.busy;
  const disabled = busy || !upload.current;
  const error = !upload.current
    ? "errors.asset_scope_changed"
    : (failure ?? (upload.error ? uploadMessage(upload.error) : undefined));
  useEffect(() => {
    const held = life.current;
    held.mounted = true;
    return () => {
      held.mounted = false;
      ++held.revision;
    };
  }, []);
  useLayoutEffect(() => {
    if (!upload.current) return;
    if (error) alert.current?.focus({ preventScroll: true });
    else if (completed || upload.phase === "cancelled") input.current?.focus({ preventScroll: true });
    else if (upload.phase === "selected") submit.current?.focus({ preventScroll: true });
  }, [error, selection, upload, upload.phase, completed]);
  const reject = (key: string) => {
    upload.cancel();
    setSelection(undefined);
    setCompleted(false);
    setFailure(key);
  };
  const choose = (files: FileList | readonly File[]) => {
    if (disabled || pending.current || !files.length) return;
    if (files.length !== 1) {
      reject("content_files.errors.multiple");
      return;
    }
    setCompleted(false);
    setFailure(undefined);
    const file = files[0]!;
    upload.select(file);
    setSelection({ name: file.name, size: file.size });
  };
  const perform = async () => {
    if (disabled || pending.current || completed) return;
    pending.current = true;
    const revision = ++life.current.revision;
    const alive = () => life.current.mounted && life.current.revision === revision && upload.current;
    setWorking(true);
    setFailure(undefined);
    onBusyChange?.(true);
    try {
      const asset =
        upload.readyAsset ??
        (await (upload.recovery === "confirm"
          ? upload.retryComplete()
          : upload.recovery === "restart"
            ? upload.restart()
            : upload.start()));
      if (!alive() || !asset) return;
      await onUploaded(asset);
      if (alive()) setCompleted(true);
    } catch (reason) {
      if (alive()) setFailure(uploadMessage(reason));
    } finally {
      if (alive()) {
        pending.current = false;
        setWorking(false);
        onBusyChange?.(false);
      }
    }
  };
  const cancel = () => {
    if (!upload.busy) return;
    ++life.current.revision;
    pending.current = false;
    upload.cancel();
    setWorking(false);
    setCompleted(false);
    setFailure(undefined);
    onBusyChange?.(false);
  };
  const label = completed
    ? "upload"
    : upload.readyAsset
      ? "use_file"
      : upload.recovery === "confirm"
        ? "check_upload"
        : upload.recovery === "restart"
          ? "upload_again"
          : upload.recovery === "prepare"
            ? "retry_upload"
            : "upload";
  return (
    <div className="space-y-3" aria-busy={busy}>
      <div
        className="rounded-md border border-dashed border-subtle p-3 text-13 text-secondary"
        aria-label={t("content_files.drop")}
        onDragOver={(event) => {
          event.preventDefault();
          if (event.dataTransfer) event.dataTransfer.dropEffect = disabled ? "none" : "copy";
        }}
        onDrop={(event) => {
          event.preventDefault();
          if (disabled || pending.current) return;
          const { files, items } = event.dataTransfer;
          if (files.length > 1 || items.length > 1) {
            reject("content_files.errors.multiple");
            return;
          }
          if ([...items].some((item) => item.webkitGetAsEntry?.()?.isDirectory)) {
            reject("content_files.errors.directory");
            return;
          }
          if (files.length !== 1 || [...items].some((item) => item.kind !== "file")) {
            reject("content_files.errors.not_file");
            return;
          }
          choose(files);
        }}
      >
        {t("content_files.drop")}
      </div>
      <label className="block text-13" htmlFor={id}>
        {t("content_files.choose")}
      </label>
      <input
        ref={input}
        id={id}
        type="file"
        disabled={disabled}
        aria-invalid={!!error}
        aria-describedby={error ? `${id}-error` : undefined}
        className="block max-w-full text-13"
        onChange={(event) => {
          if (event.target.files) choose(event.target.files);
          event.target.value = "";
        }}
      />
      {selection && (
        <p className="text-13 break-words">
          {selection.name}
          <span className="ml-2 text-secondary">{selection.size} B</span>
        </p>
      )}
      <p aria-live="polite" aria-atomic="true" className="text-13 text-secondary">
        {t(`content_files.phases.${upload.phase}`)}
      </p>
      {error && (
        <p ref={alert} id={`${id}-error`} role="alert" tabIndex={-1} className="text-13 text-danger-primary">
          {t(error)}
        </p>
      )}
      <div className="flex gap-2">
        <Button
          ref={submit}
          type="button"
          variant="secondary"
          disabled={disabled || completed || (upload.phase !== "selected" && !upload.readyAsset && !upload.recovery)}
          onClick={() => {
            void perform();
          }}
        >
          {t(`content_files.${label}`)}
        </Button>
        {upload.busy && (
          <Button type="button" variant="secondary" onClick={cancel}>
            {t("content_files.cancel_upload")}
          </Button>
        )}
      </div>
    </div>
  );
});
function uploadMessage(error: unknown): string {
  if (error instanceof FileSelectionError) return `content_files.errors.${error.kind}`;
  if (error instanceof UploadFailure) return `content_files.errors.${error.kind}`;
  return errorMessageKey(error);
}

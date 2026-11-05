/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "@nerve/i18n";
import type { TNameDescriptionLoader } from "@nerve/types";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { AutosaveRetention } from "./autosave-retention";

/** Local input revisions only; the caller owns persistence and the authoritative entity. */
export function useConfirmedAutosave<T>({
  initial,
  same,
  save,
  status,
  disabled = false,
  prepareSave,
  retention,
  canSave,
  resumesOnEdit,
}: {
  initial: T;
  same: (left: T, right: T) => boolean;
  save: (value: T) => Promise<void>;
  status: (value: TNameDescriptionLoader) => void;
  disabled?: boolean;
  prepareSave?: () => (value: T) => Promise<void>;
  retention?: AutosaveRetention<T>;
  canSave?: (value: T) => boolean;
  /** False keeps preserved input paused while editing; only an explicit resume may send it. */
  resumesOnEdit?: (value: T) => boolean;
}) {
  const [owner] = useState(() => {
    const kept = retention?.read();
    const dirty = !!kept?.dirty && !same(kept.value, initial);
    return {
      active: false,
      eligible: !canSave || canSave(dirty ? kept!.value : initial),
      value: dirty ? kept!.value : initial,
      confirmed: dirty ? kept!.confirmed : initial,
      latestBase: initial,
      revision: kept?.revision ?? 0,
      attempted: kept?.attempted ?? -1,
      inFlight: 0,
      writing: [] as T[],
      dirty,
      replacing: false,
      paused: dirty,
      conflict: dirty && !same(kept!.confirmed, initial),
      failure: dirty ? kept?.failure : undefined,
      prepared: undefined as ((value: T) => Promise<void>) | undefined,
      timer: undefined as ReturnType<typeof setTimeout> | undefined,
    };
  });
  const [value, setValue] = useState(owner.value);
  const [replacing, setReplacing] = useState(false);
  const [failure, setFailure] = useState(owner.failure);
  const [conflict, setConflict] = useState(owner.conflict);
  const [needsResume, setNeedsResume] = useState(owner.paused);
  const latest = useRef({ save, status, same, disabled, prepareSave, retention, canSave, resumesOnEdit });
  useLayoutEffect(() => {
    latest.current = { save, status, same, disabled, prepareSave, retention, canSave, resumesOnEdit };
  }, [save, status, same, disabled, prepareSave, retention, canSave, resumesOnEdit]);
  const remember = useCallback(() => {
    latest.current.retention?.store({
      value: owner.value,
      confirmed: owner.confirmed,
      revision: owner.revision,
      attempted: owner.attempted,
      dirty: owner.dirty,
      failure: owner.failure,
    });
  }, [owner]);
  const fail = useCallback(
    (next: { error: unknown } | undefined) => {
      owner.failure = next;
      setFailure(next);
    },
    [owner]
  );
  const pause = useCallback(
    (paused: boolean, changedBase = false) => {
      owner.paused = paused;
      owner.conflict = changedBase;
      setNeedsResume(paused);
      setConflict(changedBase);
    },
    [owner]
  );
  useLayoutEffect(() => {
    owner.active = true;
    return () => {
      owner.active = false;
      clearTimeout(owner.timer);
      // A disappearing canonical owner suspends input. Normal route exit keeps the original last-edit flush.
      if (
        !owner.replacing &&
        !owner.paused &&
        owner.dirty &&
        owner.eligible &&
        owner.attempted !== owner.revision &&
        !latest.current.disabled &&
        (latest.current.retention?.canFlushOnUnmount ?? true)
      ) {
        owner.attempted = owner.revision;
        void Promise.resolve()
          .then(() => (owner.prepared ?? latest.current.save)(owner.value))
          .catch(() => undefined);
      }
      remember();
    };
  }, [owner, remember]);
  useLayoutEffect(() => {
    owner.latestBase = initial;
    if (!owner.dirty || same(owner.value, initial)) {
      owner.confirmed = initial;
      owner.value = initial;
      owner.dirty = false;
      setValue(initial);
      fail(undefined);
      pause(false);
    } else if (retention && !same(owner.confirmed, initial) && !owner.writing.some((input) => same(input, initial))) {
      clearTimeout(owner.timer);
      pause(true, true);
    } else {
      owner.confirmed = initial;
    }
    remember();
  }, [owner, initial, same, retention, remember, fail, pause]);

  const send = useCallback(async () => {
    clearTimeout(owner.timer);
    if (
      !owner.active ||
      owner.replacing ||
      owner.paused ||
      !owner.dirty ||
      (latest.current.canSave && !latest.current.canSave(owner.value)) ||
      latest.current.disabled ||
      owner.attempted === owner.revision
    )
      return;
    const revision = owner.revision;
    const input = owner.value;
    owner.attempted = revision;
    owner.inFlight += 1;
    owner.writing.push(input);
    fail(undefined);
    remember();
    latest.current.status("submitting");
    try {
      await (owner.prepared ?? latest.current.save)(input);
      if (!owner.active) return;
      if (revision === owner.revision) {
        owner.confirmed = input;
        owner.dirty = false;
        pause(false);
        latest.current.status("submitted");
      }
    } catch (error) {
      if (owner.active && revision === owner.revision) {
        fail({ error });
        latest.current.status("failed");
      }
    } finally {
      owner.inFlight -= 1;
      owner.writing.splice(owner.writing.indexOf(input), 1);
      if (owner.active) remember();
    }
  }, [owner, fail, pause, remember]);
  const change = useCallback(
    (next: T) => {
      if (!owner.active || owner.replacing || latest.current.disabled || latest.current.same(next, owner.value)) return;
      owner.prepared = latest.current.prepareSave?.();
      owner.value = next;
      owner.eligible = !latest.current.canSave || latest.current.canSave(next);
      owner.revision += 1;
      owner.dirty = owner.inFlight > 0 || !latest.current.same(next, owner.confirmed);
      setValue(next);
      fail(undefined);
      clearTimeout(owner.timer);
      // Editing preserved input is allowed, but cannot silently resolve a changed canonical base
      // or resend what the caller must verify again first.
      if (!owner.conflict && (latest.current.resumesOnEdit?.(next) ?? true)) pause(false);
      remember();
      latest.current.status(owner.dirty ? "submitting" : "saved");
      if (owner.dirty && !owner.paused) owner.timer = setTimeout(() => void send(), 1500);
    },
    [owner, send, fail, pause, remember]
  );
  const retry = useCallback(() => {
    if (!owner.active || !owner.dirty || owner.inFlight > 0 || owner.paused) return;
    owner.attempted = -1;
    void send();
  }, [owner, send]);
  const resume = useCallback(() => {
    if (!owner.active || !owner.dirty || owner.inFlight > 0 || latest.current.disabled) return;
    owner.confirmed = owner.latestBase;
    owner.prepared = latest.current.prepareSave?.();
    owner.attempted = -1;
    pause(false);
    remember();
    void send();
  }, [owner, send, pause, remember]);
  const discard = useCallback(() => {
    if (!owner.active || owner.inFlight > 0 || owner.replacing) return;
    clearTimeout(owner.timer);
    owner.value = owner.latestBase;
    owner.confirmed = owner.latestBase;
    owner.dirty = false;
    owner.revision += 1;
    owner.attempted = owner.revision;
    owner.prepared = undefined;
    setValue(owner.value);
    fail(undefined);
    pause(false);
    remember();
    latest.current.status("saved");
  }, [owner, fail, pause, remember]);
  const replaceWith = useCallback(
    async (confirm: () => Promise<T>) => {
      if (!owner.active || owner.replacing || latest.current.disabled) throw new SessionChangedError();
      clearTimeout(owner.timer);
      const attempted = owner.attempted === owner.revision;
      const revision = ++owner.revision;
      if (attempted) owner.attempted = revision;
      owner.replacing = true;
      setReplacing(true);
      remember();
      latest.current.status("submitting");
      try {
        const confirmed = await confirm();
        if (!owner.active || revision !== owner.revision) throw new SessionChangedError();
        owner.value = confirmed;
        owner.confirmed = confirmed;
        owner.latestBase = confirmed;
        owner.dirty = false;
        owner.prepared = undefined;
        owner.attempted = revision;
        setValue(confirmed);
        fail(undefined);
        pause(false);
        latest.current.status("submitted");
        return confirmed;
      } catch (error) {
        if (owner.active && revision === owner.revision) {
          if (owner.dirty) fail({ error });
          latest.current.status(owner.dirty ? "failed" : "saved");
        }
        throw error;
      } finally {
        owner.replacing = false;
        if (owner.active) {
          setReplacing(false);
          remember();
        }
      }
    },
    [owner, fail, pause, remember]
  );
  return { value, change, failure, retry, replaceWith, replacing, conflict, needsResume, resume, discard };
}

export function PreservedInput({
  conflict,
  resume,
  discard,
  keepEditing = false,
}: {
  conflict: boolean;
  resume(): void;
  discard(): void;
  keepEditing?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <div role="status" className="space-x-2 text-13">
      <span>{t(conflict ? "content_files.input.conflict" : "content_files.input.preserved")}</span>
      <button type="button" onClick={resume}>
        {t(
          keepEditing
            ? "content_files.input.keep_editing"
            : conflict
              ? "content_files.input.keep"
              : "content_files.input.save"
        )}
      </button>
      <button type="button" onClick={discard}>
        {t("content_files.input.latest")}
      </button>
    </div>
  );
}

export function AutosaveError({ failure, retry }: { failure: { error: unknown } | undefined; retry?: () => void }) {
  const message = useRefusalMessage();
  const { t } = useTranslation();
  if (!failure || failure.error instanceof SessionChangedError) return null;
  return (
    <div role="alert">
      {message(failure.error)}{" "}
      {retry && (
        <button type="button" onClick={retry}>
          {t("common.retry")}
        </button>
      )}
    </div>
  );
}

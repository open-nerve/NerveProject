/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "@nerve/i18n";
import type { TNameDescriptionLoader } from "@nerve/types";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";

/** Local input revisions only; the caller owns persistence and the authoritative entity. */
export function useConfirmedAutosave<T>({
  initial,
  same,
  save,
  status,
  disabled = false,
  prepareSave,
}: {
  initial: T;
  same: (left: T, right: T) => boolean;
  save: (value: T) => Promise<void>;
  status: (value: TNameDescriptionLoader) => void;
  disabled?: boolean;
  prepareSave?: () => (value: T) => Promise<void>;
}) {
  const [value, setValue] = useState(initial);
  const [replacing, setReplacing] = useState(false);
  const [failure, setFailure] = useState<{ error: unknown }>();
  const [owner] = useState(() => ({
    active: false,
    value: initial,
    confirmed: initial,
    revision: 0,
    attempted: -1,
    inFlight: 0,
    dirty: false,
    replacing: false,
    prepared: undefined as ((value: T) => Promise<void>) | undefined,
    timer: undefined as ReturnType<typeof setTimeout> | undefined,
  }));
  const latest = useRef({ save, status, same, disabled, prepareSave });
  useLayoutEffect(() => {
    latest.current = { save, status, same, disabled, prepareSave };
  }, [save, status, same, disabled, prepareSave]);
  useLayoutEffect(() => {
    owner.active = true;
    return () => {
      owner.active = false;
      clearTimeout(owner.timer);
      // Preserve the existing last-edit flush, but never resend an attempted or refused revision.
      if (!owner.replacing && owner.dirty && owner.attempted !== owner.revision && !latest.current.disabled) {
        owner.attempted = owner.revision;
        void Promise.resolve()
          .then(() => (owner.prepared ?? latest.current.save)(owner.value))
          .catch(() => undefined);
      }
    };
  }, [owner]);
  useLayoutEffect(() => {
    owner.confirmed = initial;
    if (!owner.dirty || same(owner.value, initial)) {
      owner.value = initial;
      owner.dirty = false;
      setValue(initial);
    }
  }, [owner, initial, same]);

  const send = useCallback(async () => {
    clearTimeout(owner.timer);
    if (
      !owner.active ||
      owner.replacing ||
      !owner.dirty ||
      latest.current.disabled ||
      owner.attempted === owner.revision
    )
      return;
    const revision = owner.revision;
    const input = owner.value;
    owner.attempted = revision;
    owner.inFlight += 1;
    setFailure(undefined);
    latest.current.status("submitting");
    try {
      await (owner.prepared ?? latest.current.save)(input);
      if (!owner.active) return;
      if (revision === owner.revision) {
        owner.confirmed = input;
        owner.dirty = false;
        latest.current.status("submitted");
      }
    } catch (error) {
      if (owner.active && revision === owner.revision) {
        setFailure({ error });
        latest.current.status("failed");
      }
    } finally {
      owner.inFlight -= 1;
    }
  }, [owner]);
  const change = useCallback(
    (next: T) => {
      if (!owner.active || owner.replacing || latest.current.disabled) return;
      owner.prepared = latest.current.prepareSave?.();
      owner.value = next;
      owner.revision += 1;
      owner.dirty = owner.inFlight > 0 || !latest.current.same(next, owner.confirmed);
      setValue(next);
      setFailure(undefined);
      clearTimeout(owner.timer);
      latest.current.status(owner.dirty ? "submitting" : "saved");
      if (owner.dirty) owner.timer = setTimeout(() => void send(), 1500);
    },
    [owner, send]
  );
  const retry = useCallback(() => {
    if (!owner.active || !owner.dirty || owner.inFlight > 0) return;
    owner.attempted = -1;
    void send();
  }, [owner, send]);
  const replaceWith = useCallback(
    async (confirm: () => Promise<T>) => {
      if (!owner.active || owner.replacing || latest.current.disabled) throw new SessionChangedError();
      clearTimeout(owner.timer);
      const attempted = owner.attempted === owner.revision;
      const revision = ++owner.revision;
      if (attempted) owner.attempted = revision;
      owner.replacing = true;
      setReplacing(true);
      latest.current.status("submitting");
      try {
        const confirmed = await confirm();
        if (!owner.active || revision !== owner.revision) throw new SessionChangedError();
        owner.value = confirmed;
        owner.confirmed = confirmed;
        owner.dirty = false;
        owner.prepared = undefined;
        owner.attempted = revision;
        setValue(confirmed);
        setFailure(undefined);
        latest.current.status("submitted");
        return confirmed;
      } catch (error) {
        if (owner.active && revision === owner.revision) {
          if (owner.dirty) setFailure({ error });
          latest.current.status(owner.dirty ? "failed" : "saved");
        }
        throw error;
      } finally {
        owner.replacing = false;
        if (owner.active) setReplacing(false);
      }
    },
    [owner]
  );
  return { value, change, failure, retry, replaceWith, replacing };
}

export function AutosaveError({ failure, retry }: { failure: { error: unknown } | undefined; retry: () => void }) {
  const message = useRefusalMessage();
  const { t } = useTranslation();
  if (!failure || failure.error instanceof SessionChangedError) return null;
  return (
    <div role="alert">
      {message(failure.error)}{" "}
      <button type="button" onClick={retry}>
        {t("common.retry")}
      </button>
    </div>
  );
}

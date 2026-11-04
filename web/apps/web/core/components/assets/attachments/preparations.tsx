/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useRef, useState } from "react";
import { observer } from "mobx-react";
import type { Asset } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { errorMessageKey } from "@/lib/error-messages";
import type { DraftPreparationsResource } from "@/store/asset/draft-preparations-resource";
import { DeleteAttachmentDialog } from "./delete-dialog";
export const DraftPreparations = observer(function DraftPreparations({
  resource,
}: {
  resource: DraftPreparationsResource;
}) {
  const { t } = useTranslation();
  const [opened, setOpened] = useState<DraftPreparationsResource>();
  const [selection, setSelection] = useState<{ resource: DraftPreparationsResource; asset: Asset }>();
  const refresh = useRef<HTMLButtonElement>(null);
  if (!resource.current) return null;
  const open = opened === resource;
  const selected = selection?.resource === resource ? selection.asset : undefined;
  const read = () => {
    setOpened(resource);
    void resource.retryRead().catch(() => undefined);
  };
  return (
    <section className="space-y-3">
      <Button ref={refresh} type="button" variant="secondary" disabled={resource.loading} onClick={read}>
        {t(open ? "content_files.refresh" : "content_files.preparations.open")}
      </Button>
      {open && (
        <>
          <p className="text-13 text-secondary">{t("content_files.preparations.warning")}</p>
          <p aria-live="polite" className="text-13 text-secondary">
            {resource.loading
              ? t("content_files.reading")
              : resource.rows.length === 0
                ? t("content_files.preparations.empty")
                : null}
          </p>
          {resource.error !== undefined && (
            <p role="alert" className="text-13 text-danger-primary">
              {t(errorMessageKey(resource.error))}
            </p>
          )}
          {resource.uncertainIDs.length > 0 && (
            <p className="text-13 text-secondary">{t("content_files.unknown_delete")}</p>
          )}
          <ul className="space-y-2">
            {resource.rows.map((asset) => (
              <li key={asset.id} className="space-y-1 rounded-md border border-subtle p-3 text-13">
                <p className="break-words">{asset.name}</p>
                <p className="text-secondary">
                  {t(`content_files.states.${asset.state}`)} · {t(`content_files.purposes.${asset.purpose}`)} ·{" "}
                  {asset.size} B
                </p>
                <time className="text-12 text-secondary" dateTime={asset.created_at}>
                  {new Date(asset.created_at).toLocaleString()}
                </time>
                <div>
                  <Button
                    type="button"
                    variant="error-outline"
                    aria-label={`${t("content_files.discard")} ${asset.name}`}
                    onClick={() => {
                      refresh.current?.focus();
                      setSelection({ resource, asset });
                    }}
                  >
                    {t("content_files.discard")}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
          {resource.nextCursor && (
            <Button
              type="button"
              variant="secondary"
              disabled={resource.loading}
              onClick={() => {
                void resource.more().catch(() => undefined);
              }}
            >
              {t("content_files.more")}
            </Button>
          )}
        </>
      )}
      {selected && (
        <DeleteAttachmentDialog
          key={selected.id}
          view={resource.view}
          file={selected}
          discard
          uncertain={resource.uncertainIDs.includes(selected.id)}
          remove={(id) => resource.remove(id)}
          verify={() => resource.retryRead()}
          onClose={() => setSelection(undefined)}
        />
      )}
    </section>
  );
});

/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useLayoutEffect, useRef, useState } from "react";
import { cn } from "@nerve/utils";
import { useTranslation } from "@nerve/i18n";
import { useSessionSWR } from "@/lib/use-session-swr";
import { followInSession } from "@/lib/in-session";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
import { DescriptionVersionsDropdown } from "./dropdown";
import { DescriptionVersionsModal } from "./modal";

/** Display metadata, not either domain's wire protocol. */
export type DescriptionVersionOption = { id: string; ownerId: string; lastSavedAt: string };
export type TDescriptionVersionEntityInformation = {
  createdAt: Date;
  createdByDisplayName: string;
  id: string;
  isRestoreDisabled: boolean;
};
type Props = {
  className?: string;
  entityInformation: TDescriptionVersionEntityInformation;
  scopeKey: string;
  versions: readonly DescriptionVersionOption[] | undefined;
  listError?: unknown;
  retryList: () => Promise<unknown>;
  loadMore?: () => Promise<unknown>;
  readDescription: (versionId: string) => Promise<string>;
  handleRestore: (descriptionHTML: string) => Promise<unknown>;
  projectId?: string;
  workspaceSlug: string;
};
export function DescriptionVersionsRoot(props: Props) {
  return (
    <DescriptionVersionsContent
      key={JSON.stringify([props.scopeKey, props.workspaceSlug, props.projectId, props.entityInformation.id])}
      {...props}
    />
  );
}
function DescriptionVersionsContent(props: Props) {
  const {
    className,
    entityInformation,
    scopeKey,
    versions,
    listError,
    retryList,
    loadMore,
    readDescription,
    handleRestore,
    projectId,
    workspaceSlug,
  } = props;
  const { t } = useTranslation();
  const message = useRefusalMessage();
  const [activeId, setActiveId] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [readError, setReadError] = useState<unknown>();
  const active = useRef(false);
  const reading = useRef(false);
  useLayoutEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  const {
    data: html,
    error,
    mutate,
  } = useSessionSWR(
    activeId
      ? ["DESCRIPTION_VERSION_HTML", scopeKey, workspaceSlug, projectId ?? "", entityInformation.id, activeId]
      : null,
    (_scope, _ws, _project, _entity, version) => readDescription(version),
    { dedupingInterval: 0 }
  );
  const index = versions?.findIndex((row) => row.id === activeId) ?? -1;
  const selected = index >= 0 ? versions?.[index] : undefined;
  const readList = async (read: () => Promise<unknown>) => {
    if (reading.current) return;
    reading.current = true;
    setPending(true);
    setReadError(undefined);
    await followInSession(read, {
      failed: (reason) => {
        if (active.current) setReadError(reason);
      },
    });
    reading.current = false;
    if (active.current) setPending(false);
  };
  return (
    <>
      {activeId && (
        <DescriptionVersionsModal
          key={activeId}
          activeVersionDescription={html}
          activeVersionDetails={selected}
          handleClose={() => setActiveId(null)}
          handleNavigation={(direction) => {
            const next = index + (direction === "prev" ? -1 : 1);
            if (next >= 0 && next < (versions?.length ?? 0)) setActiveId(versions![next]!.id);
          }}
          handleRestore={handleRestore}
          isNextDisabled={index < 0 || index >= (versions?.length ?? 0) - 1}
          isPrevDisabled={index <= 0}
          isOpen
          isRestoreDisabled={entityInformation.isRestoreDisabled}
          projectId={projectId}
          workspaceSlug={workspaceSlug}
          error={error}
          retry={() => mutate()}
        />
      )}
      <div className={cn(className)}>
        <DescriptionVersionsDropdown
          disabled={!versions?.length}
          entityInformation={entityInformation}
          onVersionClick={setActiveId}
          versions={versions}
        />
        {(readError ?? listError) != null && (
          <div role="alert">
            {message(readError ?? listError)}{" "}
            <button type="button" disabled={pending} onClick={() => void readList(retryList)}>
              {t("common.retry")}
            </button>
          </div>
        )}
        {loadMore && (
          <button type="button" disabled={pending} onClick={() => void readList(loadMore)}>
            {t("common.actions.load_more")}
          </button>
        )}
      </div>
    </>
  );
}

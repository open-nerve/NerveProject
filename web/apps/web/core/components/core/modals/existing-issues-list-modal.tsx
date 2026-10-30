/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useLayoutEffect, useMemo, useState } from "react";
import { observer } from "mobx-react";
import { Combobox } from "@headlessui/react";
import { CloseOutline, SearchOutline } from "@makeplane/propel/icons";
import { Check } from "lucide-react";
import { Switch } from "@makeplane/propel/components/switch";
import type { Issue } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssueSearch } from "@/hooks/use-issue-search";
import { useRefusalMessage, useRefusalToast } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IssueSearchQuery } from "@/services/issue/issue-search.service";
import type { IssueReadToken } from "@/store/issue/root.store";
import { IssueSearchModalEmptyState } from "./issue-search-modal-empty-state";

/** The result of one selection command; only failed/unconfirmed IDs may be resubmitted. */
export type SelectionSubmitResult = {
  confirmedIds: readonly string[];
  failures: readonly { id: string; error: unknown }[];
};
type Props = {
  workspaceSlug: string | undefined;
  projectId?: string;
  isOpen: boolean;
  handleClose: () => void;
  searchParams: Omit<IssueSearchQuery, "project_id" | "q">;
  handleOnSubmit: (ids: readonly string[]) => Promise<SelectionSubmitResult>;
  workspaceLevelToggle?: boolean;
  shouldHideIssue?: (issue: Issue) => boolean;
};
type Choice = { id: string; proof: IssueReadToken };

export const ExistingIssuesListModal = observer(function ExistingIssuesListModal(props: Props) {
  const { rootIssueStore: root } = useIssueDetail();
  const workspace = props.workspaceSlug
    ? root.rootStore.workspaceRoot.getWorkspaceBySlug(props.workspaceSlug)
    : undefined;
  const identity = root.permissionReads.identity(workspace?.id, props.projectId ? [props.projectId] : []);
  const scope = JSON.stringify([props.workspaceSlug, props.projectId, props.searchParams, props.isOpen, identity]);
  return <ExistingIssuesScope key={scope} {...props} />;
});

function ExistingIssuesScope(props: Props) {
  const [workspaceLevel, setWorkspaceLevel] = useState(false);
  return (
    <ExistingIssuesSelection
      key={String(workspaceLevel)}
      {...props}
      workspaceLevel={workspaceLevel}
      setWorkspaceLevel={setWorkspaceLevel}
    />
  );
}

const ExistingIssuesSelection = observer(function ExistingIssuesSelection({
  workspaceSlug,
  projectId,
  isOpen,
  handleClose,
  searchParams,
  handleOnSubmit,
  workspaceLevelToggle = false,
  shouldHideIssue,
  workspaceLevel,
  setWorkspaceLevel,
}: Props & { workspaceLevel: boolean; setWorkspaceLevel: (value: boolean) => void }) {
  const { t } = useTranslation();
  const { rootIssueStore: root } = useIssueDetail();
  const [term, setTerm] = useState("");
  const query: IssueSearchQuery = {
    ...searchParams,
    ...(!workspaceLevel && projectId ? { project_id: projectId } : {}),
    ...(term ? { q: term } : {}),
  };
  const search = useIssueSearch(root, workspaceSlug, query, isOpen);
  const workspaceID = workspaceSlug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug)?.id : undefined;
  const projectIDs = query.project_id ? [query.project_id] : [];
  const permission = root.permissionReads.identity(workspaceID, projectIDs);
  const owner = useMemo(() => ({ root, permission, active: false, submitting: false }), [root, permission]);
  const [selection, setSelection] = useState<{ owner: typeof owner; choices: Choice[] }>({ owner, choices: [] });
  const [pendingOwner, setPendingOwner] = useState<typeof owner>();
  const refusal = useRefusalToast();
  const refusalMessage = useRefusalMessage();
  useLayoutEffect(() => {
    owner.active = isOpen;
    return () => {
      owner.active = false;
    };
  }, [owner, isOpen]);
  const current = () =>
    owner.active &&
    root.active &&
    !!workspaceID &&
    root.permissionReads.identity(workspaceID, projectIDs) === owner.permission;
  const visible = (choice: Choice) => {
    const issue = root.issues.getIssueById(choice.id);
    return (
      !!issue &&
      issue.workspace_id === workspaceID &&
      (!query.project_id || issue.project_id === query.project_id) &&
      root.isReadCurrent(choice.proof, projectIDs, workspaceID) &&
      root.isReadCurrent(choice.proof, [issue.project_id], workspaceID) &&
      !shouldHideIssue?.(issue)
    );
  };
  const selected = selection.owner === owner ? selection.choices.filter(visible) : [];
  const issues = search.ids.flatMap((id) => {
    const issue = root.issues.getIssueById(id);
    return issue && !shouldHideIssue?.(issue) ? [issue] : [];
  });
  const pending = pendingOwner === owner;
  const setChoices = (choices: Choice[]) => setSelection({ owner, choices });
  const choose = (ids: string[]) => {
    if (!current() || pending) return;
    const proof = search.proof;
    setChoices(
      [...new Set(ids)].flatMap((id) => {
        const held = selected.find((choice) => choice.id === id);
        return held ? [held] : proof && issues.some((issue) => issue.id === id) ? [{ id, proof }] : [];
      })
    );
  };
  const close = () => {
    owner.active = false;
    handleClose();
  };
  const submit = async () => {
    if (!current() || owner.submitting || !selected.length) return;
    const ids = selected.map((choice) => choice.id);
    owner.submitting = true;
    setPendingOwner(owner);
    try {
      const answer = await handleOnSubmit(ids);
      if (!current()) return;
      const confirmed = new Set(answer.confirmedIds);
      const failed = new Set(answer.failures.map((failure) => failure.id));
      if ([...confirmed, ...failed].some((id) => !ids.includes(id)) || [...confirmed].some((id) => failed.has(id)))
        throw new Error("Invalid selection confirmation");
      setChoices(selected.filter((choice) => !confirmed.has(choice.id)));
      if (confirmed.size === ids.length) close();
      else refusal(answer.failures[0]?.error ?? new Error("Selection was not confirmed"));
    } catch (error) {
      if (current() && !(error instanceof SessionChangedError)) refusal(error);
    } finally {
      owner.submitting = false;
      if (current()) setPendingOwner(undefined);
    }
  };
  const allSelected = issues.length > 0 && issues.every((issue) => selected.some((choice) => choice.id === issue.id));
  const identifier = (issue: Issue) => (
    <span className="shrink-0 text-11 text-secondary">
      {root.rootStore.projectRoot.project.getProjectIdentifierById(issue.project_id)}-{issue.sequence_id}
    </span>
  );
  return (
    <ModalCore isOpen={isOpen} handleClose={close} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <ModalTitle className="px-4 pt-4 text-16 font-medium">{t("issue.add.existing")}</ModalTitle>
      <Combobox as="div" multiple value={selected.map((choice) => choice.id)} onChange={choose} disabled={pending}>
        {search.error !== undefined && (
          <div role="alert" className="p-3">
            {refusalMessage(search.error)}{" "}
            <button type="button" onClick={search.retry}>
              {t("common.retry")}
            </button>
          </div>
        )}
        <div className="relative m-1">
          <SearchOutline
            className="pointer-events-none absolute top-3.5 left-4 h-5 w-5 text-secondary"
            aria-hidden="true"
          />
          <Combobox.Input
            aria-label={t("common.search.placeholder")}
            className="h-12 w-full bg-transparent pr-4 pl-11 text-13 outline-none"
            placeholder={t("common.search.placeholder")}
            value={term}
            onChange={(event) => setTerm(event.target.value)}
          />
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2 p-3 text-13">
          <div className="flex flex-wrap gap-2">
            {selected.length ? (
              selected.map(({ id }) => {
                const issue = root.issues.getIssueById(id)!;
                return (
                  <span key={id} className="flex items-center gap-1 rounded-md border border-subtle p-1">
                    {identifier(issue)}
                    <button
                      type="button"
                      disabled={pending}
                      aria-label={`${t("common.remove")} ${issue.name}`}
                      onClick={() => choose(selected.filter((choice) => choice.id !== id).map((choice) => choice.id))}
                    >
                      <CloseOutline className="h-3 w-3" />
                    </button>
                  </span>
                );
              })
            ) : (
              <span>{t("issue.select.empty")}</span>
            )}
          </div>
          {workspaceLevelToggle && (
            <label className="flex items-center gap-2">
              <Switch
                size="sm"
                disabled={pending}
                checked={workspaceLevel}
                onCheckedChange={setWorkspaceLevel}
                aria-label={t("common.workspace_level")}
              />
              {t("common.workspace_level")}
            </label>
          )}
        </div>
        <Combobox.Options
          as="ul"
          static
          modal={false}
          className="vertical-scrollbar max-h-80 overflow-y-auto p-2 text-13"
        >
          {!search.loading &&
            search.error === undefined &&
            issues.map((issue) => (
              <Combobox.Option
                key={issue.id}
                as="li"
                value={issue.id}
                className={({ active }) =>
                  `my-0.5 flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 ${active ? "bg-layer-1" : ""}`
                }
              >
                <Check
                  aria-hidden="true"
                  className={`h-4 w-4 ${selected.some((choice) => choice.id === issue.id) ? "" : "invisible"}`}
                />
                <span
                  className="h-1.5 w-1.5 rounded-full"
                  style={{ backgroundColor: root.rootStore.state.getStateById(issue.state_id)?.color }}
                />
                {identifier(issue)}
                <span className="truncate">{issue.name}</span>
              </Combobox.Option>
            ))}
        </Combobox.Options>
        {search.loading && (
          <p role="status" className="p-3">
            {t("common.loading")}
          </p>
        )}
        {!search.loading && search.error === undefined && issues.length === 0 && (
          <IssueSearchModalEmptyState
            issues={issues}
            searchTerm={term}
            debouncedSearchTerm={term}
            isSearching={false}
          />
        )}
      </Combobox>
      <div className="flex items-center justify-between p-3">
        <Button
          variant="link"
          disabled={pending || issues.length === 0}
          onClick={() =>
            choose(
              allSelected
                ? selected
                    .filter((choice) => !issues.some((issue) => issue.id === choice.id))
                    .map((choice) => choice.id)
                : [...selected.map((choice) => choice.id), ...issues.map((issue) => issue.id)]
            )
          }
        >
          {allSelected ? t("issue.select.deselect_all") : t("issue.select.select_all")}
        </Button>
        <div className="flex gap-2">
          <Button variant="secondary" size="lg" onClick={close}>
            {t("common.cancel")}
          </Button>
          <Button variant="primary" size="lg" onClick={submit} loading={pending} disabled={pending || !selected.length}>
            {pending ? t("common.adding") : t("issue.select.add_selected")}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
});

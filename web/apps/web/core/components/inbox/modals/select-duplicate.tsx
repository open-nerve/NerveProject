/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useRef, useState } from "react";
import { observer } from "mobx-react";
import { Combobox } from "@headlessui/react";
import { SearchOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssueSearch } from "@/hooks/use-issue-search";
import { useRefusalMessage } from "@/hooks/use-refusal-toast";
import { IssueSearchModalEmptyState } from "@/components/core/modals/issue-search-modal-empty-state";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";

type Props = {
  /** The work item marked, and its project's: the intake detail's, never the address's (P5a spec 2.8). */
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  isOpen: boolean;
  value?: string | null;
  /** Closes the modal: a cancel, or once the work item is marked. */
  onClose: () => void;
  /** Marks the work item a duplicate of the one chosen; a refusal rejects, said by the caller. */
  onSubmit: (issueId: string) => void | Promise<void>;
};

/**
 * The modal that marks a work item of a project's intake a duplicate of another (P5a spec 2.8, R17): named by its
 * title (ModalTitle), its list its content, the candidates M4's search finds in the project, never the work item itself,
 * read for the work item and the project it is given. A candidate chosen is sent (onSubmit), the modal closed once
 * nerve has marked it, by neither Escape nor a click outside while it marks it; a refusal leaves it open, to choose
 * again.
 */
export const SelectDuplicateInboxIssueModal = observer(function SelectDuplicateInboxIssueModal(props: Props) {
  const { workspaceSlug, projectId, issueId } = props;
  const { rootIssueStore: root } = useIssueDetail();
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
  const identity = root.permissionReads.identity(workspace?.id, [projectId], true);
  return props.isOpen ? (
    <DuplicateSelection key={JSON.stringify([workspaceSlug, projectId, issueId, identity])} {...props} />
  ) : null;
});

const DuplicateSelection = observer(function DuplicateSelection(props: Props) {
  const { workspaceSlug, projectId, issueId, isOpen, value, onClose, onSubmit } = props;
  const { t } = useTranslation();
  const { rootIssueStore: root } = useIssueDetail();
  const [term, setTerm] = useState("");
  // whether the work item is being marked: the modal waits for nerve then
  const [marking, setMarking] = useState(false);
  // An intake item is not an Issue source. Its duplicate picker reads the project's candidates only.
  const search = useIssueSearch(root, workspaceSlug, { project_id: projectId, ...(term ? { q: term } : {}) }, isOpen);
  const capture = useDetailScope(workspaceSlug, projectId, issueId, true);
  const message = useRefusalMessage();
  const submitted = useRef(false);
  const issues = search.ids.flatMap((id) => {
    const issue = root.issues.getIssueById(id);
    return issue && issue.id !== issueId ? [issue] : [];
  });
  const choose = async (id: string | null) => {
    if (!id || id === issueId || submitted.current || !search.proof || !search.ids.includes(id)) return;
    const current = capture();
    const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
    const issue = root.issues.getIssueById(id);
    if (
      !current() ||
      !workspace ||
      !issue ||
      issue.project_id !== projectId ||
      issue.workspace_id !== workspace.id ||
      !root.isReadCurrent(search.proof, [projectId], workspace.id)
    )
      return;
    submitted.current = true;
    setMarking(true);
    try {
      // the intake's action model marks it, and says a refusal
      await onSubmit(id);
    } catch {
      submitted.current = false;
      if (current()) setMarking(false);
      return;
    }
    if (current()) onClose();
  };
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={marking ? undefined : onClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
      <ModalTitle className="px-4 pt-4 text-16 font-medium">{t("inbox_issue.actions.mark_as_duplicate")}</ModalTitle>
      <Combobox as="div" value={value ?? null} onChange={(id: string | null) => void choose(id)}>
        <div className="relative m-1">
          <SearchOutline
            className="pointer-events-none absolute top-3.5 left-4 h-5 w-5 text-secondary"
            aria-hidden="true"
          />
          <Combobox.Input
            aria-label={t("common.search.placeholder")}
            placeholder={t("common.search.placeholder")}
            className="h-12 w-full bg-transparent pr-4 pl-11 text-13 outline-none"
            value={term}
            onChange={(event) => setTerm(event.target.value)}
          />
        </div>
        {search.error !== undefined && (
          <div role="alert" className="p-3">
            {message(search.error)}{" "}
            <button type="button" onClick={search.retry}>
              {t("common.retry")}
            </button>
          </div>
        )}
        {search.loading && (
          <p role="status" className="p-3">
            {t("common.loading")}
          </p>
        )}
        {!search.loading && search.error === undefined && issues.length === 0 && (
          <IssueSearchModalEmptyState
            issueCount={issues.length}
            searchTerm={term}
            debouncedSearchTerm={term}
            isSearching={false}
          />
        )}
        <Combobox.Options
          as="ul"
          static
          modal={false}
          className="vertical-scrollbar max-h-80 overflow-y-auto p-2 text-13"
        >
          {issues.map((issue) => (
            <Combobox.Option
              as="li"
              key={issue.id}
              value={issue.id}
              className="my-0.5 flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-secondary data-focus:bg-layer-1"
            >
              <span
                className="h-1.5 w-1.5 shrink-0 rounded-full"
                style={{ backgroundColor: root.rootStore.state.getStateById(issue.state_id)?.color }}
              />
              <IssueIdentifier projectId={issue.project_id} issueId={issue.id} size="xs" variant="secondary" />
              <span className="truncate">{issue.name}</span>
            </Combobox.Option>
          ))}
        </Combobox.Options>
      </Combobox>
    </ModalCore>
  );
});

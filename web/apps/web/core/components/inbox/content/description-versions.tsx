/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import {
  DescriptionVersionsRoot,
  type TDescriptionVersionEntityInformation,
} from "@/components/core/description-versions";
import type {
  TIntakeDescriptionVersionDetails,
  TIntakeDescriptionVersionsListResponse,
} from "@/services/inbox/intake-description-version";
import { useSessionSWR } from "@/lib/use-session-swr";

type Props = {
  className?: string;
  entityInformation: TDescriptionVersionEntityInformation;
  fetchHandlers: {
    listDescriptionVersions: (entityId: string) => Promise<TIntakeDescriptionVersionsListResponse>;
    retrieveDescriptionVersion: (entityId: string, versionId: string) => Promise<TIntakeDescriptionVersionDetails>;
  };
  handleRestore: (html: string) => void;
  workspaceSlug: string;
  projectId: string;
};
/** M7 retains its legacy service and editor callback until its own native API lands. */
export function IntakeDescriptionVersions(props: Props) {
  const { entityInformation, fetchHandlers, workspaceSlug, projectId, handleRestore } = props;
  const id = entityInformation.id;
  const { data, error, mutate } = useSessionSWR(
    ["INTAKE_DESCRIPTION_VERSIONS", workspaceSlug, projectId, id],
    (_ws, _project, entity) => fetchHandlers.listDescriptionVersions(entity)
  );
  return (
    <DescriptionVersionsRoot
      {...props}
      scopeKey="intake"
      versions={data?.results.map((row) => ({ id: row.id, ownerId: row.owned_by, lastSavedAt: row.last_saved_at }))}
      listError={error}
      retryList={() => mutate()}
      readDescription={async (version) =>
        (await fetchHandlers.retrieveDescriptionVersion(id, version)).description_html ?? "<p></p>"
      }
      handleRestore={async (html) => {
        handleRestore(html);
      }}
    />
  );
}

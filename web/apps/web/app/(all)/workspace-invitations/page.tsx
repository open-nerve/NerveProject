/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate, useSearchParams } from "react-router";
import { BoxesOutline, CloseOutline, TickOutline, UserOutline } from "@makeplane/propel/icons";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { EmptySpace, EmptySpaceItem } from "@/components/ui/empty-space";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser } from "@/hooks/store/user";
import { useInvitationPreview } from "@/hooks/use-invitation-preview";
// wrappers
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

function WorkspaceInvitationPage() {
  // router
  const navigate = useNavigate();
  // query params
  const [searchParams] = useSearchParams();
  const invitation_id = searchParams.get("invitation_id");
  const token = searchParams.get("token");
  // store hooks
  const { data: currentUser } = useUser();
  const { acceptInvitation, declineInvitation } = useWorkspace();

  const { data: invitationDetail, error } = useInvitationPreview(invitation_id, token);

  const handleAccept = async () => {
    if (!invitationDetail || !token) return;
    try {
      // nerve accepts the invitation of the caller's own address alone
      await acceptInvitation(invitationDetail.id, token);
      navigate(`/${invitationDetail.workspace_slug}`);
    } catch (err: unknown) {
      console.error(err);
    }
  };

  const handleReject = async () => {
    if (!invitationDetail || !token) return;
    try {
      await declineInvitation(invitationDetail.id, token);
      navigate("/");
    } catch (err: unknown) {
      console.error(err);
    }
  };

  return (
    <AuthenticationWrapper pageType={EPageTypes.PUBLIC}>
      <div className="flex h-full w-full flex-col items-center justify-center px-3">
        {invitationDetail && !invitationDetail.declined ? (
          error ? (
            <div className="shadow-2xl flex w-full flex-col space-y-4 rounded-sm border border-subtle bg-surface-1 px-4 py-8 text-center md:w-1/3">
              <h2 className="text-18 uppercase">INVITATION NOT FOUND</h2>
            </div>
          ) : (
            <EmptySpace
              title={`You have been invited to ${invitationDetail.workspace_name}`}
              description="Your workspace is where you'll create projects, collaborate on your work items, and organize different streams of work in your Nerve account."
            >
              <EmptySpaceItem Icon={TickOutline} title="Accept" action={handleAccept} />
              <EmptySpaceItem Icon={CloseOutline} title="Ignore" action={handleReject} />
            </EmptySpace>
          )
        ) : error || invitationDetail?.declined ? (
          <EmptySpace
            title="This invitation link is not active anymore."
            description="Your workspace is where you'll create projects, collaborate on your work items, and organize different streams of work in your Nerve account."
            link={{ text: "Or start from an empty project", href: "/" }}
          >
            {!currentUser ? (
              <EmptySpaceItem Icon={UserOutline} title="Sign in to continue" href="/" />
            ) : (
              <EmptySpaceItem Icon={BoxesOutline} title="Continue to home" href="/" />
            )}
          </EmptySpace>
        ) : (
          <div className="flex h-full w-full items-center justify-center">
            <LogoSpinner />
          </div>
        )}
      </div>
    </AuthenticationWrapper>
  );
}

export default observer(WorkspaceInvitationPage);

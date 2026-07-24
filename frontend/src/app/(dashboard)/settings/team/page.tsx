"use client";

import { useState } from "react";
import { Users, Plus, Trash2, Shield } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  useTeamMembers,
  useOrgRoles,
  useInviteMember,
  useRemoveMember,
  type TeamMember,
} from "@/queries/team";
import { motion } from "motion/react";

function InviteDialog({ onClose }: { onClose: () => void }) {
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [role, setRole] = useState("member");
  const invite = useInviteMember();
  const { data: rolesData } = useOrgRoles();
  const roles = rolesData?.data ?? [];

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    invite.mutate(
      { email, first_name: firstName, last_name: lastName, role },
      { onSuccess: () => onClose() }
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <Card className="relative z-10 w-full max-w-md">
        <CardContent className="p-6">
          <h2 className="text-lg font-semibold mb-4">Invite Team Member</h2>
          <form onSubmit={handleSubmit} className="space-y-3">
            <input
              type="email"
              placeholder="Email address"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
            />
            <div className="grid grid-cols-2 gap-3">
              <input
                placeholder="First name"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                required
                className="rounded-lg border border-input bg-background px-3 py-2 text-sm"
              />
              <input
                placeholder="Last name"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                required
                className="rounded-lg border border-input bg-background px-3 py-2 text-sm"
              />
            </div>
            <select
              value={role}
              onChange={(e) => setRole(e.target.value)}
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
            >
              {roles.length > 0 ? (
                roles.map((r) => (
                  <option key={r.id} value={r.name}>{r.name}</option>
                ))
              ) : (
                <>
                  <option value="admin">Admin</option>
                  <option value="member">Member</option>
                  <option value="viewer">Viewer</option>
                </>
              )}
            </select>
            <div className="flex gap-2 pt-2">
              <Button type="button" variant="outline" onClick={onClose} className="flex-1">
                Cancel
              </Button>
              <Button type="submit" disabled={invite.isPending} className="flex-1">
                {invite.isPending ? "Sending..." : "Send Invite"}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

const STATUS_COLORS: Record<string, string> = {
  active: "bg-emerald-50 text-emerald-700",
  invited: "bg-blue-50 text-blue-700",
  inactive: "bg-neutral-100 text-neutral-600",
};

export default function TeamSettingsPage() {
  const [showInvite, setShowInvite] = useState(false);
  const { data, isLoading } = useTeamMembers();
  const removeMember = useRemoveMember();

  const members: TeamMember[] = data?.data ?? [];

  return (
    <motion.div className="space-y-6" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Team</h1>
          <p className="text-sm text-muted-foreground">
            Manage members and their roles in your organization
          </p>
        </div>
        <Button onClick={() => setShowInvite(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Invite Member
        </Button>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="animate-pulse rounded-xl border border-border bg-card px-5 py-4">
              <div className="h-4 bg-neutral-200 rounded w-1/4 mb-2" />
              <div className="h-3 bg-neutral-100 rounded w-1/3" />
            </div>
          ))}
        </div>
      ) : members.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <Users className="h-12 w-12 text-muted-foreground/40 mb-4" />
            <h3 className="text-lg font-medium">No team members</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Invite your first team member to get started.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {members.map((member) => (
            <div
              key={member.id}
              className="flex items-center gap-4 rounded-xl border border-border bg-card px-5 py-4"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary text-xs font-medium text-primary-foreground">
                {member.first_name[0]}{member.last_name[0]}
              </div>

              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-medium">
                    {member.first_name} {member.last_name}
                  </h3>
                  <Badge className={STATUS_COLORS[member.status] ?? STATUS_COLORS.inactive}>
                    {member.status}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground">{member.email}</p>
              </div>

              <div className="flex items-center gap-2">
                {member.roles.map((role) => (
                  <Badge key={role} variant="outline" className="gap-1">
                    <Shield className="h-3 w-3" />
                    {role}
                  </Badge>
                ))}
              </div>

              <button
                onClick={() => {
                  if (confirm(`Remove ${member.first_name} ${member.last_name}?`)) {
                    removeMember.mutate(member.id);
                  }
                }}
                className="text-muted-foreground hover:text-destructive transition-colors"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}

      {showInvite && <InviteDialog onClose={() => setShowInvite(false)} />}
    </motion.div>
  );
}

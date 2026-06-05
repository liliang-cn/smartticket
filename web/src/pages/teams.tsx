import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { UsersRound, Plus, Pencil, Trash2, ChevronRight, ChevronLeft, UserPlus, UserMinus } from "lucide-react";
import { toast } from "sonner";
import {
  useTeams,
  useTeamMembers,
  useCreateTeam,
  useUpdateTeam,
  useDeleteTeam,
  useAddTeamMember,
  useRemoveTeamMember,
  type Team,
  type TeamMember,
} from "@/features/teams/api";
import { useUsers } from "@/features/users/api";
import { apiError } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/misc";
import { useReveal } from "@/lib/use-reveal";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// --- Team form dialog --------------------------------------------------------

interface TeamFormState {
  name: string;
  description: string;
}

function emptyTeamForm(): TeamFormState {
  return { name: "", description: "" };
}

function TeamFormDialog({
  team,
  trigger,
}: {
  team?: Team;
  trigger?: React.ReactNode;
}) {
  const { t } = useTranslation("teams");
  const [open, setOpen] = useState(false);
  const isEdit = team != null;
  const create = useCreateTeam();
  const update = useUpdateTeam(team?.id ?? 0);
  const pending = isEdit ? update.isPending : create.isPending;

  const [form, setForm] = useState<TeamFormState>(emptyTeamForm);

  useEffect(() => {
    if (open) {
      setForm(
        isEdit && team
          ? { name: team.name, description: team.description }
          : emptyTeamForm()
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim()) {
      toast.error(t("validation.nameRequired"));
      return;
    }
    const payload = { name: form.name.trim(), description: form.description.trim() };
    try {
      if (isEdit) {
        await update.mutateAsync(payload);
        toast.success(t("toast.updated"));
      } else {
        await create.mutateAsync(payload);
        toast.success(t("toast.created"));
      }
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, isEdit ? t("toast.updateFailed") : t("toast.createFailed")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button>
            <Plus /> {t("newTeam")}
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? t("form.titleEdit") : t("form.titleCreate")}</DialogTitle>
          <DialogDescription>{t("form.description")}</DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="t-name">{t("form.name")}</Label>
            <Input
              id="t-name"
              placeholder={t("form.namePlaceholder")}
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="t-desc">{t("form.description_field")}</Label>
            <Input
              id="t-desc"
              placeholder={t("form.descriptionPlaceholder")}
              value={form.description}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
            />
          </div>

          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button type="submit" disabled={pending}>
              {pending
                ? isEdit ? t("form.savingPending") : t("form.creatingPending")
                : isEdit ? t("form.submitEdit") : t("form.submitCreate")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// --- Add member dialog -------------------------------------------------------

function AddMemberDialog({
  teamId,
  teamName,
  existingMemberIds,
}: {
  teamId: number;
  teamName: string;
  existingMemberIds: Set<number>;
}) {
  const { t } = useTranslation("teams");
  const [open, setOpen] = useState(false);
  const [selectedUserId, setSelectedUserId] = useState<string>("");
  const addMember = useAddTeamMember(teamId);

  // Fetch users (first page, no search — good enough for a picker)
  const { data: usersPage } = useUsers({ page: 1, page_size: 100 });
  const availableUsers = (usersPage?.items ?? []).filter(
    (u) => !existingMemberIds.has(u.id as unknown as number)
  );

  async function handleAdd() {
    if (!selectedUserId) return;
    try {
      await addMember.mutateAsync(Number(selectedUserId));
      toast.success(t("toast.memberAdded"));
      setSelectedUserId("");
      setOpen(false);
    } catch (err) {
      toast.error(apiError(err, t("toast.addMemberFailed")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="secondary" size="sm">
          <UserPlus />
          {t("members.addMember")}
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t("members.addDialog.title")}</DialogTitle>
          <DialogDescription>
            {t("members.addDialog.description", { teamName })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-1.5">
          <Label>{t("members.addDialog.userLabel")}</Label>
          <Select value={selectedUserId} onValueChange={setSelectedUserId}>
            <SelectTrigger>
              <SelectValue placeholder={t("members.addDialog.selectPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              {availableUsers.map((u) => (
                <SelectItem key={u.id} value={String(u.id)}>
                  {u.first_name && u.last_name
                    ? `${u.first_name} ${u.last_name} (${u.email})`
                    : u.email}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="ghost">
              {t("actions.cancel", { ns: "common" })}
            </Button>
          </DialogClose>
          <Button onClick={handleAdd} disabled={!selectedUserId || addMember.isPending}>
            {addMember.isPending
              ? t("members.addDialog.submitting")
              : t("members.addDialog.submit")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// --- Members panel -----------------------------------------------------------

function MembersPanel({
  team,
  onClose,
}: {
  team: Team;
  onClose: () => void;
}) {
  const { t } = useTranslation("teams");
  const { data: members, isLoading } = useTeamMembers(team.id);
  const removeMember = useRemoveTeamMember(team.id);

  const memberIdSet = new Set((members ?? []).map((m) => m.id as unknown as number));

  async function handleRemove(member: TeamMember) {
    try {
      await removeMember.mutateAsync(member.id as unknown as number);
      toast.success(t("toast.memberRemoved"));
    } catch (err) {
      toast.error(apiError(err, t("toast.removeMemberFailed")));
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" onClick={onClose}>
            <ChevronLeft />
          </Button>
          <h2 className="text-lg font-semibold">
            {t("members.title", { name: team.name })}
          </h2>
        </div>
        <AddMemberDialog
          teamId={team.id}
          teamName={team.name}
          existingMemberIds={memberIdSet}
        />
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("members.table.name")}</th>
              <th className="px-4 py-3 font-medium">{t("members.table.email")}</th>
              <th className="px-4 py-3 font-medium">{t("members.table.role")}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 4 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : members && members.length > 0 ? (
              members.map((member) => (
                <tr
                  key={member.id}
                  className="border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5 font-medium text-foreground">
                    {member.first_name && member.last_name
                      ? `${member.first_name} ${member.last_name}`
                      : member.username || member.email}
                  </td>
                  <td className="px-4 py-3.5 text-muted-foreground">{member.email}</td>
                  <td className="px-4 py-3.5">
                    <Badge tone="neutral" className="capitalize">
                      {member.role}
                    </Badge>
                  </td>
                  <td className="px-2 py-3.5 text-right">
                    <Button
                      variant="ghost"
                      size="icon"
                      title={t("members.remove")}
                      disabled={removeMember.isPending}
                      onClick={() => handleRemove(member)}
                    >
                      <UserMinus />
                    </Button>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={4} className="px-4 py-12 text-center">
                  <UsersRound className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    {t("members.empty")}
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>
    </div>
  );
}

// --- Page --------------------------------------------------------------------

export function TeamsPage() {
  const { t } = useTranslation("teams");
  const { data: teams, isLoading } = useTeams();
  const deleteTeam = useDeleteTeam();
  const ref = useReveal<HTMLDivElement>();

  const [toDelete, setToDelete] = useState<{ id: number; name: string } | null>(null);
  const [selectedTeam, setSelectedTeam] = useState<Team | null>(null);

  async function confirmDelete() {
    if (!toDelete) return;
    try {
      await deleteTeam.mutateAsync(toDelete.id);
      toast.success(t("toast.deleted"));
      setToDelete(null);
      if (selectedTeam?.id === toDelete.id) setSelectedTeam(null);
    } catch (err) {
      toast.error(apiError(err, t("toast.deleteFailed")));
    }
  }

  if (selectedTeam) {
    return (
      <div ref={ref} className="w-full">
        <MembersPanel
          team={selectedTeam}
          onClose={() => setSelectedTeam(null)}
        />
      </div>
    );
  }

  return (
    <div ref={ref} className="w-full">
      <div data-reveal className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {t("page.eyebrow")}
          </div>
          <h1 className="mt-1 text-3xl">{t("page.title")}</h1>
        </div>
        <TeamFormDialog />
      </div>

      <Card data-reveal className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("table.name")}</th>
              <th className="px-4 py-3 font-medium">{t("table.description")}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 4 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 3 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : teams && teams.length > 0 ? (
              teams.map((team) => (
                <tr
                  key={team.id}
                  className="cursor-pointer border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                  onClick={() => setSelectedTeam(team)}
                >
                  <td className="px-4 py-3.5 font-medium text-foreground">
                    <div className="flex items-center gap-2">
                      {team.name}
                      <ChevronRight className="size-4 text-muted-foreground/60" />
                    </div>
                  </td>
                  <td className="px-4 py-3.5 text-muted-foreground">
                    {team.description || (
                      <span className="text-muted-foreground/40">—</span>
                    )}
                  </td>
                  <td className="px-2 py-3.5">
                    <div
                      className="flex items-center justify-end gap-1"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <TeamFormDialog
                        team={team}
                        trigger={
                          <Button variant="ghost" size="icon" title={t("form.titleEdit")}>
                            <Pencil />
                          </Button>
                        }
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        title={t("deleteDialog.title")}
                        onClick={() => setToDelete({ id: team.id, name: team.name })}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={3} className="px-4 py-16 text-center">
                  <UsersRound className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">{t("empty")}</p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      <Dialog open={toDelete != null} onOpenChange={(v) => !v && setToDelete(null)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("deleteDialog.title")}</DialogTitle>
            <DialogDescription>
              {t("deleteDialog.description", { name: toDelete?.name })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost">
                {t("actions.cancel", { ns: "common" })}
              </Button>
            </DialogClose>
            <Button
              type="button"
              variant="destructive"
              disabled={deleteTeam.isPending}
              onClick={confirmDelete}
            >
              {deleteTeam.isPending ? t("deleteDialog.confirmPending") : t("deleteDialog.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

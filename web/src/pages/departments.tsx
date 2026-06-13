import { useState } from "react";
import { Network, Plus, Pencil, Trash2, ChevronRight } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import {
  useDepartments,
  useCreateDepartment,
  useUpdateDepartment,
  useDeleteDepartment,
  type Department,
} from "@/features/departments/api";
import { useUsers } from "@/features/users/api";
import { apiError } from "@/lib/api";
import { Card } from "@/components/ui/card";
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
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// ---- Tree helpers ----------------------------------------------------------

interface TreeNode {
  dept: Department;
  depth: number;
}

function buildTree(departments: Department[]): TreeNode[] {
  const byParent = new Map<number | null, Department[]>();
  for (const d of departments) {
    const key = d.parent_id ?? null;
    if (!byParent.has(key)) byParent.set(key, []);
    byParent.get(key)!.push(d);
  }

  const result: TreeNode[] = [];

  function walk(parentId: number | null, depth: number) {
    const children = byParent.get(parentId) ?? [];
    // Sort alphabetically for stability
    children.sort((a, b) => a.name.localeCompare(b.name));
    for (const child of children) {
      result.push({ dept: child, depth });
      walk(child.id, depth + 1);
    }
  }

  walk(null, 0);
  return result;
}

function managerName(dept: Department): string {
  if (!dept.manager) return "—";
  const full = `${dept.manager.first_name} ${dept.manager.last_name}`.trim();
  return full || dept.manager.username;
}

// ---- Create / Edit dialog --------------------------------------------------

const NONE = "__none__";

interface DeptFormDialogProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  /** When set, we are editing an existing department. */
  editing?: Department;
  /** Preset parent_id (used for "Add child" action). */
  presetParentId?: number | null;
  departments: Department[];
}

function DeptFormDialog({
  open,
  onOpenChange,
  editing,
  presetParentId,
  departments,
}: DeptFormDialogProps) {
  const { t } = useTranslation("departments");
  const create = useCreateDepartment();
  const update = useUpdateDepartment();

  const [name, setName] = useState(editing?.name ?? "");
  const [parentId, setParentId] = useState<string>(
    editing != null
      ? editing.parent_id != null
        ? String(editing.parent_id)
        : NONE
      : presetParentId != null
      ? String(presetParentId)
      : NONE
  );
  const [managerId, setManagerId] = useState<string>(
    editing?.manager_id != null ? String(editing.manager_id) : NONE
  );

  // Fetch all users to populate manager select (admin endpoint, same auth)
  const { data: usersPage } = useUsers({ page: 1, page_size: 200 });
  const staffUsers = (usersPage?.items ?? []).filter(
    (u) => u.role !== "customer" && u.is_active
  );

  // Reset form when dialog opens
  function handleOpenChange(v: boolean) {
    if (v) {
      setName(editing?.name ?? "");
      setParentId(
        editing != null
          ? editing.parent_id != null
            ? String(editing.parent_id)
            : NONE
          : presetParentId != null
          ? String(presetParentId)
          : NONE
      );
      setManagerId(
        editing?.manager_id != null ? String(editing.manager_id) : NONE
      );
    }
    onOpenChange(v);
  }

  // Departments selectable as parent (exclude self + descendants to avoid cycles)
  const selfId = editing?.id;
  const parentOptions = departments.filter((d) => {
    if (selfId == null) return true;
    return d.id !== selfId;
  });

  const isPending = create.isPending || update.isPending;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      toast.error(t("validation.nameRequired"));
      return;
    }

    const payload = {
      name: name.trim(),
      parent_id: parentId !== NONE ? Number(parentId) : null,
      manager_id: managerId !== NONE ? Number(managerId) : null,
    };

    try {
      if (editing) {
        await update.mutateAsync({ id: editing.id, ...payload });
        toast.success(t("toast.updated"));
      } else {
        await create.mutateAsync(payload);
        toast.success(t("toast.created"));
      }
      handleOpenChange(false);
    } catch (err) {
      // Check for cycle error from backend
      const errMsg = apiError(err, "");
      if (errMsg.toLowerCase().includes("cycle")) {
        toast.error(t("toast.cycleError"));
      } else {
        toast.error(
          editing
            ? apiError(err, t("toast.updateFailed"))
            : apiError(err, t("toast.createFailed"))
        );
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {editing ? t("form.editTitle") : t("form.createTitle")}
          </DialogTitle>
          <DialogDescription>
            {editing ? t("form.editDescription") : t("form.createDescription")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="dept-name">{t("form.name")}</Label>
            <Input
              id="dept-name"
              placeholder={t("form.namePlaceholder")}
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={200}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="dept-parent">{t("form.parent")}</Label>
            <Select value={parentId} onValueChange={setParentId}>
              <SelectTrigger id="dept-parent">
                <SelectValue placeholder={t("form.parentNone")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NONE}>{t("form.parentNone")}</SelectItem>
                {parentOptions.map((d) => (
                  <SelectItem key={d.id} value={String(d.id)}>
                    {d.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="dept-manager">{t("form.manager")}</Label>
            <Select value={managerId} onValueChange={setManagerId}>
              <SelectTrigger id="dept-manager">
                <SelectValue placeholder={t("form.managerNone")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NONE}>{t("form.managerNone")}</SelectItem>
                {staffUsers.map((u) => {
                  const label =
                    `${u.first_name} ${u.last_name}`.trim() || u.username;
                  return (
                    <SelectItem key={u.id} value={String(u.id)}>
                      {label}
                    </SelectItem>
                  );
                })}
              </SelectContent>
            </Select>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => handleOpenChange(false)}
              disabled={isPending}
            >
              {t("actions.cancel", { ns: "common" })}
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending
                ? editing
                  ? t("form.savingEdit")
                  : t("form.creating")
                : editing
                ? t("form.submitEdit")
                : t("form.submitCreate")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ---- Delete confirmation dialog -------------------------------------------

function DeleteDialog({
  dept,
  open,
  onClose,
}: {
  dept: Department | null;
  open: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation("departments");
  const del = useDeleteDepartment();

  async function handleConfirm() {
    if (!dept) return;
    try {
      await del.mutateAsync(dept.id);
      toast.success(t("toast.deleted"));
      onClose();
    } catch (err) {
      toast.error(apiError(err, t("toast.deleteFailed")));
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t("deleteDialog.title")}</DialogTitle>
          <DialogDescription>
            {t("deleteDialog.description", { name: dept?.name })}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            type="button"
            variant="ghost"
            onClick={onClose}
            disabled={del.isPending}
          >
            {t("actions.cancel", { ns: "common" })}
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={del.isPending}
            onClick={handleConfirm}
          >
            {del.isPending
              ? t("deleteDialog.confirmPending")
              : t("deleteDialog.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---- Page ------------------------------------------------------------------

export function DepartmentsPage() {
  const { t } = useTranslation("departments");
  const { data: departments, isLoading } = useDepartments();
  const ref = useReveal<HTMLDivElement>();

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Department | undefined>(undefined);
  const [presetParent, setPresetParent] = useState<number | null>(null);
  const [toDelete, setToDelete] = useState<Department | null>(null);

  function openCreate(parentId?: number | null) {
    setEditing(undefined);
    setPresetParent(parentId ?? null);
    setFormOpen(true);
  }

  function openEdit(dept: Department) {
    setEditing(dept);
    setPresetParent(null);
    setFormOpen(true);
  }

  const tree = buildTree(departments ?? []);

  return (
    <div ref={ref} className="w-full">
      <div data-reveal className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {t("page.eyebrow")}
          </div>
          <h1 className="mt-1 text-3xl">{t("page.title")}</h1>
        </div>
        <Button onClick={() => openCreate()}>
          <Plus /> {t("newDepartment")}
        </Button>
      </div>

      <Card data-reveal className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("table.name")}</th>
              <th className="px-4 py-3 font-medium">{t("table.manager")}</th>
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
            ) : tree.length > 0 ? (
              tree.map(({ dept, depth }) => (
                <tr
                  key={dept.id}
                  className="border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5">
                    <div
                      className="flex items-center gap-1"
                      style={{ paddingLeft: `${depth * 1.25}rem` }}
                    >
                      {depth > 0 && (
                        <ChevronRight className="size-3.5 shrink-0 text-muted-foreground/50" />
                      )}
                      <span className="font-medium text-foreground">
                        {dept.name}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-3.5 text-sm text-muted-foreground">
                    {managerName(dept)}
                  </td>
                  <td className="px-2 py-3.5 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        title={t("actions.addChildTitle")}
                        onClick={() => openCreate(dept.id)}
                      >
                        <Plus className="size-4" />
                        {t("actions.addChild")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        title={t("actions.editTitle")}
                        onClick={() => openEdit(dept)}
                      >
                        <Pencil className="size-4" />
                        {t("actions.edit")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        title={t("actions.deleteTitle")}
                        onClick={() => setToDelete(dept)}
                      >
                        <Trash2 className="size-4" />
                        {t("actions.delete")}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={3} className="px-4 py-16 text-center">
                  <Network className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    {t("empty")}
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {/* Create / Edit dialog */}
      <DeptFormDialog
        open={formOpen}
        onOpenChange={setFormOpen}
        editing={editing}
        presetParentId={presetParent}
        departments={departments ?? []}
      />

      {/* Delete confirmation */}
      <DeleteDialog
        dept={toDelete}
        open={toDelete != null}
        onClose={() => setToDelete(null)}
      />
    </div>
  );
}

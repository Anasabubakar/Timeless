"use client";

import { useState } from "react";
import { Plus, Search, Filter, MoreHorizontal } from "lucide-react";
import { useCompanies, useCreateCompany } from "@/queries/companies";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import type { Company } from "@/types";

export default function CompaniesPage() {
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const { data, isLoading } = useCompanies(1, 50, search);

  const companies: Company[] = (data as any)?.companies || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Companies</h1>
          <p className="text-sm text-muted-foreground">
            Track and manage potential sponsor companies
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex h-8 items-center gap-1.5 rounded-lg bg-primary px-3 text-xs font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-3.5 w-3.5" />
          Add Company
        </button>
      </div>

      <div className="flex items-center gap-2">
        <div className="flex h-8 flex-1 items-center gap-2 rounded-lg border border-input bg-background px-3">
          <Search className="h-3.5 w-3.5 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search companies..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
        </div>
        <button className="flex h-8 items-center gap-1.5 rounded-lg border border-input px-3 text-xs text-muted-foreground hover:bg-accent">
          <Filter className="h-3.5 w-3.5" />
          Filter
        </button>
      </div>

      {isLoading ? (
        <div className="h-64 animate-pulse rounded-xl border border-border bg-muted/30" />
      ) : companies.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-12">
          <p className="text-sm text-muted-foreground">No companies found</p>
          <button
            onClick={() => setShowCreate(true)}
            className="mt-2 text-xs font-medium text-primary hover:underline"
          >
            Add your first company
          </button>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-border">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Company</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Industry</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Employees</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Score</th>
                <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Status</th>
                <th className="w-10 px-4 py-2.5"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {companies.map((company) => (
                <tr key={company.id} className="transition-colors hover:bg-muted/20">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted text-xs font-medium">
                        {company.name[0]}
                      </div>
                      <div>
                        <p className="text-sm font-medium">{company.name}</p>
                        <p className="text-xs text-muted-foreground">{company.domain || company.website}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">
                    {company.industry?.name || "—"}
                  </td>
                  <td className="px-4 py-3 text-sm text-muted-foreground">
                    {company.employee_count || "—"}
                  </td>
                  <td className="px-4 py-3">
                    {company.score ? (
                      <div className="flex items-center gap-2">
                        <div className="h-1.5 w-12 overflow-hidden rounded-full bg-muted">
                          <div
                            className="h-full rounded-full bg-emerald-500"
                            style={{ width: `${company.score}%` }}
                          />
                        </div>
                        <span className="text-xs font-medium">{company.score}</span>
                      </div>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                        company.status === "active"
                          ? "bg-emerald-50 text-emerald-700"
                          : "bg-neutral-100 text-neutral-600"
                      }`}
                    >
                      {company.status}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <button className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-accent">
                      <MoreHorizontal className="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <CreateCompanyDialog open={showCreate} onOpenChange={setShowCreate} />
    </div>
  );
}

function CreateCompanyDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const createCompany = useCreateCompany();
  const [form, setForm] = useState({
    name: "",
    domain: "",
    website: "",
    description: "",
    employee_count: "",
    headquarters: "",
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createCompany.mutate(
      {
        name: form.name,
        domain: form.domain || undefined,
        website: form.website || undefined,
        description: form.description || undefined,
        employee_count: form.employee_count || undefined,
        headquarters: form.headquarters || undefined,
      },
      {
        onSuccess: () => {
          onOpenChange(false);
          setForm({ name: "", domain: "", website: "", description: "", employee_count: "", headquarters: "" });
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Company</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 mt-4">
          <input
            placeholder="Company name"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            className="h-9 w-full rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
            required
          />
          <div className="grid grid-cols-2 gap-3">
            <input
              placeholder="Domain (e.g. stripe.com)"
              value={form.domain}
              onChange={(e) => setForm({ ...form, domain: e.target.value })}
              className="h-9 rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
            />
            <input
              placeholder="Website URL"
              value={form.website}
              onChange={(e) => setForm({ ...form, website: e.target.value })}
              className="h-9 rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
            />
          </div>
          <textarea
            placeholder="Description"
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
            className="min-h-[60px] w-full rounded-[10px] border border-neutral-200 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
          />
          <div className="grid grid-cols-2 gap-3">
            <input
              placeholder="Employee count"
              value={form.employee_count}
              onChange={(e) => setForm({ ...form, employee_count: e.target.value })}
              className="h-9 rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
            />
            <input
              placeholder="Headquarters"
              value={form.headquarters}
              onChange={(e) => setForm({ ...form, headquarters: e.target.value })}
              className="h-9 rounded-[10px] border border-neutral-200 px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10"
            />
          </div>
          <DialogFooter>
            <button
              type="button"
              onClick={() => onOpenChange(false)}
              className="h-8 rounded-lg border border-neutral-200 px-3 text-xs font-medium hover:bg-neutral-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={createCompany.isPending}
              className="h-8 rounded-lg bg-neutral-900 px-3 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50"
            >
              {createCompany.isPending ? "Adding..." : "Add Company"}
            </button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

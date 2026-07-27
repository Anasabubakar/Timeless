"use client";

import { useState } from "react";
import { Plus, Search, Filter, Mail, MoreHorizontal } from "lucide-react";
import { useContacts, useCreateContact, useDeleteContact } from "@/queries/contacts";
import { Avatar } from "@/components/ui/avatar";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import type { Contact } from "@/types";
import { motion } from "motion/react";
import { useNewParam } from "@/hooks/use-new-param";

export default function ContactsPage() {
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  useNewParam(() => setShowCreate(true));
  const { data, isLoading } = useContacts({ search });
  const deleteContact = useDeleteContact();

  const contacts: Contact[] = (data as any)?.contacts || [];

  return (
    <motion.div className="space-y-6" initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.3 }}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Contacts</h1>
          <p className="text-sm text-muted-foreground">
            Decision makers and key contacts at sponsor companies
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex h-8 items-center gap-1.5 rounded-lg bg-primary px-3 text-xs font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-3.5 w-3.5" />
          Add Contact
        </button>
      </div>

      <div className="flex items-center gap-2">
        <div className="flex h-8 flex-1 items-center gap-2 rounded-lg border border-input bg-background px-3">
          <Search className="h-3.5 w-3.5 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search contacts..."
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
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-28 animate-pulse rounded-xl border border-border bg-muted/30" />
          ))}
        </div>
      ) : contacts.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-12">
          <p className="text-sm text-muted-foreground">No contacts found</p>
          <button
            onClick={() => setShowCreate(true)}
            className="mt-2 text-xs font-medium text-primary hover:underline"
          >
            Add your first contact
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {contacts.map((contact) => (
            <div
              key={contact.id}
              className="flex items-start gap-3 rounded-xl border border-border bg-card p-4 transition-shadow hover:shadow-sm"
            >
              <Avatar
                alt={`${contact.first_name} ${contact.last_name}`}
                size="md"
              />
              <div className="min-w-0 flex-1">
                <div className="flex items-start justify-between">
                  <div>
                    <p className="text-sm font-medium">
                      {contact.first_name} {contact.last_name}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {contact.title}
                      {contact.company && ` at ${contact.company.name}`}
                    </p>
                  </div>
                  <button
                    onClick={() => deleteContact.mutate(contact.id)}
                    className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-accent"
                  >
                    <MoreHorizontal className="h-4 w-4" />
                  </button>
                </div>
                {contact.email && (
                  <div className="mt-2 flex items-center gap-1 text-[11px] text-muted-foreground">
                    <Mail className="h-3 w-3" />
                    {contact.email}
                  </div>
                )}
                {contact.last_contacted_at && (
                  <p className="mt-1.5 text-[10px] text-muted-foreground/60">
                    Last contacted {new Date(contact.last_contacted_at).toLocaleDateString()}
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      <CreateContactDialog open={showCreate} onOpenChange={setShowCreate} />
    </motion.div>
  );
}

function CreateContactDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const createContact = useCreateContact();
  const [form, setForm] = useState({
    first_name: "",
    last_name: "",
    email: "",
    title: "",
    phone: "",
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createContact.mutate(form, {
      onSuccess: () => {
        onOpenChange(false);
        setForm({ first_name: "", last_name: "", email: "", title: "", phone: "" });
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Contact</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 mt-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <input
              placeholder="First name"
              value={form.first_name}
              onChange={(e) => setForm({ ...form, first_name: e.target.value })}
              className="h-9 rounded-[10px] border border-neutral-200 bg-background px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:focus:ring-neutral-100/10"
              required
            />
            <input
              placeholder="Last name"
              value={form.last_name}
              onChange={(e) => setForm({ ...form, last_name: e.target.value })}
              className="h-9 rounded-[10px] border border-neutral-200 bg-background px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:focus:ring-neutral-100/10"
              required
            />
          </div>
          <input
            placeholder="Email"
            type="email"
            value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
            className="h-9 w-full rounded-[10px] border border-neutral-200 bg-background px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:focus:ring-neutral-100/10"
          />
          <input
            placeholder="Job title"
            value={form.title}
            onChange={(e) => setForm({ ...form, title: e.target.value })}
            className="h-9 w-full rounded-[10px] border border-neutral-200 bg-background px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:focus:ring-neutral-100/10"
          />
          <input
            placeholder="Phone"
            value={form.phone}
            onChange={(e) => setForm({ ...form, phone: e.target.value })}
            className="h-9 w-full rounded-[10px] border border-neutral-200 bg-background px-3 text-sm outline-none focus:ring-2 focus:ring-neutral-900/10 dark:border-neutral-700 dark:focus:ring-neutral-100/10"
          />
          <DialogFooter>
            <button
              type="button"
              onClick={() => onOpenChange(false)}
              className="h-8 rounded-lg border border-neutral-200 px-3 text-xs font-medium hover:bg-neutral-50 dark:border-neutral-700 dark:hover:bg-neutral-800"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={createContact.isPending}
              className="h-8 rounded-lg bg-neutral-900 px-3 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200"
            >
              {createContact.isPending ? "Creating..." : "Create"}
            </button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

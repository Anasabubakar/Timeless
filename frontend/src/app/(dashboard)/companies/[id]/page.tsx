"use client";

import { useParams } from "next/navigation";
import { ArrowLeft, Globe, Users, Building2, MapPin, Linkedin, Mail } from "lucide-react";
import Link from "next/link";
import { useCompany } from "@/queries/companies";
import { useSponsors } from "@/queries/sponsors";
import { useContacts } from "@/queries/contacts";
import type { Sponsor, Contact } from "@/types";

export default function CompanyDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: company, isLoading } = useCompany(id);
  const { data: sponsorsData } = useSponsors({ limit: 20 });
  const { data: contactsData } = useContacts({ limit: 50 });

  const sponsors: Sponsor[] = ((sponsorsData as any)?.sponsors || []).filter(
    (s: Sponsor) => s.company_id === id
  );
  const contacts: Contact[] = ((contactsData as any)?.contacts || []).filter(
    (c: Contact) => c.company_id === id
  );

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="h-48 animate-pulse rounded-xl bg-muted/30" />
      </div>
    );
  }

  if (!company) {
    return (
      <div className="flex flex-col items-center py-12">
        <p className="text-sm text-muted-foreground">Company not found</p>
        <Link href="/companies" className="mt-2 text-xs text-primary hover:underline">
          Back to companies
        </Link>
      </div>
    );
  }

  const c = company as any;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          href="/companies"
          className="flex h-7 w-7 items-center justify-center rounded-md hover:bg-accent"
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted text-sm font-semibold">
          {c.name[0]}
        </div>
        <div className="flex-1">
          <h1 className="text-xl font-semibold tracking-tight">{c.name}</h1>
          {c.industry && (
            <p className="text-xs text-muted-foreground">{c.industry.name}</p>
          )}
        </div>
        {c.score && (
          <div className="flex items-center gap-2">
            <div className="h-2 w-16 overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-emerald-500"
                style={{ width: `${c.score}%` }}
              />
            </div>
            <span className="text-xs font-medium">{c.score}/100</span>
          </div>
        )}
      </div>

      <div className="grid grid-cols-3 gap-6">
        <div className="col-span-2 space-y-6">
          {c.description && (
            <div className="rounded-xl border border-border p-4">
              <h3 className="text-xs font-medium text-muted-foreground mb-2">About</h3>
              <p className="text-sm leading-relaxed">{c.description}</p>
            </div>
          )}

          <div className="rounded-xl border border-border p-4">
            <h3 className="text-xs font-medium text-muted-foreground mb-3">
              Sponsorship History ({sponsors.length})
            </h3>
            {sponsors.length === 0 ? (
              <p className="text-xs text-muted-foreground">No sponsorships yet</p>
            ) : (
              <div className="space-y-2">
                {sponsors.map((s) => (
                  <div
                    key={s.id}
                    className="flex items-center justify-between rounded-lg bg-muted/30 px-3 py-2"
                  >
                    <div>
                      <p className="text-xs font-medium">
                        {s.campaign?.name || "Campaign"}
                      </p>
                      <p className="text-[10px] text-muted-foreground">
                        Stage: {s.stage}
                      </p>
                    </div>
                    <div className="text-right">
                      {s.deal_value && (
                        <p className="text-xs font-medium">
                          ${s.deal_value.toLocaleString()}
                        </p>
                      )}
                      {s.tier && (
                        <span className="text-[10px] text-muted-foreground">{s.tier}</span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="rounded-xl border border-border p-4">
            <h3 className="text-xs font-medium text-muted-foreground mb-3">
              Contacts ({contacts.length})
            </h3>
            {contacts.length === 0 ? (
              <p className="text-xs text-muted-foreground">No contacts linked</p>
            ) : (
              <div className="space-y-2">
                {contacts.map((contact) => (
                  <div
                    key={contact.id}
                    className="flex items-center justify-between rounded-lg bg-muted/30 px-3 py-2"
                  >
                    <div className="flex items-center gap-2">
                      <div className="flex h-7 w-7 items-center justify-center rounded-full bg-muted text-[10px] font-medium">
                        {contact.first_name[0]}{contact.last_name[0]}
                      </div>
                      <div>
                        <p className="text-xs font-medium">
                          {contact.first_name} {contact.last_name}
                        </p>
                        <p className="text-[10px] text-muted-foreground">{contact.title}</p>
                      </div>
                    </div>
                    {contact.email && (
                      <a
                        href={`mailto:${contact.email}`}
                        className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground hover:bg-accent"
                      >
                        <Mail className="h-3 w-3" />
                      </a>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="space-y-4">
          <div className="rounded-xl border border-border p-4 space-y-3">
            <h3 className="text-xs font-medium text-muted-foreground">Details</h3>
            {c.website && (
              <InfoRow icon={Globe} label="Website" value={c.website} isLink />
            )}
            {c.domain && (
              <InfoRow icon={Globe} label="Domain" value={c.domain} />
            )}
            {c.employee_count && (
              <InfoRow icon={Users} label="Employees" value={c.employee_count} />
            )}
            {c.headquarters && (
              <InfoRow icon={MapPin} label="HQ" value={c.headquarters} />
            )}
            {c.annual_revenue && (
              <InfoRow icon={Building2} label="Revenue" value={c.annual_revenue} />
            )}
            {c.linkedin_url && (
              <InfoRow icon={Linkedin} label="LinkedIn" value="View Profile" isLink href={c.linkedin_url} />
            )}
            {c.founded_year && (
              <InfoRow icon={Building2} label="Founded" value={String(c.founded_year)} />
            )}
          </div>

          {c.tags && c.tags.length > 0 && (
            <div className="rounded-xl border border-border p-4">
              <h3 className="text-xs font-medium text-muted-foreground mb-2">Tags</h3>
              <div className="flex flex-wrap gap-1">
                {c.tags.map((tag: string) => (
                  <span
                    key={tag}
                    className="rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-medium"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          )}

          <div className="rounded-xl border border-border p-4">
            <h3 className="text-xs font-medium text-muted-foreground mb-1">Status</h3>
            <span
              className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                c.status === "active"
                  ? "bg-emerald-50 text-emerald-700"
                  : "bg-neutral-100 text-neutral-600"
              }`}
            >
              {c.status}
            </span>
            {c.source && (
              <p className="mt-2 text-[10px] text-muted-foreground">
                Source: {c.source}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function InfoRow({
  icon: Icon,
  label,
  value,
  isLink,
  href,
}: {
  icon: any;
  label: string;
  value: string;
  isLink?: boolean;
  href?: string;
}) {
  return (
    <div className="flex items-center gap-2">
      <Icon className="h-3.5 w-3.5 text-muted-foreground" />
      <span className="text-[11px] text-muted-foreground w-16">{label}</span>
      {isLink ? (
        <a
          href={href || value}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs font-medium text-primary hover:underline truncate"
        >
          {value}
        </a>
      ) : (
        <span className="text-xs font-medium truncate">{value}</span>
      )}
    </div>
  );
}

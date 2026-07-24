"use client";

import { useState, useCallback } from "react";
import { UploadCloud, Check, ArrowRight, Database, Users, Building2, User } from "lucide-react";
import { api } from "@/lib/api";

const ENTITIES = [
  { key: "sponsors", label: "Sponsors", icon: Database },
  { key: "contacts", label: "Contacts", icon: User },
  { key: "companies", label: "Companies", icon: Building2 },
];

export default function ImportPage() {
  const [entity, setEntity] = useState("sponsors");
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<Record<string, string>[]>([]);
  const [results, setResults] = useState<{ count: number; errors: number; results?: any[] } | null>(null);
  const [loading, setLoading] = useState(false);

  const handleFile = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const f = e.target.files?.[0];
      if (!f) return;
      setFile(f);
      setResults(null);
      const reader = new FileReader();
      reader.onload = () => {
        const text = reader.result as string;
        const lines = text.split("\n").slice(0, 6);
        const headers = lines[0]?.split(",").map((s) => s.trim()) ?? [];
        const rows = lines.slice(1).map((line) => {
          const values = line.split(",").map((s) => s.trim());
          const obj: Record<string, string> = {};
          headers.forEach((h, i) => (obj[h] = values[i] ?? ""));
          return obj;
        });
        setPreview(rows);
      };
      reader.readAsText(f);
    },
    []
  );

  const handleImport = async () => {
    if (!file) return;
    setLoading(true);
    setResults(null);
    try {
      const formData = new FormData();
      formData.append("file", file);
      formData.append("entity", entity);
      const data = await api.post(`/import/${entity}`, formData);
      setResults(data as any);
    } catch (e: any) {
      setResults({ count: 0, errors: 1, results: [{ error: e.message || "Import failed" }] });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Import Data</h1>
        <p className="text-sm text-muted-foreground">Upload CSV files to import sponsors, contacts, or companies.</p>
      </div>
      <div className="rounded-xl border border-border bg-card p-5 space-y-6">
        <div>
          <label className="text-[11px] font-medium text-muted-foreground mb-2 block">Select Entity</label>
          <div className="flex gap-2">
            {ENTITIES.map((eItem) => {
              const Icon = eItem.icon;
              return (
                <button
                  key={eItem.key}
                  onClick={() => setEntity(eItem.key)}
                  className={\`flex items-center gap-2 rounded-lg border px-3 py-2 text-xs font-medium transition-colors \${entity === eItem.key ? "border-neutral-900 bg-neutral-900 text-white dark:border-neutral-100 dark:bg-neutral-100 dark:text-neutral-900" : "border-neutral-200 hover:bg-neutral-50 dark:border-neutral-700 dark:hover:bg-neutral-800"}\`}
                >
                  <Icon className="h-3.5 w-3.5" />
                  {eItem.label}
                </button>
              );
            })}
          </div>
        </div>
        <div className="rounded-lg border border-dashed border-neutral-200 bg-neutral-50/40 dark:border-neutral-700 dark:bg-neutral-900/40 p-6 text-center transition-colors hover:bg-neutral-50 dark:hover:bg-neutral-900/60">
          <input type="file" accept=".csv" onChange={handleFile} className="hidden" id="csv-upload" />
          <label htmlFor="csv-upload" className="cursor-pointer flex flex-col items-center gap-3">
            <div className="h-10 w-10 rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center">
              <UploadCloud className="h-5 w-5 text-muted-foreground" />
            </div>
            <div className="space-y-0.5">
              <p className="text-sm font-medium">{file ? file.name : "Upload CSV file"}</p>
              <p className="text-xs text-muted-foreground">Click to browse or drag and drop</p>
            </div>
          </label>
        </div>
        {preview.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-xs font-medium">Preview (first 5 rows)</h4>
            <div className="overflow-x-auto rounded-lg border border-border">
              <table className="w-full text-xs">
                <thead className="bg-neutral-50 dark:bg-neutral-800">
                  <tr>
                    {Object.keys(preview[0] || {}).map((k) => (
                      <th key={k} className="text-left px-2 py-1.5 font-medium text-muted-foreground border-b border-border truncate max-w-[140px]">{k}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {preview.map((row, i) => (
                    <tr key={i} className="border-b border-border last:border-0">
                      {Object.values(row).map((v, j) => (
                        <td key={j} className="px-2 py-1.5 truncate max-w-[140px]">{v}</td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
        <div className="flex items-center gap-3 pt-2">
          <button onClick={handleImport} disabled={!file || loading} className="h-9 rounded-lg bg-neutral-900 px-4 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-40 dark:bg-neutral-100 dark:text-neutral-900 dark:hover:bg-neutral-200">{loading ? "Importing..." : \`Import \${entity}\`}</button>
        </div>
        {results && (
          <div className="rounded-xl border border-border bg-card p-4 space-y-3">
            <div className="flex items-center gap-2">
              <Check className="h-4 w-4 text-emerald-600" />
              <h4 className="text-sm font-medium">Import Complete</h4>
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div className="rounded-lg border border-border bg-neutral-50 dark:bg-neutral-900/40 p-3"><p className="text-[10px] text-muted-foreground">Inserted</p><p className="text-lg font-semibold">{results.count}</p></div>
              <div className="rounded-lg border border-border bg-neutral-50 dark:bg-neutral-900/40 p-3"><p className="text-[10px] text-muted-foreground">Errors</p><p className="text-lg font-semibold">{results.errors}</p></div>
              <div className="rounded-lg border border-border bg-neutral-50 dark:bg-neutral-900/40 p-3"><p className="text-[10px] text-muted-foreground">Entity</p><p className="text-sm font-medium truncate">{entity}</p></div>
            </div>
            {results.results && results.results.length > 0 && <div className="text-xs text-muted-foreground">Processed {results.results.length} rows successfully.</div>}
          </div>
        )}
      </div>
    </div>
  );
}

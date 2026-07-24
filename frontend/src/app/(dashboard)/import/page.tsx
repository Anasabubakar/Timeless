"use client";

import { useState, useCallback, useMemo } from "react";
import { Upload, FileUp, CheckCircle2, AlertCircle, ArrowRight } from "lucide-react";
import { api } from "@/lib/api";

export default function ImportPage() {
  const [entity, setEntity] = useState("sponsors");
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string[][]>([]);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ imported: number; errors: string[] } | null>(null);

  const handleFile = useCallback((f: File) => {
    setFile(f);
    setResult(null);
    const reader = new FileReader();
    reader.onload = (e) => {
      const text = e.target?.result as string;
      const lines = text.split(/\r?\n/).filter((l) => l.trim() !== "");
      const rows: string[][] = [];
      for (const line of lines) {
        const cols: string[] = [];
        let current = "";
        let inQuotes = false;
        for (let i = 0; i < line.length; i++) {
          const ch = line[i];
          if (ch === '"') {
            inQuotes = !inQuotes;
          } else if (ch === "," && !inQuotes) {
            cols.push(current);
            current = "";
          } else {
            current += ch;
          }
        }
        cols.push(current);
        rows.push(cols);
      }
      setPreview(rows.slice(0, 5));
    };
    reader.readAsText(f);
  }, []);

  const onDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      const f = e.dataTransfer.files?.[0];
      if (f && f.type === "text/csv") handleFile(f);
    },
    [handleFile]
  );

  const onImport = async () => {
    if (!file) return;
    setLoading(true);
    setResult(null);
    try {
      const formData = new FormData();
      formData.append("file", file);
      const data = await api.upload<{ imported: number; errors: string[] }>(
        `/import/${entity}`,
        formData
      );
      setResult(data);
    } catch (e: any) {
      setResult({ imported: 0, errors: [e.message || "Import failed"] });
    } finally {
      setLoading(false);
    }
  };

  const entities = useMemo(
    () => [
      { value: "sponsors", label: "Sponsors" },
      { value: "contacts", label: "Contacts" },
      { value: "companies", label: "Companies" },
    ],
    []
  );

  return (
    <div className="max-w-2xl mx-auto px-6 py-10 space-y-8">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Import CSV</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Upload a CSV file to bulk import records into SponsorOS.
        </p>
      </div>

      <div className="space-y-3">
        <label className="text-sm font-medium">Entity</label>
        <div className="flex gap-2">
          {entities.map((e) => (
            <button
              key={e.value}
              onClick={() => setEntity(e.value)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors ${
                entity === e.value
                  ? "bg-primary text-primary-foreground border-primary"
                  : "bg-card border-border text-muted-foreground hover:text-foreground hover:border-foreground/20"
              }`}
            >
              {e.label}
            </button>
          ))}
        </div>
      </div>

      <div
        onDragOver={(e) => e.preventDefault()}
        onDrop={onDrop}
        className="border-2 border-dashed border-border rounded-xl p-8 text-center hover:border-foreground/20 transition-colors bg-muted/20"
      >
        <div className="flex flex-col items-center gap-3">
          <div className="h-10 w-10 rounded-full bg-muted flex items-center justify-center">
            <Upload className="h-5 w-5 text-muted-foreground" />
          </div>
          <div className="text-sm font-medium">Drop your CSV file here</div>
          <p className="text-xs text-muted-foreground">or click to select a file</p>
          <input
            type="file"
            accept=".csv"
            className="hidden"
            id="csv-file"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) handleFile(f);
            }}
          />
          <label
            htmlFor="csv-file"
            className="inline-flex items-center gap-1.5 rounded-lg bg-foreground px-3 py-1.5 text-xs font-medium text-background hover:bg-foreground/90 transition-colors cursor-pointer"
          >
            <FileUp className="h-3.5 w-3.5" />
            Select CSV
          </label>
          {file && (
            <p className="text-xs text-muted-foreground mt-1">{file.name}</p>
          )}
        </div>
      </div>

      {preview.length > 0 && (
        <div className="space-y-2">
          <div className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Preview (first 5 rows)</div>
          <div className="overflow-x-auto rounded-lg border border-border">
            <table className="w-full text-xs">
              <thead className="bg-muted/40">
                <tr>
                  {preview[0].map((cell, i) => (
                    <th key={i} className="text-left px-3 py-2 font-medium text-muted-foreground whitespace-nowrap">
                      {cell || `Col ${i + 1}`}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {preview.slice(1).map((row, ri) => (
                  <tr key={ri} className="hover:bg-muted/20">
                    {row.map((cell, ci) => (
                      <td key={ci} className="px-3 py-2 text-foreground whitespace-nowrap truncate max-w-[160px]">
                        {cell || "—"}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <button
        onClick={onImport}
        disabled={!file || loading}
        className="inline-flex items-center gap-2 rounded-lg bg-neutral-900 px-4 py-2.5 text-sm font-medium text-white hover:bg-neutral-800 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
      >
        {loading ? (
          <span>Importing...</span>
        ) : (
          <>
            <ArrowRight className="h-4 w-4" />
            Import
          </>
        )}
      </button>

      {result && (
        <div className="rounded-xl border border-border bg-card p-5 space-y-3">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-5 w-5 text-emerald-500" />
            <span className="font-semibold text-sm">Import complete</span>
          </div>
          <p className="text-sm text-muted-foreground">
            Imported <strong className="text-foreground">{result.imported}</strong> records.
          </p>
          {result.errors.length > 0 && (
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm font-medium text-red-500">
                <AlertCircle className="h-4 w-4" />
                Errors
              </div>
              <ul className="text-xs text-muted-foreground list-disc pl-4 space-y-1">
                {result.errors.map((err, i) => (
                  <li key={i}>{err}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

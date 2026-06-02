import { useState } from "react";
import { Link } from "react-router-dom";
import { Loader2, Sparkles, FileText } from "lucide-react";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import { useKnowledgeAsk, type AskResult } from "@/features/knowledge/api";
import { apiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

export function AskAiDialog() {
  const { t } = useTranslation("knowledge");
  const [open, setOpen] = useState(false);
  const [question, setQuestion] = useState("");
  const [result, setResult] = useState<AskResult | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const ask = useKnowledgeAsk();

  const submit = () => {
    const q = question.trim();
    if (!q || ask.isPending) return;
    setErrorMsg(null);
    ask.mutate(
      { question: q },
      {
        onSuccess: (data) => {
          setResult(data);
        },
        onError: (err) => {
          const msg = apiError(err, t("ask_ai.error_default"));
          setErrorMsg(msg);
          toast.error(msg);
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="secondary">
          <Sparkles className="size-4" /> {t("ask_ai.trigger")}
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="size-4 text-primary" /> {t("ask_ai.dialog_title")}
          </DialogTitle>
          <DialogDescription>
            {t("ask_ai.dialog_description")}
          </DialogDescription>
        </DialogHeader>

        <Textarea
          rows={3}
          placeholder={t("ask_ai.question_placeholder")}
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
              e.preventDefault();
              submit();
            }
          }}
        />

        <div className="flex items-center justify-between">
          <span className="font-mono text-[11px] text-muted-foreground">
            {t("ask_ai.shortcut_hint")}
          </span>
          <Button onClick={submit} disabled={ask.isPending || !question.trim()}>
            {ask.isPending ? (
              <>
                <Loader2 className="size-4 animate-spin" /> {t("ask_ai.thinking")}
              </>
            ) : (
              <>
                <Sparkles className="size-4" /> {t("ask_ai.ask_button")}
              </>
            )}
          </Button>
        </div>

        {/* Results */}
        {ask.isPending && (
          <div className="flex items-center gap-2 rounded-lg border border-border bg-muted/30 px-4 py-3 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" /> {t("ask_ai.searching_message")}
          </div>
        )}

        {!ask.isPending && errorMsg && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-300">
            {errorMsg}
          </div>
        )}

        {!ask.isPending && !errorMsg && result && (
          <div className="max-h-[50vh] space-y-4 overflow-y-auto">
            <div className="rounded-lg border border-border bg-muted/30 px-4 py-3">
              <div className="mb-2 font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
                {t("ask_ai.answer_label")}
              </div>
              <p className="whitespace-pre-wrap text-sm leading-relaxed text-foreground">
                {result.answer}
              </p>
            </div>

            {result.citations.length > 0 && (
              <div>
                <div className="mb-2 font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
                  {t("ask_ai.sources_label")}
                </div>
                <ul className="space-y-1.5">
                  {result.citations.map((c, i) => (
                    <li key={`${c.article_id}-${i}`}>
                      <Link
                        to={`/knowledge/${c.article_id}`}
                        onClick={() => setOpen(false)}
                        className="group flex items-center justify-between gap-3 rounded-md border border-border/60 px-3 py-2 text-sm transition-colors hover:border-primary/40 hover:bg-accent/50"
                      >
                        <span className="flex min-w-0 items-center gap-2">
                          <FileText className="size-4 shrink-0 text-muted-foreground group-hover:text-primary" />
                          <span className="truncate text-foreground group-hover:text-primary">
                            {c.title}
                          </span>
                        </span>
                        <Badge tone="amber">
                          {Math.round(c.score * 100)}%
                        </Badge>
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

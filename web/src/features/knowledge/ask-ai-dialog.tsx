import { useState } from "react";
import { Link } from "react-router-dom";
import { Loader2, Sparkles, FileText } from "lucide-react";
import { toast } from "sonner";
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
          const msg = apiError(err, "Ask AI failed");
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
          <Sparkles className="size-4" /> Ask AI
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="size-4 text-primary" /> Ask the knowledge base
          </DialogTitle>
          <DialogDescription>
            Ask a question and get an AI answer grounded in your articles.
          </DialogDescription>
        </DialogHeader>

        <Textarea
          rows={3}
          placeholder="e.g. How do I reset a customer's password?"
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
            ⌘/Ctrl + Enter to ask
          </span>
          <Button onClick={submit} disabled={ask.isPending || !question.trim()}>
            {ask.isPending ? (
              <>
                <Loader2 className="size-4 animate-spin" /> Thinking…
              </>
            ) : (
              <>
                <Sparkles className="size-4" /> Ask
              </>
            )}
          </Button>
        </div>

        {/* Results */}
        {ask.isPending && (
          <div className="flex items-center gap-2 rounded-lg border border-border bg-muted/30 px-4 py-3 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" /> Searching the knowledge
            base and composing an answer…
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
                Answer
              </div>
              <p className="whitespace-pre-wrap text-sm leading-relaxed text-foreground">
                {result.answer}
              </p>
            </div>

            {result.citations.length > 0 && (
              <div>
                <div className="mb-2 font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
                  Sources
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

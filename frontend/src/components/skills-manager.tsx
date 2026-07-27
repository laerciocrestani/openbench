import { useCallback, useEffect, useState } from "react"

import { AppService } from "../../bindings/github.com/laerciocrestani/openbench"
import type { SkillView } from "../../bindings/github.com/laerciocrestani/openbench/internal/desktop"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { Loader2, Plus, RotateCcw, Trash2, X } from "lucide-react"

function errText(e: unknown): string {
  if (e instanceof Error) return e.message
  return String(e)
}

type EditorState =
  | { mode: "closed" }
  | { mode: "create" }
  | { mode: "edit"; skill: SkillView }

export function SkillsManager({ active }: { active: boolean }) {
  const [skills, setSkills] = useState<SkillView[]>([])
  const [dir, setDir] = useState("")
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [editor, setEditor] = useState<EditorState>({ mode: "closed" })

  const reload = useCallback(async () => {
    setBusy(true)
    setError(null)
    try {
      const res = await AppService.ListSkills()
      setSkills(res?.skills ?? [])
      setDir(res?.dir || "")
    } catch (e) {
      setError(errText(e))
    } finally {
      setBusy(false)
    }
  }, [])

  useEffect(() => {
    if (active) void reload()
  }, [active, reload])

  const toggle = async (skill: SkillView, enabled: boolean) => {
    setError(null)
    try {
      await AppService.SetSkillEnabled(skill.id, enabled)
      await reload()
    } catch (e) {
      setError(errText(e))
    }
  }

  const reset = async (id: string) => {
    setError(null)
    try {
      await AppService.ResetSkill(id)
      await reload()
      setEditor({ mode: "closed" })
    } catch (e) {
      setError(errText(e))
    }
  }

  const remove = async (id: string) => {
    setError(null)
    try {
      await AppService.DeleteSkill(id)
      await reload()
      setEditor({ mode: "closed" })
    } catch (e) {
      setError(errText(e))
    }
  }

  if (editor.mode !== "closed") {
    return (
      <SkillEditor
        skill={editor.mode === "edit" ? editor.skill : null}
        onCancel={() => setEditor({ mode: "closed" })}
        onSaved={async () => {
          setEditor({ mode: "closed" })
          await reload()
        }}
        onReset={
          editor.mode === "edit" && editor.skill.builtin
            ? () => void reset(editor.skill.id)
            : undefined
        }
        onDelete={
          editor.mode === "edit" && !editor.skill.builtin
            ? () => void remove(editor.skill.id)
            : undefined
        }
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="text-[11px] text-muted-foreground">
        Skills ativas entram no system prompt do Chat IA. A builtin{" "}
        <span className="font-medium text-foreground">docker-debug</span> ajuda a
        depurar containers Compose e orienta o fluxo{" "}
        <span className="font-medium text-foreground">Corrigir com IA</span> após
        falha de orquestração.
      </p>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="flex items-center gap-2">
        <Button
          size="sm"
          variant="outline"
          onClick={() => setEditor({ mode: "create" })}
          disabled={busy}
        >
          <Plus />
          Nova skill
        </Button>
        <Button size="sm" variant="ghost" onClick={() => void reload()} disabled={busy}>
          {busy ? <Loader2 className="animate-spin" /> : null}
          Atualizar
        </Button>
      </div>

      <ul className="flex flex-col gap-2">
        {skills.map((s) => (
          <li
            key={s.id}
            className="flex flex-col gap-2 rounded-lg border border-border px-3 py-2"
          >
            <div className="flex items-start gap-3">
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-1.5">
                  <button
                    type="button"
                    className="truncate text-left text-sm font-medium hover:underline"
                    onClick={() => setEditor({ mode: "edit", skill: s })}
                  >
                    {s.name || s.id}
                  </button>
                  {s.builtin ? (
                    <Badge variant="secondary" className="text-[10px]">
                      builtin
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-[10px]">
                      custom
                    </Badge>
                  )}
                  {s.customized && (
                    <Badge variant="outline" className="text-[10px]">
                      editada
                    </Badge>
                  )}
                </div>
                <p className="mt-0.5 font-mono text-[10px] text-muted-foreground">{s.id}</p>
                {s.description ? (
                  <p className="mt-1 line-clamp-2 text-[11px] text-muted-foreground">
                    {s.description}
                  </p>
                ) : null}
              </div>
              <Switch
                checked={!!s.enabled}
                onCheckedChange={(checked) => void toggle(s, !!checked)}
                title={s.enabled ? "Desativar" : "Ativar"}
              />
            </div>
            <div className="flex flex-wrap gap-1">
              <Button
                size="sm"
                variant="ghost"
                onClick={() => setEditor({ mode: "edit", skill: s })}
              >
                Editar
              </Button>
              {s.builtin && s.customized ? (
                <Button size="sm" variant="ghost" onClick={() => void reset(s.id)}>
                  <RotateCcw />
                  Restaurar
                </Button>
              ) : null}
              {!s.builtin ? (
                <Button size="sm" variant="ghost" onClick={() => void remove(s.id)}>
                  <Trash2 />
                  Apagar
                </Button>
              ) : null}
            </div>
          </li>
        ))}
        {!busy && skills.length === 0 && (
          <li className="text-sm text-muted-foreground">Nenhuma skill encontrada.</li>
        )}
      </ul>

      {dir ? (
        <p className="font-mono text-[11px] text-muted-foreground">Skills: {dir}</p>
      ) : null}
    </div>
  )
}

function SkillEditor({
  skill,
  onCancel,
  onSaved,
  onReset,
  onDelete,
}: {
  skill: SkillView | null
  onCancel: () => void
  onSaved: () => void | Promise<void>
  onReset?: () => void
  onDelete?: () => void
}) {
  const isNew = skill === null
  const [id, setId] = useState(skill?.id ?? "")
  const [name, setName] = useState(skill?.name ?? "")
  const [description, setDescription] = useState(skill?.description ?? "")
  const [body, setBody] = useState(
    skill?.body ??
      "Playbook da skill.\n\n## Quando aplicar\n\n...\n\n## Passos\n\n1. ...\n",
  )
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const save = async () => {
    setBusy(true)
    setError(null)
    try {
      await AppService.SaveSkill(id.trim(), name.trim(), description.trim(), body.trim())
      await onSaved()
    } catch (e) {
      setError(errText(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <h3 className="text-sm font-medium">
          {isNew ? "Nova skill" : `Editar: ${skill?.name || skill?.id}`}
        </h3>
        <Button
          size="icon-sm"
          variant="ghost"
          className="ml-auto"
          onClick={onCancel}
          title="Voltar"
        >
          <X />
        </Button>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="skill-id">ID</Label>
        <Input
          id="skill-id"
          value={id}
          onChange={(e) => setId(e.target.value)}
          disabled={!isNew}
          placeholder="ex.: minha-skill"
          className="font-mono text-xs"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="skill-name">Nome</Label>
        <Input
          id="skill-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Debug Docker Compose"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="skill-desc">Descrição</Label>
        <Input
          id="skill-desc"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Curta, para a lista"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="skill-body">Corpo (Markdown)</Label>
        <Textarea
          id="skill-body"
          value={body}
          onChange={(e) => setBody(e.target.value)}
          className="min-h-48 font-mono text-xs"
        />
      </div>

      <div className="flex flex-wrap gap-2">
        {onReset ? (
          <Button type="button" variant="outline" size="sm" onClick={onReset} disabled={busy}>
            <RotateCcw />
            Restaurar padrão
          </Button>
        ) : null}
        {onDelete ? (
          <Button type="button" variant="outline" size="sm" onClick={onDelete} disabled={busy}>
            <Trash2 />
            Apagar
          </Button>
        ) : null}
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="ml-auto"
          onClick={onCancel}
          disabled={busy}
        >
          Cancelar
        </Button>
        <Button type="button" size="sm" onClick={() => void save()} disabled={busy}>
          {busy ? <Loader2 className="animate-spin" /> : null}
          Salvar
        </Button>
      </div>
    </div>
  )
}

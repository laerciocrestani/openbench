import { useState, type ReactNode } from "react"
import {
  FolderGit2,
  FolderOpen,
  FolderPlus,
  GitBranchPlus,
  Loader2,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

export type ProjectSetupStep = "choice" | "create" | "clone"

export type ProjectSetupDialogProps = {
  open: boolean
  busy: boolean
  step: ProjectSetupStep
  initPath: string | null
  error: string | null
  onOpenChange: (open: boolean) => void
  onStep: (step: ProjectSetupStep) => void
  onAbrir: () => void
  onCreate: (parentDir: string, name: string) => void
  onClone: (url: string, parentDir: string, name: string) => void
  onPickDirectory: (title: string) => Promise<string | null>
  onConfirmInit: (addAll: boolean) => void
  onCancelInit: () => void
}

function ChoiceCard({
  icon,
  title,
  description,
  onClick,
  disabled,
}: {
  icon: ReactNode
  title: string
  description: string
  onClick: () => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "flex w-full items-start gap-3 rounded-xl border bg-background px-4 py-3 text-left transition-colors",
        "hover:bg-muted/50 disabled:pointer-events-none disabled:opacity-50"
      )}
    >
      <span className="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
        {icon}
      </span>
      <span className="min-w-0 flex-1">
        <span className="block font-medium">{title}</span>
        <span className="mt-0.5 block text-sm text-muted-foreground">{description}</span>
      </span>
    </button>
  )
}

export function ProjectSetupDialog({
  open,
  busy,
  step,
  initPath,
  error,
  onOpenChange,
  onStep,
  onAbrir,
  onCreate,
  onClone,
  onPickDirectory,
  onConfirmInit,
  onCancelInit,
}: ProjectSetupDialogProps) {
  const [name, setName] = useState("")
  const [parentDir, setParentDir] = useState("")
  const [url, setUrl] = useState("")
  const [initAddAll, setInitAddAll] = useState(true)

  const resetForms = () => {
    setName("")
    setParentDir("")
    setUrl("")
  }

  const handleOpenChange = (next: boolean) => {
    if (!next && busy) return
    if (!next) resetForms()
    onOpenChange(next)
  }

  const pickParent = async (title: string) => {
    const path = await onPickDirectory(title)
    if (path) setParentDir(path)
  }

  return (
    <>
      <Dialog open={open && !initPath} onOpenChange={handleOpenChange}>
        <DialogContent className="sm:max-w-md">
          {step === "choice" && (
            <>
              <DialogHeader>
                <DialogTitle>Projeto</DialogTitle>
                <DialogDescription>
                  Abra um repositório existente, crie um novo ou clone de um remote.
                </DialogDescription>
              </DialogHeader>
              <div className="flex flex-col gap-2">
                <ChoiceCard
                  icon={<FolderOpen className="size-4" />}
                  title="Abrir projeto"
                  description="Escolha uma pasta local com git (ou inicie um repo)."
                  onClick={onAbrir}
                  disabled={busy}
                />
                <ChoiceCard
                  icon={<FolderPlus className="size-4" />}
                  title="Criar projeto"
                  description="Cria uma pasta com o nome do projeto e roda git init."
                  onClick={() => {
                    resetForms()
                    onStep("create")
                  }}
                  disabled={busy}
                />
                <ChoiceCard
                  icon={<GitBranchPlus className="size-4" />}
                  title="Clonar repositório"
                  description="Baixa qualquer remote git para uma pasta local."
                  onClick={() => {
                    resetForms()
                    onStep("clone")
                  }}
                  disabled={busy}
                />
              </div>
            </>
          )}

          {step === "create" && (
            <>
              <DialogHeader>
                <DialogTitle>Criar projeto</DialogTitle>
                <DialogDescription>
                  Informe o nome e a pasta onde o projeto será criado.
                </DialogDescription>
              </DialogHeader>
              <div className="flex flex-col gap-3">
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ob-create-name">Nome do projeto</Label>
                  <Input
                    id="ob-create-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="meu-app"
                    disabled={busy}
                    autoFocus
                  />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ob-create-parent">Diretório de destino</Label>
                  <div className="flex gap-2">
                    <Input
                      id="ob-create-parent"
                      value={parentDir}
                      onChange={(e) => setParentDir(e.target.value)}
                      placeholder="/Users/…/projects"
                      disabled={busy}
                      className="min-w-0 flex-1"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      disabled={busy}
                      onClick={() => void pickParent("Destino do projeto")}
                    >
                      <FolderOpen />
                    </Button>
                  </div>
                  {name.trim() && parentDir.trim() ? (
                    <p className="truncate text-xs text-muted-foreground">
                      Será criado em {parentDir.replace(/\/$/, "")}/{name.trim()}
                    </p>
                  ) : null}
                </div>
                {error ? <p className="text-sm text-destructive">{error}</p> : null}
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => {
                    resetForms()
                    onStep("choice")
                  }}
                >
                  Voltar
                </Button>
                <Button
                  disabled={busy || !name.trim() || !parentDir.trim()}
                  onClick={() => onCreate(parentDir.trim(), name.trim())}
                >
                  {busy ? <Loader2 className="animate-spin" /> : <FolderPlus />}
                  Criar
                </Button>
              </DialogFooter>
            </>
          )}

          {step === "clone" && (
            <>
              <DialogHeader>
                <DialogTitle>Clonar repositório</DialogTitle>
                <DialogDescription>
                  Cole a URL do remote (HTTPS ou SSH) e escolha o destino.
                </DialogDescription>
              </DialogHeader>
              <div className="flex flex-col gap-3">
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ob-clone-url">URL do repositório</Label>
                  <Input
                    id="ob-clone-url"
                    value={url}
                    onChange={(e) => setUrl(e.target.value)}
                    placeholder="git@github.com:org/repo.git"
                    disabled={busy}
                    autoFocus
                  />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ob-clone-name">Nome da pasta (opcional)</Label>
                  <Input
                    id="ob-clone-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="derivado da URL se vazio"
                    disabled={busy}
                  />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ob-clone-parent">Diretório de destino</Label>
                  <div className="flex gap-2">
                    <Input
                      id="ob-clone-parent"
                      value={parentDir}
                      onChange={(e) => setParentDir(e.target.value)}
                      placeholder="/Users/…/projects"
                      disabled={busy}
                      className="min-w-0 flex-1"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      disabled={busy}
                      onClick={() => void pickParent("Destino do clone")}
                    >
                      <FolderOpen />
                    </Button>
                  </div>
                </div>
                {error ? <p className="text-sm text-destructive">{error}</p> : null}
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => {
                    resetForms()
                    onStep("choice")
                  }}
                >
                  Voltar
                </Button>
                <Button
                  disabled={busy || !url.trim() || !parentDir.trim()}
                  onClick={() => onClone(url.trim(), parentDir.trim(), name.trim())}
                >
                  {busy ? <Loader2 className="animate-spin" /> : <GitBranchPlus />}
                  Clonar
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(initPath)}
        onOpenChange={(next) => {
          if (!next && !busy) onCancelInit()
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia className="bg-amber-500/15 text-amber-700 dark:text-amber-400">
              <FolderGit2 />
            </AlertDialogMedia>
            <AlertDialogTitle>Pasta sem repositório git</AlertDialogTitle>
            <AlertDialogDescription>
              <span className="block break-all font-medium text-foreground">
                {initPath}
              </span>
              <span className="mt-2 block">
                Esta pasta não é um repositório git. Deseja iniciar um repositório
                com <code className="text-xs">git init</code>?
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <label className="flex items-start gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-sm">
            <Checkbox
              checked={initAddAll}
              onCheckedChange={(v) => setInitAddAll(v === true)}
              disabled={busy}
              className="mt-0.5"
            />
            <span>
              <span className="font-medium">Adicionar todos os arquivos</span>
              <span className="mt-0.5 block text-muted-foreground">
                Roda <code className="text-xs">git add .</code> depois do init.
                Depois você ainda pode adicionar arquivo a arquivo no dashboard.
              </span>
            </span>
          </label>
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy} onClick={onCancelInit}>
              Cancelar
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={busy}
              onClick={() => onConfirmInit(initAddAll)}
            >
              {busy ? <Loader2 className="animate-spin" /> : null}
              Iniciar repositório
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

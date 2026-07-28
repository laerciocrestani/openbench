import { useEffect, useState } from "react"
import { FolderGit2, Link2, Loader2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

export type RemoteOriginDialogProps = {
  open: boolean
  busy: boolean
  error: string | null
  defaultName: string
  onOpenChange: (open: boolean) => void
  onSetURL: (url: string) => void
  onCreateGitHub: (name: string, visibility: "public" | "private", description: string) => void
}

export function RemoteOriginDialog({
  open,
  busy,
  error,
  defaultName,
  onOpenChange,
  onSetURL,
  onCreateGitHub,
}: RemoteOriginDialogProps) {
  const [tab, setTab] = useState<"url" | "create">("url")
  const [url, setUrl] = useState("")
  const [name, setName] = useState(defaultName)
  const [visibility, setVisibility] = useState<"public" | "private">("private")
  const [description, setDescription] = useState("")

  useEffect(() => {
    if (!open) return
    setTab("url")
    setUrl("")
    setName(defaultName)
    setVisibility("private")
    setDescription("")
  }, [open, defaultName])

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && busy) return
        onOpenChange(next)
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Link2 className="size-4" />
            Remote origin
          </DialogTitle>
          <DialogDescription>
            Configure o <code className="text-xs">origin</code> para Sync, Push e PR.
            Cole a URL de um repositório existente ou crie um novo no GitHub.
          </DialogDescription>
        </DialogHeader>

        <Tabs
          value={tab}
          onValueChange={(v) => setTab(v === "create" ? "create" : "url")}
        >
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="url" disabled={busy}>
              Usar URL
            </TabsTrigger>
            <TabsTrigger value="create" disabled={busy}>
              Criar no GitHub
            </TabsTrigger>
          </TabsList>

          <TabsContent value="url" className="mt-3 flex flex-col gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ob-remote-url">URL do repositório</Label>
              <Input
                id="ob-remote-url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://github.com/org/repo.git"
                disabled={busy}
                autoFocus
              />
              <p className="text-xs text-muted-foreground">
                HTTPS ou SSH — equivalente a{" "}
                <code className="text-[11px]">git remote add origin …</code>
              </p>
            </div>
            {error && tab === "url" ? (
              <p className="text-sm text-destructive">{error}</p>
            ) : null}
            <DialogFooter className="px-0 sm:justify-end">
              <Button
                variant="outline"
                disabled={busy}
                onClick={() => onOpenChange(false)}
              >
                Cancelar
              </Button>
              <Button
                disabled={busy || !url.trim()}
                onClick={() => onSetURL(url.trim())}
              >
                {busy ? <Loader2 className="animate-spin" /> : <Link2 />}
                Salvar origin
              </Button>
            </DialogFooter>
          </TabsContent>

          <TabsContent value="create" className="mt-3 flex flex-col gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ob-remote-name">Nome no GitHub</Label>
              <Input
                id="ob-remote-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="org/meu-repo"
                disabled={busy}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Visibilidade</Label>
              <Select
                value={visibility}
                onValueChange={(v) =>
                  setVisibility(v === "public" ? "public" : "private")
                }
                disabled={busy}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="private">Privado</SelectItem>
                  <SelectItem value="public">Público</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="ob-remote-desc">Descrição (opcional)</Label>
              <Input
                id="ob-remote-desc"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                disabled={busy}
              />
            </div>
            <p className="text-xs text-muted-foreground">
              Usa <code className="text-[11px]">gh repo create</code> e configura o
              origin. O push fica para o fluxo Push depois.
            </p>
            {error && tab === "create" ? (
              <p className="text-sm text-destructive">{error}</p>
            ) : null}
            <DialogFooter className="px-0 sm:justify-end">
              <Button
                variant="outline"
                disabled={busy}
                onClick={() => onOpenChange(false)}
              >
                Cancelar
              </Button>
              <Button
                disabled={busy || !name.trim()}
                onClick={() =>
                  onCreateGitHub(name.trim(), visibility, description.trim())
                }
              >
                {busy ? <Loader2 className="animate-spin" /> : <FolderGit2 />}
                Criar repositório
              </Button>
            </DialogFooter>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}

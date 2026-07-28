import type { LucideIcon } from "lucide-react"
import {
  ArrowDown,
  ArrowDownUp,
  ArrowUp,
  CircleHelp,
  FolderOpen,
  GitCommit,
  GitMerge,
  GitPullRequest,
  RefreshCw,
  Stethoscope,
  Trash2,
} from "lucide-react"

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

type ToolbarActionHelp = {
  name: string
  icon: LucideIcon
  summary: string
  details: string
}

const TOOLBAR_ACTIONS: ToolbarActionHelp[] = [
  {
    name: "Abrir",
    icon: FolderOpen,
    summary: "Escolhe ou troca o projeto (repositório git) ativo.",
    details: "Abre o seletor de pasta e carrega o dashboard desse path.",
  },
  {
    name: "Atualizar",
    icon: RefreshCw,
    summary: "Recarrega o estado do repositório atual.",
    details: "Atualiza branch, dirty, ahead/behind, PR e Docker sem alterar git.",
  },
  {
    name: "Commit",
    icon: GitCommit,
    summary: "Gera preview de commit com IA e confirma a mensagem.",
    details:
      "O menu ao lado oferece Commit & Push, Create Branch & Commit e Commit & Create PR, conforme você está na base ou em feature.",
  },
  {
    name: "Push",
    icon: ArrowUp,
    summary: "Envia commits locais para o upstream.",
    details:
      "Aparece quando há commits à frente (↑). No primeiro push da branch, publica o remoto e configura tracking.",
  },
  {
    name: "Pull",
    icon: ArrowDown,
    summary: "Atualiza a branch atual com o remoto (fast-forward).",
    details:
      "Faz fetch e pull --ff-only quando a branch está ↓ atrás. Em feature, também atualiza a base local (main) sem trocar de branch. Exige working tree limpa.",
  },
  {
    name: "Doctor",
    icon: Stethoscope,
    summary: "Diagnostica a saúde do repositório.",
    details:
      "Aponta problemas (divergência, base atrasada, PR em conflito, etc.) e pode sugerir correções antes de commit/push/PR.",
  },
  {
    name: "Pull Request",
    icon: GitPullRequest,
    summary: "Cria ou revisa a PR da branch atual.",
    details:
      "Gera título/corpo com IA e abre no GitHub. Disponível quando há commits à frente da base e ainda não existe PR aberta.",
  },
  {
    name: "Merge PR",
    icon: GitMerge,
    summary: "Mergeia a PR aberta no GitHub.",
    details:
      "Só habilita com PR aberta, não-draft, sem conflitos e com checks ok. Depois do merge, costuma ser hora de Sync da base.",
  },
  {
    name: "Sync",
    icon: ArrowDownUp,
    summary: "Sincroniza só a base local com origin.",
    details:
      "Faz fetch e fast-forward de main/master. Em feature, atualiza o ref local da base sem puxar a feature. Use quando a base está ↓ atrás (ex.: pós-merge). Exige working tree limpa.",
  },
  {
    name: "Hygiene",
    icon: Trash2,
    summary: "Limpa branches já mergeadas ou absorvidas.",
    details:
      "Faz checkout automático na base (main) e remove candidatos locais e/ou no GitHub. Assim a branch em que você estava também pode ser apagada. Não faz Sync (fast-forward) da base.",
  },
]

export function ToolbarHelpDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] w-[min(48rem,calc(100%-2rem))] max-w-none flex-col gap-0 overflow-hidden p-0 sm:max-w-none">
        <DialogHeader className="shrink-0 space-y-2 border-b px-4 py-3 text-left">
          <DialogTitle className="flex items-center gap-2">
            <CircleHelp className="size-4" />
            Ações da toolbar
          </DialogTitle>
          <DialogDescription>
            Papel de cada botão do fluxo git no openbench. O destaque “próximo”
            indica a ação recomendada pelo estado atual do repo.
          </DialogDescription>
        </DialogHeader>

        <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden">
          <ul className="flex flex-col gap-3 px-4 py-3">
            {TOOLBAR_ACTIONS.map((action) => {
              const Icon = action.icon
              return (
                <li
                  key={action.name}
                  className="flex gap-3 rounded-lg border bg-muted/20 px-3 py-2.5"
                >
                  <div className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md border bg-background">
                    <Icon className="size-3.5 text-muted-foreground" aria-hidden />
                  </div>
                  <div className="min-w-0 flex flex-col gap-1">
                    <p className="text-sm font-medium">{action.name}</p>
                    <p className="text-sm text-foreground/90">{action.summary}</p>
                    <p className="text-xs leading-relaxed text-muted-foreground">
                      {action.details}
                    </p>
                  </div>
                </li>
              )
            })}
          </ul>
        </div>
      </DialogContent>
    </Dialog>
  )
}

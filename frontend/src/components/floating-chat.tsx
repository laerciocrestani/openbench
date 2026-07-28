import { Bot, Sparkles, X } from "lucide-react"

import { ProjectChatPanel } from "@/components/project-chat-panel"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export function FloatingChat({
  projectPath,
  open,
  onOpenChange,
  bottomOffset = 16,
}: {
  projectPath: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Distância do fundo (ex.: altura do status-bar). */
  bottomOffset?: number
}) {
  if (!projectPath) return null

  const bottomStyle = { bottom: bottomOffset }

  return (
    <>
      {open ? (
        <div
          style={bottomStyle}
          className={cn(
            "fixed right-4 z-50 flex w-[min(100vw-2rem,26rem)] flex-col overflow-hidden",
            "h-[min(72vh,40rem)] rounded-xl border border-border bg-popover text-popover-foreground shadow-xl",
          )}
        >
          <div className="flex shrink-0 items-center gap-2 border-b px-3 py-2">
            <Bot className="size-4 text-muted-foreground" />
            <span className="text-sm font-medium">Chat IA</span>
            <span className="text-[11px] text-muted-foreground">flutuante</span>
            <Button
              variant="ghost"
              size="icon-xs"
              className="ml-auto"
              title="Fechar chat"
              onClick={() => onOpenChange(false)}
            >
              <X />
            </Button>
          </div>
          <div className="min-h-0 flex-1">
            <ProjectChatPanel
              projectPath={projectPath}
              visible={open}
              className="bg-popover"
              hideChrome
            />
          </div>
        </div>
      ) : (
        <button
          type="button"
          style={bottomStyle}
          className={cn(
            "fixed right-4 z-50 rounded-full p-[2px] shadow-md",
            "bg-[linear-gradient(90deg,#14b8a6_0%,#84cc16_50%,#eab308_100%)]",
            "hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          )}
          onClick={() => onOpenChange(true)}
          title="Abrir chat com a IA"
        >
          <span className="flex h-10 items-center gap-1.5 rounded-full bg-white px-4 text-black">
            <Sparkles className="size-4 fill-black" />
            <span className="text-sm font-semibold tracking-wide">AI</span>
          </span>
        </button>
      )}
    </>
  )
}

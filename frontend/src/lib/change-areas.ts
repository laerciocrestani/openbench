/** Agrupa paths por área semântica — espelha internal/ai.ChangeAreasFromPaths. */

export type ChangeAreaGroup<T extends { path: string }> = {
  key: string
  label: string
  files: T[]
}

export function changeAreaKey(path: string): string {
  const normalized = path.replace(/\\/g, "/").replace(/\/+/g, "/")
  const lastSlash = normalized.lastIndexOf("/")
  const dir = lastSlash >= 0 ? normalized.slice(0, lastSlash) : "."
  const base = lastSlash >= 0 ? normalized.slice(lastSlash + 1) : normalized
  const extDot = base.lastIndexOf(".")
  const name = extDot > 0 ? base.slice(0, extDot) : base
  const lowerDir = dir.toLowerCase()

  if (
    lowerDir.includes("/commands") ||
    lowerDir.includes("/controllers") ||
    lowerDir.includes("/models") ||
    lowerDir.includes("/services") ||
    lowerDir.includes("/handlers")
  ) {
    return `${dir}/${name}`
  }
  return dir || "."
}

/** Rótulo curto para o ramo (últimos 2 segmentos). */
export function changeAreaLabel(key: string): string {
  if (!key || key === ".") return "raiz"
  const parts = key.replace(/\\/g, "/").split("/").filter(Boolean)
  if (parts.length <= 2) return parts.join("/") || "raiz"
  return parts.slice(-2).join("/")
}

export function fileBasename(path: string): string {
  const normalized = path.replace(/\\/g, "/")
  const i = normalized.lastIndexOf("/")
  return i >= 0 ? normalized.slice(i + 1) : normalized
}

export function groupFilesByArea<T extends { path: string }>(
  files: T[],
): ChangeAreaGroup<T>[] {
  const order: string[] = []
  const map = new Map<string, T[]>()

  for (const f of files) {
    const key = changeAreaKey(f.path)
    if (!map.has(key)) {
      map.set(key, [])
      order.push(key)
    }
    map.get(key)!.push(f)
  }

  return order.map((key) => ({
    key,
    label: changeAreaLabel(key),
    files: map.get(key)!,
  }))
}

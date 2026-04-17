import type { editor as MonacoEditorType } from "monaco-editor"

interface SelectionProvider {
  host: HTMLElement
  getSelectedText: () => string
}

const providers = new Set<SelectionProvider>()

function getClosestElement(target: EventTarget | null): Element | null {
  if (target instanceof Element) return target
  if (target instanceof Node) return target.parentElement
  return null
}

function getMonacoSelectionText(editor: MonacoEditorType.IStandaloneCodeEditor): string {
  const model = editor.getModel()
  if (!model) return ""

  const selections = editor.getSelections() ?? []
  const parts = selections
    .filter((selection) => !selection.isEmpty())
    .map((selection) => model.getValueInRange(selection))
    .filter((part) => part.length > 0)

  return parts.join(model.getEOL())
}

export function registerMonacoSelectionProvider(editor: MonacoEditorType.IStandaloneCodeEditor): () => void {
  const host = editor.getContainerDomNode()
  const provider: SelectionProvider = {
    host,
    getSelectedText: () => getMonacoSelectionText(editor),
  }
  providers.add(provider)
  return () => {
    providers.delete(provider)
  }
}

export function getSelectionFromRegisteredEditors(target: EventTarget | null): string {
  const element = getClosestElement(target)
  if (!element) return ""

  for (const provider of providers) {
    if (provider.host.contains(element)) {
      return provider.getSelectedText()
    }
  }

  return ""
}

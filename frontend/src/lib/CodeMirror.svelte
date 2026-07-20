<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { EditorView, keymap, placeholder as placeholderExtension } from '@codemirror/view'
  import { EditorState } from '@codemirror/state'
  import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
  import { markdown } from '@codemirror/lang-markdown'

  // Props: a two-way bound markdown string, placeholder text, and a save callback
  // invoked when the user presses the platform save shortcut inside the editor.
  let {
    value = $bindable(''),
    placeholder = '',
    onSave
  }: { value?: string; placeholder?: string; onSave?: () => void } = $props()

  let container: HTMLDivElement
  let editorView: EditorView | undefined

  // focus moves the keyboard caret into the markdown body.
  export function focus(): void {
    editorView?.focus()
  }

  // darkTheme gives the editor a palette consistent with the rest of the app.
  const darkTheme = EditorView.theme(
    {
      '&': { color: 'var(--text)', backgroundColor: 'var(--surface-input)', height: '100%' },
      '.cm-content': { caretColor: 'var(--accent)', fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace' },
      '.cm-cursor': { borderLeftColor: 'var(--accent)' },
      '.cm-scroller': { overflow: 'auto' },
      '&.cm-focused': { outline: 'none' }
    },
    { dark: true }
  )

  // buildExtensions assembles the editor behaviour, including the save shortcut.
  function buildExtensions() {
    const saveShortcut = keymap.of([
      {
        key: 'Mod-s',
        run: () => {
          onSave?.()
          return true
        }
      }
    ])
    const changeListener = EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        value = update.state.doc.toString()
      }
    })
    return [
      history(),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      saveShortcut,
      markdown(),
      placeholderExtension(placeholder),
      EditorView.lineWrapping,
      changeListener,
      darkTheme
    ]
  }

  // Create the editor once the container element exists.
  onMount(() => {
    editorView = new EditorView({
      parent: container,
      state: EditorState.create({ doc: value, extensions: buildExtensions() })
    })
  })

  onDestroy(() => editorView?.destroy())

  // Push programmatic value changes (such as selecting a node) into the editor
  // without disturbing the caret while the user is typing.
  $effect(() => {
    const incoming = value
    if (editorView && incoming !== editorView.state.doc.toString()) {
      editorView.dispatch({ changes: { from: 0, to: editorView.state.doc.length, insert: incoming } })
    }
  })
</script>

<div class="codemirror-host" bind:this={container}></div>

<style>
  .codemirror-host {
    height: 100%;
    min-height: 0;
    border: 1px solid var(--border-input);
    border-radius: 6px;
    overflow: hidden;
  }
</style>

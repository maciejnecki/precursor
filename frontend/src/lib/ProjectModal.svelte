<script lang="ts">
  import EmojiButton from './EmojiButton.svelte'
  import { randomGlyph } from './types'
  import {
    closeProjectModal,
    createProject,
    projectEditId,
    projectModalOpen,
    projects,
    updateProject
  } from './store'

  // Editable field copies for the project being created or edited.
  let name = $state('')
  let description = $state('')
  let icon = $state(randomGlyph())

  let nameInput: HTMLInputElement | undefined = $state()

  // editing reports whether the modal is editing an existing project rather than
  // creating one, which drives the heading, button label, and submit action.
  let editing = $derived($projectEditId !== null)

  // Seed the fields each time the modal opens: from the edited project in edit mode,
  // or fresh values in create mode. Focus lands on the name field either way.
  $effect(() => {
    if ($projectModalOpen) {
      const existing = $projects.find((project) => project.id === $projectEditId)
      if (existing) {
        name = existing.name
        description = existing.description
        icon = existing.icon || randomGlyph()
      } else {
        name = ''
        description = ''
        icon = randomGlyph()
      }
      queueMicrotask(() => nameInput?.focus())
    }
  })

  // submit persists the project according to the mode, then closes the modal.
  async function submit(): Promise<void> {
    if (editing && $projectEditId) {
      await updateProject($projectEditId, name, description, icon)
    } else {
      await createProject(name, description, icon)
    }
    closeProjectModal()
  }

  // handleNameKeydown lets Enter submit the form so a project can be saved without
  // reaching for the mouse.
  function handleNameKeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter') {
      event.preventDefault()
      void submit()
    }
  }
</script>

{#if $projectModalOpen}
  <!-- The backdrop dismisses on click; keyboard users close the modal with Escape,
       so it is presentational to assistive technology. -->
  <div class="overlay" role="presentation" onclick={closeProjectModal}></div>
  <div class="panel">
    <h2>{editing ? 'Edit Project' : 'New Project'}</h2>
    <div class="row">
      <EmojiButton bind:value={icon} />
    </div>
    <input
      type="text"
      placeholder="Project name"
      bind:value={name}
      bind:this={nameInput}
      onkeydown={handleNameKeydown}
    />
    <input type="text" placeholder="Description" bind:value={description} />
    <button type="button" class="primary submit" onclick={submit}>{editing ? 'Save' : 'Create'}</button>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.55);
    z-index: 80;
  }

  .panel {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 90;
    width: 420px;
    max-width: 90vw;
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 18px;
    background-color: var(--surface-raised);
    backdrop-filter: var(--blur-panel);
    border: 1px solid var(--border);
    border-radius: 10px;
  }

  h2 {
    margin: 0;
    font-size: 18px;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  /* The create button spans the modal as a prominent, taller call to action,
     matching the task composer's submit button. */
  .submit {
    width: 100%;
    padding: 12px;
    font-weight: 600;
  }
</style>

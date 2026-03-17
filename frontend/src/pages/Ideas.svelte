<script>
  import { onMount, onDestroy } from 'svelte'
  import { push } from 'svelte-spa-router'

  const wails = window['go']?.['main']?.['App']

  const STATUSES = [
    { key: 'new', label: 'New' },
    { key: 'researching', label: 'Researching' },
    { key: 'researched', label: 'Researched' },
    { key: 'developing', label: 'In Development' },
    { key: 'paused', label: 'Paused' },
  ]

  const CATEGORIES = ['saas', 'app', 'tool', 'service']
  const PRIORITIES = ['high', 'medium', 'low']

  let ideas = []
  let loading = true
  let search = ''
  let searching = false
  let expandedIdea = null
  let showQuickAdd = false
  let quickTitle = ''
  let quickDesc = ''
  let quickCategory = 'app'
  let quickPriority = 'medium'
  let researchingIds = {}
  let convertingId = null
  let dragId = null
  let dragOverCol = null

  // Detail editing
  let editingNotes = ''
  let editingEffort = ''
  let editingRevenue = ''
  let savingNotes = false

  // Suggested task add
  let newTaskTitle = ''
  let newTaskDesc = ''

  onMount(async () => {
    await loadIdeas()
    const rt = window.runtime
    if (rt) {
      rt.EventsOn('idea:status-changed', handleStatusChanged)
      rt.EventsOn('idea:researched', handleResearched)
    }
  })

  onDestroy(() => {
    const rt = window.runtime
    if (rt) {
      rt.EventsOff('idea:status-changed')
      rt.EventsOff('idea:researched')
    }
  })

  async function loadIdeas() {
    loading = true
    try {
      ideas = await wails.GetIdeas('all')
    } catch (e) {
      console.error('Failed to load ideas:', e)
      ideas = []
    }
    loading = false
  }

  function handleStatusChanged(ideaId, status) {
    ideas = ideas.map(i => i.id === ideaId ? { ...i, status } : i)
    if (expandedIdea?.id === ideaId) {
      expandedIdea = { ...expandedIdea, status }
    }
  }

  async function handleResearched(ideaId) {
    researchingIds = { ...researchingIds, [ideaId]: false }
    try {
      const updated = await wails.GetIdea(ideaId)
      ideas = ideas.map(i => i.id === ideaId ? updated : i)
      if (expandedIdea?.id === ideaId) {
        expandedIdea = updated
        editingNotes = updated.notes || ''
        editingEffort = updated.estimatedEffort || ''
        editingRevenue = updated.potentialRevenue || ''
      }
    } catch (e) {
      console.error('Failed to refresh idea:', e)
    }
  }

  function ideasForStatus(status) {
    return ideas.filter(i => i.status === status)
  }

  // Search
  let searchTimeout
  function handleSearchInput() {
    clearTimeout(searchTimeout)
    if (!search.trim()) {
      loadIdeas()
      return
    }
    searchTimeout = setTimeout(async () => {
      searching = true
      try {
        ideas = await wails.SearchIdeas(search)
      } catch (e) {
        console.error('Search failed:', e)
      }
      searching = false
    }, 400)
  }

  // Quick add
  async function createIdea() {
    if (!quickTitle.trim()) return
    try {
      const idea = await wails.CreateIdea(quickTitle, quickDesc, quickCategory, quickPriority, [])
      ideas = [idea, ...ideas]
      quickTitle = ''
      quickDesc = ''
      quickCategory = 'app'
      quickPriority = 'medium'
      showQuickAdd = false
    } catch (e) {
      console.error('Failed to create idea:', e)
    }
  }

  function handleQuickAddKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      createIdea()
    }
    if (e.key === 'Escape') {
      showQuickAdd = false
    }
  }

  // Expand/detail
  async function openIdea(idea) {
    try {
      expandedIdea = await wails.GetIdea(idea.id)
    } catch {
      expandedIdea = idea
    }
    editingNotes = expandedIdea.notes || ''
    editingEffort = expandedIdea.estimatedEffort || ''
    editingRevenue = expandedIdea.potentialRevenue || ''
  }

  function closeDetail() {
    expandedIdea = null
    newTaskTitle = ''
    newTaskDesc = ''
  }

  // Auto research
  async function autoResearch(ideaId) {
    researchingIds = { ...researchingIds, [ideaId]: true }
    try {
      await wails.AutoResearchIdea(ideaId)
    } catch (e) {
      console.error('Auto research failed:', e)
      researchingIds = { ...researchingIds, [ideaId]: false }
    }
  }

  // Convert to project
  async function convertToProject(ideaId) {
    convertingId = ideaId
    try {
      const project = await wails.ConvertIdeaToProject(ideaId)
      ideas = ideas.map(i => i.id === ideaId ? { ...i, status: 'developing', projectId: project.id } : i)
      if (expandedIdea?.id === ideaId) {
        expandedIdea = { ...expandedIdea, status: 'developing', projectId: project.id }
      }
      push(`/project/${project.id}`)
    } catch (e) {
      console.error('Convert failed:', e)
    }
    convertingId = null
  }

  // Move status
  async function moveToStatus(idea, newStatus) {
    const prev = idea.status
    ideas = ideas.map(i => i.id === idea.id ? { ...i, status: newStatus } : i)
    if (expandedIdea?.id === idea.id) {
      expandedIdea = { ...expandedIdea, status: newStatus }
    }
    try {
      await wails.UpdateIdea(idea.id, idea.title, idea.description, idea.category, idea.priority, newStatus, idea.tags || [])
    } catch (e) {
      console.error('Move failed:', e)
      ideas = ideas.map(i => i.id === idea.id ? { ...i, status: prev } : i)
    }
  }

  // Save notes
  async function saveNotes() {
    if (!expandedIdea) return
    savingNotes = true
    try {
      const updated = await wails.UpdateIdeaNotes(expandedIdea.id, editingNotes, editingEffort, editingRevenue)
      ideas = ideas.map(i => i.id === updated.id ? updated : i)
      expandedIdea = updated
    } catch (e) {
      console.error('Save notes failed:', e)
    }
    savingNotes = false
  }

  // Suggested tasks
  async function addSuggestedTask() {
    if (!newTaskTitle.trim() || !expandedIdea) return
    try {
      const updated = await wails.AddSuggestedTask(expandedIdea.id, newTaskTitle, newTaskDesc)
      ideas = ideas.map(i => i.id === updated.id ? updated : i)
      expandedIdea = updated
      newTaskTitle = ''
      newTaskDesc = ''
    } catch (e) {
      console.error('Add task failed:', e)
    }
  }

  async function toggleTaskStatus(taskIndex, currentStatus) {
    if (!expandedIdea) return
    const newStatus = currentStatus === 'accepted' ? 'pending' : 'accepted'
    try {
      const updated = await wails.UpdateSuggestedTaskStatus(expandedIdea.id, taskIndex, newStatus)
      ideas = ideas.map(i => i.id === updated.id ? updated : i)
      expandedIdea = updated
    } catch (e) {
      console.error('Toggle task failed:', e)
    }
  }

  // Delete
  async function deleteIdea(id) {
    try {
      await wails.DeleteIdea(id)
      ideas = ideas.filter(i => i.id !== id)
      if (expandedIdea?.id === id) closeDetail()
    } catch (e) {
      console.error('Delete failed:', e)
    }
  }

  // Drag and drop
  function handleDragStart(e, idea) {
    dragId = idea.id
    e.dataTransfer.effectAllowed = 'move'
  }

  function handleDragOver(e, statusKey) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    dragOverCol = statusKey
  }

  function handleDragLeave() {
    dragOverCol = null
  }

  function handleDrop(e, statusKey) {
    e.preventDefault()
    dragOverCol = null
    if (!dragId) return
    const idea = ideas.find(i => i.id === dragId)
    if (idea && idea.status !== statusKey) {
      moveToStatus(idea, statusKey)
    }
    dragId = null
  }

  function handleDragEnd() {
    dragId = null
    dragOverCol = null
  }

  // Helpers
  function categoryColor(cat) {
    const colors = { saas: '#3b82f6', app: '#22c55e', tool: '#f59e0b', service: '#a855f7' }
    return colors[cat] || '#71717a'
  }

  function priorityColor(p) {
    const colors = { high: '#ef4444', medium: '#f59e0b', low: '#71717a' }
    return colors[p] || '#71717a'
  }

  function researchTypeIcon(type) {
    const icons = { competitor: '\u2694', suggestion: '\u2699', finding: '\ud83d\udcca' }
    return icons[type] || '\ud83d\udcdd'
  }

  function timeAgo(dateStr) {
    if (!dateStr) return ''
    const diff = Date.now() - new Date(dateStr).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 60) return `${mins}m ago`
    const hrs = Math.floor(mins / 60)
    if (hrs < 24) return `${hrs}h ago`
    const days = Math.floor(hrs / 24)
    return `${days}d ago`
  }
</script>

<div class="ideas-page">
  <div class="page-header">
    <div class="header-left">
      <h1>Ideas</h1>
      <span class="count">{ideas.length} ideas</span>
    </div>
    <div class="header-right">
      <div class="search-wrap">
        <input
          class="search-input"
          type="text"
          placeholder="Search ideas..."
          bind:value={search}
          on:input={handleSearchInput}
        />
        {#if searching}
          <span class="search-spinner"></span>
        {/if}
      </div>
      <button class="btn-primary" on:click={() => { showQuickAdd = !showQuickAdd }}>+ New Idea</button>
    </div>
  </div>

  {#if showQuickAdd}
    <div class="quick-add">
      <input
        class="quick-add-title"
        type="text"
        placeholder="Idea title..."
        bind:value={quickTitle}
        on:keydown={handleQuickAddKeydown}
        autofocus
      />
      <textarea
        class="quick-add-desc"
        placeholder="Brief description (optional)"
        bind:value={quickDesc}
        on:keydown={handleQuickAddKeydown}
        rows="2"
      ></textarea>
      <div class="quick-add-row">
        <div class="quick-add-options">
          <select bind:value={quickCategory}>
            {#each CATEGORIES as cat}
              <option value={cat}>{cat}</option>
            {/each}
          </select>
          <select bind:value={quickPriority}>
            {#each PRIORITIES as p}
              <option value={p}>{p}</option>
            {/each}
          </select>
        </div>
        <div class="quick-add-actions">
          <button class="btn-secondary" on:click={() => { showQuickAdd = false }}>Cancel</button>
          <button class="btn-primary" on:click={createIdea} disabled={!quickTitle.trim()}>Add</button>
        </div>
      </div>
    </div>
  {/if}

  {#if loading}
    <div class="empty-state">Loading ideas...</div>
  {:else if ideas.length === 0 && !search}
    <div class="empty-state">
      <span class="empty-icon">\ud83d\udca1</span>
      <p>No ideas yet. Click "+ New Idea" to get started.</p>
    </div>
  {:else}
    <div class="kanban-board">
      {#each STATUSES as col}
        {@const colIdeas = ideasForStatus(col.key)}
        <div
          class="kanban-column"
          class:drag-over={dragOverCol === col.key}
          on:dragover={(e) => handleDragOver(e, col.key)}
          on:dragleave={handleDragLeave}
          on:drop={(e) => handleDrop(e, col.key)}
        >
          <div class="column-header">
            <span class="column-title">{col.label}</span>
            <span class="column-count">{colIdeas.length}</span>
          </div>
          <div class="column-items">
            {#each colIdeas as idea (idea.id)}
              <div
                class="kanban-card"
                class:dragging={dragId === idea.id}
                draggable="true"
                on:dragstart={(e) => handleDragStart(e, idea)}
                on:dragend={handleDragEnd}
                on:click={() => openIdea(idea)}
              >
                <div class="card-top">
                  <span class="card-title">{idea.title}</span>
                  <span class="priority-dot" style="background: {priorityColor(idea.priority)}" title="{idea.priority} priority"></span>
                </div>
                {#if idea.category}
                  <span class="category-badge" style="background: {categoryColor(idea.category)}20; color: {categoryColor(idea.category)}">{idea.category}</span>
                {/if}
                {#if idea.tags && idea.tags.length > 0}
                  <div class="card-tags">
                    {#each idea.tags.slice(0, 3) as tag}
                      <span class="tag">{tag}</span>
                    {/each}
                    {#if idea.tags.length > 3}
                      <span class="tag tag-more">+{idea.tags.length - 3}</span>
                    {/if}
                  </div>
                {/if}
                <div class="card-footer">
                  {#if idea.research && idea.research.length > 0}
                    <span class="research-count">\ud83d\udcdd {idea.research.length}</span>
                  {/if}
                  <span class="card-time">{timeAgo(idea.updatedAt)}</span>
                </div>
                {#if researchingIds[idea.id]}
                  <div class="card-researching">
                    <span class="pulse-dot"></span> Researching...
                  </div>
                {/if}
              </div>
            {/each}
            {#if colIdeas.length === 0}
              <div class="column-empty">Drop ideas here</div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Detail Modal -->
{#if expandedIdea}
  <div class="modal-overlay" on:click={closeDetail}>
    <div class="detail-modal" on:click|stopPropagation>
      <div class="detail-header">
        <div class="detail-header-left">
          <h2>{expandedIdea.title}</h2>
          <div class="detail-meta">
            {#if expandedIdea.category}
              <span class="category-badge" style="background: {categoryColor(expandedIdea.category)}20; color: {categoryColor(expandedIdea.category)}">{expandedIdea.category}</span>
            {/if}
            <span class="priority-dot-label">
              <span class="priority-dot" style="background: {priorityColor(expandedIdea.priority)}"></span>
              {expandedIdea.priority}
            </span>
            <span class="detail-status">{expandedIdea.status}</span>
          </div>
        </div>
        <button class="btn-icon" on:click={closeDetail}>\u2715</button>
      </div>

      {#if expandedIdea.description}
        <div class="detail-section">
          <p class="detail-desc">{expandedIdea.description}</p>
        </div>
      {/if}

      <!-- Actions bar -->
      <div class="detail-actions">
        <div class="status-buttons">
          {#each STATUSES as s}
            <button
              class="status-btn"
              class:active={expandedIdea.status === s.key}
              on:click={() => moveToStatus(expandedIdea, s.key)}
            >{s.label}</button>
          {/each}
        </div>
        <div class="action-buttons">
          {#if expandedIdea.status !== 'researching'}
            <button
              class="btn-small"
              on:click={() => autoResearch(expandedIdea.id)}
              disabled={researchingIds[expandedIdea.id]}
            >
              {#if researchingIds[expandedIdea.id]}
                Researching...
              {:else}
                \ud83d\udd2c Auto Research
              {/if}
            </button>
          {/if}
          {#if !expandedIdea.projectId}
            <button
              class="btn-small convert-btn"
              on:click={() => convertToProject(expandedIdea.id)}
              disabled={convertingId === expandedIdea.id}
            >
              {#if convertingId === expandedIdea.id}
                Converting...
              {:else}
                \ud83d\ude80 Convert to Project
              {/if}
            </button>
          {:else}
            <button class="btn-small" on:click={() => push(`/project/${expandedIdea.projectId}`)}>
              \u2192 View Project
            </button>
          {/if}
        </div>
      </div>

      <!-- Research entries -->
      <div class="detail-section">
        <h3>Research ({expandedIdea.research?.length || 0})</h3>
        {#if expandedIdea.research && expandedIdea.research.length > 0}
          <div class="research-list">
            {#each expandedIdea.research as entry}
              <div class="research-entry">
                <div class="research-entry-header">
                  <span class="research-icon">{researchTypeIcon(entry.type)}</span>
                  <span class="research-title">{entry.title}</span>
                  <span class="research-time">{timeAgo(entry.createdAt)}</span>
                </div>
                <div class="research-content">{entry.content}</div>
                {#if entry.source}
                  <span class="research-source">{entry.source}</span>
                {/if}
              </div>
            {/each}
          </div>
        {:else}
          <p class="empty-sub">No research yet. Click "Auto Research" to start.</p>
        {/if}
      </div>

      <!-- Suggested tasks -->
      <div class="detail-section">
        <h3>Suggested Tasks ({expandedIdea.suggestedTasks?.length || 0})</h3>
        {#if expandedIdea.suggestedTasks && expandedIdea.suggestedTasks.length > 0}
          <div class="tasks-list">
            {#each expandedIdea.suggestedTasks as task, idx}
              <div class="task-item" class:accepted={task.status === 'accepted'}>
                <button class="task-check" on:click={() => toggleTaskStatus(idx, task.status)}>
                  {task.status === 'accepted' ? '\u2713' : '\u25cb'}
                </button>
                <div class="task-info">
                  <span class="task-title">{task.title}</span>
                  {#if task.description}
                    <span class="task-desc">{task.description}</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
        <div class="add-task-form">
          <input type="text" placeholder="Task title..." bind:value={newTaskTitle} on:keydown={(e) => e.key === 'Enter' && addSuggestedTask()} />
          <button class="btn-small" on:click={addSuggestedTask} disabled={!newTaskTitle.trim()}>Add</button>
        </div>
      </div>

      <!-- Notes & business info -->
      <div class="detail-section">
        <h3>Notes & Business</h3>
        <div class="form-group">
          <label>Notes</label>
          <textarea rows="3" bind:value={editingNotes} placeholder="Free-form notes..."></textarea>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label>Estimated Effort</label>
            <input type="text" bind:value={editingEffort} placeholder="e.g. 2 weeks" />
          </div>
          <div class="form-group">
            <label>Potential Revenue</label>
            <input type="text" bind:value={editingRevenue} placeholder="e.g. $5k/mo" />
          </div>
        </div>
        <button class="btn-primary btn-sm" on:click={saveNotes} disabled={savingNotes}>
          {savingNotes ? 'Saving...' : 'Save Notes'}
        </button>
      </div>

      {#if expandedIdea.tags && expandedIdea.tags.length > 0}
        <div class="detail-section">
          <h3>Tags</h3>
          <div class="detail-tags">
            {#each expandedIdea.tags as tag}
              <span class="tag">{tag}</span>
            {/each}
          </div>
        </div>
      {/if}

      <div class="detail-footer">
        <span class="detail-date">Created {timeAgo(expandedIdea.createdAt)}</span>
        {#if expandedIdea.lastResearchedAt}
          <span class="detail-date">Last researched {timeAgo(expandedIdea.lastResearchedAt)}</span>
        {/if}
        <button class="btn-danger-text" on:click={() => deleteIdea(expandedIdea.id)}>Delete Idea</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .ideas-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
    height: 100%;
    overflow: hidden;
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-shrink: 0;
  }

  .header-left {
    display: flex;
    align-items: baseline;
    gap: 10px;
  }

  .header-left h1 {
    font-size: 22px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .count {
    font-size: 13px;
    color: var(--text-muted);
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .search-wrap {
    position: relative;
  }

  .search-input {
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 8px 12px;
    font-size: 13px;
    width: 220px;
    outline: none;
    transition: border-color var(--transition-fast);
  }

  .search-input:focus {
    border-color: var(--accent);
  }

  .search-spinner {
    position: absolute;
    right: 10px;
    top: 50%;
    transform: translateY(-50%);
    width: 14px;
    height: 14px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  /* Quick Add */
  .quick-add {
    background: var(--bg-card);
    border: 1px solid var(--accent);
    border-radius: var(--radius-md);
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex-shrink: 0;
  }

  .quick-add-title {
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 8px 10px;
    font-size: 14px;
    font-weight: 500;
    outline: none;
    font-family: inherit;
  }

  .quick-add-title:focus {
    border-color: var(--accent);
  }

  .quick-add-desc {
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 8px 10px;
    font-size: 13px;
    outline: none;
    resize: none;
    font-family: inherit;
  }

  .quick-add-desc:focus {
    border-color: var(--accent);
  }

  .quick-add-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .quick-add-options {
    display: flex;
    gap: 8px;
  }

  .quick-add-options select {
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 5px 8px;
    font-size: 12px;
    outline: none;
    font-family: inherit;
    text-transform: capitalize;
  }

  .quick-add-actions {
    display: flex;
    gap: 8px;
  }

  /* Kanban Board */
  .kanban-board {
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    gap: 12px;
    flex: 1;
    min-height: 0;
    overflow-x: auto;
  }

  .kanban-column {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    display: flex;
    flex-direction: column;
    min-width: 200px;
    overflow: hidden;
    transition: border-color var(--transition-fast);
  }

  .kanban-column.drag-over {
    border-color: var(--accent);
    background: rgba(124, 58, 237, 0.03);
  }

  .column-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 14px;
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
  }

  .column-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-primary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .column-count {
    font-size: 11px;
    color: var(--text-muted);
    background: var(--bg-primary);
    padding: 2px 8px;
    border-radius: 10px;
  }

  .column-items {
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: 1;
    overflow-y: auto;
  }

  .column-empty {
    text-align: center;
    color: var(--text-muted);
    font-size: 12px;
    padding: 20px 0;
    opacity: 0.5;
  }

  /* Kanban Cards */
  .kanban-card {
    background: var(--bg-primary);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: 12px;
    cursor: pointer;
    transition: all var(--transition-fast);
    display: flex;
    flex-direction: column;
    gap: 8px;
    user-select: none;
  }

  .kanban-card:hover {
    border-color: var(--border);
    transform: translateY(-1px);
  }

  .kanban-card.dragging {
    opacity: 0.4;
  }

  .card-top {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 8px;
  }

  .card-title {
    font-size: 13px;
    color: var(--text-primary);
    font-weight: 500;
    line-height: 1.3;
  }

  .priority-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
    margin-top: 3px;
  }

  .category-badge {
    font-size: 10px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
    text-transform: uppercase;
    width: fit-content;
    letter-spacing: 0.3px;
  }

  .card-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .tag {
    font-size: 10px;
    color: var(--text-muted);
    background: var(--bg-card);
    padding: 2px 6px;
    border-radius: 3px;
    border: 1px solid var(--border-subtle);
  }

  .tag-more {
    color: var(--accent);
    background: var(--accent-subtle);
    border-color: transparent;
  }

  .card-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .research-count {
    font-size: 11px;
    color: var(--text-muted);
  }

  .card-time {
    font-size: 10px;
    color: var(--text-muted);
    margin-left: auto;
  }

  .card-researching {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: var(--accent);
    padding-top: 4px;
    border-top: 1px solid var(--border-subtle);
  }

  .pulse-dot {
    width: 6px;
    height: 6px;
    background: var(--accent);
    border-radius: 50%;
    animation: pulse 1.5s ease-in-out infinite;
  }

  /* Empty state */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--text-muted);
    font-size: 14px;
    gap: 8px;
  }

  .empty-icon {
    font-size: 48px;
  }

  /* Detail Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .detail-modal {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 24px;
    width: 680px;
    max-width: 90vw;
    max-height: 85vh;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }

  .detail-header h2 {
    font-size: 18px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 6px;
  }

  .detail-meta {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .priority-dot-label {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 12px;
    color: var(--text-secondary);
    text-transform: capitalize;
  }

  .detail-status {
    font-size: 11px;
    color: var(--accent);
    background: var(--accent-subtle);
    padding: 2px 8px;
    border-radius: 4px;
    text-transform: capitalize;
    font-weight: 500;
  }

  .detail-section {
    border-top: 1px solid var(--border-subtle);
    padding-top: 14px;
  }

  .detail-section h3 {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 10px;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .detail-desc {
    font-size: 13px;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .detail-actions {
    display: flex;
    flex-direction: column;
    gap: 10px;
    background: var(--bg-primary);
    border-radius: var(--radius-md);
    padding: 12px;
  }

  .status-buttons {
    display: flex;
    gap: 2px;
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    padding: 2px;
  }

  .status-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    padding: 5px 10px;
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    border-radius: 4px;
    transition: all var(--transition-fast);
    font-family: inherit;
  }

  .status-btn:hover {
    color: var(--text-primary);
  }

  .status-btn.active {
    background: var(--accent-subtle);
    color: var(--accent);
  }

  .action-buttons {
    display: flex;
    gap: 8px;
  }

  .convert-btn {
    color: var(--success);
    border-color: var(--success);
  }

  .convert-btn:hover {
    background: rgba(34, 197, 94, 0.1);
  }

  /* Research entries */
  .research-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .research-entry {
    background: var(--bg-primary);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    padding: 10px 12px;
  }

  .research-entry-header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 6px;
  }

  .research-icon {
    font-size: 14px;
  }

  .research-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .research-time {
    font-size: 10px;
    color: var(--text-muted);
    margin-left: auto;
  }

  .research-content {
    font-size: 12px;
    color: var(--text-secondary);
    line-height: 1.5;
    white-space: pre-wrap;
    max-height: 200px;
    overflow-y: auto;
  }

  .research-source {
    display: inline-block;
    font-size: 10px;
    color: var(--text-muted);
    margin-top: 6px;
    background: var(--bg-card);
    padding: 1px 6px;
    border-radius: 3px;
  }

  .empty-sub {
    font-size: 12px;
    color: var(--text-muted);
  }

  /* Suggested tasks */
  .tasks-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 10px;
  }

  .task-item {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 6px 8px;
    border-radius: var(--radius-sm);
    transition: background var(--transition-fast);
  }

  .task-item:hover {
    background: var(--bg-primary);
  }

  .task-check {
    background: none;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 14px;
    padding: 0;
    line-height: 1;
    flex-shrink: 0;
    margin-top: 1px;
  }

  .task-item.accepted .task-check {
    color: var(--success);
  }

  .task-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .task-title {
    font-size: 13px;
    color: var(--text-primary);
  }

  .task-item.accepted .task-title {
    text-decoration: line-through;
    color: var(--text-muted);
  }

  .task-desc {
    font-size: 11px;
    color: var(--text-muted);
  }

  .add-task-form {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .add-task-form input {
    flex: 1;
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 6px 10px;
    font-size: 12px;
    outline: none;
    font-family: inherit;
  }

  .add-task-form input:focus {
    border-color: var(--accent);
  }

  /* Notes form */
  .form-group {
    margin-bottom: 10px;
  }

  .form-group label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 4px;
  }

  .form-group input,
  .form-group textarea {
    width: 100%;
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 8px 10px;
    font-size: 13px;
    font-family: inherit;
    outline: none;
    transition: border-color var(--transition-fast);
    box-sizing: border-box;
  }

  .form-group input:focus,
  .form-group textarea:focus {
    border-color: var(--accent);
  }

  .form-group textarea {
    resize: vertical;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .detail-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .detail-footer {
    display: flex;
    align-items: center;
    gap: 14px;
    border-top: 1px solid var(--border-subtle);
    padding-top: 12px;
  }

  .detail-date {
    font-size: 11px;
    color: var(--text-muted);
  }

  /* Buttons */
  .btn-primary {
    background: var(--accent);
    color: white;
    border: none;
    border-radius: var(--radius-sm);
    padding: 8px 16px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: background var(--transition-fast);
    font-family: inherit;
  }

  .btn-primary:hover {
    background: var(--accent-hover);
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary.btn-sm {
    padding: 6px 12px;
    font-size: 12px;
  }

  .btn-secondary {
    background: var(--bg-card);
    color: var(--text-secondary);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 8px 16px;
    font-size: 13px;
    cursor: pointer;
    transition: all var(--transition-fast);
    font-family: inherit;
  }

  .btn-secondary:hover {
    background: var(--bg-card-hover);
    color: var(--text-primary);
  }

  .btn-small {
    background: transparent;
    color: var(--accent);
    border: 1px solid var(--accent);
    border-radius: var(--radius-sm);
    padding: 4px 10px;
    font-size: 11px;
    cursor: pointer;
    transition: all var(--transition-fast);
    font-family: inherit;
    white-space: nowrap;
  }

  .btn-small:hover {
    background: var(--accent-subtle);
  }

  .btn-small:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-icon {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px 6px;
    border-radius: 4px;
    font-size: 16px;
    transition: all var(--transition-fast);
  }

  .btn-icon:hover {
    background: var(--bg-card-hover);
    color: var(--text-primary);
  }

  .btn-danger-text {
    background: none;
    border: none;
    color: var(--error);
    cursor: pointer;
    font-size: 11px;
    padding: 0;
    margin-left: auto;
    opacity: 0.7;
    transition: opacity var(--transition-fast);
    font-family: inherit;
  }

  .btn-danger-text:hover {
    opacity: 1;
  }

  @keyframes spin {
    to { transform: translateY(-50%) rotate(360deg); }
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.3; }
  }
</style>

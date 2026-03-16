<script>
  import { push } from 'svelte-spa-router'
  import { onMount, onDestroy } from 'svelte'

  export let params = {}

  const wails = window['go']?.['main']?.['App']

  let project = null
  let client = null
  let tasks = []
  let stats = null
  let loading = true
  let projectActivity = []

  // Task modal
  let taskModal = false
  let taskMode = 'create'
  let editingTask = { projectId: '', title: '', description: '', status: 'todo', priority: 'medium', assignedTo: '', estimatedHours: 0, tags: [] }
  let tagInput = ''
  let deleteConfirm = null

  const columns = [
    { key: 'todo', label: 'To Do', color: '#71717a' },
    { key: 'in_progress', label: 'In Progress', color: '#3b82f6' },
    { key: 'in_review', label: 'In Review', color: '#f59e0b' },
    { key: 'done', label: 'Done', color: '#22c55e' },
  ]

  function priorityColor(p) {
    return { urgent: '#ef4444', high: '#f97316', medium: '#3b82f6', low: '#71717a' }[p] || '#71717a'
  }

  function statusColor(s) {
    return { active: '#22c55e', paused: '#f59e0b', completed: '#3b82f6', archived: '#71717a' }[s] || '#71717a'
  }

  async function loadProject() {
    try {
      loading = true
      project = await wails.GetProject(params.id)
      if (project.clientId) {
        try { client = await wails.GetClient(project.clientId) } catch(e) { client = null }
      }
      await loadTasks()
      await loadStats()
    } catch (e) {
      console.error('Failed to load project:', e)
    } finally {
      loading = false
    }
  }

  async function loadTasks() {
    try {
      tasks = await wails.GetTasks(params.id, '') || []
    } catch (e) {
      tasks = []
    }
  }

  async function loadStats() {
    try {
      stats = await wails.GetProjectStats(params.id)
    } catch (e) {
      stats = null
    }
  }

  function tasksByStatus(status) {
    return tasks.filter(t => t.status === status)
  }

  function openCreateTask(status) {
    taskModal = true
    taskMode = 'create'
    editingTask = { projectId: params.id, title: '', description: '', status, priority: 'medium', assignedTo: '', estimatedHours: 0, tags: [] }
    tagInput = ''
  }

  function openEditTask(task) {
    taskModal = true
    taskMode = 'edit'
    editingTask = { ...task, tags: task.tags || [] }
    tagInput = ''
  }

  async function saveTask() {
    try {
      if (taskMode === 'create') {
        await wails.CreateTask(editingTask)
      } else {
        await wails.UpdateTask(editingTask)
      }
      taskModal = false
      await loadTasks()
      await loadStats()
    } catch (e) {
      console.error('Failed to save task:', e)
    }
  }

  async function moveTask(task, newStatus) {
    try {
      await wails.UpdateTask({ ...task, status: newStatus })
      await loadTasks()
      await loadStats()
    } catch (e) {
      console.error('Failed to move task:', e)
    }
  }

  async function deleteTask() {
    if (!deleteConfirm) return
    try {
      await wails.DeleteTask(deleteConfirm.id)
      deleteConfirm = null
      await loadTasks()
      await loadStats()
    } catch (e) {
      console.error('Failed to delete task:', e)
    }
  }

  function addTag() {
    const tag = tagInput.trim()
    if (tag && !editingTask.tags.includes(tag)) {
      editingTask.tags = [...editingTask.tags, tag]
    }
    tagInput = ''
  }

  function removeTag(tag) {
    editingTask.tags = editingTask.tags.filter(t => t !== tag)
  }

  function handleTagKeydown(e) {
    if (e.key === 'Enter') {
      e.preventDefault()
      addTag()
    }
  }

  function timeAgo(ts) {
    if (!ts) return ''
    const diff = Date.now() - new Date(ts).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'just now'
    if (mins < 60) return mins + 'm ago'
    const hrs = Math.floor(mins / 60)
    if (hrs < 24) return hrs + 'h ago'
    return Math.floor(hrs / 24) + 'd ago'
  }

  async function loadProjectActivity() {
    try {
      projectActivity = await wails.GetActivityForEntity('project', params.id) || []
    } catch (e) {
      projectActivity = []
    }
  }

  function goBack() {
    push('/clients')
  }

  let unsubActivity = null

  onMount(() => {
    const runtime = window.runtime
    if (runtime) {
      runtime.EventsOn('hivemind:new-activity', () => {
        loadProjectActivity()
      })
      unsubActivity = () => runtime.EventsOff('hivemind:new-activity')
    }
  })

  onDestroy(() => {
    if (unsubActivity) unsubActivity()
  })

  $: if (params.id) { loadProject(); loadProjectActivity() }
</script>

<div class="project-page">
  {#if loading}
    <div class="loading">Loading project...</div>
  {:else if !project}
    <div class="loading">Project not found</div>
  {:else}
    <!-- Project Header -->
    <div class="project-header">
      <div class="header-top">
        <button class="btn-back" on:click={goBack}>← Clients</button>
        <div class="header-badges">
          <span class="badge" style="background: {statusColor(project.status)}20; color: {statusColor(project.status)}">{project.status}</span>
          <span class="badge" style="background: {priorityColor(project.priority)}20; color: {priorityColor(project.priority)}">{project.priority}</span>
        </div>
      </div>
      <h1>{project.name}</h1>
      <div class="header-meta">
        {#if client}<span class="meta-item">◈ {client.name}</span>{/if}
        {#if project.stack}<span class="meta-item">⚙ {project.stack}</span>{/if}
        {#if project.repoUrl}<a class="meta-link" href={project.repoUrl} target="_blank" rel="noopener">⬡ Repository</a>{/if}
        {#if project.startDate}<span class="meta-item">Start: {project.startDate.split('T')[0]}</span>{/if}
        {#if project.dueDate}<span class="meta-item">Due: {project.dueDate.split('T')[0]}</span>{/if}
      </div>
      {#if stats}
        <div class="stats-bar">
          <div class="stat"><span class="stat-num">{stats.total}</span> Total</div>
          <div class="stat"><span class="stat-num" style="color: #71717a">{stats.todo}</span> To Do</div>
          <div class="stat"><span class="stat-num" style="color: #3b82f6">{stats.inProgress}</span> In Progress</div>
          <div class="stat"><span class="stat-num" style="color: #f59e0b">{stats.inReview}</span> Review</div>
          <div class="stat"><span class="stat-num" style="color: #22c55e">{stats.done}</span> Done</div>
          {#if stats.total > 0}
            <div class="progress-bar">
              <div class="progress-fill" style="width: {(stats.done / stats.total * 100)}%"></div>
            </div>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Kanban Board -->
    <div class="kanban">
      {#each columns as col (col.key)}
        <div class="kanban-column">
          <div class="column-header">
            <div class="column-title">
              <span class="column-dot" style="background: {col.color}"></span>
              {col.label}
              <span class="column-count">{tasksByStatus(col.key).length}</span>
            </div>
            <button class="btn-add-task" on:click={() => openCreateTask(col.key)} title="Add task">+</button>
          </div>
          <div class="column-body">
            {#each tasksByStatus(col.key) as task (task.id)}
              <div class="task-card" on:click={() => openEditTask(task)}>
                <div class="task-top">
                  <span class="task-title">{task.title}</span>
                  <span class="priority-dot" style="background: {priorityColor(task.priority)}" title={task.priority}></span>
                </div>
                {#if task.description}
                  <div class="task-desc">{task.description}</div>
                {/if}
                {#if task.tags && task.tags.length > 0}
                  <div class="task-tags">
                    {#each task.tags as tag}
                      <span class="tag">{tag}</span>
                    {/each}
                  </div>
                {/if}
                <div class="task-footer">
                  {#if task.assignedTo}
                    <span class="task-assignee">⬡ {task.assignedTo}</span>
                  {:else}
                    <span class="task-unassigned">Unassigned</span>
                  {/if}
                  <div class="task-move" on:click|stopPropagation>
                    {#each columns as target}
                      {#if target.key !== task.status}
                        <button class="move-btn" on:click={() => moveTask(task, target.key)} title="Move to {target.label}">
                          <span class="move-dot" style="background: {target.color}"></span>
                        </button>
                      {/if}
                    {/each}
                  </div>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>

    <!-- Project Activity Feed -->
    {#if projectActivity.length > 0}
      <div class="project-activity">
        <div class="pa-header">
          <span class="pa-title">Activity</span>
          <span class="pa-count">{projectActivity.length}</span>
        </div>
        <div class="pa-list">
          {#each projectActivity as entry}
            <div class="pa-item">
              <span class="pa-action" class:created={entry.action === 'created'} class:updated={entry.action === 'updated'} class:deleted={entry.action === 'deleted'}>
                {entry.action === 'created' ? '+' : entry.action === 'updated' ? '~' : '-'}
              </span>
              <span class="pa-summary">{entry.summary}</span>
              <span class="pa-instance">{entry.instanceId}</span>
              <span class="pa-time">{timeAgo(entry.timestamp)}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>

<!-- Task Modal -->
{#if taskModal}
  <div class="modal-overlay" on:click={() => taskModal = false}>
    <div class="modal" on:click|stopPropagation>
      <h2>{taskMode === 'create' ? 'New Task' : 'Edit Task'}</h2>
      <div class="form-group">
        <label>Title *</label>
        <input type="text" bind:value={editingTask.title} placeholder="Task title" />
      </div>
      <div class="form-group">
        <label>Description</label>
        <textarea bind:value={editingTask.description} placeholder="Task description..." rows="3"></textarea>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Status</label>
          <select bind:value={editingTask.status}>
            <option value="todo">To Do</option>
            <option value="in_progress">In Progress</option>
            <option value="in_review">In Review</option>
            <option value="done">Done</option>
          </select>
        </div>
        <div class="form-group">
          <label>Priority</label>
          <select bind:value={editingTask.priority}>
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
            <option value="urgent">Urgent</option>
          </select>
        </div>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Assigned To</label>
          <input type="text" bind:value={editingTask.assignedTo} placeholder="Agent ID or name" />
        </div>
        <div class="form-group">
          <label>Estimated Hours</label>
          <input type="number" bind:value={editingTask.estimatedHours} min="0" step="0.5" />
        </div>
      </div>
      <div class="form-group">
        <label>Tags</label>
        <div class="tag-input-wrap">
          {#each editingTask.tags as tag}
            <span class="tag-chip">{tag} <button class="tag-remove" on:click={() => removeTag(tag)}>×</button></span>
          {/each}
          <input class="tag-input" type="text" bind:value={tagInput} on:keydown={handleTagKeydown} placeholder="Add tag..." />
        </div>
      </div>
      <div class="modal-actions">
        {#if taskMode === 'edit'}
          <button class="btn-danger-solid" on:click={() => { deleteConfirm = editingTask; taskModal = false }}>Delete</button>
        {/if}
        <div style="flex:1"></div>
        <button class="btn-secondary" on:click={() => taskModal = false}>Cancel</button>
        <button class="btn-primary" on:click={saveTask} disabled={!editingTask.title}>Save</button>
      </div>
    </div>
  </div>
{/if}

{#if deleteConfirm}
  <div class="modal-overlay" on:click={() => deleteConfirm = null}>
    <div class="modal modal-small" on:click|stopPropagation>
      <h2>Delete Task</h2>
      <p>Delete <strong>{deleteConfirm.title}</strong>?</p>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => deleteConfirm = null}>Cancel</button>
        <button class="btn-danger-solid" on:click={deleteTask}>Delete</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .project-page {
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .loading {
    text-align: center;
    color: var(--text-secondary);
    padding: 48px 0;
  }

  .project-header {
    padding: 20px 24px 16px;
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
  }

  .header-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .btn-back {
    background: transparent;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 13px;
    padding: 4px 0;
    transition: color var(--transition-fast);
  }

  .btn-back:hover {
    color: var(--accent);
  }

  .header-badges {
    display: flex;
    gap: 6px;
  }

  .project-header h1 {
    font-size: 20px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 8px;
  }

  .header-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    font-size: 12px;
    color: var(--text-secondary);
    margin-bottom: 12px;
  }

  .meta-item {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .meta-link {
    color: var(--accent);
    text-decoration: none;
  }

  .meta-link:hover {
    text-decoration: underline;
  }

  .stats-bar {
    display: flex;
    align-items: center;
    gap: 16px;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .stat {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .stat-num {
    font-weight: 700;
    font-size: 14px;
    color: var(--text-primary);
  }

  .progress-bar {
    flex: 1;
    max-width: 200px;
    height: 4px;
    background: var(--border);
    border-radius: 2px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: var(--success);
    border-radius: 2px;
    transition: width var(--transition-normal);
  }

  /* Kanban */
  .kanban {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
    padding: 16px 24px;
    flex: 1;
    overflow-x: auto;
    min-height: 0;
  }

  .kanban-column {
    background: var(--bg-sidebar);
    border-radius: var(--radius-md);
    display: flex;
    flex-direction: column;
    min-height: 0;
    min-width: 220px;
  }

  .column-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 12px 8px;
    flex-shrink: 0;
  }

  .column-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .column-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }

  .column-count {
    font-size: 11px;
    color: var(--text-muted);
    font-weight: 400;
  }

  .btn-add-task {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    width: 24px;
    height: 24px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all var(--transition-fast);
  }

  .btn-add-task:hover {
    border-color: var(--accent);
    color: var(--accent);
  }

  .column-body {
    flex: 1;
    overflow-y: auto;
    padding: 4px 8px 12px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .task-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    padding: 10px;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .task-card:hover {
    border-color: var(--accent);
  }

  .task-top {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 8px;
    margin-bottom: 4px;
  }

  .task-title {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
    line-height: 1.3;
  }

  .priority-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
    margin-top: 4px;
  }

  .task-desc {
    font-size: 11px;
    color: var(--text-muted);
    margin-bottom: 6px;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .task-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-bottom: 6px;
  }

  .tag {
    font-size: 10px;
    background: var(--accent-subtle);
    color: var(--accent);
    padding: 1px 6px;
    border-radius: 3px;
  }

  .task-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 11px;
  }

  .task-assignee {
    color: var(--text-secondary);
  }

  .task-unassigned {
    color: var(--text-muted);
    font-style: italic;
  }

  .task-move {
    display: flex;
    gap: 3px;
    opacity: 0;
    transition: opacity var(--transition-fast);
  }

  .task-card:hover .task-move {
    opacity: 1;
  }

  .move-btn {
    background: transparent;
    border: 1px solid var(--border);
    border-radius: 3px;
    width: 18px;
    height: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all var(--transition-fast);
    padding: 0;
  }

  .move-btn:hover {
    background: var(--bg-card-hover);
  }

  .move-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
  }

  /* Shared badge style */
  .badge {
    font-size: 10px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 4px;
    text-transform: uppercase;
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
  }

  .btn-primary:hover { background: var(--accent-hover); }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-secondary {
    background: var(--bg-card);
    color: var(--text-secondary);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 8px 16px;
    font-size: 13px;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .btn-secondary:hover {
    background: var(--bg-card-hover);
    color: var(--text-primary);
  }

  .btn-danger-solid {
    background: var(--error);
    color: white;
    border: none;
    border-radius: var(--radius-sm);
    padding: 8px 16px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
  }

  .btn-danger-solid:hover { opacity: 0.9; }

  /* Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 24px;
    width: 480px;
    max-width: 90vw;
    max-height: 85vh;
    overflow-y: auto;
  }

  .modal-small { width: 360px; }

  .modal h2 {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 16px;
  }

  .modal p {
    font-size: 13px;
    color: var(--text-secondary);
    margin-bottom: 16px;
  }

  .form-group {
    margin-bottom: 12px;
  }

  .form-group label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 4px;
  }

  .form-group input,
  .form-group textarea,
  .form-group select {
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
  }

  .form-group input:focus,
  .form-group textarea:focus,
  .form-group select:focus {
    border-color: var(--accent);
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 16px;
  }

  .tag-input-wrap {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 6px 8px;
    min-height: 36px;
    align-items: center;
    transition: border-color var(--transition-fast);
  }

  .tag-input-wrap:focus-within {
    border-color: var(--accent);
  }

  .tag-chip {
    font-size: 11px;
    background: var(--accent-subtle);
    color: var(--accent);
    padding: 2px 6px;
    border-radius: 3px;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .tag-remove {
    background: none;
    border: none;
    color: var(--accent);
    cursor: pointer;
    font-size: 12px;
    padding: 0;
    line-height: 1;
  }

  .tag-input {
    background: transparent;
    border: none;
    color: var(--text-primary);
    font-size: 12px;
    outline: none;
    flex: 1;
    min-width: 60px;
    padding: 2px;
  }

  /* Project Activity */
  .project-activity {
    padding: 12px 24px 16px;
    border-top: 1px solid var(--border-subtle);
    flex-shrink: 0;
    max-height: 160px;
    overflow-y: auto;
  }

  .pa-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .pa-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .pa-count {
    font-size: 10px;
    color: var(--text-muted);
  }

  .pa-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .pa-item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 11px;
  }

  .pa-action {
    width: 16px;
    height: 16px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
    font-weight: 700;
    flex-shrink: 0;
    background: var(--accent-subtle);
    color: var(--accent);
  }

  .pa-action.created { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
  .pa-action.updated { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
  .pa-action.deleted { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

  .pa-summary {
    flex: 1;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .pa-instance {
    color: var(--accent);
    flex-shrink: 0;
  }

  .pa-time {
    color: var(--text-muted);
    flex-shrink: 0;
  }
</style>

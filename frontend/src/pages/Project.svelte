<script>
  import { push } from 'svelte-spa-router'
  import { onMount, onDestroy } from 'svelte'

  export let params = {}

  const wails = window['go']?.['main']?.['App']

  let project = null
  let client = null
  let businessUnit = null
  let parentProject = null
  let tasks = []
  let stats = null
  let loading = true
  let projectActivity = []
  let repoStatus = null
  let envFiles = []
  let allUsedPorts = []

  // Sub-projects
  let subProjects = []
  let subProjectModal = false
  let newSubProject = { name: '', description: '', repoUrl: '', stack: '' }

  // Local path
  let localPathInput = ''
  let settingLocalPath = false

  // Tabs - built dynamically based on project type
  let activeTab = 'tasks'
  $: tabs = project?.isGroup
    ? [
        { key: 'subprojects', label: 'Sub-projects' },
        { key: 'tasks', label: 'Tasks' },
        { key: 'repository', label: 'Repository' },
        { key: 'ports', label: 'Ports' },
        { key: 'env', label: 'Env Files' },
        { key: 'activity', label: 'Activity' },
      ]
    : [
        { key: 'tasks', label: 'Tasks' },
        { key: 'repository', label: 'Repository' },
        { key: 'ports', label: 'Ports' },
        { key: 'env', label: 'Env Files' },
        { key: 'activity', label: 'Activity' },
      ]

  // Task modal
  let taskModal = false
  let taskMode = 'create'
  let editingTask = { projectId: '', title: '', description: '', status: 'todo', priority: 'medium', assignedTo: '', estimatedHours: 0, tags: [] }
  let tagInput = ''
  let deleteConfirm = null

  // Port modal
  let portModal = false
  let newPort = { port: '', service: '', protocol: 'http' }
  let portConflicts = []
  let portStatuses = {}

  // Env editor
  let editingEnvFile = null
  let envContent = ''
  let envVars = []
  let envSaving = false

  // Cloning
  let cloning = false
  let cloneError = ''
  let pulling = false
  let pullResult = ''

  // Edit project modal
  let editModal = false
  let editingProject = {}
  let allClients = []
  let allBusinessUnits = []

  // Actions menu & move-to-group
  let actionsMenuOpen = false
  let moveToGroupModal = false
  let allGroups = []
  let actionConfirm = null // { type: 'convertToGroup' | 'makeStandalone', project }
  let actionLoading = false

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
        if (project.businessUnitId) {
          try {
            const bus = await wails.GetBusinessUnits(project.clientId) || []
            businessUnit = bus.find(b => b.id === project.businessUnitId) || null
          } catch(e) { businessUnit = null }
        }
      } else {
        client = null
        businessUnit = null
      }
      // Load parent for breadcrumbs
      if (project.parentId) {
        try { parentProject = await wails.GetProject(project.parentId) } catch(e) { parentProject = null }
      } else {
        parentProject = null
      }
      if (project.isGroup) {
        await loadSubProjects()
        if (activeTab === 'tasks') activeTab = 'subprojects'
      }
      await loadTasks()
      await loadStats()
    } catch (e) {
      console.error('Failed to load project:', e)
    } finally {
      loading = false
    }
  }

  async function loadSubProjects() {
    try { subProjects = await wails.GetSubProjects(params.id) || [] } catch (e) { subProjects = [] }
  }

  async function openCreateSubProject() {
    subProjectModal = true
    newSubProject = { name: '', description: '', repoUrl: '', stack: '' }
  }

  async function saveSubProject() {
    if (!newSubProject.name) return
    try {
      await wails.CreateSubProject(params.id, newSubProject.name, newSubProject.description, newSubProject.repoUrl, newSubProject.stack)
      subProjectModal = false
      await loadSubProjects()
      await loadProject()
    } catch (e) { console.error('Failed to create sub-project:', e) }
  }

  async function deleteSubProject(subId) {
    try {
      await wails.DeleteSubProject(subId)
      await loadSubProjects()
      await loadProject()
    } catch (e) { console.error('Failed to delete sub-project:', e) }
  }

  function getSubProjectLocalStatus(subId) {
    return wails.CheckLocalRepo(subId)
  }

  async function setLocalPathManual() {
    if (!localPathInput.trim()) return
    settingLocalPath = true
    try {
      await wails.SetLocalPath(params.id, localPathInput.trim())
      localPathInput = ''
      await loadRepoStatus()
    } catch (e) { alert('Error: ' + e) }
    finally { settingLocalPath = false }
  }

  async function openInFinder() {
    try { await wails.OpenInFinder(params.id) } catch (e) { console.error(e) }
  }

  async function loadTasks() {
    try { tasks = await wails.GetTasks(params.id, '') || [] } catch (e) { tasks = [] }
  }

  async function loadStats() {
    try { stats = await wails.GetProjectStats(params.id) } catch (e) { stats = null }
  }

  async function loadRepoStatus() {
    try { repoStatus = await wails.CheckLocalRepo(params.id) } catch (e) { repoStatus = null }
  }

  async function loadEnvFiles() {
    try { envFiles = await wails.ScanEnvFiles(params.id) || [] } catch (e) { envFiles = [] }
  }

  async function loadAllPorts() {
    try { allUsedPorts = await wails.GetAllUsedPorts() || [] } catch (e) { allUsedPorts = [] }
  }

  async function checkPortStatuses() {
    if (!project?.ports) return
    const statuses = {}
    for (const p of project.ports) {
      try { statuses[p.port] = await wails.CheckPortInUse(p.port) } catch { statuses[p.port] = false }
    }
    portStatuses = statuses
  }

  function tasksByStatus(status) {
    return tasks.filter(t => t.status === status)
  }

  // --- Task CRUD ---
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
      if (taskMode === 'create') await wails.CreateTask(editingTask)
      else await wails.UpdateTask(editingTask)
      taskModal = false
      await loadTasks()
      await loadStats()
    } catch (e) { console.error('Failed to save task:', e) }
  }

  async function moveTask(task, newStatus) {
    try {
      await wails.UpdateTask({ ...task, status: newStatus })
      await loadTasks()
      await loadStats()
    } catch (e) { console.error('Failed to move task:', e) }
  }

  async function deleteTask() {
    if (!deleteConfirm) return
    try {
      await wails.DeleteTask(deleteConfirm.id)
      deleteConfirm = null
      await loadTasks()
      await loadStats()
    } catch (e) { console.error('Failed to delete task:', e) }
  }

  function addTag() {
    const tag = tagInput.trim()
    if (tag && !editingTask.tags.includes(tag)) editingTask.tags = [...editingTask.tags, tag]
    tagInput = ''
  }
  function removeTag(tag) { editingTask.tags = editingTask.tags.filter(t => t !== tag) }
  function handleTagKeydown(e) { if (e.key === 'Enter') { e.preventDefault(); addTag() } }

  // --- Repo ---
  async function cloneRepo() {
    cloning = true
    cloneError = ''
    try {
      await wails.CloneRepository(params.id)
      await loadRepoStatus()
    } catch (e) {
      cloneError = e.toString()
    } finally {
      cloning = false
    }
  }

  async function pullLatest() {
    pulling = true
    pullResult = ''
    try {
      pullResult = await wails.PullLatest(params.id)
      await loadRepoStatus()
    } catch (e) {
      pullResult = 'Error: ' + e.toString()
    } finally {
      pulling = false
    }
  }

  async function openTerminal() {
    try { await wails.OpenInTerminal(params.id) } catch (e) { console.error(e) }
  }

  async function setManualPath() {
    const path = prompt('Enter the local path to the repository:')
    if (!path) return
    try {
      await wails.SetLocalPath(params.id, path)
      await loadRepoStatus()
    } catch (e) { alert('Error: ' + e) }
  }

  // --- Ports ---
  async function openAddPort() {
    portModal = true
    newPort = { port: '', service: '', protocol: 'http' }
    portConflicts = []
  }

  async function checkNewPortConflicts() {
    if (!newPort.port) { portConflicts = []; return }
    try { portConflicts = await wails.CheckPortConflicts(parseInt(newPort.port)) || [] } catch { portConflicts = [] }
  }

  async function addPort() {
    if (!newPort.port || !newPort.service) return
    try {
      project = await wails.AddPort(params.id, parseInt(newPort.port), newPort.service, newPort.protocol)
      portModal = false
      await loadAllPorts()
      checkPortStatuses()
    } catch (e) { alert('Error: ' + e) }
  }

  async function removePort(port) {
    try {
      project = await wails.RemovePort(params.id, port)
      await loadAllPorts()
      checkPortStatuses()
    } catch (e) { console.error(e) }
  }

  function getPortConflict(port) {
    return allUsedPorts.filter(p => p.port === port && p.projectId !== params.id).map(p => p.projectName)
  }

  // --- Env Files ---
  async function openEnvEditor(filename) {
    editingEnvFile = filename
    envContent = ''
    envVars = []
    try {
      envContent = await wails.GetEnvFileContent(params.id, filename)
      envVars = await wails.GetEnvVariables(params.id, filename) || []
    } catch (e) {
      envContent = ''
      envVars = []
    }
  }

  async function saveEnvContent() {
    envSaving = true
    try {
      await wails.SaveEnvFileContent(params.id, editingEnvFile, envContent)
      envVars = await wails.GetEnvVariables(params.id, editingEnvFile) || []
    } catch (e) { alert('Save error: ' + e) }
    finally { envSaving = false }
  }

  async function createFromExample(exampleFile) {
    try {
      await wails.CreateEnvFromExample(params.id, exampleFile)
      await loadEnvFiles()
    } catch (e) { alert('Error: ' + e) }
  }

  function closeEnvEditor() {
    editingEnvFile = null
    envContent = ''
    envVars = []
  }

  // --- Project Actions (convert, move, standalone) ---
  function toggleActionsMenu() { actionsMenuOpen = !actionsMenuOpen }
  function closeActionsMenu() { actionsMenuOpen = false }

  async function confirmConvertToGroup() {
    actionConfirm = { type: 'convertToGroup' }
    actionsMenuOpen = false
  }

  async function confirmMakeStandalone() {
    actionConfirm = { type: 'makeStandalone' }
    actionsMenuOpen = false
  }

  async function executeAction() {
    if (!actionConfirm) return
    actionLoading = true
    try {
      if (actionConfirm.type === 'convertToGroup') {
        await wails.ConvertToGroup(params.id)
      } else if (actionConfirm.type === 'makeStandalone') {
        await wails.MoveProjectToStandalone(params.id)
      }
      actionConfirm = null
      await loadProject()
    } catch (e) {
      console.error('Action failed:', e)
      alert('Error: ' + e)
    } finally {
      actionLoading = false
    }
  }

  async function openMoveToGroup() {
    actionsMenuOpen = false
    try { allGroups = await wails.GetAllGroups() || [] } catch { allGroups = [] }
    // Filter out current project and current parent from group list
    allGroups = allGroups.filter(g => g.id !== params.id && g.id !== project?.parentId)
    moveToGroupModal = true
  }

  async function moveToGroup(groupId) {
    actionLoading = true
    try {
      await wails.MoveProjectToGroup(params.id, groupId)
      moveToGroupModal = false
      await loadProject()
    } catch (e) {
      console.error('Move failed:', e)
      alert('Error: ' + e)
    } finally {
      actionLoading = false
    }
  }

  // --- Edit Project ---
  async function openEditProject() {
    editingProject = { ...project }
    try { allClients = await wails.GetClients() || [] } catch { allClients = [] }
    if (editingProject.clientId) {
      try { allBusinessUnits = await wails.GetBusinessUnits(editingProject.clientId) || [] } catch { allBusinessUnits = [] }
    } else {
      allBusinessUnits = []
    }
    editModal = true
  }

  async function onClientChange() {
    editingProject.businessUnitId = ''
    if (editingProject.clientId) {
      try { allBusinessUnits = await wails.GetBusinessUnits(editingProject.clientId) || [] } catch { allBusinessUnits = [] }
    } else {
      allBusinessUnits = []
    }
  }

  async function saveProject() {
    try {
      await wails.UpdateProject(editingProject)
      editModal = false
      await loadProject()
    } catch (e) { console.error('Failed to save:', e) }
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
    try { projectActivity = await wails.GetActivityForEntity('project', params.id) || [] } catch (e) { projectActivity = [] }
  }

  function goBack() { push('/projects') }

  let unsubActivity = null
  let unsubCloning = null
  let unsubCloned = null

  onMount(() => {
    const rt = window.runtime
    if (rt) {
      rt.EventsOn('hivemind:new-activity', () => loadProjectActivity())
      unsubActivity = () => rt.EventsOff('hivemind:new-activity')
      rt.EventsOn('project:cloning', () => { cloning = true })
      unsubCloning = () => rt.EventsOff('project:cloning')
      rt.EventsOn('project:cloned', () => { cloning = false; loadRepoStatus() })
      unsubCloned = () => rt.EventsOff('project:cloned')
    }
  })

  onDestroy(() => {
    if (unsubActivity) unsubActivity()
    if (unsubCloning) unsubCloning()
    if (unsubCloned) unsubCloned()
  })

  $: if (params.id) {
    loadProject()
    loadProjectActivity()
    loadRepoStatus()
    loadEnvFiles()
    loadAllPorts()
  }

  $: if (project?.ports) checkPortStatuses()
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
        <div class="breadcrumb">
          <button class="btn-back" on:click={() => push('/projects')}>Projects</button>
          {#if parentProject}
            <span class="breadcrumb-sep">/</span>
            <button class="btn-back" on:click={() => push(`/project/${parentProject.id}`)}>{parentProject.name}</button>
          {/if}
          <span class="breadcrumb-sep">/</span>
          <span class="breadcrumb-current">{project.name}</span>
        </div>
        <div class="header-right-actions">
          <button class="btn-edit" on:click={openEditProject}>Edit</button>
          <div class="actions-menu-wrap">
            <button class="btn-edit" on:click={toggleActionsMenu} title="More actions">...</button>
            {#if actionsMenuOpen}
              <div class="actions-backdrop" on:click={closeActionsMenu}></div>
              <div class="actions-dropdown">
                {#if !project.isGroup}
                  <button class="dropdown-item" on:click={confirmConvertToGroup}>Convert to Group</button>
                {/if}
                <button class="dropdown-item" on:click={openMoveToGroup}>Move to Group...</button>
                {#if project.parentId}
                  <button class="dropdown-item" on:click={confirmMakeStandalone}>Make Standalone</button>
                {/if}
              </div>
            {/if}
          </div>
          <div class="header-badges">
            <span class="badge" style="background: {statusColor(project.status)}20; color: {statusColor(project.status)}">{project.status}</span>
            <span class="badge" style="background: {priorityColor(project.priority)}20; color: {priorityColor(project.priority)}">{project.priority}</span>
          </div>
        </div>
      </div>
      <h1>{project.name}</h1>
      {#if project.description}<p class="project-desc">{project.description}</p>{/if}
      <div class="header-meta">
        {#if client}
          <span class="meta-item">◈ {client.name}{#if businessUnit} / {businessUnit.name}{/if}</span>
        {:else}
          <span class="meta-item meta-muted">(No client)</span>
        {/if}
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

    <!-- Tab Navigation -->
    <div class="tab-bar">
      {#each tabs as tab}
        <button class="tab-btn" class:active={activeTab === tab.key} on:click={() => activeTab = tab.key}>
          {tab.label}
          {#if tab.key === 'subprojects' && subProjects.length}
            <span class="tab-count">{subProjects.length}</span>
          {/if}
          {#if tab.key === 'ports' && project.ports?.length}
            <span class="tab-count">{project.ports.length}</span>
          {/if}
          {#if tab.key === 'env' && envFiles.length}
            <span class="tab-count">{envFiles.length}</span>
          {/if}
        </button>
      {/each}
    </div>

    <!-- Tab Content -->
    <div class="tab-content">
      <!-- SUB-PROJECTS TAB -->
      {#if activeTab === 'subprojects'}
        <div class="section-content">
          <div class="section-top-bar">
            <span class="section-label">Sub-projects</span>
            <button class="btn-small" on:click={openCreateSubProject}>+ Add Sub-project</button>
          </div>
          {#if subProjects.length === 0}
            <div class="empty-section">
              <p>No sub-projects yet. Add one to get started.</p>
              <button class="btn-primary" on:click={openCreateSubProject}>Add Sub-project</button>
            </div>
          {:else}
            <div class="sub-projects-grid">
              {#each subProjects as sub (sub.id)}
                <div class="sub-project-card" on:click={() => push(`/project/${sub.id}`)}>
                  <div class="sub-top">
                    <span class="sub-icon">&#9776;</span>
                    <span class="sub-name">{sub.name}</span>
                    <div class="sub-badges">
                      <span class="badge" style="background: {statusColor(sub.status)}20; color: {statusColor(sub.status)}">{sub.status}</span>
                    </div>
                  </div>
                  {#if sub.description}<div class="sub-desc">{sub.description}</div>{/if}
                  {#if sub.repoUrl}<div class="sub-repo mono">{sub.repoUrl}</div>{/if}
                  {#if sub.stack}<div class="sub-stack">{sub.stack}</div>{/if}
                  <div class="sub-actions" on:click|stopPropagation>
                    <button class="btn-icon btn-danger" on:click={() => { deleteConfirm = sub }} title="Remove">✕</button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>

      <!-- TASKS TAB -->
      {:else if activeTab === 'tasks'}
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
                      <div class="task-desc-text">{task.description}</div>
                    {/if}
                    {#if task.tags && task.tags.length > 0}
                      <div class="task-tags">
                        {#each task.tags as tag}<span class="tag">{tag}</span>{/each}
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

      <!-- REPOSITORY TAB -->
      {:else if activeTab === 'repository'}
        <div class="section-content">
          {#if !project.repoUrl}
            <div class="empty-section">
              <p>No repository URL configured.</p>
              <button class="btn-secondary" on:click={openEditProject}>Set Repository URL</button>
            </div>
          {:else}
            <div class="repo-url-row">
              <span class="repo-label">URL:</span>
              <a class="repo-link" href={project.repoUrl} target="_blank" rel="noopener">{project.repoUrl}</a>
            </div>

            {#if repoStatus?.isCloned}
              <div class="repo-info">
                <div class="local-path-section">
                  <div class="info-item">
                    <span class="info-label">Local Path</span>
                    <div class="local-path-row">
                      <span class="local-path-icon">&#128193;</span>
                      <span class="info-value mono">{repoStatus.localPath}</span>
                    </div>
                  </div>
                  <div class="local-path-actions">
                    <button class="btn-secondary btn-compact" on:click={openTerminal}>Open in Terminal</button>
                    <button class="btn-secondary btn-compact" on:click={openInFinder}>Open in Finder</button>
                  </div>
                </div>
                <div class="info-grid">
                  <div class="info-item">
                    <span class="info-label">Branch</span>
                    <span class="info-value"><span class="branch-badge">{repoStatus.currentBranch}</span></span>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Last Commit</span>
                    <span class="info-value">{repoStatus.lastCommit || 'N/A'}</span>
                  </div>
                  <div class="info-item">
                    <span class="info-label">Uncommitted Changes</span>
                    <span class="info-value">
                      {#if repoStatus.hasUncommittedChanges}
                        <span class="status-dot warning"></span> Yes
                      {:else}
                        <span class="status-dot ok"></span> Clean
                      {/if}
                    </span>
                  </div>
                </div>
                <div class="repo-actions">
                  <button class="btn-secondary" on:click={pullLatest} disabled={pulling}>
                    {pulling ? 'Pulling...' : 'Pull Latest'}
                  </button>
                </div>
                {#if pullResult}
                  <pre class="pull-output">{pullResult}</pre>
                {/if}
              </div>
            {:else}
              <div class="repo-not-cloned">
                <p>Repository is not cloned locally.</p>
                <div class="repo-actions">
                  <button class="btn-primary" on:click={cloneRepo} disabled={cloning}>
                    {cloning ? 'Cloning...' : 'Clone Repository'}
                  </button>
                </div>
                {#if cloning}
                  <div class="clone-progress">
                    <div class="spinner"></div>
                    <span>Cloning repository...</span>
                  </div>
                {/if}
                {#if cloneError}
                  <div class="error-msg">{cloneError}</div>
                {/if}
                <div class="manual-path-section">
                  <span class="info-label">Or set path manually</span>
                  <div class="manual-path-row">
                    <input class="path-input" type="text" bind:value={localPathInput} placeholder="/Users/you/Projects/my-project" />
                    <button class="btn-secondary btn-compact" on:click={setLocalPathManual} disabled={settingLocalPath || !localPathInput.trim()}>
                      {settingLocalPath ? 'Setting...' : 'Set Path'}
                    </button>
                  </div>
                </div>
              </div>
            {/if}
          {/if}
        </div>

      <!-- PORTS TAB -->
      {:else if activeTab === 'ports'}
        <div class="section-content">
          <div class="section-top-bar">
            <span class="section-label">Registered Ports</span>
            <button class="btn-small" on:click={openAddPort}>+ Add Port</button>
          </div>
          {#if !project.ports || project.ports.length === 0}
            <div class="empty-section"><p>No ports registered for this project.</p></div>
          {:else}
            <div class="ports-table">
              <div class="port-row port-header-row">
                <span class="port-col-status"></span>
                <span class="port-col-num">Port</span>
                <span class="port-col-service">Service</span>
                <span class="port-col-protocol">Protocol</span>
                <span class="port-col-conflicts">Conflicts</span>
                <span class="port-col-actions"></span>
              </div>
              {#each project.ports as p}
                {@const inUse = portStatuses[p.port]}
                {@const conflicts = getPortConflict(p.port)}
                <div class="port-row">
                  <span class="port-col-status">
                    <span class="status-dot" class:ok={!inUse} class:error={inUse} title={inUse ? 'In use' : 'Free'}></span>
                  </span>
                  <span class="port-col-num mono">{p.port}</span>
                  <span class="port-col-service">{p.service}</span>
                  <span class="port-col-protocol">{p.protocol}</span>
                  <span class="port-col-conflicts">
                    {#if conflicts.length > 0}
                      <span class="conflict-warn">⚠ {conflicts.join(', ')}</span>
                    {:else}
                      <span class="no-conflict">—</span>
                    {/if}
                  </span>
                  <span class="port-col-actions">
                    <button class="btn-icon btn-danger" on:click={() => removePort(p.port)} title="Remove">✕</button>
                  </span>
                </div>
              {/each}
            </div>
          {/if}
        </div>

      <!-- ENV FILES TAB -->
      {:else if activeTab === 'env'}
        <div class="section-content">
          {#if !repoStatus?.isCloned}
            <div class="empty-section"><p>Clone the repository first to manage env files.</p></div>
          {:else if editingEnvFile}
            <div class="env-editor">
              <div class="env-editor-header">
                <button class="btn-back" on:click={closeEnvEditor}>← Back</button>
                <span class="env-filename">{editingEnvFile}</span>
                <button class="btn-primary" on:click={saveEnvContent} disabled={envSaving}>
                  {envSaving ? 'Saving...' : 'Save'}
                </button>
              </div>
              <textarea class="env-textarea" bind:value={envContent} spellcheck="false"></textarea>
              {#if envVars.length > 0}
                <div class="env-vars-list">
                  <div class="env-vars-header">Parsed Variables ({envVars.length})</div>
                  {#each envVars as v}
                    <div class="env-var-row">
                      <span class="env-var-key">{v.key}</span>
                      <span class="env-var-eq">=</span>
                      <span class="env-var-value">{v.value}</span>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          {:else}
            <div class="section-top-bar">
              <span class="section-label">Environment Files</span>
              <button class="btn-small" on:click={loadEnvFiles}>Refresh</button>
            </div>
            {#if envFiles.length === 0}
              <div class="empty-section"><p>No .env files detected.</p></div>
            {:else}
              <div class="env-list">
                {#each envFiles as ef}
                  <div class="env-file-row">
                    <div class="env-file-info">
                      <span class="env-file-name mono">{ef.name}</span>
                      <div class="env-file-badges">
                        <span class="env-badge" class:exists={ef.exists} class:missing={!ef.exists}>
                          {ef.exists ? 'exists' : 'missing'}
                        </span>
                        {#if ef.exampleExists}
                          <span class="env-badge example">has .example</span>
                        {/if}
                      </div>
                    </div>
                    <div class="env-file-actions">
                      {#if ef.exists}
                        <button class="btn-small" on:click={() => openEnvEditor(ef.name)}>Edit</button>
                      {:else if ef.exampleExists}
                        <button class="btn-small" on:click={() => createFromExample(ef.name + '.example')}>Create from example</button>
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          {/if}
        </div>

      <!-- ACTIVITY TAB -->
      {:else if activeTab === 'activity'}
        <div class="section-content">
          {#if projectActivity.length === 0}
            <div class="empty-section"><p>No activity recorded yet.</p></div>
          {:else}
            <div class="activity-list">
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
          {/if}
        </div>
      {/if}
    </div>
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
      {#if deleteConfirm.parentId !== undefined && deleteConfirm.parentId !== null}
        <h2>Delete Sub-project</h2>
        <p>Delete <strong>{deleteConfirm.name}</strong>? This will also delete its tasks.</p>
        <div class="modal-actions">
          <button class="btn-secondary" on:click={() => deleteConfirm = null}>Cancel</button>
          <button class="btn-danger-solid" on:click={() => { deleteSubProject(deleteConfirm.id); deleteConfirm = null }}>Delete</button>
        </div>
      {:else}
        <h2>Delete Task</h2>
        <p>Delete <strong>{deleteConfirm.title}</strong>?</p>
        <div class="modal-actions">
          <button class="btn-secondary" on:click={() => deleteConfirm = null}>Cancel</button>
          <button class="btn-danger-solid" on:click={deleteTask}>Delete</button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Port Modal -->
{#if portModal}
  <div class="modal-overlay" on:click={() => portModal = false}>
    <div class="modal modal-small" on:click|stopPropagation>
      <h2>Add Port</h2>
      <div class="form-group">
        <label>Port *</label>
        <input type="number" bind:value={newPort.port} placeholder="3000" on:input={checkNewPortConflicts} />
      </div>
      <div class="form-group">
        <label>Service *</label>
        <input type="text" bind:value={newPort.service} placeholder="e.g. frontend, api, database" />
      </div>
      <div class="form-group">
        <label>Protocol</label>
        <select bind:value={newPort.protocol}>
          <option value="http">HTTP</option>
          <option value="https">HTTPS</option>
          <option value="tcp">TCP</option>
          <option value="ws">WebSocket</option>
        </select>
      </div>
      {#if portConflicts.length > 0}
        <div class="conflict-warning">⚠ Port {newPort.port} is used by: {portConflicts.join(', ')}</div>
      {/if}
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => portModal = false}>Cancel</button>
        <button class="btn-primary" on:click={addPort} disabled={!newPort.port || !newPort.service}>Add</button>
      </div>
    </div>
  </div>
{/if}

<!-- Edit Project Modal -->
{#if editModal}
  <div class="modal-overlay" on:click={() => editModal = false}>
    <div class="modal" on:click|stopPropagation>
      <h2>Edit Project</h2>
      <div class="form-group">
        <label>Name *</label>
        <input type="text" bind:value={editingProject.name} placeholder="Project name" />
      </div>
      <div class="form-group">
        <label>Description</label>
        <textarea bind:value={editingProject.description} placeholder="Project description..." rows="3"></textarea>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Client (optional)</label>
          <select bind:value={editingProject.clientId} on:change={onClientChange}>
            <option value="">(No client)</option>
            {#each allClients as c}
              <option value={c.id}>{c.name}</option>
            {/each}
          </select>
        </div>
        <div class="form-group">
          <label>Business Unit (optional)</label>
          <select bind:value={editingProject.businessUnitId} disabled={!editingProject.clientId}>
            <option value="">(None)</option>
            {#each allBusinessUnits as bu}
              <option value={bu.id}>{bu.name}</option>
            {/each}
          </select>
        </div>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Status</label>
          <select bind:value={editingProject.status}>
            <option value="active">Active</option>
            <option value="paused">Paused</option>
            <option value="completed">Completed</option>
            <option value="archived">Archived</option>
          </select>
        </div>
        <div class="form-group">
          <label>Priority</label>
          <select bind:value={editingProject.priority}>
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
            <option value="urgent">Urgent</option>
          </select>
        </div>
      </div>
      <div class="form-group">
        <label>Stack</label>
        <input type="text" bind:value={editingProject.stack} placeholder="e.g. React Native, Node.js" />
      </div>
      <div class="form-group">
        <label>Repository URL</label>
        <input type="text" bind:value={editingProject.repoUrl} placeholder="https://github.com/..." />
      </div>
      <div class="form-row">
        <div class="form-group">
          <label>Start Date</label>
          <input type="date" bind:value={editingProject.startDate} />
        </div>
        <div class="form-group">
          <label>Due Date</label>
          <input type="date" bind:value={editingProject.dueDate} />
        </div>
      </div>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => editModal = false}>Cancel</button>
        <button class="btn-primary" on:click={saveProject} disabled={!editingProject.name}>Save</button>
      </div>
    </div>
  </div>
{/if}

<!-- Sub-project Modal -->
{#if subProjectModal}
  <div class="modal-overlay" on:click={() => subProjectModal = false}>
    <div class="modal" on:click|stopPropagation>
      <h2>Add Sub-project</h2>
      <div class="form-group">
        <label>Name *</label>
        <input type="text" bind:value={newSubProject.name} placeholder="e.g. worklife-backend" />
      </div>
      <div class="form-group">
        <label>Description</label>
        <textarea bind:value={newSubProject.description} placeholder="Sub-project description..." rows="2"></textarea>
      </div>
      <div class="form-group">
        <label>Repository URL</label>
        <input type="text" bind:value={newSubProject.repoUrl} placeholder="https://github.com/..." />
      </div>
      <div class="form-group">
        <label>Stack</label>
        <input type="text" bind:value={newSubProject.stack} placeholder="e.g. Node.js, Express" />
      </div>
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => subProjectModal = false}>Cancel</button>
        <button class="btn-primary" on:click={saveSubProject} disabled={!newSubProject.name}>Create</button>
      </div>
    </div>
  </div>
{/if}

<!-- Action Confirm Dialog -->
{#if actionConfirm}
  <div class="modal-overlay" on:click={() => actionConfirm = null}>
    <div class="modal modal-small" on:click|stopPropagation>
      {#if actionConfirm.type === 'convertToGroup'}
        <h2>Convert to Group</h2>
        <p>Convert <strong>{project?.name}</strong> into a project group? All existing data (tasks, ports, env files) will be preserved. You'll be able to add sub-projects to it.</p>
      {:else if actionConfirm.type === 'makeStandalone'}
        <h2>Make Standalone</h2>
        <p>Remove <strong>{project?.name}</strong> from its parent group and make it a top-level project? All data will be preserved.</p>
      {/if}
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => actionConfirm = null}>Cancel</button>
        <button class="btn-primary" on:click={executeAction} disabled={actionLoading}>
          {actionLoading ? 'Working...' : 'Confirm'}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Move to Group Modal -->
{#if moveToGroupModal}
  <div class="modal-overlay" on:click={() => moveToGroupModal = false}>
    <div class="modal" on:click|stopPropagation>
      <h2>Move to Group</h2>
      <p>Select a group to move <strong>{project?.name}</strong> into. All data will be preserved.</p>
      {#if allGroups.length === 0}
        <div class="empty-section"><p>No groups available. Create a group first or convert a project to a group.</p></div>
      {:else}
        <div class="group-pick-list">
          {#each allGroups as g (g.id)}
            <button class="group-pick-item" on:click={() => moveToGroup(g.id)} disabled={actionLoading}>
              <span class="group-pick-icon">📁</span>
              <div class="group-pick-info">
                <span class="group-pick-name">{g.name}</span>
                {#if g.description}<span class="group-pick-desc">{g.description}</span>{/if}
              </div>
            </button>
          {/each}
        </div>
      {/if}
      <div class="modal-actions">
        <button class="btn-secondary" on:click={() => moveToGroupModal = false}>Cancel</button>
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

  /* Header */
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

  .header-right-actions {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .breadcrumb {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .breadcrumb-sep {
    color: var(--text-muted);
    font-size: 12px;
  }

  .breadcrumb-current {
    color: var(--text-primary);
    font-size: 13px;
    font-weight: 500;
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
  .btn-back:hover { color: var(--accent); }

  .btn-edit {
    background: var(--bg-card);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: var(--radius-sm);
    padding: 4px 12px;
    font-size: 12px;
    cursor: pointer;
    transition: all var(--transition-fast);
  }
  .btn-edit:hover { border-color: var(--accent); color: var(--accent); }

  .header-badges { display: flex; gap: 6px; }

  .project-header h1 {
    font-size: 20px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .project-desc {
    font-size: 13px;
    color: var(--text-secondary);
    margin-bottom: 8px;
    line-height: 1.4;
  }

  .header-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    font-size: 12px;
    color: var(--text-secondary);
    margin-bottom: 12px;
  }
  .meta-item { display: flex; align-items: center; gap: 4px; }
  .meta-muted { color: var(--text-muted); font-style: italic; }
  .meta-link { color: var(--accent); text-decoration: none; }
  .meta-link:hover { text-decoration: underline; }

  .stats-bar {
    display: flex;
    align-items: center;
    gap: 16px;
    font-size: 12px;
    color: var(--text-secondary);
  }
  .stat { display: flex; align-items: center; gap: 4px; }
  .stat-num { font-weight: 700; font-size: 14px; color: var(--text-primary); }

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

  /* Tabs */
  .tab-bar {
    display: flex;
    gap: 0;
    padding: 0 24px;
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
  }

  .tab-btn {
    background: transparent;
    border: none;
    border-bottom: 2px solid transparent;
    color: var(--text-secondary);
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition-fast);
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .tab-btn:hover { color: var(--text-primary); }
  .tab-btn.active {
    color: var(--accent);
    border-bottom-color: var(--accent);
  }
  .tab-count {
    font-size: 10px;
    background: var(--accent-subtle);
    color: var(--accent);
    padding: 1px 6px;
    border-radius: 8px;
  }

  .tab-content {
    flex: 1;
    overflow: auto;
    min-height: 0;
  }

  /* Kanban */
  .kanban {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
    padding: 16px 24px;
    height: 100%;
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

  .column-dot { width: 8px; height: 8px; border-radius: 50%; }
  .column-count { font-size: 11px; color: var(--text-muted); font-weight: 400; }

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
  .btn-add-task:hover { border-color: var(--accent); color: var(--accent); }

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
  .task-card:hover { border-color: var(--accent); }

  .task-top {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 8px;
    margin-bottom: 4px;
  }
  .task-title { font-size: 13px; font-weight: 500; color: var(--text-primary); line-height: 1.3; }
  .priority-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; margin-top: 4px; }

  .task-desc-text {
    font-size: 11px;
    color: var(--text-muted);
    margin-bottom: 6px;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .task-tags { display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: 6px; }
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
  .task-assignee { color: var(--text-secondary); }
  .task-unassigned { color: var(--text-muted); font-style: italic; }

  .task-move {
    display: flex;
    gap: 3px;
    opacity: 0;
    transition: opacity var(--transition-fast);
  }
  .task-card:hover .task-move { opacity: 1; }

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
  .move-btn:hover { background: var(--bg-card-hover); }
  .move-dot { width: 6px; height: 6px; border-radius: 50%; }

  /* Section content (repo, ports, env, activity) */
  .section-content {
    padding: 20px 24px;
  }

  .section-top-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .section-label {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .empty-section {
    text-align: center;
    padding: 40px 0;
    color: var(--text-muted);
    font-size: 13px;
  }

  /* Repository */
  .repo-url-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
    font-size: 13px;
  }
  .repo-label { color: var(--text-secondary); font-weight: 500; }
  .repo-link { color: var(--accent); text-decoration: none; }
  .repo-link:hover { text-decoration: underline; }

  .repo-info {
    background: var(--bg-sidebar);
    border-radius: var(--radius-md);
    padding: 16px;
  }

  .info-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    margin-bottom: 16px;
  }

  .info-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .info-label { font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.3px; }
  .info-value { font-size: 13px; color: var(--text-primary); }
  .mono { font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace; font-size: 12px; }

  .branch-badge {
    background: var(--accent-subtle);
    color: var(--accent);
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
  }

  .status-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }
  .status-dot.ok { background: var(--success); }
  .status-dot.warning { background: var(--warning); }
  .status-dot.error { background: var(--error); }

  .repo-actions {
    display: flex;
    gap: 8px;
  }

  .repo-not-cloned {
    background: var(--bg-sidebar);
    border-radius: var(--radius-md);
    padding: 24px;
    text-align: center;
  }
  .repo-not-cloned p {
    color: var(--text-secondary);
    font-size: 13px;
    margin-bottom: 16px;
  }
  .repo-not-cloned .repo-actions { justify-content: center; margin-bottom: 12px; }

  .clone-progress {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin-top: 12px;
    color: var(--text-secondary);
    font-size: 13px;
  }

  .spinner {
    width: 16px;
    height: 16px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .error-msg {
    color: var(--error);
    font-size: 12px;
    margin-top: 8px;
    text-align: center;
  }

  .pull-output {
    margin-top: 12px;
    background: var(--bg-primary);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    padding: 10px 12px;
    font-size: 12px;
    font-family: 'SF Mono', 'Fira Code', monospace;
    color: var(--text-secondary);
    white-space: pre-wrap;
    max-height: 120px;
    overflow-y: auto;
  }

  /* Ports */
  .ports-table {
    background: var(--bg-sidebar);
    border-radius: var(--radius-md);
    overflow: hidden;
  }

  .port-row {
    display: grid;
    grid-template-columns: 32px 80px 1fr 80px 1fr 40px;
    align-items: center;
    padding: 10px 12px;
    border-bottom: 1px solid var(--border-subtle);
    font-size: 13px;
  }
  .port-row:last-child { border-bottom: none; }

  .port-header-row {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.3px;
    background: var(--bg-card);
  }

  .port-col-num { color: var(--text-primary); font-weight: 500; }
  .port-col-service { color: var(--text-secondary); }
  .port-col-protocol { color: var(--text-muted); }
  .port-col-status { display: flex; justify-content: center; }

  .conflict-warn { color: var(--warning); font-size: 12px; }
  .no-conflict { color: var(--text-muted); }

  /* Env Files */
  .env-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .env-file-row {
    background: var(--bg-sidebar);
    border-radius: var(--radius-md);
    padding: 12px 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .env-file-info { display: flex; align-items: center; gap: 12px; }
  .env-file-name { font-size: 14px; font-weight: 500; color: var(--text-primary); }

  .env-file-badges { display: flex; gap: 6px; }
  .env-badge {
    font-size: 10px;
    padding: 2px 8px;
    border-radius: 4px;
    font-weight: 500;
  }
  .env-badge.exists { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
  .env-badge.missing { background: rgba(239, 68, 68, 0.15); color: #ef4444; }
  .env-badge.example { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }

  .env-file-actions { display: flex; gap: 8px; }

  /* Env Editor */
  .env-editor { display: flex; flex-direction: column; gap: 12px; }

  .env-editor-header {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .env-filename {
    flex: 1;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
    font-family: 'SF Mono', 'Fira Code', monospace;
  }

  .env-textarea {
    width: 100%;
    min-height: 300px;
    background: #1a1a2e;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    color: #e0e0e0;
    padding: 16px;
    font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
    font-size: 13px;
    line-height: 1.6;
    resize: vertical;
    outline: none;
    transition: border-color var(--transition-fast);
  }
  .env-textarea:focus { border-color: var(--accent); }

  .env-vars-list {
    background: var(--bg-sidebar);
    border-radius: var(--radius-md);
    overflow: hidden;
  }
  .env-vars-header {
    padding: 10px 16px;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.3px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .env-var-row {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 6px 16px;
    border-bottom: 1px solid var(--border-subtle);
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 12px;
  }
  .env-var-row:last-child { border-bottom: none; }
  .env-var-key { color: #7c3aed; font-weight: 500; }
  .env-var-eq { color: var(--text-muted); }
  .env-var-value { color: var(--text-primary); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  /* Activity */
  .activity-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .pa-item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    padding: 8px 12px;
    background: var(--bg-sidebar);
    border-radius: var(--radius-sm);
  }

  .pa-action {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 700;
    flex-shrink: 0;
    background: var(--accent-subtle);
    color: var(--accent);
  }
  .pa-action.created { background: rgba(34, 197, 94, 0.15); color: #22c55e; }
  .pa-action.updated { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
  .pa-action.deleted { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

  .pa-summary { flex: 1; color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .pa-instance { color: var(--accent); flex-shrink: 0; font-size: 11px; }
  .pa-time { color: var(--text-muted); flex-shrink: 0; }

  /* Shared */
  .badge {
    font-size: 10px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 4px;
    text-transform: uppercase;
  }

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
  .btn-secondary:hover { background: var(--bg-card-hover); color: var(--text-primary); }
  .btn-secondary:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-small {
    background: transparent;
    color: var(--accent);
    border: 1px solid var(--accent);
    border-radius: var(--radius-sm);
    padding: 4px 12px;
    font-size: 11px;
    cursor: pointer;
    transition: all var(--transition-fast);
  }
  .btn-small:hover { background: var(--accent-subtle); }

  .btn-icon {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px 6px;
    border-radius: 4px;
    font-size: 14px;
    transition: all var(--transition-fast);
  }
  .btn-icon:hover { background: var(--bg-card-hover); color: var(--text-primary); }
  .btn-icon.btn-danger:hover { color: var(--error); }

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

  .conflict-warning {
    background: rgba(245, 158, 11, 0.1);
    border: 1px solid rgba(245, 158, 11, 0.3);
    border-radius: var(--radius-sm);
    padding: 8px 12px;
    font-size: 12px;
    color: var(--warning);
    margin-bottom: 8px;
  }

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

  .form-group { margin-bottom: 12px; }

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

  .form-group select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
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
  .tag-input-wrap:focus-within { border-color: var(--accent); }

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

  /* Sub-projects */
  .sub-projects-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 10px;
  }

  .sub-project-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: 14px;
    cursor: pointer;
    transition: all var(--transition-fast);
  }
  .sub-project-card:hover { border-color: var(--accent); background: var(--bg-card-hover); }

  .sub-top {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 6px;
  }

  .sub-icon {
    color: var(--text-muted);
    font-size: 14px;
  }

  .sub-name {
    font-weight: 600;
    font-size: 14px;
    color: var(--text-primary);
    flex: 1;
  }

  .sub-badges { display: flex; gap: 4px; flex-shrink: 0; }

  .sub-desc {
    font-size: 12px;
    color: var(--text-secondary);
    margin-bottom: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sub-repo {
    font-size: 11px;
    color: var(--text-muted);
    margin-bottom: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sub-stack {
    font-size: 11px;
    color: var(--accent);
    margin-bottom: 4px;
  }

  .sub-actions {
    display: flex;
    gap: 4px;
    justify-content: flex-end;
    opacity: 0;
    transition: opacity var(--transition-fast);
  }
  .sub-project-card:hover .sub-actions { opacity: 1; }

  /* Local path */
  .local-path-section {
    margin-bottom: 16px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .local-path-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 4px;
  }

  .local-path-icon { font-size: 16px; }

  .local-path-actions {
    display: flex;
    gap: 8px;
    margin-top: 10px;
  }

  .btn-compact {
    padding: 5px 12px;
    font-size: 12px;
  }

  .manual-path-section {
    margin-top: 20px;
    padding-top: 16px;
    border-top: 1px solid var(--border-subtle);
  }

  .manual-path-row {
    display: flex;
    gap: 8px;
    margin-top: 6px;
  }

  .path-input {
    flex: 1;
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    padding: 8px 10px;
    font-size: 13px;
    font-family: 'SF Mono', 'Fira Code', monospace;
    outline: none;
    transition: border-color var(--transition-fast);
  }
  .path-input:focus { border-color: var(--accent); }
  .path-input::placeholder { color: var(--text-muted); }

  /* Actions dropdown menu */
  .actions-menu-wrap {
    position: relative;
  }

  .actions-backdrop {
    position: fixed;
    inset: 0;
    z-index: 49;
  }

  .actions-dropdown {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 4px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    min-width: 180px;
    padding: 4px;
    z-index: 50;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  }

  .dropdown-item {
    display: block;
    width: 100%;
    background: transparent;
    border: none;
    color: var(--text-secondary);
    padding: 8px 12px;
    font-size: 13px;
    text-align: left;
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }
  .dropdown-item:hover {
    background: var(--accent-subtle);
    color: var(--accent);
  }

  /* Group picker list */
  .group-pick-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-height: 300px;
    overflow-y: auto;
    margin-bottom: 8px;
  }

  .group-pick-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    background: var(--bg-sidebar);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    cursor: pointer;
    transition: all var(--transition-fast);
    text-align: left;
    color: var(--text-primary);
  }
  .group-pick-item:hover {
    border-color: var(--accent);
    background: var(--bg-card-hover);
  }
  .group-pick-item:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .group-pick-icon { font-size: 18px; flex-shrink: 0; }

  .group-pick-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow: hidden;
  }

  .group-pick-name {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .group-pick-desc {
    font-size: 11px;
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>

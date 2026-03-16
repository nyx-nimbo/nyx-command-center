<script>
  const columns = [
    {
      title: 'To Do',
      items: [
        { title: 'Integrate email summarization', priority: 'medium' },
        { title: 'Calendar conflict detector', priority: 'low' },
        { title: 'Slack notification bridge', priority: 'medium' },
      ]
    },
    {
      title: 'In Progress',
      items: [
        { title: 'Build Nyx voice interface', priority: 'high' },
        { title: 'Dashboard analytics widgets', priority: 'medium' },
      ]
    },
    {
      title: 'Done',
      items: [
        { title: 'MongoDB Atlas sync pipeline', priority: 'high' },
        { title: 'Initial project scaffolding', priority: 'high' },
      ]
    },
  ]

  function priorityClass(p) {
    return `priority-${p}`
  }
</script>

<div class="ideas-page">
  <div class="ideas-header">
    <p class="ideas-subtitle">Kanban board for tracking ideas and tasks</p>
  </div>

  <div class="kanban-board">
    {#each columns as column}
      <div class="kanban-column">
        <div class="column-header">
          <span class="column-title">{column.title}</span>
          <span class="column-count">{column.items.length}</span>
        </div>
        <div class="column-items">
          {#each column.items as item}
            <div class="kanban-card">
              <div class="kanban-card-title">{item.title}</div>
              <span class="priority-tag {priorityClass(item.priority)}">{item.priority}</span>
            </div>
          {/each}
        </div>
      </div>
    {/each}
  </div>
</div>

<style>
  .ideas-page {
    display: flex;
    flex-direction: column;
    gap: 20px;
    height: 100%;
  }

  .ideas-subtitle {
    font-size: 13px;
    color: var(--text-muted);
  }

  .kanban-board {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
    flex: 1;
  }

  .kanban-column {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .column-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px;
    border-bottom: 1px solid var(--border-subtle);
  }

  .column-title {
    font-size: 13px;
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
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: 1;
  }

  .kanban-card {
    background: var(--bg-primary);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: 14px;
    cursor: pointer;
    transition: all var(--transition-fast);
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .kanban-card:hover {
    border-color: var(--border);
    transform: translateY(-1px);
  }

  .kanban-card-title {
    font-size: 13px;
    color: var(--text-primary);
    font-weight: 500;
  }

  .priority-tag {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    width: fit-content;
  }

  .priority-high {
    color: var(--error);
    background: rgba(239, 68, 68, 0.1);
  }

  .priority-medium {
    color: var(--warning);
    background: rgba(245, 158, 11, 0.1);
  }

  .priority-low {
    color: var(--text-muted);
    background: rgba(113, 113, 122, 0.1);
  }
</style>

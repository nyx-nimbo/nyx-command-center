<script>
  const stats = [
    { label: 'Active Ideas', value: '12', change: '+3', icon: '◆' },
    { label: 'Emails Today', value: '24', change: '+8', icon: '✉' },
    { label: 'Events This Week', value: '7', change: '0', icon: '▦' },
    { label: 'Tasks Completed', value: '38', change: '+5', icon: '✓' },
  ]

  const activityFeed = [
    { time: '2 min ago', text: 'Nyx processed 3 new emails', type: 'info' },
    { time: '15 min ago', text: 'Calendar sync completed', type: 'success' },
    { time: '1 hr ago', text: 'New idea captured: Voice Interface', type: 'info' },
    { time: '2 hr ago', text: 'MongoDB Atlas backup completed', type: 'success' },
    { time: '4 hr ago', text: 'Email digest generated', type: 'info' },
  ]
</script>

<div class="dashboard">
  <div class="stats-grid">
    {#each stats as stat}
      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-icon">{stat.icon}</span>
          <span class="stat-change" class:positive={stat.change.startsWith('+') && stat.change !== '+0'}>
            {stat.change}
          </span>
        </div>
        <div class="stat-value">{stat.value}</div>
        <div class="stat-label">{stat.label}</div>
      </div>
    {/each}
  </div>

  <div class="dashboard-grid">
    <div class="card activity-card">
      <div class="card-header">
        <h2 class="card-title">Activity Feed</h2>
        <span class="card-badge">Live</span>
      </div>
      <div class="activity-list">
        {#each activityFeed as item}
          <div class="activity-item">
            <div class="activity-dot" class:success={item.type === 'success'}></div>
            <div class="activity-content">
              <span class="activity-text">{item.text}</span>
              <span class="activity-time">{item.time}</span>
            </div>
          </div>
        {/each}
      </div>
    </div>

    <div class="card status-card">
      <div class="card-header">
        <h2 class="card-title">System Status</h2>
      </div>
      <div class="status-list">
        <div class="status-row">
          <span class="status-name">MongoDB Atlas</span>
          <span class="status-badge online">Connected</span>
        </div>
        <div class="status-row">
          <span class="status-name">SQLite Cache</span>
          <span class="status-badge online">Active</span>
        </div>
        <div class="status-row">
          <span class="status-name">Gmail API</span>
          <span class="status-badge pending">Pending Setup</span>
        </div>
        <div class="status-row">
          <span class="status-name">Google Calendar</span>
          <span class="status-badge pending">Pending Setup</span>
        </div>
      </div>
    </div>
  </div>
</div>

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
  }

  .stat-card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    padding: 20px;
    transition: border-color var(--transition-fast);
  }

  .stat-card:hover {
    border-color: var(--border);
  }

  .stat-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }

  .stat-icon {
    font-size: 18px;
    color: var(--accent);
  }

  .stat-change {
    font-size: 12px;
    color: var(--text-muted);
    font-weight: 500;
  }

  .stat-change.positive {
    color: var(--success);
  }

  .stat-value {
    font-size: 28px;
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .stat-label {
    font-size: 12px;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .dashboard-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .card {
    background: var(--bg-card);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    padding: 20px;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .card-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .card-badge {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--success);
    background: rgba(34, 197, 94, 0.1);
    padding: 3px 8px;
    border-radius: var(--radius-sm);
  }

  .activity-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .activity-item {
    display: flex;
    align-items: flex-start;
    gap: 10px;
  }

  .activity-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--accent);
    margin-top: 6px;
    flex-shrink: 0;
  }

  .activity-dot.success {
    background: var(--success);
  }

  .activity-content {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .activity-text {
    font-size: 13px;
    color: var(--text-primary);
  }

  .activity-time {
    font-size: 11px;
    color: var(--text-muted);
  }

  .status-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .status-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 0;
    border-bottom: 1px solid var(--border-subtle);
  }

  .status-row:last-child {
    border-bottom: none;
  }

  .status-name {
    font-size: 13px;
    color: var(--text-secondary);
  }

  .status-badge {
    font-size: 11px;
    font-weight: 500;
    padding: 3px 8px;
    border-radius: var(--radius-sm);
  }

  .status-badge.online {
    color: var(--success);
    background: rgba(34, 197, 94, 0.1);
  }

  .status-badge.pending {
    color: var(--warning);
    background: rgba(245, 158, 11, 0.1);
  }

  @media (max-width: 900px) {
    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }
    .dashboard-grid {
      grid-template-columns: 1fr;
    }
  }
</style>

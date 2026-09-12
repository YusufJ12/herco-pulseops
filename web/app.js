document.addEventListener('DOMContentLoaded', () => {
  const targetsList = document.getElementById('targets-list');
  const statCount = document.getElementById('stat-count');
  const statLatency = document.getElementById('stat-latency');
  const statUptime = document.getElementById('stat-uptime');
  const statHealth = document.getElementById('stat-health');
  const btnRefresh = document.getElementById('btn-refresh');

  const modal = document.getElementById('modal-target');
  const btnOpenModal = document.getElementById('btn-open-modal');
  const btnCloseModal = document.getElementById('btn-close-modal');
  const btnCancelModal = document.getElementById('btn-cancel-modal');
  const formAddTarget = document.getElementById('form-add-target');

  btnOpenModal.addEventListener('click', () => modal.classList.remove('hidden'));
  const closeModal = () => {
    modal.classList.add('hidden');
    formAddTarget.reset();
  };
  btnCloseModal.addEventListener('click', closeModal);
  btnCancelModal.addEventListener('click', closeModal);

  // Fetch and render
  async function loadTargets() {
    try {
      const res = await fetch('/api/targets');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const summaries = await res.json();
      render(summaries);
    } catch (err) {
      console.error('Failed fetching targets:', err);
      targetsList.innerHTML = `
        <div class="p-6 bg-red-950/40 border border-red-800 rounded-xl text-red-300 text-center">
          <i class="fa-solid fa-triangle-exclamation mb-2 text-xl"></i>
          <div>Failed to connect to PulseOps daemon. Retrying...</div>
        </div>
      `;
    }
  }

  function render(summaries) {
    if (!summaries || summaries.length === 0) {
      targetsList.innerHTML = `
        <div class="text-center py-16 bg-slate-900/30 border border-slate-800/80 rounded-xl">
          <i class="fa-solid fa-satellite-dish text-4xl text-slate-600 mb-3"></i>
          <div class="text-base font-medium text-slate-300">No targets monitored yet</div>
          <div class="text-xs text-slate-500 mt-1">Add your first URL above to start telemetry probes.</div>
        </div>
      `;
      updateStats([], 0, 100);
      return;
    }

    let totalLat = 0;
    let countedLat = 0;
    let totalUptime = 0;
    let allUp = true;

    for (const s of summaries) {
      const p = s.last_probe;
      if (p) {
        totalLat += p.latency_ms;
        countedLat++;
        if (!p.is_up) allUp = false;
      }
      totalUptime += s.uptime_percent;
    }

    targetsList.innerHTML = summaries.map(renderTargetCard).join('');

    const avgLat = countedLat > 0 ? Math.round(totalLat / countedLat) : 0;
    const avgUptime = summaries.length > 0 ? (totalUptime / summaries.length).toFixed(1) : 100;
    updateStats(summaries, avgLat, avgUptime, allUp);
    updateChart(summaries);
  }

  // Chart.js instance management
  let telemetryChart = null;
  const palette = [
    { border: '#10b981', bg: 'rgba(16, 185, 129, 0.15)' }, // emerald
    { border: '#38bdf8', bg: 'rgba(56, 189, 248, 0.15)' },  // sky
    { border: '#f59e0b', bg: 'rgba(245, 158, 11, 0.15)' },  // amber
    { border: '#a855f7', bg: 'rgba(168, 85, 247, 0.15)' },  // purple
  ];

  async function updateChart(summaries) {
    const canvas = document.getElementById('latencyChart');
    if (!canvas) return;

    if (!summaries || summaries.length === 0) {
      if (telemetryChart) {
        telemetryChart.destroy();
        telemetryChart = null;
      }
      return;
    }

    try {
      // Fetch history for all targets
      const datasets = [];
      let commonLabels = [];

      for (let i = 0; i < summaries.length; i++) {
        const s = summaries[i];
        const res = await fetch(`/api/targets/${s.target.id}/history?limit=25`);
        if (!res.ok) continue;
        const history = await res.json();
        
        // Reverse so oldest is left, latest is right
        const chron = [...history].reverse();
        const color = palette[i % palette.length];

        if (chron.length > commonLabels.length) {
          commonLabels = chron.map(h => {
            const d = new Date(h.created_at);
            return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
          });
        }

        datasets.push({
          label: `${s.target.name} (${s.target.url})`,
          data: chron.map(h => h.latency_ms),
          borderColor: color.border,
          backgroundColor: color.bg,
          borderWidth: 2,
          fill: true,
          tension: 0.35,
          pointRadius: 3,
          pointHoverRadius: 6,
        });
      }

      if (telemetryChart) {
        telemetryChart.data.labels = commonLabels;
        telemetryChart.data.datasets = datasets;
        telemetryChart.update('none');
      } else {
        const ctx = canvas.getContext('2d');
        telemetryChart = new Chart(ctx, {
          type: 'line',
          data: {
            labels: commonLabels,
            datasets: datasets
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            interaction: {
              mode: 'index',
              intersect: false
            },
            plugins: {
              legend: {
                labels: {
                  color: '#94a3b8',
                  font: { family: 'monospace', size: 11 }
                }
              },
              tooltip: {
                backgroundColor: '#0f172a',
                titleColor: '#f8fafc',
                bodyColor: '#38bdf8',
                borderColor: '#334155',
                borderWidth: 1,
                callbacks: {
                  label: (ctx) => ` ${ctx.dataset.label}: ${ctx.parsed.y} ms`
                }
              }
            },
            scales: {
              x: {
                grid: { color: 'rgba(51, 65, 85, 0.4)' },
                ticks: { color: '#64748b', font: { family: 'monospace', size: 10 } }
              },
              y: {
                title: { display: true, text: 'Latency (ms)', color: '#64748b', font: { family: 'monospace', size: 11 } },
                grid: { color: 'rgba(51, 65, 85, 0.4)' },
                ticks: { color: '#64748b', font: { family: 'monospace', size: 10 } },
                suggestedMin: 0
              }
            }
          }
        });
      }
    } catch (err) {
      console.warn('Failed rendering telemetry chart:', err);
    }
  }

  function updateStats(summaries, avgLat, avgUptime, allUp = true) {
    statCount.textContent = summaries.length;
    statLatency.textContent = `${avgLat} ms`;
    statUptime.textContent = `${avgUptime}%`;

    if (summaries.length === 0) {
      statHealth.innerHTML = '<span class="text-slate-400">Idle</span>';
    } else if (allUp) {
      statHealth.innerHTML = '<i class="fa-solid fa-circle-check text-emerald-400"></i> All Up';
      statHealth.className = 'text-2xl font-bold text-emerald-400 flex items-center gap-2';
    } else {
      statHealth.innerHTML = '<i class="fa-solid fa-triangle-exclamation text-rose-500"></i> Issues Detected';
      statHealth.className = 'text-2xl font-bold text-rose-400 flex items-center gap-2';
    }
  }

  // Global functions for inline action buttons
  window.triggerProbe = async (id) => {
    try {
      const res = await fetch(`/api/targets/${id}/probe`, { method: 'POST' });
      if (res.ok) {
        await loadTargets();
      }
    } catch (err) {
      alert('Probe error: ' + err.message);
    }
  };

  window.deleteTarget = async (id) => {
    if (!confirm('Stop monitoring this target?')) return;
    try {
      const res = await fetch(`/api/targets/${id}`, { method: 'DELETE' });
      if (res.ok) {
        await loadTargets();
      }
    } catch (err) {
      alert('Delete error: ' + err.message);
    }
  };

  // Add target form submit
  formAddTarget.addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = document.getElementById('input-name').value.trim();
    const url = document.getElementById('input-url').value.trim();
    const btn = document.getElementById('btn-submit-target');

    btn.disabled = true;
    btn.innerHTML = '<i class="fa-solid fa-spinner fa-spin"></i> Probing...';

    try {
      const res = await fetch('/api/targets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, url }),
      });

      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || 'Failed adding target');
      }

      closeModal();
      await loadTargets();
    } catch (err) {
      alert('Error: ' + err.message);
    } finally {
      btn.disabled = false;
      btn.innerHTML = 'Save & Probe';
    }
  });

  btnRefresh.addEventListener('click', loadTargets);

  // Initial load & periodic poll
  loadTargets();
  setInterval(loadTargets, 15000);
});

function escapeHtml(str) {
  if (!str) return '';
  return str.replace(/[&<>"']/g, m => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
  }[m]));
}

function getStatusBadge(isUp, statusCode) {
  if (statusCode > 0) {
    return `HTTP ${statusCode}`;
  }
  return isUp ? 'UP' : 'DOWN';
}

function getLatencyBadgeClass(latency, isUp) {
  if (latency > 1500 || !isUp) {
    return 'text-rose-400 border-rose-500/30 bg-rose-500/10';
  }
  if (latency > 500) {
    return 'text-amber-400 border-amber-500/30 bg-amber-500/10';
  }
  return 'text-emerald-400 border-emerald-500/30 bg-emerald-500/10';
}

function renderTargetCard(s) {
  const p = s.last_probe;
  const isUp = Boolean(p?.is_up);
  const statusCode = p?.status_code ?? 0;
  const latency = p?.latency_ms ?? 0;
  const sslDays = p?.ssl_expiry_days ?? 0;
  const uptime = s.uptime_percent.toFixed(1);

  let headersObj = {};
  if (p?.headers) {
    try { headersObj = JSON.parse(p.headers); } catch (_) {}
  }

  const vercelCache = headersObj['x-vercel-cache'] || '';
  const latColor = getLatencyBadgeClass(latency, isUp);
  const statusBadge = getStatusBadge(isUp, statusCode);

  return `
    <div class="bg-slate-900/70 border border-slate-800 rounded-xl p-5 hover:border-slate-700 transition">
      <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        
        <!-- Left Info -->
        <div class="flex items-start gap-4">
          <div class="mt-1">
            ${isUp 
              ? '<span class="flex h-3.5 w-3.5 relative"><span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span><span class="relative inline-flex rounded-full h-3.5 w-3.5 bg-emerald-500"></span></span>' 
              : '<span class="relative inline-flex rounded-full h-3.5 w-3.5 bg-rose-500"></span>'
            }
          </div>
          <div>
            <div class="flex items-center gap-2">
              <h3 class="font-bold text-white text-base">${escapeHtml(s.target.name)}</h3>
              <span class="text-xs font-mono px-2 py-0.5 rounded ${isUp ? 'bg-emerald-500/20 text-emerald-300' : 'bg-rose-500/20 text-rose-300'} font-semibold">
                ${statusBadge}
              </span>
              ${vercelCache ? `<span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-sky-900/40 text-sky-300 border border-sky-800">Vercel: ${vercelCache}</span>` : ''}
            </div>
            
            <a href="${escapeHtml(s.target.url)}" target="_blank" class="text-xs font-mono text-slate-400 hover:text-emerald-400 transition flex items-center gap-1 mt-1">
              ${escapeHtml(s.target.url)} <i class="fa-solid fa-arrow-up-right-from-square text-[10px]"></i>
            </a>

            ${p?.error_msg && !isUp ? `<div class="text-xs text-rose-400 mt-2 font-mono"><i class="fa-solid fa-circle-exclamation mr-1"></i>${escapeHtml(p.error_msg)}</div>` : ''}
          </div>
        </div>

        <!-- Telemetry Metrics & Actions -->
        <div class="flex flex-wrap items-center gap-4 text-xs font-mono">
          <!-- Latency -->
          <div class="px-3 py-1.5 rounded-lg border ${latColor} flex items-center gap-1.5">
            <i class="fa-solid fa-bolt text-[11px]"></i>
            <span>${latency} ms</span>
          </div>

          <!-- SSL Expiry -->
          ${sslDays > 0 ? `
            <div class="px-3 py-1.5 rounded-lg border border-slate-700 bg-slate-800/60 text-slate-300 flex items-center gap-1.5">
              <i class="fa-solid fa-shield-halved text-emerald-400 text-[11px]"></i>
              <span>SSL: ${sslDays}d</span>
            </div>
          ` : ''}

          <!-- Uptime SLA -->
          <div class="px-3 py-1.5 rounded-lg border border-slate-700 bg-slate-800/60 text-slate-300 flex items-center gap-1.5">
            <i class="fa-solid fa-chart-pie text-sky-400 text-[11px]"></i>
            <span>SLA: ${uptime}%</span>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-2">
            <button onclick="window.triggerProbe(${s.target.id})" title="Instant Probe" class="p-2 bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-emerald-400 rounded-lg transition border border-slate-700">
              <i class="fa-solid fa-play text-xs"></i>
            </button>
            <button onclick="window.deleteTarget(${s.target.id})" title="Remove Monitor" class="p-2 bg-slate-800 hover:bg-rose-950/60 text-slate-400 hover:text-rose-400 rounded-lg transition border border-slate-700">
              <i class="fa-regular fa-trash-can text-xs"></i>
            </button>
          </div>

        </div>

      </div>
    </div>
  `;
}

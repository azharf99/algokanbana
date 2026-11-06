<script lang="ts">
  import {
    apiFetch,
    app,
    connectSSE,
    copyToClipboard,
    disconnectSSE,
    fmtDateTimeWIB,
    navigate,
    notify,
    refreshBootstrap,
    type RouteKey,
  } from './lib/state.svelte';
  import LoginScreen from './components/LoginScreen.svelte';
  import CommandPalette from './components/CommandPalette.svelte';
  import DashboardView from './routes/DashboardView.svelte';
  import TaskboardView from './routes/TaskboardView.svelte';
  import StudentsView from './routes/StudentsView.svelte';
  import FeedbackView from './routes/FeedbackView.svelte';
  import ScheduleView from './routes/ScheduleView.svelte';
  import AssessmentView from './routes/AssessmentView.svelte';
  import SurveysView from './routes/SurveysView.svelte';
  import CommView from './routes/CommView.svelte';
  import ProfileView from './routes/ProfileView.svelte';
  import TeamLeadAndBenchmarkView from './routes/TeamLeadAndBenchmarkView.svelte';

  let mobileNavOpen = $state(false);

  const navItems: { key: RouteKey; label: string; icon: string }[] = [
    { key: 'taskboard', label: 'Papan Tugas', icon: '📋' },
    { key: 'dashboard', label: 'Dasbor & KPI', icon: '📊' },
    { key: 'students', label: 'Siswa (55)', icon: '🎓' },
    { key: 'regular-feedback', label: 'Regular Feedback', icon: '📝' },
    { key: 'schedule', label: 'Jadwal & Reschedule', icon: '🗓️' },
    { key: 'assessment', label: 'Assessment', icon: '🏆' },
    { key: 'surveys', label: 'Survei Siswa', icon: '📑' },
    { key: 'communication', label: 'Bantuan & Chat', icon: '💬' },
    { key: 'profile', label: 'Profil & Telegram', icon: '👤' },
  ];

  async function initSession() {
    try {
      const me = await apiFetch('/api/v1/auth/me');
      if (me.authenticated) {
        await refreshBootstrap();
        connectSSE();
      }
    } catch {
      // unauthenticated
    } finally {
      app.ready = true;
    }

    const path = window.location.pathname;
    if (path.startsWith('/portal/')) {
      const seg = path.replace('/portal/', '').split('/')[0] as RouteKey;
      if (seg) app.route = seg;
    }
  }

  async function switchRole(role: 'tutor' | 'team_lead') {
    await apiFetch('/api/v1/auth/role', {
      method: 'POST',
      body: JSON.stringify({ role }),
    });
    await refreshBootstrap();
    if (role === 'team_lead') {
      navigate('tl-demo');
      notify('Beralih ke Mode Demo Team Lead (Devika Wulan Rahmadani)', 'info');
    } else {
      navigate('dashboard');
      notify('Kembali ke Mode Tutor Pengajar', 'info');
    }
  }

  async function triggerManualSync() {
    await apiFetch('/api/v1/sync/trigger', { method: 'POST' });
    await refreshBootstrap();
    notify('Sinkronisasi cache lokal selesai (< 1 ms)!', 'info');
  }

  async function handleLogout() {
    disconnectSSE();
    await apiFetch('/api/v1/auth/logout', { method: 'POST' });
    app.authenticated = false;
    app.session = null;
  }

  function onGlobalKey(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      app.paletteOpen = !app.paletteOpen;
    } else if (e.key === 'Escape' && app.paletteOpen) {
      app.paletteOpen = false;
    }
  }

  $effect(() => {
    initSession();
  });
</script>

<svelte:window onkeydown={onGlobalKey} />

{#if !app.ready}
  <div class="min-h-screen bg-slate-950 flex items-center justify-center text-xs text-slate-400">
    Memuat AlgoKanbana...
  </div>
{:else if !app.authenticated}
  <LoginScreen />
{:else}
  <div class="min-h-screen bg-slate-950 text-slate-100 flex">
    <!-- Left Sidebar -->
    <aside class="w-64 shrink-0 bg-slate-900/95 border-r border-slate-800/90 hidden lg:flex flex-col justify-between p-4">
      <div class="space-y-5">
        <!-- Brand -->
        <div class="flex items-center justify-between px-1">
          <div class="flex items-center gap-2.5">
            <div class="w-9 h-9 rounded-xl bg-indigo-600 flex items-center justify-center font-black text-sm text-white shadow-md shadow-indigo-600/30">
              AK
            </div>
            <div>
              <div class="text-sm font-extrabold text-white leading-none">AlgoKanbana</div>
              <div class="text-[11px] text-slate-400 mt-0.5">Algonova Tutor Portal</div>
            </div>
          </div>
          <span class="text-[10px] font-bold uppercase px-2 py-0.5 rounded bg-indigo-500/20 text-indigo-300">
            {app.session?.activeRole === 'team_lead' ? 'TL' : 'Tutor'}
          </span>
        </div>

        <!-- Primary Navigation -->
        <nav class="space-y-1">
          {#each navItems as item}
            <button
              onclick={() => navigate(item.key)}
              class="w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-semibold transition cursor-pointer {app.route === item.key ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/25' : 'text-slate-300 hover:bg-slate-800/80 hover:text-white'}"
            >
              <span class="flex items-center gap-2.5">
                <span>{item.icon}</span>
                <span>{item.label}</span>
              </span>
              {#if item.key === 'assessment' && app.badges.assessmentOpen}
                <span class="w-2 h-2 rounded-full bg-amber-400"></span>
              {/if}
              {#if item.key === 'communication' && app.badges.tutorUnread > 0}
                <span class="px-1.5 py-0.2 rounded-full text-[10px] bg-emerald-500 text-slate-950 font-black">
                  {app.badges.tutorUnread}
                </span>
              {/if}
            </button>
          {/each}

          <div class="pt-3 mt-3 border-t border-slate-800 space-y-1">
            <div class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider text-slate-500">Mode Presentasi Pimpinan</div>
            <button
              onclick={() => navigate('tl-demo')}
              class="w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-semibold transition cursor-pointer {app.route === 'tl-demo' ? 'bg-amber-500/20 border border-amber-500/40 text-amber-300' : 'text-amber-300/80 hover:bg-slate-800'}"
            >
              <span>👩‍💼 Meja Team Lead Demo</span>
              {#if app.badges.tlUnread > 0}
                <span class="px-1.5 rounded-full text-[10px] bg-amber-400 text-slate-950 font-black">{app.badges.tlUnread}</span>
              {/if}
            </button>
            <button
              onclick={() => navigate('benchmark')}
              class="w-full flex items-center gap-2 px-3 py-2 rounded-xl text-xs font-semibold transition cursor-pointer {app.route === 'benchmark' ? 'bg-emerald-500/20 border border-emerald-500/40 text-emerald-300' : 'text-emerald-400/90 hover:bg-slate-800'}"
            >
              <span>⚡ Benchmark vs Aplikasi Lama</span>
            </button>
          </div>
        </nav>

        <!-- Jadwal Minggu Ini Mini-Rail -->
        <div class="p-3 rounded-2xl bg-slate-950 border border-slate-800 space-y-2.5">
          <div class="flex items-center justify-between text-[11px]">
            <span class="font-bold text-slate-300">Jadwal Terdekat (WIB)</span>
            <button onclick={() => navigate('schedule')} class="text-indigo-400 hover:underline cursor-pointer">Semua</button>
          </div>
          <div class="space-y-2 max-h-52 overflow-y-auto">
            {#each (app.upcomingLessons || []).slice(0, 4) as ls}
              <div class="p-2.5 rounded-xl bg-slate-900 border border-slate-800/80 text-[11px] space-y-1.5">
                <div class="font-bold text-white truncate">{ls.groupTitle}</div>
                <div class="text-emerald-400">{fmtDateTimeWIB(ls.startsAt)}</div>
                <div class="flex items-center gap-1.5">
                  <button onclick={() => copyToClipboard(ls.meetingUrl, 'Link Kelas')} class="px-2 py-0.5 rounded bg-slate-800 text-slate-300 hover:text-white cursor-pointer">
                    Salin Link
                  </button>
                  <a href={ls.prepUrl} target="_blank" rel="noreferrer" class="px-2 py-0.5 rounded bg-indigo-600/20 text-indigo-300">
                    Prep
                  </a>
                </div>
              </div>
            {/each}
          </div>
        </div>
      </div>

      <!-- Footer User -->
      <div class="pt-3 border-t border-slate-800 flex items-center justify-between text-xs">
        <div class="truncate pr-2">
          <div class="font-bold text-white truncate">{app.tutor?.fullName || 'Azhar Faturohman'}</div>
          <div class="text-[11px] text-slate-500">LMS #{app.tutor?.lmsUserId}</div>
        </div>
        <button onclick={handleLogout} class="px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-rose-600/30 text-slate-300 hover:text-rose-300 cursor-pointer">
          Keluar
        </button>
      </div>
    </aside>

    <!-- Main Content Area -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- Top Telemetry & Action Bar -->
      <header class="h-14 border-b border-slate-800/90 bg-slate-900/70 backdrop-blur-md px-4 lg:px-6 flex items-center justify-between gap-3 sticky top-0 z-30">
        <div class="flex items-center gap-2.5">
          <button onclick={() => (mobileNavOpen = !mobileNavOpen)} class="lg:hidden px-2.5 py-1.5 rounded-lg bg-slate-800 text-xs">
            ☰ Menu
          </button>

          <button
            onclick={() => (app.paletteOpen = true)}
            class="px-3 py-1.5 rounded-xl bg-slate-950 border border-slate-800 hover:border-slate-700 text-xs text-slate-400 flex items-center gap-3 cursor-pointer"
          >
            <span>🔎 Cari cepat siswa, grup, halaman...</span>
            <kbd class="px-1.5 py-0.5 rounded bg-slate-800 text-[10px] text-slate-300">Ctrl+K</kbd>
          </button>
        </div>

        <div class="flex items-center gap-2.5 text-xs">
          <!-- Live Server Speed Pill -->
          <button
            onclick={() => navigate('benchmark')}
            title="Klik untuk melihat detail perbandingan performa"
            class="hidden sm:inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 font-mono text-[11px] cursor-pointer"
          >
            <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
            Go API: {app.telemetry.serverMs.toFixed(2)} ms
          </button>

          <!-- Role Switcher -->
          <div class="inline-flex rounded-xl bg-slate-950 border border-slate-800 p-0.5">
            <button
              onclick={() => switchRole('tutor')}
              class="px-2.5 py-1 rounded-lg text-[11px] font-semibold cursor-pointer {app.session?.activeRole !== 'team_lead' ? 'bg-indigo-600 text-white' : 'text-slate-400'}"
            >
              Tutor
            </button>
            <button
              onclick={() => switchRole('team_lead')}
              class="px-2.5 py-1 rounded-lg text-[11px] font-semibold cursor-pointer {app.session?.activeRole === 'team_lead' ? 'bg-amber-500 text-slate-950' : 'text-slate-400'}"
            >
              TL Demo
            </button>
          </div>

          <button onclick={triggerManualSync} title="Sinkronisasi data" class="px-2.5 py-1.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 cursor-pointer">
            ↻
          </button>
        </div>
      </header>

      <!-- Mobile Drawer Nav -->
      {#if mobileNavOpen}
        <div class="lg:hidden bg-slate-900 border-b border-slate-800 p-3 grid grid-cols-2 gap-1.5 text-xs">
          {#each navItems as item}
            <button
              onclick={() => {
                mobileNavOpen = false;
                navigate(item.key);
              }}
              class="p-2 rounded-lg text-left {app.route === item.key ? 'bg-indigo-600 text-white font-bold' : 'bg-slate-950 text-slate-300'}"
            >
              {item.icon} {item.label}
            </button>
          {/each}
          <button onclick={() => { mobileNavOpen = false; navigate('tl-demo'); }} class="p-2 rounded-lg text-left bg-slate-950 text-amber-300">👩‍💼 Team Lead Demo</button>
          <button onclick={() => { mobileNavOpen = false; navigate('benchmark'); }} class="p-2 rounded-lg text-left bg-slate-950 text-emerald-300">⚡ Benchmark</button>
        </div>
      {/if}

      <!-- Page View -->
      <main class="flex-1 p-4 lg:p-6 max-w-7xl w-full mx-auto">
        {#if app.route === 'dashboard'}
          <DashboardView />
        {:else if app.route === 'taskboard'}
          <TaskboardView />
        {:else if app.route === 'students'}
          <StudentsView />
        {:else if app.route === 'regular-feedback'}
          <FeedbackView />
        {:else if app.route === 'schedule'}
          <ScheduleView />
        {:else if app.route === 'assessment'}
          <AssessmentView />
        {:else if app.route === 'surveys'}
          <SurveysView />
        {:else if app.route === 'communication'}
          <CommView />
        {:else if app.route === 'profile'}
          <ProfileView />
        {:else if app.route === 'tl-demo'}
          <TeamLeadAndBenchmarkView mode="tl-demo" />
        {:else if app.route === 'benchmark'}
          <TeamLeadAndBenchmarkView mode="benchmark" />
        {/if}
      </main>
    </div>
  </div>

  <CommandPalette />
{/if}

<!-- Toast Notifications -->
<div class="fixed bottom-4 right-4 z-50 space-y-2 pointer-events-none">
  {#each app.toasts as t (t.id)}
    <div
      class="pointer-events-auto px-4 py-2.5 rounded-xl shadow-xl border text-xs font-semibold {t.kind === 'error'
        ? 'bg-rose-950/95 border-rose-500/50 text-rose-200'
        : t.kind === 'info'
          ? 'bg-slate-900/95 border-indigo-500/40 text-indigo-200'
          : 'bg-emerald-950/95 border-emerald-500/50 text-emerald-200'}"
    >
      {t.message}
    </div>
  {/each}
</div>

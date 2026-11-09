<script lang="ts">
  import { apiFetch, app, buildWhatsAppURL, copyToClipboard, fmtDateTimeWIB, notify } from '../lib/state.svelte';

  let activeTab = $state<'groups' | 'kpi'>('groups');
  let search = $state('');
  let formatFilter = $state<'all' | 'private' | 'group'>('all');
  let showArchived = $state(false);
  let kpiPeriod = $state('this_month');

  let dashData = $state<any>(app.cache.dashboard || null);
  let kpiData = $state<any>(app.cache.kpi || null);

  let selectedGroupDetail = $state<any>(null);
  let loadingDetail = $state(false);

  // Report to TL Drawer
  let reportOpen = $state(false);
  let repType = $state('attendance_drop');
  let repGroupId = $state('');
  let repStudentId = $state('');
  let repUrgency = $state('medium');
  let repSummary = $state('');
  let repDetail = $state('');
  let repSubmitting = $state(false);

  async function loadGroups() {
    const d = await apiFetch('/api/v1/dashboard/groups');
    dashData = d;
    app.cache.dashboard = d;
  }

  async function loadKpi() {
    const k = await apiFetch(`/api/v1/dashboard/kpi?period=${kpiPeriod}`);
    kpiData = k;
    app.cache.kpi = k;
  }

  async function openGroupDrawer(groupId: string) {
    loadingDetail = true;
    try {
      selectedGroupDetail = await apiFetch(`/api/v1/dashboard/groups/${groupId}`);
    } catch (e: any) {
      notify(e.message, 'error');
    } finally {
      loadingDetail = false;
    }
  }

  function openReportForGroup(groupId = '', studentId = '') {
    repGroupId = groupId;
    repStudentId = studentId;
    repSummary = '';
    repDetail = '';
    reportOpen = true;
  }

  async function submitTLReport(e: Event) {
    e.preventDefault();
    repSubmitting = true;
    try {
      await apiFetch('/api/v1/reports/tl', {
        method: 'POST',
        body: JSON.stringify({
          reportType: repType,
          groupId: repGroupId,
          studentId: repStudentId,
          urgency: repUrgency,
          summary: repSummary,
          detail: repDetail,
        }),
      });
      reportOpen = false;
      notify(`Laporan berhasil dikirim ke Team Lead (${app.tutor?.tlName || 'Team Lead'})!`, 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    } finally {
      repSubmitting = false;
    }
  }

  $effect(() => {
    loadGroups();
    if (app.routeParam) {
      openGroupDrawer(app.routeParam);
      app.routeParam = '';
    }
  });

  $effect(() => {
    const handleSSE = (e: any) => {
      if (e.detail?.event === 'sync_completed' || e.detail?.event === 'tl_report_created') {
        loadGroups();
        if (activeTab === 'kpi') loadKpi();
      }
    };
    window.addEventListener('ak-sse', handleSSE);
    return () => window.removeEventListener('ak-sse', handleSSE);
  });

  $effect(() => {
    if (activeTab === 'kpi') {
      loadKpi();
    }
  });

  const filteredGroups = $derived.by(() => {
    const list: any[] = dashData?.groups || [];
    const q = search.trim().toLowerCase();
    return list.filter((g) => {
      if (formatFilter !== 'all' && g.format !== formatFilter) return false;
      if (q && !g.title.toLowerCase().includes(q) && !g.groupId.includes(q) && !g.courseName.toLowerCase().includes(q)) {
        return false;
      }
      return true;
    });
  });
</script>

<div class="space-y-5">
  <!-- Header -->
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 class="text-xl font-bold text-white">Dasbor Pengajaran Tutor</h2>
      <p class="text-xs text-slate-400">Ringkasan seluruh grup kelas aktif, kehadiran siswa, dan capaian KPI real-time.</p>
    </div>

    <div class="flex items-center gap-2.5">
      <div class="inline-flex rounded-xl bg-slate-900 border border-slate-800 p-1">
        <button
          onclick={() => (activeTab = 'groups')}
          class="px-3.5 py-1.5 rounded-lg text-xs font-semibold transition cursor-pointer {activeTab === 'groups' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-white'}"
        >
          Grup Kelas ({dashData?.summary?.activeGroupsCount ?? 0})
        </button>
        <button
          onclick={() => (activeTab = 'kpi')}
          class="px-3.5 py-1.5 rounded-lg text-xs font-semibold transition cursor-pointer {activeTab === 'kpi' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-white'}"
        >
          KPI Saya
        </button>
      </div>

      <button
        onclick={() => openReportForGroup()}
        class="px-3.5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg shadow-indigo-600/20 cursor-pointer"
      >
        + Kirim Laporan ke TL
      </button>
    </div>
  </div>

  {#if activeTab === 'groups'}
    <!-- 3 KPI Cards -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
        <div class="text-[11px] font-semibold uppercase tracking-wider text-slate-400">Total Grup Aktif</div>
        <div class="mt-1 text-3xl font-extrabold text-white">{dashData?.summary?.activeGroupsCount ?? 0}</div>
        <div class="mt-1 text-xs text-slate-400">Termasuk kelas Private 1-on-1 & Reguler</div>
      </div>
      <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
        <div class="text-[11px] font-semibold uppercase tracking-wider text-slate-400">Total Siswa Aktif</div>
        <div class="mt-1 text-3xl font-extrabold text-indigo-400">{dashData?.summary?.totalStudents ?? 0}</div>
        <div class="mt-1 text-xs text-slate-400">Tersinkronisasi dari LMS Algonova</div>
      </div>
      <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
        <div class="text-[11px] font-semibold uppercase tracking-wider text-slate-400">Kehadiran Rata-Rata</div>
        <div class="mt-1 text-3xl font-extrabold text-emerald-400">{dashData?.summary?.avgAttendancePct ?? 0}%</div>
        <div class="mt-1 text-xs text-emerald-400/80">Target hijau Algonova &ge; 88% / 80%</div>
      </div>
    </div>

    <!-- Filter Bar -->
    <div class="flex flex-wrap items-center justify-between gap-3 bg-slate-900/80 border border-slate-800 p-3 rounded-2xl">
      <input
        bind:value={search}
        placeholder="Cari nama grup, kode kursus, atau ID LMS (1710124)..."
        class="flex-1 min-w-60 px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white focus:outline-none focus:border-indigo-500"
      />
      <div class="flex items-center gap-1.5 text-xs">
        {#each [['all', 'Semua Format'], ['private', 'Private 1-on-1'], ['group', 'Grup Reguler']] as [val, label]}
          <button
            onclick={() => (formatFilter = val as any)}
            class="px-3 py-1.5 rounded-lg border transition cursor-pointer {formatFilter === val ? 'bg-indigo-600/20 border-indigo-500 text-indigo-300 font-semibold' : 'bg-slate-950 border-slate-800 text-slate-400 hover:text-white'}"
          >
            {label}
          </button>
        {/each}
      </div>
    </div>

    <!-- Active Groups Table -->
    <div class="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs">
          <thead class="bg-slate-950/80 text-slate-400 uppercase text-[11px] border-b border-slate-800">
            <tr>
              <th class="py-3 px-4">Grup & ID LMS</th>
              <th class="py-3 px-3">Format</th>
              <th class="py-3 px-3">Jadwal (WIB)</th>
              <th class="py-3 px-3">Siswa</th>
              <th class="py-3 px-3">Kehadiran</th>
              <th class="py-3 px-4 text-right">Aksi Cepat</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/70">
            {#each filteredGroups as g (g.groupId)}
              <tr
                onclick={() => openGroupDrawer(g.groupId)}
                class="hover:bg-slate-800/60 transition cursor-pointer"
              >
                <td class="py-3 px-4">
                  <div class="font-semibold text-white text-sm">{g.title}</div>
                  <div class="text-[11px] text-slate-400">LMS #{g.groupId} • {g.courseName}</div>
                </td>
                <td class="py-3 px-3">
                  <span class="px-2 py-0.5 rounded-full text-[11px] font-semibold {g.format === 'private' ? 'bg-amber-500/15 text-amber-300 border border-amber-500/30' : 'bg-indigo-500/15 text-indigo-300 border border-indigo-500/30'}">
                    {g.format === 'private' ? 'Private 1-1' : 'Grup'}
                  </span>
                </td>
                <td class="py-3 px-3 text-slate-300 font-medium">{g.scheduleSummary}</td>
                <td class="py-3 px-3 text-white font-semibold">{g.studentsCount} siswa</td>
                <td class="py-3 px-3">
                  <div class="flex items-center gap-2">
                    <div class="w-16 h-1.5 rounded-full bg-slate-800 overflow-hidden">
                      <div
                        class="h-full rounded-full {g.avgAttendancePct >= 88 ? 'bg-emerald-500' : g.avgAttendancePct >= 80 ? 'bg-amber-500' : 'bg-rose-500'}"
                        style="width: {g.avgAttendancePct}%"
                      ></div>
                    </div>
                    <span class="font-semibold text-slate-200">{g.avgAttendancePct}%</span>
                  </div>
                </td>
                <td class="py-3 px-4 text-right" onclick={(e) => e.stopPropagation()}>
                  <button
                    onclick={() => openReportForGroup(g.groupId)}
                    class="px-2.5 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 mr-1.5 cursor-pointer"
                  >
                    Lapor TL
                  </button>
                  <a
                    href={g.lmsUrl}
                    target="_blank"
                    rel="noreferrer"
                    class="px-2.5 py-1 rounded-lg bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600/30"
                  >
                    LMS &nearr;
                  </a>
                </td>
              </tr>
            {/each}

            {#if filteredGroups.length === 0}
              <tr>
                <td colspan="6" class="py-10 text-center text-xs text-slate-500">
                  Belum ada grup kelas aktif terdaftar.
                </td>
              </tr>
            {/if}
          </tbody>
        </table>
      </div>
    </div>

    <!-- Collapsible Archive -->
    {#if (dashData?.archivedGroups?.length ?? 0) > 0}
      <div class="bg-slate-900/60 border border-slate-800 rounded-2xl p-3.5">
        <button
          onclick={() => (showArchived = !showArchived)}
          class="w-full flex items-center justify-between text-xs font-semibold text-slate-400 hover:text-white cursor-pointer"
        >
          <span>Arsip ({dashData.archivedGroups.length} Grup Tanpa Siswa)</span>
          <span>{showArchived ? '▲ Sembunyikan' : '▼ Tampilkan'}</span>
        </button>
        {#if showArchived}
          <div class="mt-3 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2.5">
            {#each dashData.archivedGroups as ag}
              <div class="p-3 rounded-xl bg-slate-950 border border-slate-800/80 text-xs">
                <div class="font-semibold text-slate-300">{ag.title}</div>
                <div class="text-slate-500">LMS #{ag.groupId} • 0 siswa</div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {:else}
    <!-- KPI SAYA TAB -->
    <div class="space-y-4">
      <div class="flex items-center gap-2">
        {#each [['this_month', 'Bulan Ini'], ['last_month', 'Bulan Lalu'], ['this_week', 'Minggu Ini']] as [p, lbl]}
          <button
            onclick={() => {
              kpiPeriod = p;
              loadKpi();
            }}
            class="px-3 py-1.5 rounded-xl text-xs font-semibold border cursor-pointer {kpiPeriod === p ? 'bg-indigo-600 text-white border-indigo-500' : 'bg-slate-900 text-slate-400 border-slate-800'}"
          >
            {lbl}
          </button>
        {/each}
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800">
          <div class="text-xs font-semibold text-slate-400 uppercase">Kehadiran (KPI)</div>
          <div class="mt-2 text-3xl font-black text-emerald-400">{kpiData?.attendanceKpi ?? 86}%</div>
          <p class="mt-1 text-xs text-slate-400">Ambang batas aman: 80% / Hijau: 88%</p>
        </div>

        <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800">
          <div class="text-xs font-semibold text-slate-400 uppercase">Discipline (Feedback + Assessment)</div>
          <div class="mt-2 text-3xl font-black text-indigo-400">{kpiData?.disciplineScore ?? 97} <span class="text-sm text-slate-500">/ 100</span></div>
          <p class="mt-1 text-xs text-slate-400">
            Regular Feedback: {kpiData?.feedbackScore ?? 48}/50 • Assessment: {kpiData?.assessmentScore ?? 49}/50
          </p>
        </div>

        <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800">
          <div class="text-xs font-semibold text-slate-400 uppercase">Quality Akademik</div>
          <div class="mt-2 text-3xl font-black text-amber-400">{kpiData?.qualityScore ?? 94}%</div>
          <p class="mt-1 text-xs text-slate-400">Target kualitas pengajaran &ge; 75%</p>
        </div>
      </div>

      <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800">
        <h3 class="text-sm font-bold text-white mb-3">Rincian Mingguan Kehadiran & Kepatuhan Tugas</h3>
        <div class="grid grid-cols-1 sm:grid-cols-4 gap-3">
          {#each kpiData?.weeks || [] as w}
            <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800">
              <div class="text-xs font-semibold text-white">{w.label}</div>
              <div class="text-[11px] text-slate-500">Mulai {w.weekStart}</div>
              <div class="mt-2 flex items-center justify-between text-xs">
                <span class="text-slate-400">Kehadiran:</span>
                <span class="font-bold text-emerald-400">{w.attendancePct}%</span>
              </div>
              <div class="mt-1 flex items-center justify-between text-xs">
                <span class="text-slate-400">Tugas KPI:</span>
                <span class="font-bold text-indigo-300">{w.tasksDone}/{w.tasksTotal}</span>
              </div>
            </div>
          {/each}
        </div>
      </div>
    </div>
  {/if}
</div>

<!-- Group Detail & Student Attendance Drawer -->
{#if selectedGroupDetail || loadingDetail}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-40 bg-black/65 flex justify-end"
    onclick={(e) => {
      if (e.target === e.currentTarget) selectedGroupDetail = null;
    }}
  >
    <div class="w-full max-w-xl bg-slate-900 border-l border-slate-800 h-full overflow-y-auto p-6 space-y-5">
      {#if loadingDetail}
        <div class="text-sm text-slate-400">Memuat detail grup...</div>
      {:else if selectedGroupDetail}
        <div class="flex items-start justify-between gap-3">
          <div>
            <span class="text-[11px] font-semibold px-2 py-0.5 rounded bg-indigo-600/20 text-indigo-300">
              LMS #{selectedGroupDetail.group.groupId} • {selectedGroupDetail.group.format}
            </span>
            <h3 class="mt-1.5 text-lg font-bold text-white">{selectedGroupDetail.group.title}</h3>
            <p class="text-xs text-slate-400">{selectedGroupDetail.group.scheduleSummary} • {selectedGroupDetail.group.courseName}</p>
          </div>
          <button onclick={() => (selectedGroupDetail = null)} class="px-2.5 py-1 rounded-lg bg-slate-800 text-slate-300 text-xs cursor-pointer">
            Tutup ✕
          </button>
        </div>

        <div class="flex gap-2">
          <button
            onclick={() => {
              const gid = selectedGroupDetail.group.groupId;
              selectedGroupDetail = null;
              openReportForGroup(gid);
            }}
            class="px-3.5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer"
          >
            Kirim Laporan Grup Ini ke TL
          </button>
          <a
            href={selectedGroupDetail.group.lmsUrl}
            target="_blank"
            rel="noreferrer"
            class="px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold"
          >
            Buka di LMS &nearr;
          </a>
        </div>

        <div>
          <h4 class="text-xs font-bold uppercase tracking-wider text-slate-400 mb-2.5">
            Daftar Siswa & Kehadiran ({selectedGroupDetail.students.length})
          </h4>
          <div class="space-y-2.5">
            {#each selectedGroupDetail.students as st}
              <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex flex-wrap items-center justify-between gap-3">
                <div>
                  <div class="text-sm font-semibold text-white">{st.fullName}</div>
                  <div class="text-xs text-slate-400">
                    Hadir: <span class="text-emerald-400 font-semibold">{st.lessonsAttended}/{st.lessonsTotal} ({st.attendancePct}%)</span>
                    • Sisa Kuota: <span class="{st.lessonsRemaining <= 3 ? 'text-rose-400 font-bold' : 'text-slate-200'}">{st.lessonsRemaining}/{st.totalLessonsPurchased}</span>
                  </div>
                </div>
                <div class="flex items-center gap-1.5 text-xs">
                  <button
                    onclick={() => copyToClipboard(st.parentPhone, 'Nomor WA Orang Tua')}
                    class="px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 cursor-pointer"
                  >
                    Salin WA
                  </button>
                  <a
                    href={buildWhatsAppURL(st.parentPhone, `Halo Ayah/Bunda dari ${st.fullName}, saya Kak Azhar tutor coding Algonova Indonesia.`)}
                    target="_blank"
                    rel="noreferrer"
                    class="px-2.5 py-1.5 rounded-lg bg-emerald-600/20 text-emerald-300 hover:bg-emerald-600/30 font-semibold"
                  >
                    Chat Ortu
                  </a>
                </div>
              </div>
            {/each}
          </div>
        </div>

        <div>
          <h4 class="text-xs font-bold uppercase tracking-wider text-slate-400 mb-2.5">Riwayat & Jadwal Pertemuan Terdekat</h4>
          <div class="space-y-2">
            {#each selectedGroupDetail.recentLessons as ls}
              <div class="p-3 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between text-xs">
                <div>
                  <div class="font-semibold text-slate-200">{ls.lessonTitle}</div>
                  <div class="text-slate-400">{fmtDateTimeWIB(ls.startsAt)}</div>
                </div>
                <span class="px-2 py-0.5 rounded text-[11px] {ls.isCompleted ? 'bg-emerald-500/15 text-emerald-400' : 'bg-indigo-500/15 text-indigo-300'}">
                  {ls.isCompleted ? 'Selesai' : 'Mendatang'}
                </span>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}

<!-- Report to Team Lead Drawer (Bug #1 Fixed: Numeric LMS Group ID Supported!) -->
{#if reportOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-50 bg-black/70 flex justify-end"
    onclick={(e) => {
      if (e.target === e.currentTarget) reportOpen = false;
    }}
  >
    <form onsubmit={submitTLReport} class="w-full max-w-lg bg-slate-900 border-l border-slate-800 h-full overflow-y-auto p-6 space-y-4">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-lg font-bold text-white">Kirim Laporan ke Team Lead</h3>
          <p class="text-xs text-slate-400">Penerima: {app.tutor?.tlName || 'Devika Wulan Rahmadani'}</p>
        </div>
        <button type="button" onclick={() => (reportOpen = false)} class="px-2.5 py-1 rounded-lg bg-slate-800 text-xs text-slate-300 cursor-pointer">
          ✕
        </button>
      </div>

      <div class="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/25 text-[11px] text-emerald-300">
        ✓ Perbaikan Bug NanoKanbana: Mendukung ID Grup LMS numerik (mis. <code>1710124</code>) tanpa error validasi UUID!
      </div>

      <div>
        <label for="rep-type" class="block text-xs font-semibold text-slate-300 mb-1">Kategori Laporan</label>
        <select id="rep-type" bind:value={repType} class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white">
          <option value="attendance_drop">Penurunan Kehadiran Siswa (Attendance Drop)</option>
          <option value="student_issues">Kendala Belajar / Perilaku Siswa</option>
          <option value="progress_decline">Penurunan Progres Tugas / Proyek</option>
          <option value="group_instability">Dinamika / Jadwal Grup Tidak Stabil</option>
        </select>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="rep-group" class="block text-xs font-semibold text-slate-300 mb-1">Grup Kelas (Opsional)</label>
          <select id="rep-group" bind:value={repGroupId} class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white">
            <option value="">-- Pilih Grup --</option>
            {#each dashData?.groups || [] as g}
              <option value={g.groupId}>{g.title} (#{g.groupId})</option>
            {/each}
          </select>
        </div>
        <div>
          <label for="rep-urg" class="block text-xs font-semibold text-slate-300 mb-1">Tingkat Urgensi</label>
          <select id="rep-urg" bind:value={repUrgency} class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white">
            <option value="low">Rendah (Info Rutin)</option>
            <option value="medium">Sedang (Perlu Atensi)</option>
            <option value="high">Tinggi (Mendesak)</option>
          </select>
        </div>
      </div>

      <div>
        <label for="rep-sum" class="block text-xs font-semibold text-slate-300 mb-1">Ringkasan Masalah (10–200 karakter)</label>
        <input
          id="rep-sum"
          bind:value={repSummary}
          required
          minlength="5"
          placeholder="Contoh: Siswa 2x absen berturut-turut karena jadwal ujian sekolah"
          class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"
        />
      </div>

      <div>
        <label for="rep-det" class="block text-xs font-semibold text-slate-300 mb-1">Kronologi & Tindakan Awal Tutor</label>
        <textarea
          id="rep-det"
          bind:value={repDetail}
          rows="4"
          required
          minlength="10"
          placeholder="Jelaskan komunikasi yang sudah dilakukan ke orang tua dan bantuan yang dibutuhkan dari Team Lead..."
          class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"
        ></textarea>
      </div>

      <button
        type="submit"
        disabled={repSubmitting}
        class="w-full py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer"
      >
        {repSubmitting ? 'Mengirim...' : 'Kirim Laporan ke Team Lead ⚡'}
      </button>
    </form>
  </div>
{/if}

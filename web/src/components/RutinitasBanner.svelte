<script lang="ts">
  import { apiFetch, navigate } from '../lib/state.svelte';

  interface RoutinesData {
    feedbackRoutine: { filled: number; total: number; pct: number; label: string };
    graduationRoutine: { pending: number; label: string };
    attendanceRoutine: { pct: number; done: number; total: number; label: string };
    assessmentRoutine: { cycleCode: string; status: string; label: string };
    surveyRoutine: { pendingCount: number; deadlineAt: string; label: string };
  }

  let data = $state<RoutinesData | null>(null);
  let loading = $state(true);

  async function loadSummary() {
    try {
      const res = await apiFetch<RoutinesData>('/api/v1/routines/summary');
      data = res;
    } catch {
      // ignore
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    loadSummary();
  });
</script>

{#if loading}
  <div class="p-3 rounded-2xl bg-slate-900/40 border border-slate-800/80 animate-pulse flex items-center gap-2 text-xs text-slate-500">
    <span class="w-2 h-2 rounded-full bg-indigo-500 animate-ping"></span>
    Memuat Ringkasan Rutinitas Mingguan...
  </div>
{:else if data}
  <div class="p-4 rounded-2xl bg-gradient-to-r from-slate-900/90 via-slate-900/70 to-indigo-950/40 border border-slate-800/90 shadow-sm space-y-3">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span class="text-xs font-black uppercase tracking-wider text-indigo-400">⚡ Rutinitas Mingguan Tutor</span>
        <span class="text-[11px] text-slate-400">• Sinkronisasi Operasional Algonova</span>
      </div>
      <button
        onclick={loadSummary}
        class="text-[11px] text-slate-400 hover:text-slate-200 transition cursor-pointer"
        title="Segarkan data rutinitas"
      >
        ↻ Refresh
      </button>
    </div>

    <div class="grid grid-cols-2 md:grid-cols-5 gap-2.5">
      <!-- 1. Regular Feedback -->
      <div class="p-3 rounded-xl bg-slate-950/80 border border-slate-800 flex flex-col justify-between">
        <div class="flex items-center justify-between">
          <span class="text-[10px] uppercase font-bold text-slate-400">Umpan Balik</span>
          <span class="text-[10px] font-bold px-1.5 py-0.5 rounded {data.feedbackRoutine.pct >= 90 ? 'bg-emerald-500/20 text-emerald-300' : 'bg-amber-500/20 text-amber-300'}">
            {data.feedbackRoutine.pct}%
          </span>
        </div>
        <div class="mt-2">
          <div class="text-sm font-black text-white">{data.feedbackRoutine.filled} / {data.feedbackRoutine.total}</div>
          <div class="text-[10px] text-slate-400 truncate">{data.feedbackRoutine.label}</div>
        </div>
        <button
          onclick={() => navigate('regular-feedback')}
          class="mt-2.5 w-full py-1 text-center rounded-lg bg-indigo-600/20 hover:bg-indigo-600/40 text-indigo-300 text-[11px] font-semibold cursor-pointer transition"
        >
          Isi Feedback →
        </button>
      </div>

      <!-- 2. Kelulusan Siswa -->
      <div class="p-3 rounded-xl bg-slate-950/80 border border-slate-800 flex flex-col justify-between">
        <div class="flex items-center justify-between">
          <span class="text-[10px] uppercase font-bold text-slate-400">Kelulusan</span>
          <span class="text-[10px] font-bold px-1.5 py-0.5 rounded bg-indigo-500/20 text-indigo-300">
            {data.graduationRoutine.pending} Siap
          </span>
        </div>
        <div class="mt-2">
          <div class="text-sm font-black text-white">{data.graduationRoutine.pending} Siswa</div>
          <div class="text-[10px] text-slate-400 truncate">{data.graduationRoutine.label}</div>
        </div>
        <button
          onclick={() => navigate('students')}
          class="mt-2.5 w-full py-1 text-center rounded-lg bg-indigo-600/20 hover:bg-indigo-600/40 text-indigo-300 text-[11px] font-semibold cursor-pointer transition"
        >
          Cek Siswa →
        </button>
      </div>

      <!-- 3. Kehadiran Siswa -->
      <div class="p-3 rounded-xl bg-slate-950/80 border border-slate-800 flex flex-col justify-between">
        <div class="flex items-center justify-between">
          <span class="text-[10px] uppercase font-bold text-slate-400">Kehadiran</span>
          <span class="text-[10px] font-bold px-1.5 py-0.5 rounded {data.attendanceRoutine.pct >= 90 ? 'bg-emerald-500/20 text-emerald-300' : 'bg-rose-500/20 text-rose-300'}">
            {data.attendanceRoutine.pct}%
          </span>
        </div>
        <div class="mt-2">
          <div class="text-sm font-black text-white">{data.attendanceRoutine.done} / {data.attendanceRoutine.total} Sesi</div>
          <div class="text-[10px] text-slate-400 truncate">{data.attendanceRoutine.label}</div>
        </div>
        <button
          onclick={() => navigate('schedule')}
          class="mt-2.5 w-full py-1 text-center rounded-lg bg-indigo-600/20 hover:bg-indigo-600/40 text-indigo-300 text-[11px] font-semibold cursor-pointer transition"
        >
          Cek Jadwal →
        </button>
      </div>

      <!-- 4. Assessment Tutor -->
      <div class="p-3 rounded-xl bg-slate-950/80 border border-slate-800 flex flex-col justify-between">
        <div class="flex items-center justify-between">
          <span class="text-[10px] uppercase font-bold text-slate-400">Assessment</span>
          <span class="text-[10px] font-bold px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300">
            {data.assessmentRoutine.status}
          </span>
        </div>
        <div class="mt-2">
          <div class="text-sm font-black text-white">{data.assessmentRoutine.cycleCode}</div>
          <div class="text-[10px] text-slate-400 truncate">{data.assessmentRoutine.label}</div>
        </div>
        <button
          onclick={() => navigate('assessment')}
          class="mt-2.5 w-full py-1 text-center rounded-lg bg-indigo-600/20 hover:bg-indigo-600/40 text-indigo-300 text-[11px] font-semibold cursor-pointer transition"
        >
          Lihat Nilai →
        </button>
      </div>

      <!-- 5. Survei Perpanjangan Siswa -->
      <div class="p-3 rounded-xl bg-slate-950/80 border border-slate-800 flex flex-col justify-between {data.surveyRoutine.pendingCount > 0 ? 'ring-1 ring-amber-500/40 bg-amber-950/10' : ''}">
        <div class="flex items-center justify-between">
          <span class="text-[10px] uppercase font-bold text-slate-400">Survei Siswa</span>
          {#if data.surveyRoutine.pendingCount > 0}
            <span class="text-[10px] font-bold px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300 animate-pulse">
              {data.surveyRoutine.pendingCount} Perlu Diisi
            </span>
          {:else}
            <span class="text-[10px] font-bold px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300">
              Lengkap
            </span>
          {/if}
        </div>
        <div class="mt-2">
          <div class="text-sm font-black text-white">{data.surveyRoutine.pendingCount} Evaluasi Pra-Renewal</div>
          <div class="text-[10px] text-slate-400 truncate">{data.surveyRoutine.label}</div>
        </div>
        <button
          onclick={() => navigate('surveys')}
          class="mt-2.5 w-full py-1 text-center rounded-lg bg-amber-500/20 hover:bg-amber-500/40 text-amber-300 text-[11px] font-semibold cursor-pointer transition"
        >
          Buka Survei →
        </button>
      </div>
    </div>
  </div>
{/if}

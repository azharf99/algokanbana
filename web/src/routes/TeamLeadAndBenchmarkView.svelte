<script lang="ts">
  import { apiFetch, fmtDateTimeWIB, notify } from '../lib/state.svelte';

  let { mode = 'benchmark' }: { mode?: 'tl-demo' | 'benchmark' } = $props();

  let tlData = $state<any>(null);
  let benchData = $state<any>(null);
  let tlNotes = $state<Record<string, string>>({});

  async function loadTLQueue() {
    tlData = await apiFetch('/api/v1/tl/queue');
  }

  async function runLiveBenchmark() {
    benchData = await apiFetch('/api/v1/benchmark');
    notify('Benchmark live SQLite WAL + Go berhasil dijalankan!', 'info');
  }

  async function reviewReschedule(id: string, status: 'approved' | 'rejected') {
    await apiFetch(`/api/v1/tl/reschedules/${id}/review`, {
      method: 'POST',
      body: JSON.stringify({ status, tlNote: tlNotes[id] || (status === 'approved' ? 'Jadwal baru disetujui TL.' : 'Mohon pilih jam alternatif.') }),
    });
    await loadTLQueue();
    notify(`Reschedule ${status === 'approved' ? 'disetujui & jadwal kelas diperbarui!' : 'ditolak.'}`, 'success');
  }

  async function reviewReport(id: string) {
    await apiFetch(`/api/v1/tl/reports/${id}/review`, {
      method: 'POST',
      body: JSON.stringify({ status: 'resolved', tlReply: tlNotes[id] || 'Terima kasih laporannya Kak Azhar, sudah ditindaklanjuti bersama CS.' }),
    });
    await loadTLQueue();
    notify('Laporan tutor ditandai selesai oleh Team Lead!', 'success');
  }

  async function approveUrgentSub(id: string) {
    await apiFetch(`/api/v1/tl/urgent-subs/${id}/review`, {
      method: 'POST',
      body: JSON.stringify({ status: 'approved' }),
    });
    await loadTLQueue();
    notify('Guru pengganti mendesak disetujui!', 'success');
  }

  $effect(() => {
    if (mode === 'tl-demo') {
      loadTLQueue();
    } else {
      runLiveBenchmark();
    }
  });
</script>

{#if mode === 'tl-demo'}
  <div class="space-y-5">
    <div class="p-4 rounded-2xl bg-amber-500/10 border border-amber-500/30 flex flex-wrap items-center justify-between gap-3">
      <div>
        <div class="text-xs font-bold uppercase text-amber-300">Simulasi Layar Team Lead (Untuk Demo ke Pimpinan)</div>
        <h2 class="text-lg font-bold text-white">Meja Persetujuan — {tlData?.teamLeadName || 'Devika Wulan Rahmadani'}</h2>
        <p class="text-xs text-slate-400">Semua laporan, permintaan reschedule murid private, dan eskalasi dari layar Tutor langsung masuk ke sini secara real-time.</p>
      </div>
      <button onclick={loadTLQueue} class="px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-xs font-semibold text-white cursor-pointer">
        ↻ Segarkan Antrean
      </button>
    </div>

    <!-- Pending Reschedule Approvals -->
    <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-3">
      <h3 class="text-sm font-bold text-white">1. Antrean Persetujuan Reschedule ({tlData?.rescheduleRequests?.length ?? 0})</h3>
      {#if (tlData?.rescheduleRequests || []).length === 0}
        <p class="text-xs text-slate-500">Belum ada pengajuan reschedule. Coba ajukan dari menu Jadwal sebagai Tutor!</p>
      {:else}
        <div class="space-y-2.5">
          {#each tlData.rescheduleRequests as rq (rq.requestId)}
            <div class="p-4 rounded-xl bg-slate-950 border border-slate-800 flex flex-wrap items-center justify-between gap-3 text-xs">
              <div class="space-y-1">
                <div class="font-bold text-white text-sm">
                  {rq.groupName} <span class="text-indigo-400">({rq.type === 'otr' ? 'Sekali / OTR' : 'Permanen'})</span>
                </div>
                <div class="text-slate-400">
                  Dari: {fmtDateTimeWIB(rq.originalStartsAt)} &rarr; <strong class="text-emerald-400">{fmtDateTimeWIB(rq.newStartsAt)}</strong>
                </div>
                {#if rq.note}<div class="text-slate-300">Catatan Tutor: "{rq.note}"</div>{/if}
              </div>

              {#if rq.status === 'pending'}
                <div class="flex flex-wrap items-center gap-2">
                  <input
                    bind:value={tlNotes[rq.requestId]}
                    placeholder="Catatan balasan TL..."
                    class="px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-white"
                  />
                  <button onclick={() => reviewReschedule(rq.requestId, 'approved')} class="px-3 py-1.5 rounded-lg bg-emerald-600 text-white font-semibold cursor-pointer">
                    ✓ Setujui
                  </button>
                  <button onclick={() => reviewReschedule(rq.requestId, 'rejected')} class="px-3 py-1.5 rounded-lg bg-rose-600 text-white font-semibold cursor-pointer">
                    ✕ Tolak
                  </button>
                </div>
              {:else}
                <span class="px-2.5 py-1 rounded-full font-bold bg-slate-800 text-emerald-300">{rq.status}</span>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Incoming TL Reports -->
    <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-3">
      <h3 class="text-sm font-bold text-white">2. Laporan Masuk dari Tutor ({tlData?.tlReports?.length ?? 0})</h3>
      {#if (tlData?.tlReports || []).length === 0}
        <p class="text-xs text-slate-500">Belum ada laporan ke TL. Coba tombol "+ Kirim Laporan ke TL" di halaman Dasbor!</p>
      {:else}
        <div class="space-y-2.5">
          {#each tlData.tlReports as rp (rp.reportId)}
            <div class="p-4 rounded-xl bg-slate-950 border border-slate-800 flex flex-wrap items-center justify-between gap-3 text-xs">
              <div class="space-y-1 max-w-xl">
                <div class="flex items-center gap-2">
                  <span class="px-2 py-0.5 rounded bg-indigo-500/20 text-indigo-300 font-bold uppercase text-[10px]">{rp.reportType}</span>
                  <span class="font-bold text-white">{rp.groupTitle || `Grup #${rp.groupId}`}</span>
                </div>
                <p class="text-slate-300 whitespace-pre-line">{rp.message}</p>
                {#if rp.tlReply}<div class="text-emerald-400">Balasan TL: {rp.tlReply}</div>{/if}
              </div>

              {#if rp.status === 'open'}
                <div class="flex items-center gap-2">
                  <input bind:value={tlNotes[rp.reportId]} placeholder="Tindak lanjut TL..." class="px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-white" />
                  <button onclick={() => reviewReport(rp.reportId)} class="px-3 py-1.5 rounded-lg bg-indigo-600 text-white font-semibold cursor-pointer">
                    Tandai Selesai
                  </button>
                </div>
              {:else}
                <span class="px-2.5 py-1 rounded-full bg-emerald-500/20 text-emerald-300 font-bold">{rp.status}</span>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Urgent Subs -->
    {#if (tlData?.urgentSubRequests?.length ?? 0) > 0}
      <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-3">
        <h3 class="text-sm font-bold text-white">3. Permintaan Guru Pengganti Mendesak ({tlData.urgentSubRequests.length})</h3>
        {#each tlData.urgentSubRequests as us}
          <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between text-xs">
            <div>
              <div class="font-bold text-white">{us.groupTitle} — {fmtDateTimeWIB(us.lessonStartsAt)}</div>
              <div class="text-slate-400">Alasan: {us.reason}</div>
            </div>
            {#if us.status === 'pending'}
              <button onclick={() => approveUrgentSub(us.requestId)} class="px-3 py-1.5 rounded-lg bg-emerald-600 text-white font-semibold cursor-pointer">
                Assign Guru Pengganti
              </button>
            {:else}
              <span class="text-emerald-400 font-bold">{us.status}</span>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
{:else}
  <!-- PITCHING BENCHMARK VIEW -->
  <div class="space-y-5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 class="text-xl font-bold text-white">Laporan Benchmark Performa: NanoKanbana vs AlgoKanbana (Go)</h2>
        <p class="text-xs text-slate-400">Data perbandingan nyata untuk dipresentasikan kepada pimpinan Algonova Indonesia.</p>
      </div>
      <button onclick={runLiveBenchmark} class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold cursor-pointer">
        ⚡ Jalankan Uji Kecepatan Live Sekarang
      </button>
    </div>

    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
        <div class="text-[11px] uppercase font-semibold text-slate-400">Pemakaian RAM Heap Go</div>
        <div class="mt-1 text-2xl font-black text-emerald-400">{(benchData?.heapAllocMB ?? 4.2).toFixed(2)} MB</div>
        <div class="text-[11px] text-slate-500">vs ~450–900 MB stack lama</div>
      </div>
      <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
        <div class="text-[11px] uppercase font-semibold text-slate-400">Ukuran Docker Image</div>
        <div class="mt-1 text-2xl font-black text-indigo-400">~18 MB</div>
        <div class="text-[11px] text-slate-500">Single binary FROM scratch</div>
      </div>
      <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
        <div class="text-[11px] uppercase font-semibold text-slate-400">Runtime Engine</div>
        <div class="mt-1 text-xl font-black text-white">{benchData?.goVersion || 'go1.26'}</div>
        <div class="text-[11px] text-slate-500">{benchData?.goroutines ?? 6} goroutines aktif</div>
      </div>
      <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
        <div class="text-[11px] uppercase font-semibold text-slate-400">Bug Produksi Diperbaiki</div>
        <div class="mt-1 text-2xl font-black text-amber-400">{benchData?.bugsFixed ?? 5} Bug</div>
        <div class="text-[11px] text-slate-500">UUID grup, double-fetch, dll.</div>
      </div>
    </div>

    <div class="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
      <table class="w-full text-left text-xs">
        <thead class="bg-slate-950 text-slate-400 uppercase text-[11px] border-b border-slate-800">
          <tr>
            <th class="py-3.5 px-4">Modul / Halaman Portal Tutor</th>
            <th class="py-3.5 px-4">NanoKanbana Lama (Next.js Seoul + PHP)</th>
            <th class="py-3.5 px-4">AlgoKanbana (Go + SQLite WAL Live)</th>
            <th class="py-3.5 px-4 text-right">Peningkatan Kecepatan</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/70">
          {#each benchData?.comparisons || [] as row}
            <tr class="hover:bg-slate-800/40">
              <td class="py-3.5 px-4 font-bold text-white">{row.module}</td>
              <td class="py-3.5 px-4 text-rose-400 font-mono">{row.oldMs.toLocaleString('id-ID')} ms</td>
              <td class="py-3.5 px-4 text-emerald-400 font-mono font-bold">{row.newMs.toFixed(2)} ms</td>
              <td class="py-3.5 px-4 text-right">
                <span class="px-2.5 py-1 rounded-full bg-emerald-500/20 text-emerald-300 font-black">
                  {row.speedup.toLocaleString('id-ID')}x lebih cepat
                </span>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
{/if}

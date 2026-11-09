<script lang="ts">
  import { apiFetch, app, copyToClipboard, fmtDateTimeWIB, fmtTimeOnlyWIB, notify, uploadFiles } from '../lib/state.svelte';

  let tab = $state<'week' | 'list'>('week');
  let schedData = $state<any>(app.cache.schedule || null);
  let selectedSlots = $state<string[]>([]);
  let formatFilter = $state<'all' | 'private' | 'group'>('all');

  // Reschedule Drawer
  let reschOpen = $state(false);
  let reschLessonId = $state('');
  let reschType = $state<'otr' | 'permanent'>('otr');
  let reschDate = $state('');
  let reschTime = $state('16:00');
  let reschNote = $state('');
  let reschProofs = $state<string[]>([]);
  let reschUploading = $state(false);
  let reschSubmitting = $state(false);

  // Request More Load Modal
  let loadModal = $state<string | null>(null);
  let loadReason = $state('');

  const days = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'];

  async function loadSchedule() {
    const d = await apiFetch('/api/v1/schedule');
    schedData = d;
    app.cache.schedule = d;
  }

  function slotKey(weekday: number, startMin: number) {
    return `${weekday}:${startMin}`;
  }

  function formatMin(m: number) {
    const h = Math.floor(m / 60);
    const min = m % 60;
    return `${String(h).padStart(2, '0')}:${String(min).padStart(2, '0')}`;
  }

  function toggleSlot(weekday: number, startMin: number, curStatus: string) {
    if (curStatus === 'booked' || curStatus === 'booked_for_group') {
      notify('Slot sudah terisi kelas aktif. Gunakan menu Reschedule untuk memindah jadwal.', 'info');
      return;
    }
    const k = slotKey(weekday, startMin);
    if (selectedSlots.includes(k)) {
      selectedSlots = selectedSlots.filter((x) => x !== k);
    } else {
      selectedSlots = [...selectedSlots, k];
    }
  }

  async function applySlotStatus(status: 'open' | 'closed') {
    if (selectedSlots.length === 0) return;
    const coords = selectedSlots.map((s) => {
      const [w, m] = s.split(':').map(Number);
      return { weekday: w, startMin: m };
    });
    await apiFetch('/api/v1/schedule/availability', {
      method: 'PUT',
      body: JSON.stringify({ slots: coords, status }),
    });
    selectedSlots = [];
    await loadSchedule();
    notify(`Status ${coords.length} slot ketersediaan berhasil diperbarui!`, 'success');
  }

  function openRescheduleDrawer(lessonId = '') {
    const list: any[] = schedData?.lessons || [];
    if (lessonId) {
      reschLessonId = lessonId;
    } else {
      const priv = list.find((l) => l.format === 'private' && !l.isCompleted) || list[0];
      reschLessonId = priv?.lessonId || '';
    }
    const d = new Date(Date.now() + 3 * 86400_000);
    reschDate = d.toISOString().slice(0, 10);
    reschTime = '10:00';
    reschNote = '';
    reschProofs = [];
    reschOpen = true;
  }

  async function handleProofUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files?.length) return;
    reschUploading = true;
    try {
      const urls = await uploadFiles(input.files, 'proof');
      reschProofs = [...reschProofs, ...urls].slice(0, 3);
      notify('Bukti konfirmasi orang tua berhasil diunggah!', 'info');
    } catch (err: any) {
      notify(err.message, 'error');
    } finally {
      reschUploading = false;
    }
  }

  async function submitReschedule(e: Event) {
    e.preventDefault();
    reschSubmitting = true;
    try {
      const newStartsAt = `${reschDate}T${reschTime}:00+07:00`;
      await apiFetch('/api/v1/reschedules', {
        method: 'POST',
        body: JSON.stringify({
          lmsLessonId: reschLessonId,
          type: reschType,
          newStartsAt,
          note: reschNote,
          proofUrls: reschProofs,
        }),
      });
      reschOpen = false;
      await loadSchedule();
      notify('Permintaan perubahan jadwal terkirim! Cek juga di Mode Demo Team Lead untuk simulasi approve.', 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    } finally {
      reschSubmitting = false;
    }
  }

  async function withdrawReschedule(id: string) {
    await apiFetch(`/api/v1/reschedules/${id}`, { method: 'DELETE' });
    await loadSchedule();
    notify('Permintaan reschedule berhasil ditarik.', 'info');
  }

  async function submitMoreLoad(e: Event) {
    e.preventDefault();
    if (!loadModal) return;
    try {
      await apiFetch('/api/v1/workload-requests', {
        method: 'POST',
        body: JSON.stringify({ reqType: loadModal, reason: loadReason }),
      });
      loadModal = null;
      loadReason = '';
      notify('Permintaan tambahan beban kerja terkirim ke Team Lead!', 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  $effect(() => {
    loadSchedule();
  });

  $effect(() => {
    const handleSSE = (e: any) => {
      if (e.detail?.event === 'sync_completed' || e.detail?.event === 'reschedule_status_changed') {
        loadSchedule();
      }
    };
    window.addEventListener('ak-sse', handleSSE);
    return () => window.removeEventListener('ak-sse', handleSSE);
  });

  const slotMap = $derived.by(() => {
    const m: Record<string, any> = {};
    (schedData?.availabilitySlots || []).forEach((s: any) => {
      m[`${s.weekday}:${s.startMin}`] = s;
    });
    return m;
  });

  const timeRows = $derived.by(() => {
    const arr: number[] = [];
    for (let m = 480; m <= 1320; m += 60) arr.push(m); // Hourly compact rows 08:00..22:00 with :00 & :30 cells
    return arr;
  });

  const filteredLessons = $derived.by(() => {
    const list: any[] = schedData?.lessons || [];
    return list.filter((l) => formatFilter === 'all' || l.format === formatFilter);
  });
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 class="text-xl font-bold text-white">Jadwal Mengajar, Slot Ketersediaan & Reschedule</h2>
      <p class="text-xs text-slate-400">
        Waktu server LMS (<code class="text-amber-300">UTC+03:00 Moskow</code>) otomatis dinormalisasi ke <strong class="text-emerald-400">WIB (UTC+07:00)</strong> dalam &lt; 2 ms.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <div class="inline-flex rounded-xl bg-slate-900 border border-slate-800 p-1">
        <button
          onclick={() => (tab = 'week')}
          class="px-3 py-1.5 rounded-lg text-xs font-semibold cursor-pointer {tab === 'week' ? 'bg-indigo-600 text-white' : 'text-slate-400'}"
        >
          Grid 217 Slot Mingguan
        </button>
        <button
          onclick={() => (tab = 'list')}
          class="px-3 py-1.5 rounded-lg text-xs font-semibold cursor-pointer {tab === 'list' ? 'bg-indigo-600 text-white' : 'text-slate-400'}"
        >
          Daftar Sesi Kelas ({filteredLessons.length})
        </button>
      </div>

      <button
        onclick={() => openRescheduleDrawer()}
        class="px-3.5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg shadow-indigo-600/20 cursor-pointer"
      >
        🗓️ Minta Penjadwalan Ulang (Reschedule)
      </button>
    </div>
  </div>

  <!-- 4-Week Workload Strip -->
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
    <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
      <div class="text-[11px] uppercase font-semibold text-slate-400">Jam Sibuk (4 Pekan)</div>
      <div class="mt-1 text-2xl font-black text-white">{schedData?.workload?.busyHours ?? 0}j</div>
    </div>
    <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
      <div class="text-[11px] uppercase font-semibold text-slate-400">Jam Luang Tersedia</div>
      <div class="mt-1 text-2xl font-black text-emerald-400">{schedData?.workload?.freeHours ?? 0}j</div>
    </div>
    <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
      <div class="text-[11px] uppercase font-semibold text-slate-400">Beban Kerja Total</div>
      <div class="mt-1 text-2xl font-black text-indigo-400">{schedData?.workload?.loadPct ?? 0}%</div>
    </div>
    <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800">
      <div class="text-[11px] uppercase font-semibold text-slate-400">Kapasitas Luang</div>
      <div class="mt-1 text-2xl font-black text-amber-400">{schedData?.workload?.freePct ?? 0}% <span class="text-xs text-slate-500">/ 160j</span></div>
    </div>
  </div>

  <!-- Active Reschedule Requests Banner -->
  {#if (schedData?.rescheduleRequests?.length ?? 0) > 0}
    <div class="p-4 rounded-2xl bg-slate-900 border border-indigo-500/30 space-y-2.5">
      <div class="text-xs font-bold uppercase tracking-wider text-indigo-300">Permintaan Perubahan Jadwal Saya</div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-2.5">
        {#each schedData.rescheduleRequests as rq (rq.requestId)}
          <div class="p-3 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between gap-2 text-xs">
            <div>
              <div class="font-bold text-white">{rq.groupName} ({rq.type === 'otr' ? 'Sekali / OTR' : 'Permanen'})</div>
              <div class="text-slate-400">Jadwal Baru: <span class="text-emerald-400 font-semibold">{fmtDateTimeWIB(rq.newStartsAt)}</span></div>
              {#if rq.tlNote}<div class="text-[11px] text-indigo-300 mt-0.5">Catatan TL: {rq.tlNote}</div>{/if}
            </div>
            <div class="flex items-center gap-2">
              <span class="px-2 py-0.5 rounded-full text-[11px] font-bold {rq.status === 'approved' ? 'bg-emerald-500/20 text-emerald-300' : rq.status === 'pending' ? 'bg-amber-500/20 text-amber-300' : 'bg-slate-800 text-slate-400'}">
                {rq.status}
              </span>
              {#if rq.status === 'pending'}
                <button onclick={() => withdrawReschedule(rq.requestId)} class="px-2 py-1 rounded bg-slate-800 text-rose-300 cursor-pointer">
                  Tarik
                </button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  {#if tab === 'week'}
    <!-- Availability Slot Editor -->
    <div class="bg-slate-900 border border-slate-800 rounded-2xl p-4 space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div class="flex flex-wrap items-center gap-3 text-xs">
          <span class="font-bold text-white">Editor Ketersediaan Slot (08:00 – 22:30 WIB):</span>
          <span class="inline-flex items-center gap-1 text-emerald-400"><span class="w-2.5 h-2.5 rounded-xs bg-emerald-500 inline-block"></span> Buka</span>
          <span class="inline-flex items-center gap-1 text-indigo-300"><span class="w-2.5 h-2.5 rounded-xs bg-indigo-600 inline-block"></span> Terisi Grup</span>
          <span class="inline-flex items-center gap-1 text-amber-300"><span class="w-2.5 h-2.5 rounded-xs bg-amber-500 inline-block"></span> Terisi Private 1-1</span>
          <span class="inline-flex items-center gap-1 text-slate-500"><span class="w-2.5 h-2.5 rounded-xs bg-slate-800 inline-block"></span> Tutup</span>
        </div>

        {#if selectedSlots.length > 0}
          <div class="flex items-center gap-2 text-xs">
            <span class="text-indigo-300 font-semibold">{selectedSlots.length} slot dipilih</span>
            <button onclick={() => applySlotStatus('open')} class="px-3 py-1 rounded-lg bg-emerald-600 text-white font-semibold cursor-pointer">Buka Slot</button>
            <button onclick={() => applySlotStatus('closed')} class="px-3 py-1 rounded-lg bg-slate-700 text-white font-semibold cursor-pointer">Tutup Slot</button>
            <button onclick={() => (selectedSlots = [])} class="px-2 py-1 rounded-lg bg-slate-800 text-slate-400 cursor-pointer">Reset</button>
          </div>
        {/if}
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-xs border-collapse">
          <thead>
            <tr class="text-slate-400 border-b border-slate-800">
              <th class="py-2 px-2 text-left w-20">Jam WIB</th>
              {#each days as d}
                <th class="py-2 px-1.5 text-center">{d}</th>
              {/each}
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/50">
            {#each timeRows as hourMin}
              <tr>
                <td class="py-1.5 px-2 font-mono text-[11px] text-slate-400">{formatMin(hourMin)}</td>
                {#each [0, 1, 2, 3, 4, 5, 6] as wd}
                  <td class="p-1">
                    <div class="grid grid-cols-2 gap-1">
                      {#each [hourMin, hourMin + 30] as sm}
                        {@const sl = slotMap[`${wd}:${sm}`]}
                        {@const st = sl?.status || 'available'}
                        {@const isSel = selectedSlots.includes(`${wd}:${sm}`)}
                        <button
                          onclick={() => toggleSlot(wd, sm, st)}
                          title="{days[wd]} {formatMin(sm)} ({st})"
                          class="h-6 rounded-md text-[10px] font-semibold transition cursor-pointer border {isSel
                            ? 'ring-2 ring-white border-white'
                            : 'border-transparent'} {st === 'booked'
                            ? 'bg-amber-500/30 text-amber-200'
                            : st === 'booked_for_group'
                              ? 'bg-indigo-600/35 text-indigo-200'
                              : st === 'available' || st === 'avail_for_group'
                                ? 'bg-emerald-500/20 hover:bg-emerald-500/35 text-emerald-300'
                                : 'bg-slate-950 hover:bg-slate-800 text-slate-500'}"
                        >
                          {sm % 60 === 0 ? ':00' : ':30'}
                        </button>
                      {/each}
                    </div>
                  </td>
                {/each}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {:else}
    <!-- Lessons List & Quick Reschedule Trigger -->
    <div class="space-y-3">
      <div class="flex gap-2 text-xs">
        {#each [['all', 'Semua Kelas'], ['private', 'Khusus Murid Private (1-on-1)'], ['group', 'Grup Reguler']] as [k, lbl]}
          <button
            onclick={() => (formatFilter = k as any)}
            class="px-3 py-1.5 rounded-xl border cursor-pointer {formatFilter === k ? 'bg-indigo-600 border-indigo-500 text-white font-semibold' : 'bg-slate-900 border-slate-800 text-slate-400'}"
          >
            {lbl}
          </button>
        {/each}
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        {#each filteredLessons as ls (ls.lessonId)}
          <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800 flex items-center justify-between gap-3">
            <div>
              <div class="flex items-center gap-2">
                <span class="text-[10px] font-bold uppercase px-2 py-0.5 rounded {ls.format === 'private' ? 'bg-amber-500/20 text-amber-300' : 'bg-indigo-500/20 text-indigo-300'}">
                  {ls.format === 'private' ? 'Private 1-1' : 'Grup'}
                </span>
                <span class="text-xs text-slate-400">Pertemuan #{ls.courseLessonNumber}</span>
              </div>
              <div class="mt-1 text-sm font-bold text-white">{ls.groupTitle}</div>
              <div class="text-xs text-emerald-400 font-medium">{fmtDateTimeWIB(ls.startsAt)} ({fmtTimeOnlyWIB(ls.endsAt)})</div>
            </div>

            <div class="flex flex-col gap-1.5 text-xs">
              <div class="flex gap-1.5">
                <button onclick={() => copyToClipboard(ls.meetingUrl, 'Link Kelas')} class="px-2.5 py-1 rounded-lg bg-slate-800 text-slate-200 cursor-pointer">
                  Link
                </button>
                <a href={ls.prepUrl} target="_blank" rel="noreferrer" class="px-2.5 py-1 rounded-lg bg-slate-800 text-slate-200">Prep</a>
              </div>
              <button
                onclick={() => openRescheduleDrawer(ls.lessonId)}
                class="px-2.5 py-1 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold cursor-pointer"
              >
                Reschedule
              </button>
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Request More Load Card -->
  <div class="p-4 rounded-2xl bg-slate-900 border border-slate-800 flex flex-wrap items-center justify-between gap-3">
    <div>
      <h3 class="text-sm font-bold text-white">Minta Tambahan Beban Mengajar</h3>
      <p class="text-xs text-slate-400">Punya jam luang tambahan? Ajukan slot murid baru atau format kelas tertentu ke Team Lead.</p>
    </div>
    <div class="flex gap-2">
      <button onclick={() => (loadModal = 'request_more_students')} class="px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-xs font-semibold text-white cursor-pointer">
        + Lebih Banyak Siswa
      </button>
      <button onclick={() => (loadModal = 'request_specific_format')} class="px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-xs font-semibold text-white cursor-pointer">
        + Minta Format Tertentu
      </button>
    </div>
  </div>
</div>

<!-- Reschedule Drawer (Private Intensive & Group) -->
{#if reschOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 bg-black/75 flex justify-end" onclick={(e) => e.target === e.currentTarget && (reschOpen = false)}>
    <form onsubmit={submitReschedule} class="w-full max-w-lg bg-slate-900 border-l border-slate-800 h-full overflow-y-auto p-6 space-y-4">
      <div class="flex items-start justify-between">
        <div>
          <h3 class="text-lg font-bold text-white">Permintaan Perubahan Jadwal (Reschedule)</h3>
          <p class="text-xs text-slate-400">Mendukung murid Private (Intensive 1-on-1) maupun Grup Reguler.</p>
        </div>
        <button type="button" onclick={() => (reschOpen = false)} class="px-2.5 py-1 rounded bg-slate-800 text-xs text-slate-300 cursor-pointer">✕</button>
      </div>

      <div>
        <label for="rs-lesson" class="block text-xs font-semibold text-slate-300 mb-1">Pilih Sesi Kelas yang Akan Dipindahkan</label>
        <select id="rs-lesson" bind:value={reschLessonId} required class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white">
          {#each schedData?.lessons || [] as l}
            <option value={l.lessonId}>
              [{l.format === 'private' ? 'PRIVATE' : 'GRUP'}] {l.groupTitle} — {fmtDateTimeWIB(l.startsAt)}
            </option>
          {/each}
        </select>
      </div>

      <div class="grid grid-cols-2 gap-2.5 text-xs">
        <button
          type="button"
          onclick={() => (reschType = 'otr')}
          class="p-3 rounded-xl border text-left cursor-pointer {reschType === 'otr' ? 'bg-indigo-600/20 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400'}"
        >
          <div class="font-bold">Sekali Saja (OTR)</div>
          <div class="text-[11px] opacity-80">Ganti jadwal 1 sesi pertemuan ini</div>
        </button>
        <button
          type="button"
          onclick={() => (reschType = 'permanent')}
          class="p-3 rounded-xl border text-left cursor-pointer {reschType === 'permanent' ? 'bg-indigo-600/20 border-indigo-500 text-white' : 'bg-slate-950 border-slate-800 text-slate-400'}"
        >
          <div class="font-bold">Permanen</div>
          <div class="text-[11px] opacity-80">Ubah jadwal rutin seterusnya</div>
        </button>
      </div>

      <div class="grid grid-cols-2 gap-3 text-xs">
        <div>
          <label for="rs-dt" class="block font-semibold text-slate-300 mb-1">Tanggal Baru (WIB)</label>
          <input id="rs-dt" type="date" bind:value={reschDate} required class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
        </div>
        <div>
          <label for="rs-tm" class="block font-semibold text-slate-300 mb-1">Jam Mulai Baru (WIB)</label>
          <input id="rs-tm" type="time" bind:value={reschTime} required class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
        </div>
      </div>

      <div>
        <label for="rs-note" class="block text-xs font-semibold text-slate-300 mb-1">Catatan Konfirmasi dengan Orang Tua</label>
        <textarea
          id="rs-note"
          bind:value={reschNote}
          rows="3"
          required
          placeholder="Contoh: Murid private izin acara sekolah, orang tua sepakat pindah ke Kamis pukul 10:00 WIB."
          class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"
        ></textarea>
      </div>

      <div>
        <label for="rs-proof" class="block text-xs font-semibold text-slate-300 mb-1">Bukti Chat Persetujuan Orang Tua (1–3 Gambar)</label>
        <input id="rs-proof" type="file" accept="image/*" multiple onchange={handleProofUpload} class="text-xs text-slate-400" />
        {#if reschUploading}<div class="text-xs text-indigo-400 mt-1">Mengunggah gambar...</div>{/if}
        {#if reschProofs.length > 0}
          <div class="mt-2 flex gap-2">
            {#each reschProofs as _}
              <span class="text-[11px] px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-300">✓ Bukti terlampir</span>
            {/each}
          </div>
        {/if}
      </div>

      <div class="p-3 rounded-xl bg-slate-950 border border-slate-800 text-[11px] text-slate-400">
        Sistem otomatis memvalidasi bentrok jadwal kelas lain serta aturan batas pengajuan pekan lalu (Senin 23:59 WIB).
      </div>

      <button type="submit" disabled={reschSubmitting} class="w-full py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer">
        {reschSubmitting ? 'Memvalidasi Jadwal...' : 'Kirim Permintaan Reschedule ⚡'}
      </button>
    </form>
  </div>
{/if}

<!-- Request More Load Modal -->
{#if loadModal}
  <div class="fixed inset-0 z-50 bg-black/75 flex items-center justify-center p-4">
    <form onsubmit={submitMoreLoad} class="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-5 space-y-4">
      <h3 class="text-base font-bold text-white">
        {loadModal === 'request_more_students' ? 'Minta Tambahan Siswa Baru' : 'Minta Format Kelas Tertentu'}
      </h3>
      <textarea
        bind:value={loadReason}
        required
        minlength="10"
        rows="3"
        placeholder="Sebutkan hari/jam ketersediaan dan kursus yang diinginkan (minimal 10 karakter)..."
        class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"
      ></textarea>
      <div class="flex justify-end gap-2">
        <button type="button" onclick={() => (loadModal = null)} class="px-3.5 py-2 rounded-xl bg-slate-800 text-xs text-slate-300 cursor-pointer">Batal</button>
        <button type="submit" class="px-4 py-2 rounded-xl bg-indigo-600 text-white text-xs font-semibold cursor-pointer">Kirim ke TL</button>
      </div>
    </form>
  </div>
{/if}

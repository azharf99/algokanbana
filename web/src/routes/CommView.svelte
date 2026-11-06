<script lang="ts">
  import { apiFetch, app, fmtDateTimeWIB, notify, refreshBootstrap, uploadFiles } from '../lib/state.svelte';

  let tab = $state<'inbox' | 'kb'>('inbox');
  let articles = $state<any[]>([]);
  let kbQuery = $state('');
  let selectedArticle = $state<any>(null);

  let threads = $state<any[]>([]);
  let activeThread = $state<any>(null);
  let replyBody = $state('');

  // Compose Request Modal
  let composeOpen = $state(false);
  let reqType = $state('lesson_issue');
  let reqPriority = $state('medium');
  let reqStudentRef = $state('');
  let reqGroupRef = $state('');
  let reqDesc = $state('');
  let reqAttachment = $state('');

  // Escalate Modal
  let escOpen = $state(false);
  let escTeam = $state('cs');
  let escSeverity = $state('high');
  let escReason = $state('');

  async function loadKB() {
    const res = await apiFetch(`/api/v1/kb/articles?q=${encodeURIComponent(kbQuery)}`);
    articles = res.articles || [];
    if (articles.length > 0 && !selectedArticle) selectedArticle = articles[0];
  }

  async function loadThreads() {
    const res = await apiFetch('/api/v1/threads');
    threads = res.threads || [];
    if (threads.length > 0 && !activeThread) {
      openThread(threads[0].threadId);
    }
  }

  async function openThread(id: string) {
    const res = await apiFetch(`/api/v1/threads/${id}`);
    activeThread = res.thread;
    await apiFetch(`/api/v1/threads/${id}/read`, { method: 'POST' });
    await refreshBootstrap();
  }

  async function sendReply(e: Event) {
    e.preventDefault();
    if (!activeThread || !replyBody.trim()) return;
    await apiFetch(`/api/v1/threads/${activeThread.threadId}/messages`, {
      method: 'POST',
      body: JSON.stringify({ body: replyBody }),
    });
    replyBody = '';
    await openThread(activeThread.threadId);
    await loadThreads();
  }

  async function handlePdfUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files?.length) return;
    const urls = await uploadFiles(input.files, 'attachment');
    if (urls[0]) reqAttachment = urls[0];
    notify('Lampiran berhasil diunggah!', 'info');
  }

  async function submitNewRequest(e: Event) {
    e.preventDefault();
    try {
      const res = await apiFetch('/api/v1/requests', {
        method: 'POST',
        body: JSON.stringify({
          requestType: reqType,
          priority: reqPriority,
          relatedStudentRef: reqStudentRef,
          relatedGroupRef: reqGroupRef,
          description: reqDesc,
          attachmentUrl: reqAttachment,
        }),
      });
      composeOpen = false;
      reqDesc = '';
      reqStudentRef = '';
      reqGroupRef = '';
      tab = 'inbox';
      await loadThreads();
      await openThread(res.thread.threadId);
      notify('Permintaan bantuan berhasil dikirim!', 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  async function submitEscalation(e: Event) {
    e.preventDefault();
    if (!activeThread) return;
    try {
      await apiFetch(`/api/v1/threads/${activeThread.threadId}/escalate`, {
        method: 'POST',
        body: JSON.stringify({ severity: escSeverity, targetTeam: escTeam, reason: escReason }),
      });
      escOpen = false;
      escReason = '';
      await openThread(activeThread.threadId);
      await loadThreads();
      notify('Percakapan berhasil dieskalasi!', 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  async function createTaskFromThread() {
    if (!activeThread) return;
    await apiFetch('/api/v1/tasks', {
      method: 'POST',
      body: JSON.stringify({
        title: `Tindak Lanjut Chat: ${activeThread.subject.slice(0, 50)}`,
        description: activeThread.lastMessage || '',
        priority: 'medium',
        threadId: activeThread.threadId,
      }),
    });
    notify('Tugas tindak lanjut ditambahkan ke Papan Tugas!', 'success');
  }

  $effect(() => {
    loadThreads();
    loadKB();
  });
</script>

<div class="space-y-4">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 class="text-xl font-bold text-white">Bantuan, Basis Pengetahuan & Komunikasi</h2>
      <p class="text-xs text-slate-400">Bebas double-fetch SSR & mendukung input nama siswa/nomor grup fleksibel (Fix Bug #2, #3, #5).</p>
    </div>

    <div class="flex items-center gap-2">
      <div class="inline-flex rounded-xl bg-slate-900 border border-slate-800 p-1">
        <button onclick={() => (tab = 'inbox')} class="px-3.5 py-1.5 rounded-lg text-xs font-semibold cursor-pointer {tab === 'inbox' ? 'bg-indigo-600 text-white' : 'text-slate-400'}">
          Komunikasi Inbox ({threads.length})
        </button>
        <button onclick={() => (tab = 'kb')} class="px-3.5 py-1.5 rounded-lg text-xs font-semibold cursor-pointer {tab === 'kb' ? 'bg-indigo-600 text-white' : 'text-slate-400'}">
          Basis Pengetahuan SOP ({articles.length})
        </button>
      </div>

      <button onclick={() => (composeOpen = true)} class="px-3.5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer">
        + Buat Permintaan Baru
      </button>
    </div>
  </div>

  {#if tab === 'inbox'}
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 min-h-112">
      <!-- Thread List -->
      <div class="bg-slate-900 border border-slate-800 rounded-2xl p-3 space-y-2 overflow-y-auto">
        {#each threads as th (th.threadId)}
          <button
            onclick={() => openThread(th.threadId)}
            class="w-full text-left p-3 rounded-xl border transition cursor-pointer {activeThread?.threadId === th.threadId ? 'bg-slate-800 border-indigo-500' : 'bg-slate-950 border-slate-800 hover:border-slate-700'}"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="text-[10px] font-bold uppercase px-2 py-0.5 rounded bg-indigo-500/20 text-indigo-300">
                &rarr; {th.targetTeam}
              </span>
              {#if th.unreadTutor > 0}
                <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
              {/if}
            </div>
            <div class="mt-1.5 text-xs font-bold text-white line-clamp-1">{th.subject}</div>
            <div class="mt-0.5 text-[11px] text-slate-400 line-clamp-1">{th.lastMessage}</div>
          </button>
        {/each}
      </div>

      <!-- Active Chat Timeline -->
      <div class="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-2xl p-4 flex flex-col justify-between">
        {#if activeThread}
          <div>
            <div class="flex flex-wrap items-center justify-between gap-2 pb-3 border-b border-slate-800">
              <div>
                <h3 class="text-sm font-bold text-white">{activeThread.subject}</h3>
                <div class="text-[11px] text-slate-400">
                  Divisi: {activeThread.targetTeam.toUpperCase()}
                  {#if activeThread.relatedStudentRef} • Siswa: {activeThread.relatedStudentRef}{/if}
                  {#if activeThread.relatedGroupRef} • Grup: #{activeThread.relatedGroupRef}{/if}
                </div>
              </div>
              <div class="flex gap-1.5 text-xs">
                <button onclick={createTaskFromThread} class="px-2.5 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 cursor-pointer">
                  + Jadikan Tugas
                </button>
                <button onclick={() => (escOpen = true)} class="px-2.5 py-1 rounded-lg bg-amber-600/20 text-amber-300 cursor-pointer">
                  ↑ Eskalasi Divisi
                </button>
              </div>
            </div>

            <div class="py-4 space-y-3 max-h-80 overflow-y-auto">
              {#each activeThread.messages || [] as m (m.messageId)}
                <div class="p-3.5 rounded-2xl text-xs {m.senderRole === 'tutor' ? 'bg-indigo-600/20 border border-indigo-500/30 ml-8' : 'bg-slate-950 border border-slate-800 mr-8'}">
                  <div class="flex items-center justify-between text-[11px] mb-1">
                    <span class="font-bold text-white">{m.senderName}</span>
                    <span class="text-slate-500">{fmtDateTimeWIB(m.createdAt)}</span>
                  </div>
                  <p class="text-slate-200 whitespace-pre-line leading-relaxed">{m.body}</p>
                  {#if m.attachmentUrl}
                    <a href={m.attachmentUrl} target="_blank" class="inline-block mt-2 text-indigo-300 underline">📎 Buka Lampiran</a>
                  {/if}
                </div>
              {/each}
            </div>
          </div>

          <form onsubmit={sendReply} class="pt-3 border-t border-slate-800 flex gap-2">
            <input
              bind:value={replyBody}
              placeholder="Tulis balasan sebagai {app.session?.activeRole === 'team_lead' ? 'Team Lead (Devika)' : 'Tutor'}..."
              class="flex-1 px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white focus:outline-none focus:border-indigo-500"
            />
            <button type="submit" class="px-4 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer">
              Kirim
            </button>
          </form>
        {/if}
      </div>
    </div>
  {:else}
    <!-- Clean Deduplicated Knowledge Base -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
      <div class="bg-slate-900 border border-slate-800 rounded-2xl p-3 space-y-2.5">
        <input
          bind:value={kbQuery}
          oninput={() => loadKB()}
          placeholder="Cari SOP reschedule, feedback, KPI..."
          class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"
        />
        {#each articles as art}
          <button
            onclick={() => (selectedArticle = art)}
            class="w-full text-left p-3 rounded-xl border cursor-pointer {selectedArticle?.articleId === art.articleId ? 'bg-slate-800 border-indigo-500' : 'bg-slate-950 border-slate-800'}"
          >
            <span class="text-[10px] font-bold text-indigo-400 uppercase">{art.category}</span>
            <div class="text-xs font-bold text-white mt-0.5">{art.title}</div>
          </button>
        {/each}
      </div>

      <div class="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-4">
        {#if selectedArticle}
          <span class="text-xs font-semibold px-2.5 py-1 rounded-full bg-indigo-500/20 text-indigo-300">{selectedArticle.category}</span>
          <h3 class="text-lg font-bold text-white">{selectedArticle.title}</h3>
          <p class="text-xs text-slate-400">{selectedArticle.summary}</p>
          <div class="p-4 rounded-xl bg-slate-950 border border-slate-800 text-xs text-slate-200 whitespace-pre-line leading-relaxed">
            {selectedArticle.contentMd}
          </div>
          <div class="flex items-center justify-between pt-2">
            <span class="text-xs text-slate-500">👍 {selectedArticle.helpfulCount} tutor terbantu artikel ini</span>
            <button onclick={() => (composeOpen = true)} class="px-3.5 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-xs font-semibold text-white cursor-pointer">
              Belum terjawab? Kirim Permintaan ke TL/CS
            </button>
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<!-- Compose Support Request Modal (Bug #2 Fixed!) -->
{#if composeOpen}
  <div class="fixed inset-0 z-50 bg-black/75 flex items-center justify-center p-4">
    <form onsubmit={submitNewRequest} class="w-full max-w-lg bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-3.5">
      <div class="flex items-center justify-between">
        <h3 class="text-base font-bold text-white">Kirim Permintaan Bantuan (TL / CS / Tech)</h3>
        <button type="button" onclick={() => (composeOpen = false)} class="text-xs text-slate-400 cursor-pointer">✕</button>
      </div>

      <div class="grid grid-cols-2 gap-3 text-xs">
        <select bind:value={reqType} class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white">
          <option value="lesson_issue">Kendala Kelas / Pelajaran (&rarr; TL)</option>
          <option value="change_schedule">Perubahan Jadwal Khusus (&rarr; TL)</option>
          <option value="parent_issue">Komunikasi Orang Tua / Paket (&rarr; CS)</option>
          <option value="student_transfer">Transfer Grup Siswa (&rarr; CS)</option>
          <option value="technical_issue">Masalah Teknis LMS (&rarr; Tech)</option>
        </select>
        <select bind:value={reqPriority} class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white">
          <option value="low">Prioritas Rendah</option>
          <option value="medium">Prioritas Sedang</option>
          <option value="high">Prioritas Tinggi</option>
          <option value="critical">Kritis</option>
        </select>
      </div>

      <div class="grid grid-cols-2 gap-3 text-xs">
        <input bind:value={reqStudentRef} placeholder="Nama atau ID Siswa (mis. Nadia Putri)" class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
        <input bind:value={reqGroupRef} placeholder="Nomor Grup LMS (mis. 1710125)" class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
      </div>

      <textarea bind:value={reqDesc} required minlength="10" rows="3" placeholder="Jelaskan detail permintaan bantuan minimal 10 karakter..." class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"></textarea>

      <div class="text-xs">
        <label for="req-pdf" class="text-slate-400 block mb-1">Lampiran PDF / Screenshot (Opsional)</label>
        <input id="req-pdf" type="file" accept=".pdf,image/*" onchange={handlePdfUpload} class="text-xs text-slate-400" />
      </div>

      <div class="flex justify-end gap-2">
        <button type="button" onclick={() => (composeOpen = false)} class="px-3.5 py-2 rounded-xl bg-slate-800 text-xs text-slate-300 cursor-pointer">Batal</button>
        <button type="submit" class="px-4 py-2 rounded-xl bg-indigo-600 text-white text-xs font-semibold cursor-pointer">Kirim Permintaan</button>
      </div>
    </form>
  </div>
{/if}

{#if escOpen}
  <div class="fixed inset-0 z-50 bg-black/75 flex items-center justify-center p-4">
    <form onsubmit={submitEscalation} class="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-5 space-y-3.5">
      <h3 class="text-base font-bold text-white">Eskalasi Percakapan Lintas Divisi</h3>
      <div class="grid grid-cols-2 gap-2 text-xs">
        <select bind:value={escTeam} class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white">
          <option value="tl">Team Lead (TL)</option>
          <option value="cs">Customer Support (CS)</option>
          <option value="ism">Academic / ISM</option>
          <option value="tech">Technical Support</option>
        </select>
        <select bind:value={escSeverity} class="px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white">
          <option value="medium">Medium</option>
          <option value="high">High</option>
          <option value="critical">Critical</option>
        </select>
      </div>
      <textarea bind:value={escReason} required minlength="10" rows="3" placeholder="Alasan eskalasi (minimal 10 karakter)..." class="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs text-white"></textarea>
      <div class="flex justify-end gap-2">
        <button type="button" onclick={() => (escOpen = false)} class="px-3.5 py-2 rounded-xl bg-slate-800 text-xs text-slate-300 cursor-pointer">Batal</button>
        <button type="submit" class="px-4 py-2 rounded-xl bg-amber-600 text-white text-xs font-semibold cursor-pointer">Eskalasi Sekarang</button>
      </div>
    </form>
  </div>
{/if}

<script lang="ts">
  import { apiFetch, app, notify, refreshBootstrap, uploadFiles } from '../lib/state.svelte';

  let firstName = $state('');
  let lastName = $state('');
  let phone = $state('');
  let bio = $state('');
  let specTags = $state('');
  let subjTags = $state('');
  let tgDeepLink = $state('');

  $effect(() => {
    if (app.tutor) {
      firstName = app.tutor.firstName || '';
      lastName = app.tutor.lastName || '';
      phone = app.tutor.phone || '';
      bio = app.tutor.bio || '';
      specTags = (app.tutor.specializationTags || []).join(', ');
      subjTags = (app.tutor.subjectTags || []).join(', ');
    }
  });

  async function saveProfile(e: Event) {
    e.preventDefault();
    try {
      await apiFetch('/api/v1/profile', {
        method: 'PATCH',
        body: JSON.stringify({
          firstName,
          lastName,
          phone,
          bio,
          specializationTags: specTags.split(',').map((s) => s.trim()).filter(Boolean),
          subjectTags: subjTags.split(',').map((s) => s.trim()).filter(Boolean),
        }),
      });
      await refreshBootstrap();
      notify('Profil tutor berhasil diperbarui!', 'success');
    } catch (err: any) {
      notify(err.message, 'error');
    }
  }

  async function handleAsset(e: Event, kind: 'avatar' | 'cv') {
    const input = e.target as HTMLInputElement;
    if (!input.files?.length) return;
    await uploadFiles(input.files, kind);
    await refreshBootstrap();
    notify(`${kind === 'cv' ? 'CV PDF' : 'Foto profil'} berhasil diperbarui!`, 'success');
  }

  async function tgAction(action: string) {
    const res = await apiFetch('/api/v1/profile/telegram', {
      method: 'POST',
      body: JSON.stringify({ action }),
    });
    if (res.data?.deepLink) tgDeepLink = res.data.deepLink;
    await refreshBootstrap();
    notify('Pengaturan Telegram diperbarui!', 'info');
  }
</script>

<div class="space-y-5 max-w-4xl">
  <div>
    <h2 class="text-xl font-bold text-white">Profil Tutor & Integrasi Notifikasi</h2>
    <p class="text-xs text-slate-400">Kelola informasi pengajar, unggah CV, dan tautkan notifikasi Bot Telegram.</p>
  </div>

  <!-- Team Lead Banner -->
  <div class="p-4 rounded-2xl bg-indigo-600/15 border border-indigo-500/30 flex items-center justify-between">
    <div>
      <div class="text-[11px] font-bold uppercase text-indigo-300">Team Lead Anda</div>
      <div class="text-base font-bold text-white">{app.tutor?.tlName || 'Belum ditugaskan'}</div>
    </div>
    <span class="text-xs px-3 py-1 rounded-full bg-emerald-500/20 text-emerald-300 font-semibold">
      Status: {app.tutor?.status || 'active'} ({app.tutor?.employmentType || 'part_time'})
    </span>
  </div>

  <form onsubmit={saveProfile} class="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-4">
    <h3 class="text-sm font-bold text-white">Informasi Profil Pengajar</h3>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
      <div>
        <label for="pf-fn" class="block text-slate-400 mb-1">Nama Depan</label>
        <input id="pf-fn" bind:value={firstName} required class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
      </div>
      <div>
        <label for="pf-ln" class="block text-slate-400 mb-1">Nama Belakang</label>
        <input id="pf-ln" bind:value={lastName} required class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
      </div>
      <div>
        <label for="pf-em" class="block text-slate-400 mb-1">Email Akun (Read-Only)</label>
        <input id="pf-em" value={app.tutor?.email || ''} disabled class="w-full px-3.5 py-2 rounded-xl bg-slate-950/50 border border-slate-800 text-slate-500" />
      </div>
      <div>
        <label for="pf-ph" class="block text-slate-400 mb-1">Nomor WhatsApp Aktif</label>
        <input id="pf-ph" bind:value={phone} class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
      </div>
    </div>

    <div class="text-xs">
      <label for="pf-bio" class="block text-slate-400 mb-1">Bio Singkat Tutor</label>
      <textarea id="pf-bio" bind:value={bio} rows="3" class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white"></textarea>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
      <div>
        <label for="pf-sp" class="block text-slate-400 mb-1">Tag Spesialisasi (pisahkan koma)</label>
        <input id="pf-sp" bind:value={specTags} class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
      </div>
      <div>
        <label for="pf-sb" class="block text-slate-400 mb-1">Bahasa Pemrograman & Tools</label>
        <input id="pf-sb" bind:value={subjTags} class="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-white" />
      </div>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2 text-xs">
      <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800">
        <div class="font-semibold text-white mb-1">Unggah Foto Profil (PNG/JPG/WEBP)</div>
        <input type="file" accept="image/*" onchange={(e) => handleAsset(e, 'avatar')} class="text-slate-400" />
      </div>
      <div class="p-3.5 rounded-xl bg-slate-950 border border-slate-800">
        <div class="font-semibold text-white mb-1">Unggah CV Tutor (PDF)</div>
        <input type="file" accept=".pdf" onchange={(e) => handleAsset(e, 'cv')} class="text-slate-400" />
        {#if app.tutor?.cvUrl}
          <a href={app.tutor.cvUrl} target="_blank" class="inline-block mt-1 text-indigo-400 underline">Unduh CV Tersimpan</a>
        {/if}
      </div>
    </div>

    <button type="submit" class="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold cursor-pointer">
      Simpan Perubahan Profil
    </button>
  </form>

  <!-- Telegram Bot Notification Card -->
  <div class="p-5 rounded-2xl bg-slate-900 border border-slate-800 flex flex-wrap items-center justify-between gap-4">
    <div>
      <h3 class="text-sm font-bold text-white">Notifikasi Bot Telegram</h3>
      <p class="text-xs text-slate-400">
        {#if app.tutor?.telegramHandle}
          Terhubung ke <strong class="text-emerald-400">{app.tutor.telegramHandle}</strong>
        {:else}
          Belum ada akun Telegram tertaut.
        {/if}
      </p>
      {#if tgDeepLink}
        <div class="mt-2 text-xs text-indigo-300">Link Aktivasi: <code class="bg-slate-950 px-2 py-0.5 rounded">{tgDeepLink}</code></div>
      {/if}
    </div>

    <div class="flex gap-2 text-xs">
      {#if app.tutor?.telegramHandle}
        <button onclick={() => tgAction('unlink')} class="px-3.5 py-2 rounded-xl bg-rose-600/20 text-rose-300 font-semibold cursor-pointer">
          Putuskan Tautan
        </button>
      {:else}
        <button onclick={() => tgAction('generate_code')} class="px-3.5 py-2 rounded-xl bg-indigo-600 text-white font-semibold cursor-pointer">
          Buat Kode Tautan
        </button>
        <button onclick={() => tgAction('relink_demo')} class="px-3.5 py-2 rounded-xl bg-slate-800 text-slate-200 cursor-pointer">
          Tautkan Ulang @azhar_faturohman
        </button>
      {/if}
    </div>
  </div>
</div>

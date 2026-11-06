package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"algokanbana/internal/config"
	"algokanbana/internal/curriculum"
	"algokanbana/internal/store"
	"algokanbana/internal/timeutil"

	"github.com/google/uuid"
)

const (
	DefaultTutorID = "8dc8052b-3720-4889-806d-5023fb799252"
	DefaultUserID  = "7b808d3d-da54-4aeb-aa0d-d4bbe244af31"
	DefaultTLID    = "43dc3434-5324-4067-b928-a447d022a5bf"
)

func EnsureSeeded(ctx context.Context, st *store.Store, mode config.AppMode) error {
	if mode == config.ModeProduction {
		// In production, never insert mock/dummy tutors or mock classes!
		return ensureStandardKBArticles(ctx, st)
	}

	empty, err := st.IsEmpty(ctx)
	if err != nil {
		return err
	}
	if empty {
		if err := runFullSeed(ctx, st); err != nil {
			return err
		}
	}
	return ensureSurveyTargetsSeeded(ctx, st)
}

func CleanAllSeededData(ctx context.Context, st *store.Store) error {
	tables := []string{
		"tutors", "groups", "students", "lessons", "lesson_students",
		"tasks", "urgent_sub_requests", "regular_feedback_reports", "mt_feedback_entries",
		"reschedule_requests", "workload_requests", "tl_reports", "assessment_submissions",
		"threads", "messages", "saved_views", "portal_events", "survey_targets", "survey_submissions",
	}
	tx, err := st.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, tbl := range tables {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", tbl)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func ensureStandardKBArticles(ctx context.Context, st *store.Store) error {
	var count int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM kb_articles").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	kbSeed := []store.KBArticle{
		{
			ArticleID: "kb-reschedule-private", Category: "Jadwal & Reschedule", Locale: "id",
			Title: "SOP Perubahan Jadwal (Reschedule) Kelas Private 1-on-1 & Grup",
			Summary: "Aturan batas pengajuan Senin 23:59 WIB, perbedaan Sekali (OTR) vs Permanen, dan syarat bukti persetujuan orang tua.",
			ContentMD: "### Ketentuan Reschedule Kelas\n1. **Sekali (One-Time Reschedule / OTR)**: Digunakan apabila murid *private/intensive* berhalangan hadir pada 1 pertemuan tertentu dan sepakat mengganti di hari/jam lain.\n2. **Permanen (`permanent`)**: Mengubah jadwal rutin mingguan untuk seterusnya (hanya berlaku untuk pertemuan mendatang dan minimal tersisa 2 sesi).\n3. **Batas Waktu Grace Period**: Sesi pekan lalu masih dapat diajukan maksimal hingga **hari Senin pukul 23:59 WIB**.\n4. **Bukti Konfirmasi**: Lampirkan 1–3 tangkapan layar persetujuan orang tua/CS.",
			HelpfulCount: 42,
		},
		{
			ArticleID: "kb-regular-feedback", Category: "Regular Feedback", Locale: "id",
			Title: "Panduan Penulisan Regular Feedback Pertemuan #4, #8, #12, dst.",
			Summary: "Standar kualitas umpan balik orang tua setiap 4 pertemuan beserta ekspor kartu PDF A4 & pengiriman WhatsApp.",
			ContentMD: "### Standar Regular Feedback Coding & Design\n- Wajib diselesaikan setiap kelipatan pertemuan ke-4 paling lambat **Minggu pukul 23:59 WIB**.\n- Pilih kode kursus dan bulan berjalan agar topik modul sesuai kurikulum.\n- Sertakan minimal satu capaian spesifik siswa dan tautan proyek (Scratch/Roblox/GitHub/Figma) bila tersedia.",
			HelpfulCount: 38,
		},
		{
			ArticleID: "kb-urgent-sub", Category: "Operasional Kelas", Locale: "id",
			Title: "Ketentuan Pengajuan Guru Pengganti Mendesak (Urgent Sub)",
			Summary: "Prosedur pengajuan guru pengganti dalam jendela 50 menit s/d 72 jam ke depan akibat kendala darurat.",
			ContentMD: "### Pengajuan Urgent Sub\n- Berlaku jika tutor mengalami kendala mendesak (sakit mendadak, pemadaman listrik/koneksi internet, musibah).\n- Wajib melampirkan minimal 1 bukti foto/surat pendukung.\n- Status pengajuan dapat dipantau langsung di Papan Tugas dan ditarik kembali jika kendala teratasi sebelum kelas dimulai.",
			HelpfulCount: 29,
		},
		{
			ArticleID: "kb-tl-report", Category: "Komunikasi & Eskalasi", Locale: "id",
			Title: "Kapan Harus Mengirimkan Laporan ke Team Lead?",
			Summary: "Kriteria pengiriman laporan siswa mangkir 2x berturut-turut, kendala pedagogi, maupun eskalasi perpanjangan.",
			ContentMD: "### Kriteria Laporan ke Team Lead\n1. Siswa izin atau mangkir 2 kali berturut-turut tanpa kabar jelas.\n2. Siswa mengalami kesulitan pemahaman materi tingkat lanjut yang memerlukan penyesuaian kurikulum.\n3. Masukan khusus dari orang tua yang membutuhkan tindak lanjut tim Customer Support.",
			HelpfulCount: 35,
		},
	}
	for _, a := range kbSeed {
		_, _ = st.DB().ExecContext(ctx, `INSERT OR IGNORE INTO kb_articles (article_id, category, locale, title, summary, content_md, helpful_count) VALUES (?,?,?,?,?,?,?)`,
			a.ArticleID, a.Category, a.Locale, a.Title, a.Summary, a.ContentMD, a.HelpfulCount)
	}
	return nil
}

func ensureSurveyTargetsSeeded(ctx context.Context, st *store.Store) error {
	var count int
	if err := st.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM survey_targets").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now().In(timeutil.JakartaLoc)
	nowStr := timeutil.FormatWIB(now)

	surveySeed := []struct {
		sid      string
		stuId    string
		name     string
		group    string
		course   string
		rem      int
		daysAway int
	}{
		{uuid.NewString(), "880101", "Kenzie Pratama", "IND 1-1 Kenzie Pratama (PP2)", "Python Pro (Year 2 - AI & Backend)", 1, 3},
		{uuid.NewString(), "880102", "Nadia Putri", "IND 1-1 Nadia Putri (RB2.0)", "Roblox Studio 2.0 (Lua Scripting)", 2, 5},
		{uuid.NewString(), "880103", "Rafa Alfarizi", "IND 1-1 Rafa Alfarizi (PS)", "Python Start (Year 1)", 3, 7},
		{uuid.NewString(), "880104", "Kirana Maheswari", "IND 1-1 Kirana Maheswari (AI)", "Artificial Intelligence & Prompt Engineering", 4, 10},
	}
	for _, sv := range surveySeed {
		deadlineStr := timeutil.FormatWIB(now.AddDate(0, 0, sv.daysAway))
		_, _ = st.DB().ExecContext(ctx, `
			INSERT OR IGNORE INTO survey_targets (survey_id, tutor_id, student_id, student_name, group_title, course_name, lessons_remaining, deadline_at, status, created_at)
			VALUES (?,?,?,?,?,?,?,?,'pending',?)`,
			sv.sid, DefaultTutorID, sv.stuId, sv.name, sv.group, sv.course, sv.rem, deadlineStr, nowStr)
	}
	return nil
}

func runFullSeed(ctx context.Context, st *store.Store) error {

	db := st.DB()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().In(timeutil.JakartaLoc)
	nowStr := timeutil.FormatWIB(now)
	weekMon := timeutil.MondayOfWeekWIB(now)

	// 1. Seed Primary Tutor Profile (Azhar Faturohman Ahidin)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tutors (
			tutor_id, user_id, lms_user_id, login, first_name, last_name, email, phone, bio,
			photo_url, cv_url, specialization_tags_json, subject_tags_json, employment_type, status,
			country_code, course_types_json, lesson_formats_json, tl_id, tl_name,
			telegram_handle, telegram_enabled, telegram_link_code, density_pref, locale_pref,
			available_roles_json, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		DefaultTutorID, DefaultUserID, 2000723, "azharfa",
		"Azhar", "Faturohman Ahidin", "azhar.ahidin@algonova.id", "+6281295001995",
		"Coding & Software Engineering Tutor di Algonova Indonesia (Python, Roblox Lua, Web Frontend, Unity & AI Track).",
		"", "",
		`["Python Pro","Roblox Studio","Web Frontend","Intensive 1-on-1","AI Track"]`,
		`["Python","Lua","JavaScript","HTML/CSS","Scratch","C# Unity"]`,
		"part_time", "active", "ID", `["coding"]`, `["group","private"]`,
		DefaultTLID, "Devika Wulan Rahmadani",
		"@azhar_faturohman", 1, "", "default", "id",
		`["tutor","team_lead"]`, nowStr,
	)
	if err != nil {
		return fmt.Errorf("seed tutor: %w", err)
	}

	// 2. Seed 27 Active Groups (9 Private Intensive 1-on-1 + 18 Regular Groups) + 5 Archived Groups
	type groupSpec struct {
		id         string
		title      string
		code       string
		name       string
		format     string
		weekday    int // 0=Mon..6=Sun
		startHour  int
		startMin   int
		stuCount   int
		attendance int
		progress   int
		milestone  int // course lesson number this week
	}

	specs := []groupSpec{
		{"1710124", "IND 1-1 Kenzie Pratama (PP2)", "PP2", "Python Pro (Year 2 - AI & Backend)", "private", 0, 16, 0, 1, 96, 88, 8},
		{"1710125", "IND 1-1 Nadia Putri (RB2.0)", "RB2.0", "Roblox Studio 2.0 (Lua Scripting)", "private", 1, 16, 30, 1, 92, 85, 4},
		{"1710126", "IND 1-1 Rafa Alfarizi (PS)", "PS", "Python Start (Year 1)", "private", 2, 15, 30, 1, 88, 80, 12},
		{"1710127", "IND 1-1 Kirana Maheswari (AI)", "AI", "Artificial Intelligence & Prompt Engineering", "private", 3, 17, 0, 1, 100, 94, 4},
		{"1710128", "IND 1-1 Bima Sakti (GD)", "GD", "Unity 3D Game Development (C#)", "private", 4, 16, 0, 1, 84, 78, 7},
		{"1710129", "IND 1-1 Evelyn Tan (FE)", "FE", "Frontend Development (HTML/CSS/JS/React)", "private", 5, 10, 0, 1, 95, 91, 8},
		{"1710130", "IND 1-1 Daffa Ramadhan (PP1)", "PP1", "Python Pro (Year 1)", "private", 5, 13, 0, 1, 90, 86, 15},
		{"1710131", "IND 1-1 Alicia Wijaya (SC)", "SC", "Scratch Visual Programming", "private", 6, 9, 30, 1, 94, 89, 16},
		{"1710132", "IND 1-1 Reyhan Saputra (MC)", "MC", "Minecraft Coding (MakeCode & Python)", "private", 2, 18, 30, 1, 78, 74, 6},

		{"1710201", "ID-PS-MON-1830 (Python Start)", "PS", "Python Start (Year 1)", "group", 0, 18, 30, 3, 89, 82, 4},
		{"1710202", "ID-RB2-TUE-1830 (Roblox 2.0)", "RB2.0", "Roblox Studio 2.0 (Lua Scripting)", "group", 1, 18, 30, 3, 87, 84, 8},
		{"1710203", "ID-SC-WED-1700 (Scratch Kids)", "SC", "Scratch Visual Programming", "group", 2, 17, 0, 3, 91, 88, 11},
		{"1710204", "ID-PP1-THU-1900 (Python Pro 1)", "PP1", "Python Pro (Year 1)", "group", 3, 19, 0, 3, 85, 79, 12},
		{"1710205", "ID-FE-FRI-1830 (Frontend Web)", "FE", "Frontend Development (HTML/CSS/JS/React)", "group", 4, 18, 30, 3, 88, 85, 5},
		{"1710206", "ID-PS-SAT-0800 (Python Start A)", "PS", "Python Start (Year 1)", "group", 5, 8, 0, 3, 84, 80, 16},
		{"1710207", "ID-RB1-SAT-1130 (Roblox 1.0)", "RB1", "Roblox Game Design 1.0", "group", 5, 11, 30, 3, 86, 83, 9},
		{"1710208", "ID-GD-SAT-1500 (Unity 3D)", "GD", "Unity 3D Game Development (C#)", "group", 5, 15, 0, 3, 82, 77, 10},
		{"1710209", "ID-PP2-SAT-1700 (Python Pro AI)", "PP2", "Python Pro (Year 2 - AI & Backend)", "group", 5, 17, 0, 3, 90, 89, 14},
		{"1710210", "ID-VK-SUN-0800 (Coding Knight)", "VK", "Coding Knight (Visual Programming)", "group", 6, 8, 0, 3, 88, 86, 7},
		{"1710211", "ID-MC-SUN-1130 (Minecraft Py)", "MC", "Minecraft Coding (MakeCode & Python)", "group", 6, 11, 30, 3, 83, 81, 13},
		{"1710212", "ID-WD-SUN-1400 (Web Design)", "WD", "Building Websites (HTML/CSS/Wix/Figma)", "group", 6, 14, 0, 2, 85, 80, 6},
		{"1710213", "ID-DG-SUN-1600 (Graphic Design)", "DG", "Graphic & Digital Design", "group", 6, 16, 0, 2, 80, 76, 5},
		{"1710214", "ID-AI-SUN-1830 (AI Explorers)", "AI", "Artificial Intelligence & Prompt Engineering", "group", 6, 18, 30, 2, 92, 90, 3},
		{"1710215", "ID-DL-MON-1430 (Digital Lit)", "DL", "Digital Literacy", "group", 0, 14, 30, 2, 81, 78, 9},
		{"1710216", "ID-JS-TUE-1500 (JS Game Dev)", "JS", "JavaScript Game Development", "group", 1, 15, 0, 2, 83, 82, 11},
		{"1710217", "ID-AP-THU-1530 (Mobile App)", "AP", "Mobile App Development (App Inventor)", "group", 3, 15, 30, 2, 79, 75, 10},
		{"1710218", "ID-PS2-FRI-1430 (Python Yr2)", "PS2", "Python Start (Year 2)", "group", 4, 14, 30, 1, 86, 84, 14},
	}

	dayNames := []string{"Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Minggu"}
	studentNames := []string{
		"Kenzie Pratama", "Nadia Putri", "Rafa Alfarizi", "Kirana Maheswari", "Bima Sakti",
		"Evelyn Tan", "Daffa Ramadhan", "Alicia Wijaya", "Reyhan Saputra",
		"Arjuna Baskoro", "Kayla Azzahra", "Darren Kusuma", "Naufal Hidayat", "Chloe Santoso",
		"Farel Nugroho", "Salsabila Rahma", "Joshua Halim", "Bagas Adi", "Aurelia Gunawan",
		"Danish Pratama", "Zahra Kamila", "Kevin Wijaya", "Rizky Fadillah", "Michelle Lim",
		"Gibran Althaf", "Keisya Maharani", "Brandon Sutanto", "Hafiz Maulana", "Tiffany Chen",
		"Aqila Yasmin", "Devon Saputra", "Rangga Wijaya", "Shania Putri", "Matthew Tanoto",
		"Fathir Rahman", "Nayla Syifa", "Jason Kurniawan", "Aldebaran Putra", "Celine Nathania",
		"Zidan Akbar", "aisyah Humaira", "ryan Setiawan", "Clarissa Ong", "Dimas Anggara",
		"Freya Laksmi", "Nathaniel Goh", "Revan Aditya", "Talitha Zahra", "Owen Chandra",
		"Galang Permana", "Vania Prameswari", "Justin Pangestu", "Ilham Ramadhan", "Elena Kusnadi", "Raka Pradipta",
	}

	stuIdx := 0
	lessonSeq := 9500100

	for _, sp := range specs {
		schedLabel := fmt.Sprintf("%s %02d:%02d WIB", dayNames[sp.weekday], sp.startHour, sp.startMin)
		_, err := tx.ExecContext(ctx, `
			INSERT INTO groups (group_id, tutor_id, title, course_code, course_name, format, schedule_summary, start_date, students_count, avg_attendance_pct, avg_progress_pct, lms_url, is_archived)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,0)`,
			sp.id, DefaultTutorID, sp.title, sp.code, sp.name, sp.format, schedLabel, "2026-07-06",
			sp.stuCount, sp.attendance, sp.progress, "https://lms.algonova.id/group/view/"+sp.id)
		if err != nil {
			return err
		}

		// Create students for this group
		groupStudentIDs := []string{}
		for k := 0; k < sp.stuCount && stuIdx < len(studentNames); k++ {
			stName := studentNames[stuIdx]
			if stName[0] >= 'a' && stName[0] <= 'z' {
				stName = string(stName[0]-32) + stName[1:]
			}
			stID := fmt.Sprintf("%d", 880101+stuIdx)
			groupStudentIDs = append(groupStudentIDs, stID)

			// Mix of low (1..3), normal (4..9), and high (10+) balances
			rem := (stuIdx*3)%14 + 1
			totalPurchased := 24
			attended := 12 + (stuIdx % 8)
			totalDone := attended + (stuIdx % 3)
			attPct := (attended * 100) / max(totalDone, 1)
			phone := fmt.Sprintf("6281288%06d", 1040+stuIdx*17)
			lastLesson := timeutil.FormatWIB(weekMon.AddDate(0, 0, -7+sp.weekday).Add(time.Duration(sp.startHour)*time.Hour))

			_, err := tx.ExecContext(ctx, `
				INSERT INTO students (
					student_id, tutor_id, full_name, group_id, group_title, group_format, course_code, course_name,
					attendance_pct, progress_pct, lessons_attended, lessons_total, lessons_remaining, total_lessons_purchased,
					last_lesson_at, parent_name, parent_phone, lms_url, diagnostic_pct, diagnostic_goal
				) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				stID, DefaultTutorID, stName, sp.id, sp.title, sp.format, sp.code, sp.name,
				attPct, sp.progress, attended, totalDone, rem, totalPurchased,
				lastLesson, "Bpk/Ibu "+stName, phone, "https://lms.algonova.id/student/view/"+stID,
				82+(stuIdx%16), "Kemandirian Algoritma & Portofolio Proyek",
			)
			if err != nil {
				return err
			}
			stuIdx++
		}

		// Generate 5 weeks of lessons (-1 week .. +3 weeks) with realistic WIB times (simulating converted LMS +03:00)
		dur := 90
		if sp.format == "private" {
			dur = 60
		}
		for w := -1; w <= 3; w++ {
			dayDate := weekMon.AddDate(0, 0, w*7+sp.weekday)
			startT := time.Date(dayDate.Year(), dayDate.Month(), dayDate.Day(), sp.startHour, sp.startMin, 0, 0, timeutil.JakartaLoc)
			endT := startT.Add(time.Duration(dur) * time.Minute)
			cLessonNo := max(1, sp.milestone+w)
			lID := fmt.Sprintf("%d", lessonSeq)
			lessonSeq++
			isDone := 0
			if endT.Before(now) {
				isDone = 1
			}
			lTitle := fmt.Sprintf("Modul %d — Pertemuan #%d (%s)", (cLessonNo-1)/4+1, cLessonNo, sp.code)
			_, err := tx.ExecContext(ctx, `
				INSERT INTO lessons (
					lesson_id, tutor_id, group_id, group_title, course_code, format, lesson_title,
					lesson_number, course_lesson_number, starts_at, ends_at, duration_minutes,
					meeting_url, prep_url, lms_url, is_completed
				) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				lID, DefaultTutorID, sp.id, sp.title, sp.code, sp.format, lTitle,
				cLessonNo, cLessonNo, timeutil.FormatWIB(startT), timeutil.FormatWIB(endT), dur,
				"https://meet.algonova.id/room-"+sp.id,
				"https://lms.algonova.id/lesson/prep/"+lID,
				"https://lms.algonova.id/group/view/"+sp.id,
				isDone,
			)
			if err != nil {
				return err
			}

			// Seed 1 pre-filled Regular Feedback report on Kenzie Pratama's milestone #8 this week so progress shows 1/N!
			if w == 0 && sp.id == "1710124" && len(groupStudentIDs) > 0 {
				fbText := curriculum.AssembleRegularFeedback(
					"id", "Kenzie Pratama", "PP2", 2,
					"full", "very_active", "all_done",
					"Kenzie berhasil membangun REST API Python pertamanya menggunakan struktur modular yang rapi.",
					"Melanjutkan integrasi database SQL dan visualisasi data menggunakan Pandas pada modul berikutnya.",
					"https://github.com/algonova-student/kenzie-pp2-api", 5,
				)
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO regular_feedback_reports (
						report_id, tutor_id, week_start, lms_lesson_id, lms_student_id, student_name, parent_phone,
						group_title, lesson_number, course_code, month_no, attendance, engagement, tasks,
						custom1, custom2, stars, project_url, assembled_text, updated_at
					) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
					uuid.NewString(), DefaultTutorID, weekMon.Format("2006-01-02"), lID, groupStudentIDs[0],
					"Kenzie Pratama", "6281288001040", sp.title, 8, "PP2", 2,
					"full", "very_active", "all_done",
					"Kenzie berhasil membangun REST API Python pertamanya menggunakan struktur modular yang rapi.",
					"Melanjutkan integrasi database SQL dan visualisasi data menggunakan Pandas pada modul berikutnya.",
					5, "https://github.com/algonova-student/kenzie-pp2-api", fbText, nowStr,
				)
			}
		}
	}

	// 5 Archived empty groups (matching audited "Arsip (5 Tanpa siswa)")
	for i := 1; i <= 5; i++ {
		gid := fmt.Sprintf("169080%d", i)
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO groups (group_id, tutor_id, title, course_code, course_name, format, schedule_summary, start_date, students_count, avg_attendance_pct, avg_progress_pct, lms_url, is_archived)
			VALUES (?,?,?,?,?,?,?,?,0,0,0,?,1)`,
			gid, DefaultTutorID, fmt.Sprintf("ARC-2026-COHORT-%d (Selesai)", i), "PS", "Python Start (Year 1)", "group", "Sabtu 13:00 WIB", "2025-09-01",
			"https://lms.algonova.id/group/view/"+gid)
	}

	// 3. Seed 217 Availability Slots (7 days x 31 half-hour slots: 07:30 to 22:30 WIB)
	bookedLookup := map[string]string{}
	for _, sp := range specs {
		sm := sp.startHour*60 + sp.startMin
		dur := 90
		if sp.format == "private" {
			dur = 60
		}
		stKind := "booked_for_group"
		if sp.format == "private" {
			stKind = "booked"
		}
		for m := sm; m < sm+dur; m += 30 {
			bookedLookup[fmt.Sprintf("%d:%d", sp.weekday, m)] = stKind
		}
	}

	cellRow := 13
	for d := 0; d < 7; d++ {
		for idx := 0; idx < 31; idx++ {
			startMin := 450 + idx*30 // 07:30 .. 22:30
			status := "available"
			if startMin < 540 && d < 5 {
				status = "not_available"
			} else if d >= 5 && startMin >= 1200 {
				status = "avail_for_group"
			}
			if bk, ok := bookedLookup[fmt.Sprintf("%d:%d", d, startMin)]; ok {
				status = bk
			}
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO availability_slots (tutor_id, weekday, start_min, status, sheet_cell, updated_at)
				VALUES (?,?,?,?,?,?)`,
				DefaultTutorID, d, startMin, status, fmt.Sprintf("AS%d", cellRow), nowStr)
			cellRow++
		}
	}

	// 4. Seed Threads & Messages first so Tasks can link to a thread
	thread1ID := uuid.NewString()
	thread2ID := uuid.NewString()
	_, _ = tx.ExecContext(ctx, `
		INSERT INTO threads (thread_id, tutor_id, subject, request_type, target_team, priority, origin, related_group_ref, related_student_ref, unread_tutor, unread_tl, status, updated_at, created_at)
		VALUES
		(?, ?, 'Koordinasi Reschedule Murid Private & Regular Feedback Pekan Ini', 'lesson_issue', 'tl', 'medium', 'direct', '1710125', 'Nadia Putri', 1, 0, 'open', ?, ?),
		(?, ?, '[SIARAN AKADEMIK] Panduan Milestone Pertemuan ke-4 & Assessment Oktober', 'broadcast', 'tl', 'low', 'broadcast', '', '', 0, 0, 'open', ?, ?)`,
		thread1ID, DefaultTutorID, nowStr, timeutil.FormatWIB(now.Add(-6*time.Hour)),
		thread2ID, DefaultTutorID, timeutil.FormatWIB(now.Add(-24*time.Hour)), timeutil.FormatWIB(now.Add(-24*time.Hour)),
	)
	_, _ = tx.ExecContext(ctx, `
		INSERT INTO messages (message_id, thread_id, sender_role, sender_name, kind, body, attachment_url, created_at)
		VALUES
		(?, ?, 'tutor', 'Azhar Faturohman Ahidin', 'text', 'Halo Kak Devika, orang tua Nadia Putri (IND 1-1 RB2.0) mengabarkan minggu depan ada kegiatan sekolah dan menanyakan opsi pindah jam sore.', '', ?),
		(?, ?, 'team_lead', 'Devika Wulan Rahmadani', 'text', 'Halo Kak Azhar! Boleh banget, silakan ajukan lewat menu Minta Penjadwalan Ulang (mode Sekali/OTR) beserta screenshot chat orang tuanya ya agar langsung aku approve.', '', ?),
		(?, ?, 'team_lead', 'Devika Wulan Rahmadani', 'broadcast', 'Semangat pagi rekan-rekan Tutor Algonova! Pastikan seluruh Regular Feedback pertemuan kelipatan ke-4 diselesaikan sebelum Minggu pukul 23:59 WIB.', '', ?)`,
		uuid.NewString(), thread1ID, timeutil.FormatWIB(now.Add(-5*time.Hour)),
		uuid.NewString(), thread1ID, timeutil.FormatWIB(now.Add(-2*time.Hour)),
		uuid.NewString(), thread2ID, timeutil.FormatWIB(now.Add(-24*time.Hour)),
	)

	// 5. Seed Kanban Tasks (assigned, overdue, completed, escalated)
	wStartStr := weekMon.Format("2006-01-02")
	tasksSeed := []store.Task{
		{
			TaskID: uuid.NewString(), TutorID: DefaultTutorID,
			Title: "Lengkapi Regular Feedback Pertemuan #4 — Nadia Putri (IND 1-1 RB2.0)",
			Description: "Susun laporan perkembangan belajar bulan ke-1 dan kirimkan ringkasan PDF/WhatsApp ke orang tua sebelum Minggu 23:59 WIB.",
			TaskType: "regular_feedback", Status: "assigned", Priority: "high",
			DueAt: timeutil.FormatWIB(now.Add(18 * time.Hour)), AssignedAt: timeutil.FormatWIB(now.Add(-12 * time.Hour)),
			MetricCode: "feedback_sla", WeekStart: wStartStr,
			RelatedGroupID: "1710125", RelatedGroupTitle: "IND 1-1 Nadia Putri (RB2.0)",
			RelatedStudentID: "880102", RelatedStudentName: "Nadia Putri", ThreadID: thread1ID,
		},
		{
			TaskID: uuid.NewString(), TutorID: DefaultTutorID,
			Title: "Follow-up Absensi Reyhan Saputra (2x izin berturut-turut)",
			Description: "Cek kendala kehadiran siswa pada kelas Minecraft Coding dan laporkan ke Team Lead jika butuh penyesuaian jadwal.",
			TaskType: "attendance_followup", Status: "overdue", Priority: "high",
			DueAt: timeutil.FormatWIB(now.Add(-3 * time.Hour)), AssignedAt: timeutil.FormatWIB(now.Add(-30 * time.Hour)),
			MetricCode: "attendance_kpi", WeekStart: wStartStr,
			RelatedGroupID: "1710132", RelatedGroupTitle: "IND 1-1 Reyhan Saputra (MC)",
			RelatedStudentID: "880109", RelatedStudentName: "Reyhan Saputra",
		},
		{
			TaskID: uuid.NewString(), TutorID: DefaultTutorID,
			Title: "Pengingat Kuota Kelas Menipis (<= 2 sesi) — Kenzie Pratama",
			Description: "Sampaikan pengingat ramah mengenai perpanjangan paket belajar kepada orang tua atau koordinasi CS.",
			TaskType: "low_balance", Status: "assigned", Priority: "medium",
			DueAt: timeutil.FormatWIB(now.Add(6 * time.Hour)), AssignedAt: timeutil.FormatWIB(now.Add(-8 * time.Hour)),
			WeekStart: wStartStr, RelatedGroupID: "1710124", RelatedGroupTitle: "IND 1-1 Kenzie Pratama (PP2)",
			RelatedStudentID: "880101", RelatedStudentName: "Kenzie Pratama",
		},
		{
			TaskID: uuid.NewString(), TutorID: DefaultTutorID,
			Title: "Selesaikan Tutor Assessment Siklus Oktober (OCT-2026-C2)",
			Description: "Kerjakan kuesioner kompetensi pedagogi & studi kasus kelas coding sebelum tenggat siklus berakhir.",
			TaskType: "assessment_due", Status: "assigned", Priority: "medium",
			DueAt: timeutil.FormatWIB(now.Add(48 * time.Hour)), AssignedAt: timeutil.FormatWIB(now.Add(-20 * time.Hour)),
			MetricCode: "assessment_sla", WeekStart: wStartStr,
		},
		{
			TaskID: uuid.NewString(), TutorID: DefaultTutorID,
			Title: "Verifikasi Rekaman & Materi Prep Grup ID-PS-MON-1830",
			Description: "Pastikan tautan proyek siswa pertemuan #4 sudah tersimpan dengan benar.",
			TaskType: "personal", Status: "completed", Priority: "low",
			DueAt: timeutil.FormatWIB(now.Add(-10 * time.Hour)), AssignedAt: timeutil.FormatWIB(now.Add(-36 * time.Hour)),
			CompletedAt: timeutil.FormatWIB(now.Add(-14 * time.Hour)), WeekStart: wStartStr,
			RelatedGroupID: "1710201", RelatedGroupTitle: "ID-PS-MON-1830 (Python Start)",
		},
		{
			TaskID: uuid.NewString(), TutorID: DefaultTutorID,
			Title: "Permintaan Pindah Grup Siswa — Daffa Ramadhan",
			Description: "Orang tua menanyakan ketersediaan slot hari Minggu pagi karena bentrok les musik sekolah.",
			TaskType: "student_transfer", Status: "escalated", Priority: "medium",
			DueAt: timeutil.FormatWIB(now.Add(24 * time.Hour)), AssignedAt: timeutil.FormatWIB(now.Add(-18 * time.Hour)),
			ReasonNote: "Membutuhkan konfirmasi CS & Team Lead untuk pengecekan kuota grup Minggu.",
			RelatedGroupID: "1710130", RelatedGroupTitle: "IND 1-1 Daffa Ramadhan (PP1)",
			RelatedStudentID: "880107", RelatedStudentName: "Daffa Ramadhan",
		},
	}
	for _, tk := range tasksSeed {
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO tasks (task_id, tutor_id, title, description, task_type, status, priority, due_at, assigned_at, completed_at, reason_note, metric_code, week_start, related_group_id, related_group_title, related_student_id, related_student_name, thread_id)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			tk.TaskID, tk.TutorID, tk.Title, tk.Description, tk.TaskType, tk.Status, tk.Priority, tk.DueAt, tk.AssignedAt, tk.CompletedAt, tk.ReasonNote, tk.MetricCode, tk.WeekStart, tk.RelatedGroupID, tk.RelatedGroupTitle, tk.RelatedStudentID, tk.RelatedStudentName, tk.ThreadID)
	}

	// 6. Seed Active Assessment Cycle + 37 Historical Cycles
	questions := []store.AssessmentQuestion{
		{
			ID: "q1_pedagogy", Program: "all", Section: "1. Manajemen Kelas & Retensi", Kind: "single", Required: true,
			Prompt: "Apa langkah pertama paling tepat ketika murid kelas Private (1-on-1) tidak hadir 10 menit setelah sesi dimulai?",
			Options: []string{
				"Tetap standby di ruang kelas, hubungi orang tua/CS sesuai SOP 15 menit awal, dan catat status kehadiran di LMS",
				"Langsung menutup ruang kelas tanpa konfirmasi",
				"Mengubah jadwal permanen tanpa persetujuan orang tua",
			},
		},
		{
			ID: "q2_feedback", Program: "coding", Section: "2. Kualitas Regular Feedback Coding", Kind: "single", Required: true,
			Prompt: "Komponen apa yang wajib tercantum pada Regular Feedback setiap pertemuan kelipatan ke-4?",
			Options: []string{
				"Capaian materi modul spesifik, evaluasi keaktifan & tugas, rekomendasi konkret berikutnya, dan tautan karya/proyek jika ada",
				"Hanya nilai angka tanpa deskripsi",
				"Salinan silabus umum tanpa menyebutkan perkembangan siswa",
			},
		},
		{
			ID: "q3_confidence", Program: "coding", Section: "2. Kualitas Regular Feedback Coding", Kind: "scale", Required: true,
			Prompt: "Seberapa siap Anda memfasilitasi debugging mandiri siswa menggunakan metode Socratic Questioning? (Skala 1-5)",
		},
		{
			ID: "q4_case", Program: "coding", Section: "3. Studi Kasus Praktis", Kind: "paragraph", Required: true,
			Prompt: "Tuliskan 2 kalimat contoh umpan balik konstruktif untuk orang tua siswa Python Start yang antusias di kelas namun belum menyelesaikan latihan rumah.",
		},
	}
	qBytes, _ := json.Marshal(questions)
	_, _ = tx.ExecContext(ctx, `
		INSERT INTO assessment_cycles (cycle_code, title, channel, deadline_at, is_open, questions_json)
		VALUES (?,?,?,?,1,?)`,
		"OCT-2026-C2", "Tutor Quality & Coding Pedagogy Check — Oktober Siklus 2", "nk",
		timeutil.FormatWIB(now.AddDate(0, 0, 5)), string(qBytes))

	scoresPattern := []int{100, 98, 100, 100, 96, 100, 98, 100, 95, 100, 98, 100}
	baseMonth := time.Date(2025, 4, 15, 12, 0, 0, 0, timeutil.JakartaLoc)
	for i := 0; i < 37; i++ {
		dt := baseMonth.AddDate(0, i/2, (i%2)*14)
		sc := scoresPattern[i%len(scoresPattern)]
		code := fmt.Sprintf("CYC-%s-%d", dt.Format("200601"), (i%2)+1)
		label := fmt.Sprintf("Assessment %d %s", (i%2)+1, dt.Format("Jan 2006"))
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO assessment_history (tutor_id, cycle_code, cycle_label, month_key, program, score_pct, max_pct, status, graded_at)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			DefaultTutorID, code, label, dt.Format("2006-01"), "Coding & Design", sc, 100, "passed", timeutil.FormatWIB(dt))
	}

	// 7. Seed Clean Deduplicated Knowledge Base Articles (Fixing Bug #5!)
	kbSeed := []store.KBArticle{
		{
			ArticleID: "kb-reschedule-private", Category: "Jadwal & Reschedule", Locale: "id",
			Title: "SOP Perubahan Jadwal (Reschedule) Kelas Private 1-on-1 & Grup",
			Summary: "Aturan batas pengajuan Senin 23:59 WIB, perbedaan Sekali (OTR) vs Permanen, dan syarat bukti persetujuan orang tua.",
			ContentMD: "### Ketentuan Reschedule Kelas\n1. **Sekali (One-Time Reschedule / OTR)**: Digunakan apabila murid *private/intensive* berhalangan hadir pada 1 pertemuan tertentu dan sepakat mengganti di hari/jam lain.\n2. **Permanen (`permanent`)**: Mengubah jadwal rutin mingguan untuk seterusnya (hanya berlaku untuk pertemuan mendatang dan minimal tersisa 2 sesi).\n3. **Batas Waktu Grace Period**: Sesi pekan lalu masih dapat diajukan maksimal hingga **hari Senin pukul 23:59 WIB**.\n4. **Bukti Konfirmasi**: Lampirkan 1–3 tangkapan layar persetujuan orang tua/CS.",
			HelpfulCount: 42,
		},
		{
			ArticleID: "kb-regular-feedback", Category: "Regular Feedback", Locale: "id",
			Title: "Panduan Penulisan Regular Feedback Pertemuan #4, #8, #12, dst.",
			Summary: "Standar kualitas umpan balik orang tua setiap 4 pertemuan beserta ekspor kartu PDF A4 & pengiriman WhatsApp.",
			ContentMD: "### Standar Regular Feedback Coding & Design\n- Wajib diselesaikan setiap kelipatan pertemuan ke-4 paling lambat **Minggu pukul 23:59 WIB**.\n- Pilih kode kursus dan bulan berjalan agar topik modul sesuai kurikulum.\n- Sertakan minimal satu capaian spesifik siswa dan tautan proyek (Scratch/Roblox/GitHub/Figma) bila tersedia.",
			HelpfulCount: 38,
		},
		{
			ArticleID: "kb-urgent-sub", Category: "Pengganti Mendesak", Locale: "id",
			Title: "Prosedur Pengajuan Guru Pengganti Mendesak (Urgent Substitute)",
			Summary: "Ketentuan jendela waktu 50 menit hingga 72 jam sebelum kelas dimulai saat kondisi darurat.",
			ContentMD: "### Kapan Menggunakan Urgent Substitute?\n- Digunakan untuk kelas yang akan dimulai dalam **50 menit s/d 72 jam ke depan** saat tutor mengalami kendala darurat (kesehatan/listrik/bencana).\n- Untuk kelas di bawah 50 menit, segera hubungi Team Lead & grup darurat Telegram secara langsung.",
			HelpfulCount: 29,
		},
		{
			ArticleID: "kb-lms-timezone", Category: "Teknis & LMS", Locale: "id",
			Title: "Sinkronisasi Zona Waktu LMS Algonova ke WIB (UTC+7)",
			Summary: "Penjelasan konversi otomatis timestamp server LMS (+03:00) menjadi Waktu Indonesia Barat (+07:00).",
			ContentMD: "### Konversi Waktu Otomatis di AlgoKanbana\nServer backoffice `lms.algonova.id` mencatat jadwal mentah dalam zona waktu `UTC+03:00`. AlgoKanbana secara otomatis mengonversi seluruh jadwal `+4 jam` ke **WIB (`Asia/Jakarta`, `UTC+07:00`)** sehingga tutor tidak perlu menghitung manual.",
			HelpfulCount: 51,
		},
		{
			ArticleID: "kb-kpi-scoring", Category: "KPI & Kualitas", Locale: "id",
			Title: "Cara Perhitungan Skor KPI Kehadiran, Discipline, dan Quality",
			Summary: "Ambang batas warna hijau/kuning/merah untuk metrik bulanan tutor Algonova.",
			ContentMD: "### Ambang Batas KPI Tutor\n- **Attendance Siswa**: Target hijau `>= 88%` (batas perhatian `80%`).\n- **Discipline**: Gabungan ketepatan waktu *Regular Feedback* (`50 poin`) dan penyelesaian *Assessment* (`50 poin`).\n- **Quality**: Hasil observasi kelas & evaluasi akademik (`>= 75%` hijau).",
			HelpfulCount: 34,
		},
	}
	for _, a := range kbSeed {
		_, _ = tx.ExecContext(ctx, `INSERT INTO kb_articles (article_id, category, locale, title, summary, content_md, helpful_count) VALUES (?,?,?,?,?,?,?)`,
			a.ArticleID, a.Category, a.Locale, a.Title, a.Summary, a.ContentMD, a.HelpfulCount)
	}

	// 8. Seed Upcoming Portal Events & Sample Saved Views
	_, _ = tx.ExecContext(ctx, `
		INSERT INTO portal_events (event_id, title, description, starts_at, ends_at, meeting_url, created_by)
		VALUES
		(?, 'Townhall & Sharing Best Practice Tutor Coding Q4', 'Sesi berbagi tips meningkatkan retensi kelas Private & demo proyek Roblox/Python terbaik.', ?, ?, 'https://meet.algonova.id/townhall-id', 'Academic Lead Indonesia'),
		(?, 'Clinic Kurikulum Python Pro & AI Track', 'Tanya jawab modul baru Machine Learning & integrasi proyek akhir siswa.', ?, ?, 'https://meet.algonova.id/python-clinic', 'Curriculum Team')`,
		uuid.NewString(), timeutil.FormatWIB(now.AddDate(0, 0, 2)), timeutil.FormatWIB(now.AddDate(0, 0, 2).Add(90*time.Minute)),
		uuid.NewString(), timeutil.FormatWIB(now.AddDate(0, 0, 5)), timeutil.FormatWIB(now.AddDate(0, 0, 5).Add(60*time.Minute)),
	)

	_, _ = tx.ExecContext(ctx, `
		INSERT INTO saved_views (view_id, tutor_id, page_key, name, query_string, created_at)
		VALUES
		(?, ?, 'students', 'Kuota Kritis (<= 3 Sesi)', 'balance=low', ?),
		(?, ?, 'students', 'Murid Private 1-on-1', 'format=private', ?)`,
		uuid.NewString(), DefaultTutorID, nowStr,
		uuid.NewString(), DefaultTutorID, nowStr,
	)

	// 9. Seed Student Survey Targets (Renewal evaluations for students near package completion)
	surveySeed := []struct {
		sid      string
		stuId    string
		name     string
		group    string
		course   string
		rem      int
		daysAway int
	}{
		{uuid.NewString(), "880101", "Kenzie Pratama", "IND 1-1 Kenzie Pratama (PP2)", "Python Pro (Year 2 - AI & Backend)", 1, 3},
		{uuid.NewString(), "880102", "Nadia Putri", "IND 1-1 Nadia Putri (RB2.0)", "Roblox Studio 2.0 (Lua Scripting)", 2, 5},
		{uuid.NewString(), "880103", "Rafa Alfarizi", "IND 1-1 Rafa Alfarizi (PS)", "Python Start (Year 1)", 3, 7},
		{uuid.NewString(), "880104", "Kirana Maheswari", "IND 1-1 Kirana Maheswari (AI)", "Artificial Intelligence & Prompt Engineering", 4, 10},
	}
	for _, sv := range surveySeed {
		deadlineStr := timeutil.FormatWIB(now.AddDate(0, 0, sv.daysAway))
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO survey_targets (survey_id, tutor_id, student_id, student_name, group_title, course_name, lessons_remaining, deadline_at, status, created_at)
			VALUES (?,?,?,?,?,?,?,?,'pending',?)`,
			sv.sid, DefaultTutorID, sv.stuId, sv.name, sv.group, sv.course, sv.rem, deadlineStr, nowStr)
	}

	return tx.Commit()
}

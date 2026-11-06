package curriculum

import (
	"fmt"
	"strings"
)

type Course struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Months      int      `json:"months"`
	Level       string   `json:"level"`
	Category    string   `json:"category"`
	ModuleFocus []string `json:"moduleFocus"`
}

var Courses = []Course{
	{Code: "DL", Name: "Digital Literacy", Months: 9, Level: "7-9 yo", Category: "coding", ModuleFocus: []string{"Pengenalan Komputer & Keamanan Internet", "Dokumen & Presentasi Kreatif", "Logika Algoritma Dasar", "Desain Grafis Dasar"}},
	{Code: "VK", Name: "Coding Knight (Visual Programming)", Months: 9, Level: "6-7 yo", Category: "coding", ModuleFocus: []string{"Urutan Perintah & Loop", "Kondisi & Event Dasar", "Animasi Interaktif", "Mini Game Cerita"}},
	{Code: "SC", Name: "Scratch Visual Programming", Months: 9, Level: "8-10 yo", Category: "coding", ModuleFocus: []string{"Sprite, Gerak & Koordinat", "Variabel, Clone & Sensor", "Game Arcade & Platformer", "Proyek Game Mandiri"}},
	{Code: "SC2", Name: "Scratch Advanced Lab", Months: 9, Level: "9-11 yo", Category: "coding", ModuleFocus: []string{"Custom Blocks & List", "Fisika Game 2D", "Sistem Skor & Level Multi-Stage", "Capstone Game Jam"}},
	{Code: "MC", Name: "Minecraft Coding (MakeCode & Python)", Months: 9, Level: "8-11 yo", Category: "coding", ModuleFocus: []string{"Agent Automation & Koordinat 3D", "Loop & Kondisional Bangunan Otomatis", "Redstone Logic & Mini-Games", "Python Scripting di Minecraft"}},
	{Code: "RB1", Name: "Roblox Game Design 1.0", Months: 9, Level: "9-12 yo", Category: "coding", ModuleFocus: []string{"3D Terrain & Obby Studio", "Dasar Bahasa Lua & Properties", "Events, Touched & Leaderstats", "Publikasi Game Roblox Pertama"}},
	{Code: "RB2.0", Name: "Roblox Studio 2.0 (Lua Scripting)", Months: 9, Level: "10-14 yo", Category: "coding", ModuleFocus: []string{"Fundamental Lua, Variabel & Fungsi", "Client-Server RemoteEvents & DataStore", "GUI Interaktif, Shop & Inventory", "Multiplayer Simulator & Optimasi"}},
	{Code: "AP", Name: "Mobile App Development (App Inventor)", Months: 9, Level: "9-12 yo", Category: "coding", ModuleFocus: []string{"UI/UX Layar Mobile & Event Button", "Sensor Perangkat, Kamera & Audio", "Penyimpanan Lokal TinyDB & API", "Aplikasi Utilitas & Edu-Game"}},
	{Code: "PS", Name: "Python Start (Year 1)", Months: 9, Level: "11-13 yo", Category: "coding", ModuleFocus: []string{"Variabel, Tipe Data & Percabangan", "Perulangan While/For & Struktur List", "Fungsi, Modul & Turtle Graphics", "Game 2D dengan Pygame Zero / PyQt Dasar"}},
	{Code: "PS2", Name: "Python Start (Year 2)", Months: 9, Level: "12-14 yo", Category: "coding", ModuleFocus: []string{"Pemrograman Berorientasi Objek (OOP)", "Aplikasi Desktop PyQt5", "Pengembangan Game Pygame Lengkap", "Pengolahan File & Proyek Tim"}},
	{Code: "PP1", Name: "Python Pro (Year 1)", Months: 9, Level: "14-17 yo", Category: "coding", ModuleFocus: []string{"Algoritma, Struktur Data & OOP Lanjut", "Pengembangan Aplikasi Desktop & Git", "Web Backend Flask & SQLite", "Analisis Data Dasar dengan Pandas"}},
	{Code: "PP2", Name: "Python Pro (Year 2 - AI & Backend)", Months: 9, Level: "15-17 yo", Category: "coding", ModuleFocus: []string{"REST API, Database SQL & Autentikasi", "Data Science, NumPy, Pandas & Matplotlib", "Machine Learning Klasifikasi & Regresi", "Computer Vision & Capstone AI"}},
	{Code: "WD", Name: "Building Websites (HTML/CSS/Wix/Figma)", Months: 9, Level: "10-13 yo", Category: "coding", ModuleFocus: []string{"Wireframing & Desain UI di Figma", "Struktur HTML5 Semantik & CSS Flexbox", "Desain Responsif & Animasi CSS", "Landing Page & Portofolio Online"}},
	{Code: "FE", Name: "Frontend Development (HTML/CSS/JS/React)", Months: 9, Level: "13-17 yo", Category: "coding", ModuleFocus: []string{"Layout Modern Flexbox, Grid & Responsif", "JavaScript ES6+, DOM & Fetch REST API", "Modularisasi, State & Komponen", "Single Page Application Portofolio"}},
	{Code: "JS", Name: "JavaScript Game Development", Months: 9, Level: "12-16 yo", Category: "coding", ModuleFocus: []string{"HTML5 Canvas & Game Loop", "Sprite Collision, Input & Audio", "OOP JavaScript & Tilemap", "Phaser 3 Web Game Publishing"}},
	{Code: "GD", Name: "Unity 3D Game Development (C#)", Months: 9, Level: "12-17 yo", Category: "coding", ModuleFocus: []string{"Unity Editor, Physics & Prefabs", "C# Scripting, Movement & Camera", "UI Canvas, Enemy AI NavMesh & Raycast", "3D Action/Adventure Game Build"}},
	{Code: "GD2", Name: "Unity Advanced & Multiplayer", Months: 9, Level: "13-17 yo", Category: "coding", ModuleFocus: []string{"ScriptableObjects & Clean Architecture", "Shader Graph, Lighting & Cinematics", "Save System & Inventory Kompleks", "Finalisasi & Optimasi Performa Game"}},
	{Code: "AI", Name: "Artificial Intelligence & Prompt Engineering", Months: 6, Level: "12-17 yo", Category: "coding", ModuleFocus: []string{"Konsep Dasar AI, LLM & Etika Digital", "Otomasi Alur Kerja, Gambar & Kode dengan AI", "Pembuatan Chatbot & Integrasi API AI"}},
	{Code: "DA", Name: "Data Analytics with Python", Months: 6, Level: "13-17 yo", Category: "coding", ModuleFocus: []string{"Data Cleaning & Eksplorasi CSV/SQL", "Visualisasi Statistik & Storytelling", "Dashboard Analitik Proyek Nyata"}},
	{Code: "CS", Name: "Cybersecurity & Networking Basics", Months: 6, Level: "12-17 yo", Category: "coding", ModuleFocus: []string{"Jaringan TCP/IP, DNS & Kriptografi", "Keamanan Web, OSINT & Linux Dasar", "Simulasi Capture The Flag (CTF)"}},
	{Code: "DG", Name: "Graphic & Digital Design", Months: 9, Level: "9-14 yo", Category: "design", ModuleFocus: []string{"Komposisi Warna, Tipografi & Ilustrasi Vektor", "Manipulasi Foto & Branding Poster", "UI/UX Prototype di Figma", "Portofolio Desain Kreatif"}},
	{Code: "UIX", Name: "UI/UX Product Design", Months: 6, Level: "12-17 yo", Category: "design", ModuleFocus: []string{"User Research, User Flow & Wireframe", "Design System & Auto-Layout Figma", "Interactive Prototyping & Usability Test"}},
	{Code: "3D", Name: "3D Modeling with Blender", Months: 9, Level: "11-16 yo", Category: "design", ModuleFocus: []string{"Mesh Modeling, Extrude & Modifiers", "UV Mapping, Materials & Lighting", "Rigging Dasar & Render Animasi Pendek"}},
	{Code: "AN", Name: "2D Digital Animation & Video", Months: 9, Level: "9-14 yo", Category: "design", ModuleFocus: []string{"Storyboard & 12 Prinsip Animasi", "Character Rigging & Keyframe", "Editing Audio-Visual & Motion Graphics"}},
	{Code: "VP", Name: "Video Production & Content Creation", Months: 6, Level: "10-15 yo", Category: "design", ModuleFocus: []string{"Scripting, Framing & Storytelling", "Video Editing, Transisi & Color Grading", "Publikasi Konten Edukatif"}},
	{Code: "CC", Name: "Competitive Coding (C++ / Python)", Months: 9, Level: "12-17 yo", Category: "coding", ModuleFocus: []string{"Kompleksitas Waktu, Sorting & Binary Search", "Rekursi, Greedy & Dynamic Programming", "Graf BFS/DFS & Simulasi Olimpiade"}},
	{Code: "IOT", Name: "Virtual Robotics & IoT", Months: 6, Level: "10-15 yo", Category: "coding", ModuleFocus: []string{"Rangkaian Sensor & Mikrokontroler Simulasi", "Pemrograman C++/MicroPython", "Otomasi Smart Home Mini"}},
	{Code: "MT1", Name: "Math Logic Grade 1-2", Months: 9, Level: "6-8 yo", Category: "math", ModuleFocus: []string{"Berhitung Cepat & Pola Bilangan", "Logika Ruang & Geometri Visual", "Pemecahan Soal Cerita Kontekstual"}},
	{Code: "MT2", Name: "Math Logic Grade 3-4", Months: 9, Level: "8-10 yo", Category: "math", ModuleFocus: []string{"Perkalian, Pembagian & Pecahan", "Kombinatorika Dasar & Teka-Teki Logika", "Strategi Pemecahan Masalah Bertahap"}},
	{Code: "MT3", Name: "Math Olympiad & Logic Grade 5-6", Months: 9, Level: "10-12 yo", Category: "math", ModuleFocus: []string{"Pecahan, Rasio, Persentase & Teori Bilangan", "Geometri Luas/Volume & Penalaran Deduktif", "Soal Tantangan Non-Standar"}},
	{Code: "MT4", Name: "Algebra & Analytical Math Grade 7-8", Months: 9, Level: "12-14 yo", Category: "math", ModuleFocus: []string{"Persamaan Linear, Sistem & Fungsi", "Teorema Geometri & Peluang", "Pemodelan Matematika Terapan"}},
	{Code: "INT-PRV", Name: "Intensive Private Coding Track (1-on-1)", Months: 12, Level: "All Ages", Category: "coding", ModuleFocus: []string{"Kurikulum Personalisasi Sesuai Kecepatan Siswa", "Pendalaman Logika & Debugging Mandiri", "Pembangunan Proyek Portofolio Khusus"}},
}

func FindCourse(code string) (Course, bool) {
	for _, c := range Courses {
		if strings.EqualFold(c.Code, code) {
			return c, true
		}
	}
	return Course{}, false
}

func topicForMonth(c Course, monthNo int) string {
	if len(c.ModuleFocus) == 0 {
		return c.Name
	}
	if monthNo < 1 {
		monthNo = 1
	}
	idx := (monthNo - 1) * len(c.ModuleFocus) / max(c.Months, 1)
	if idx >= len(c.ModuleFocus) {
		idx = len(c.ModuleFocus) - 1
	}
	return c.ModuleFocus[idx]
}

// AssembleRegularFeedback builds localized Indonesian/English parent report paragraphs
// matching Algonova's Regular Feedback milestone workflow (#4, #8, #12, ...).
func AssembleRegularFeedback(
	locale, studentName, courseCode string,
	monthNo int,
	attendance, engagement, tasks, custom1, custom2, projectURL string,
	stars int,
) string {
	course, ok := FindCourse(courseCode)
	if !ok {
		course = Course{Code: courseCode, Name: courseCode, Months: 9, ModuleFocus: []string{"Materi Pemrograman Inti"}}
	}
	topic := topicForMonth(course, monthNo)

	if strings.EqualFold(locale, "en") {
		return assembleEN(studentName, course, monthNo, topic, attendance, engagement, tasks, custom1, custom2, projectURL, stars)
	}
	return assembleID(studentName, course, monthNo, topic, attendance, engagement, tasks, custom1, custom2, projectURL, stars)
}

func assembleID(
	studentName string,
	course Course,
	monthNo int,
	topic, attendance, engagement, tasks, custom1, custom2, projectURL string,
	stars int,
) string {
	attMap := map[string]string{
		"full": "hadir penuh secara konsisten di seluruh sesi bulan ini",
		"most": "hadir di sebagian besar pertemuan dengan disiplin waktu yang baik",
		"half": "hadir di sekitar separuh sesi bulan ini sehingga beberapa materi perlu diulas kembali",
		"low":  "memiliki tingkat kehadiran yang masih rendah bulan ini sehingga pendampingan jadwal sangat disarankan",
		"none": "belum dapat hadir pada rangkaian sesi reguler periode ini",
	}
	engMap := map[string]string{
		"very_active":   "Selama kelas berlangsung, ananda sangat aktif bertanya, berdiskusi, dan antusias mencoba tantangan kode baru.",
		"focused_quiet": "Selama kelas berlangsung, ananda menyimak penjelasan dengan fokus, tenang, dan teliti saat mempraktikkan materi.",
		"passive":       "Di dalam kelas, ananda cukup mengikuti instruksi namun masih perlu lebih sering didorong untuk berani menyampaikan ide atau bertanya.",
		"struggling":    "Di beberapa bagian materi baru, ananda masih membutuhkan bimbingan bertahap agar lebih percaya diri saat memecahkan masalah.",
	}
	taskMap := map[string]string{
		"all_done":    "Seluruh latihan dan tugas praktik pada modul ini telah diselesaikan dengan tuntas dan rapi.",
		"mostly_done": "Sebagian besar latihan praktik telah dikerjakan dengan baik, dan hanya menyisakan sedikit tantangan tambahan untuk disempurnakan.",
		"struggling":  "Beberapa latihan mandiri masih perlu diselesaikan secara bertahap agar pemahaman konsep makin kuat.",
	}

	attText := attMap[attendance]
	if attText == "" {
		attText = attMap["full"]
	}
	engText := engMap[engagement]
	if engText == "" {
		engText = engMap["very_active"]
	}
	taskText := taskMap[tasks]
	if taskText == "" {
		taskText = taskMap["all_done"]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Halo Ayah/Bunda dari %s! 👋\n\n", studentName)
	fmt.Fprintf(&b, "Berikut laporan perkembangan belajar reguler pada kursus *%s* (Bulan ke-%d — Fokus: *%s*).\n\n", course.Name, monthNo, topic)
	fmt.Fprintf(&b, "• *Kehadiran*: %s %s.\n", studentName, attText)
	fmt.Fprintf(&b, "• *Keaktifan di Kelas*: %s\n", engText)
	fmt.Fprintf(&b, "• *Penyelesaian Tugas*: %s\n", taskText)

	if strings.TrimSpace(custom1) != "" {
		fmt.Fprintf(&b, "\n*Pencapaian Menonjol*:\n%s\n", strings.TrimSpace(custom1))
	}
	if strings.TrimSpace(custom2) != "" {
		fmt.Fprintf(&b, "\n*Rekomendasi Pengembangan Berikutnya*:\n%s\n", strings.TrimSpace(custom2))
	}
	if strings.TrimSpace(projectURL) != "" {
		fmt.Fprintf(&b, "\n🔗 *Karya / Proyek Siswa*: %s\n", strings.TrimSpace(projectURL))
	}
	if stars < 1 || stars > 5 {
		stars = 5
	}
	fmt.Fprintf(&b, "\n⭐ *Apresiasi Tutor*: %s (%d/5)\nTerima kasih atas dukungan Ayah/Bunda dalam mendampingi semangat belajar %s bersama Algonova Indonesia!", strings.Repeat("★", stars)+strings.Repeat("☆", 5-stars), stars, studentName)
	return b.String()
}

func assembleEN(
	studentName string,
	course Course,
	monthNo int,
	topic, attendance, engagement, tasks, custom1, custom2, projectURL string,
	stars int,
) string {
	attMap := map[string]string{
		"full": "attended all scheduled sessions consistently this month",
		"most": "attended most sessions with good punctuality",
		"half": "attended roughly half of the sessions, so reviewing missed concepts is recommended",
		"low":  "had limited attendance during this milestone window",
		"none": "was unable to attend sessions during this period",
	}
	engMap := map[string]string{
		"very_active":   "In class, the student participates actively, asks thoughtful questions, and eagerly explores coding challenges.",
		"focused_quiet": "In class, the student stays calm, attentive, and methodical when implementing each step.",
		"passive":       "The student follows guided steps well and is encouraged to share ideas more actively during discussions.",
		"struggling":    "With newer concepts, step-by-step guidance helps build stronger problem-solving confidence.",
	}
	taskMap := map[string]string{
		"all_done":    "All core exercises and practical tasks in this module were completed thoroughly.",
		"mostly_done": "Most practical exercises were completed well, with only bonus tasks remaining.",
		"struggling":  "Additional practice on independent tasks will help solidify core concepts.",
	}
	if stars < 1 || stars > 5 {
		stars = 5
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Hello Parents of %s! 👋\n\n", studentName)
	fmt.Fprintf(&b, "Here is the regular progress report for *%s* (Month %d — Focus: *%s*).\n\n", course.Name, monthNo, topic)
	fmt.Fprintf(&b, "• *Attendance*: %s %s.\n", studentName, attMap[attendance])
	fmt.Fprintf(&b, "• *Engagement*: %s\n", engMap[engagement])
	fmt.Fprintf(&b, "• *Practical Tasks*: %s\n", taskMap[tasks])
	if strings.TrimSpace(custom1) != "" {
		fmt.Fprintf(&b, "\n*Highlights*:\n%s\n", strings.TrimSpace(custom1))
	}
	if strings.TrimSpace(custom2) != "" {
		fmt.Fprintf(&b, "\n*Next Steps*:\n%s\n", strings.TrimSpace(custom2))
	}
	if strings.TrimSpace(projectURL) != "" {
		fmt.Fprintf(&b, "\n🔗 *Student Project*: %s\n", strings.TrimSpace(projectURL))
	}
	fmt.Fprintf(&b, "\n⭐ *Tutor Rating*: %s (%d/5)\nThank you for supporting %s's learning journey at Algonova Indonesia!", strings.Repeat("★", stars)+strings.Repeat("☆", 5-stars), stars, studentName)
	return b.String()
}

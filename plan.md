Evet kanka, bu işin omurgasını düzgün kurarsan baya sağlam bir ürün çıkar. Ben bunu **“recon orkestra şefi”** gibi değil de direkt **iş akışı yöneten bir daemon + CLI** olarak tasarlardım.

## 1) Ürünün tek cümlelik tanımı

**Kullanıcının recon işlerini tek komutla başlatan, duraklatan, kaldığı yerden devam ettiren, kaynakları sınırlayan ve tüm çıktıları düzenli kaydeden yerel bir iş yöneticisi.**

Yani araç şunu yapacak:

* `subfinder`, `httpx`, `katana`, `gau`, `ffuf` gibi araçları yeniden yazmayacak.
* Onları sıraya koyacak.
* Kaynaklarını yönetecek.
* Durumunu saklayacak.
* Yarıda kalınca devam ettirecek.

---

## 2) MVP’de olması gerekenler

İlk sürümde sadece şu 6 şey yeter:

### A. Job oluşturma

Kullanıcı bir hedef verir:

```bash
recon start example.com
```

Bu komut tek başına hiçbir şey yapmasın; bir pipeline oluştursun.

### B. Pipeline tanımı

Basit bir zincir:

* subdomain discovery
* alive check
* crawler
* URL collection
* directory fuzzing
* sonuçları kaydetme

### C. Pause / Resume

```bash
recon pause <job-id>
recon resume <job-id>
```

Bu özelliğin gerçek değeri burada.

### D. Progress kaydı

Her adım için:

* başladı
* kaç hedef işlendi
* yüzde kaç tamamlandı
* hata aldı mı
* nerede kaldı

### E. Kaynak limiti

Kullanıcı şunları verebilsin:

* max concurrency
* max CPU usage
* max RAM hedefi
* request rate limit
* timeout

### F. Kalıcı durum

Bilgisayar kapanırsa bile en azından:

* job listesi
* tamamlanan adımlar
* kalan kuyruk
* ayarlar
* çıktı dosyaları

SQLite ile saklansın.

---

## 3) Temel mimari

Ben bunu 5 parçaya bölerdim:

### 1. CLI

Kullanıcıyla konuşan katman.
Komutlar:

* `start`
* `pause`
* `resume`
* `status`
* `logs`
* `stop`
* `config`
* `list`

### 2. Daemon

Asıl işi yapan arka plan servis.
CLI sadece daemon’a komut gönderir.

Avantaj:

* terminal kapansa bile sistem çalışır
* state korunur
* iş sıralaması merkezi olur

### 3. Scheduler

Hangi job ne zaman çalışacak, hangi adım sırada, hangisi beklemede gibi kararları verir.

### 4. Runner / Adapter sistemi

Her dış aracı bir “plugin” gibi yönetir.

Örnek:

* `subfinder-runner`
* `httpx-runner`
* `katana-runner`
* `ffuf-runner`

Bunlar tek bir interface’e uyar:

* input alır
* komut üretir
* çalıştırır
* output parse eder
* state yazar

### 5. Storage

SQLite.
İçinde:

* jobs
* steps
* assets
* targets
* logs
* settings
* checkpoints

---

## 4) Asıl fark yaratan özellikler

Burada sıradan bir wrapper olmaktan çıkıp ürün olur.

### A. Checkpoint sistemi

Her adımın ortasında kaza olursa:

* son iyi noktayı kaydet
* oradan devam et

Mesela:

* 10.000 subdomain’in 7.200’ü işlendi
* kalan 2.800 kuyruğa yazıldı
* kapat-aç sonrası 7.201’den devam

### B. Akıllı duraklatma

Kullanıcı “pause” dediğinde:

* yeni job başlatma
* aktif işlerin güvenli kapanmasını bekle
* yarım kalan çıktıyı kaydet
* sonra tamamen dur

### C. Otomatik yavaşlama

Sistem yükünü izleyip:

* CPU yükselirse concurrency düşür
* RAM dolarsa yeni worker açma
* ağ sıkışırsa rate limit düşür

### D. Profil sistemi

Kullanıcı tek tek ayar vermesin:

* `safe`
* `balanced`
* `aggressive`

Örnek:

* `safe`: düşük kaynak, düşük concurrency
* `balanced`: varsayılan
* `aggressive`: sadece güçlü makinelerde

### E. Job template

Sık kullanılan workflow’lar:

* subdomain recon
* alive hosts
* URL collection
* JS discovery
* directory fuzzing

Kullanıcı sıfırdan pipeline yazmasın.

---

## 5) Kullanıcı akışı nasıl olmalı

Bence en önemli şey bu.

### Senaryo 1: Basit kullanım

```bash
recon start example.com
```

Araç şunu yapsın:

1. hedefi doğrula
2. job oluştur
3. pipeline kur
4. ilk aşamayı başlat
5. çıktı yolunu göster

### Senaryo 2: Yarım bırakıp çıkma

```bash
recon pause job-42
```

Araç:

* aktif işleri kapatır
* state yazar
* kuyruğu kaydeder

### Senaryo 3: Ertesi gün devam

```bash
recon resume job-42
```

Araç:

* kaldığı yerden devam eder
* eski progress’i gösterir

### Senaryo 4: Sistem çok yoruluyor

```bash
recon config set cpu.max=25
recon config set concurrency=5
```

---

## 6) Veri modeli

SQLite tarafında basit tablo yapısı yeterli.

### jobs

* id
* name
* target
* status
* created_at
* updated_at
* profile
* config_json

### steps

* id
* job_id
* name
* status
* progress
* started_at
* finished_at
* checkpoint_json

### targets

* id
* job_id
* value
* type
* status

### outputs

* id
* job_id
* step_id
* path
* kind

### logs

* id
* job_id
* step_id
* level
* message
* created_at

Bu yapı MVP için yeterli.

---

## 7) Plugin sistemi nasıl olmalı

Her tool için bir adaptör yaz.

Örnek interface:

```text
name: subfinder
input: domain list
output: subdomain list
supports_resume: true/false
resource_profile: low/medium/high
```

Runner şunları bilir:

* komutu nasıl kuracağı
* sonucu nasıl parse edeceği
* ne zaman checkpoint atacağı

Bu sayede sonra yeni araç eklemek kolay olur.

---

## 8) Güvenli ve temiz davranış

Burada önemli olan şey, aracın kaotik olmaması.

Şunlar olmalı:

* rate limit
* timeout
* retry limit
* kill timeout
* log rotation
* disk kullanım limiti
* user confirmation for destructive actions

Yani araç “yüklenmiş bir tarayıcı” değil, kontrollü bir iş yöneticisi gibi davranmalı.

---

## 9) İlk teknoloji seçimi

Bunu ben şöyle düşünürdüm:

### En mantıklı seçenek:

* **Rust**: güçlü, hızlı, tek binary, düşük kaynak
* CLI için çok uygun
* daemon yazmak için de iyi

### Alternatif:

* **Go**: hızlı geliştirme, dağıtım kolay
* plugin ve process yönetimi rahat

### Storage:

* SQLite

### UI:

* ilk başta terminal
* sonra istenirse TUI

---

## 10) İlk ekran / ilk komutlar

Bence ilk sürümde bunlar yeter:

```bash
recon start example.com
recon status
recon pause job-1
recon resume job-1
recon logs job-1
recon config
recon list
```

Status çıktısı şöyle olabilir:

```text
Job: example.com
State: running
Step: katana
Progress: 67%
CPU: 21%
RAM: 1.4 GB
Queue: 12,430 item
Last checkpoint: 14:22:10
```

---

## 11) MVP’den sonra eklenebilecek şeyler

Bunlar ikinci dalga:

* web dashboard
* job scheduling by time
* auto-pause on battery
* GitHub repo tarama
* export report
* multi-profile
* per-target rate limits
* remote execution agent
* distributed workers

Ama bunları baştan yapma.

---

## 12) En iyi ürün stratejisi

Senin için bence doğru strateji şu:

### Ürün değil, önce sorun çöz

İlk hedef:

* recon’u daha rahat yönetmek
* bilgisayarı kilitlememek
* kapat-aç sonrası devam etmek
* kaynak kullanımını düşürmek

### Sonra ürünleştir

Bunu yapan herkes zaten doğal olarak:

* başka tool eklemek ister
* rapor ister
* görselleştirme ister
* paylaşım ister

Böyle büyür.

---

## 13) Net roadmap

Ben olsam sırayı böyle kurarım:

### Faz 1

* CLI
* job oluşturma
* SQLite
* status
* pause/resume

### Faz 2

* worker runner sistemi
* tool adapter’ları
* progress tracking
* checkpoint

### Faz 3

* resource limits
* config profiles
* log viewer
* error recovery

### Faz 4

* TUI veya web dashboard
* export
* templates
* advanced orchestration

---

## 14) Bence ürünün adı ne olmalı

Kısa ve sert isimler daha iyi olur:

* `recond`
* `orchestr8`
* `flowrecon`
* `shaper`
* `grid`
* `conduct`
* `maestro`

Ben en çok `recond` veya `maestro` tonunu seviyorum.

---

İstersen bir sonraki mesajda bunu direkt **teknik spesifikasyona** çeviririm: komutlar, klasör yapısı, SQLite şeması ve ilk 10 task listesiyle birlikte.

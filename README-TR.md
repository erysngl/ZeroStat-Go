<div align="right">
  <a href="README.md">🇺🇸 English</a> | <strong>🇹🇷 Türkçe</strong>
</div>

<div align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go Sürümü" />
  <img src="https://img.shields.io/badge/HTMX-1.9.9-336699?style=for-the-badge&logo=htmx" alt="HTMX" />
  <img src="https://img.shields.io/badge/Tailwind_CSS-3.x-38B2AC?style=for-the-badge&logo=tailwind-css" alt="Tailwind CSS" />
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker" alt="Docker Uyumlu" />
</div>

<h1 align="center">ZeroStat-Go</h1>

<p align="center">
  <strong>Ultra Hafif Sistem ve Ağ Gösterge Paneli</strong>
</p>

## Genel Bakış

**ZeroStat-Go**, maksimum verimlilik için tasarlanmış yüksek performanslı, minimalist bir sunucu kaynak izleme panelidir. Ağır JavaScript frameworklerini tamamen es geçerek Go, HTMX ve Tailwind CSS kullanır. Altyapınızı izlerken sisteme yok denecek kadar az yük bindirip gerçek zamanlı görünürlük sağlar.

**CPU, RAM Bellek, Disk kapasitesi ve aktif Ağ I/O (kesin KB/s bazında)** kullanımınızı, sunucunuzu yormadan kolaylıkla izleyin.

## Temel Özellikler

- **Işık Hızında Backend:** `gopsutil` kullanan, statik olarak derlenmiş hafif bir Go altyapısıyla çalışır.
- **Sıfır JS-Framework Frontend:** Kesintisiz, anlık kısmi sayfa güncellemeleri için Go template sistemini doğrudan **HTMX**'e bağlar.
- **Dinamik Tema:** Tailwind CSS'ten gücünü alan yerleşik Aydınlık (Light) ve Karanlık (Dark) mod geçişleri.
- **Güvenli Erişim:** Metriklerinizi koruyan, oturum (Session) tabanlı sağlam bir kimlik doğrulama sistemi.
- **KB/s Ağ İzleme:** Gerçek zamanlı indirme(Rx)/yükleme(Tx) ağ hızlarını dinamik olarak ölçeklendirerek anında gösterir.
- **Dinamik Yapılandırma:** Yayınlandıktan sonra bile ayarlar paneli üzerinden port (varsayılan **9124**), şifre ve temayı değiştirebilirsiniz.
- **i18n Desteği:** Kusursuz İngilizce ve tam Türkçe dil (Localization) desteği.
- **Bulut Mimarisine (Cloud Native) Uygun:** `20MB`'ın altında boyuta sahip optimize edilmiş, ultra hafif Alpine Dockerfile ile gelir.

## Akıllı Otomasyon Motoru ve Uyarı Sistemi (Alerting)

ZeroStat-Go, sistem metriklerinizi belirlediğiniz sınırlar doğrultusunda arka planda güvenle değerlendiren güçlü ve yerleşik bir otomasyon motoruna sahiptir.

- **Gelişmiş Uyarı Mantığı (Alerting Logic):** Yanlış alarmları (false-positive) önlemek adına CPU, RAM ve Ağ (KB/s) kurallarınıza saniye bazlı bekleme süresi (**Duration**) koyabilirsiniz. Spam engellemek için ise soğuma/bekleme periyodu (**Cooldown**) desteği sunar.
- **Çok Kanallı Bildirimler (Multi-Channel):** Sınır aşıldığında entegre **Telegram Bot**, özelleştirilebilir Webhook'lar veya SMTP E-posta kanalıyla uyarıları anında iletir.
- **Dinamik Mesajlama:** `{hostname}`, `{metric}`, `{value}` ve `{duration}` gibi dinamik yer tutucuları (placeholder) kullanarak zengin bağlamlı, akıllı bildirim şablonları tasarlayabilirsiniz.
- **Güvenli Yürütme:** İstisnai durumlara karşı koruma altındaki bir "sandbox" ortamı yardımıyla ana uygulamayı (main thread) kilitlemeden kabuk komutlarını (örn. `docker stop $(docker ps -q)`) güvenle yürütebilirsiniz.

## Mimari

* **Programlama Dili:** Go (Golang)
* **Frontend (Arayüz):** HTMX + HTML/Templates
* **Stil:** Tailwind CSS (`node_modules` hantallığını yok etmek için CDN üzerinden)
* **İşletim Sistemi Köprüsü:** `shirou/gopsutil`
* **Durum/Yönlendirme:** Native `net/http` + `gorilla/sessions`

## Konfigürasyon (Yapılandırma)

ZeroStat-Go konfigürasyonu, varsayılan olarak çevresel (.env) değişkenlerle işler. Bu ayarlar daha sonradan web gösterge paneli üzerinden de değiştirilebilir.

1. Örnek konfigürasyon dosyasını kopyalayın:
   ```bash
   cp .env.example .env
   ```
2. Değişkenlerinizi düzenleyin:
   ```ini
   ZEROSTAT_PORT=9124
   ZEROSTAT_PASSWORD=sizin_guvenli_sifreniz
   SESSION_SECRET=32_baytlik_sifreleme_anahtariniz
   
   # Bildirim Seçenekleri
   TG_BOT_TOKEN=telegram_bot_tokeniniz
   TG_CHAT_ID=telegram_sohbet_id
   WEBHOOK_URL=https://kendi-webook-adresiniz.com/endpoint
   ```

## Kurulum ve Dağıtım

### Yöntem 1: Docker ile Kurulum (Önerilir)

Gerçek sunucu metriklerini konteynere doğru biçimde eşleyerek ZeroStat-Go'yu çalıştırmanın en sağlıklı ve temiz yoludur. Aşağıdaki içeriği `docker-compose.yml` adıyla kaydedin:

```yaml
services:
  zerostat:
    image: ghcr.io/erysngl/zerostat-go:latest
    container_name: zerostat-dashboard
    restart: unless-stopped
    ports:
      - "9124:9124"
    environment:
      - ZEROSTAT_PASSWORD=admin
    pid: host
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/host/root:ro
      - ./.env:/app/.env
      - ./data:/app/data
```

Sistemi başlatmadan önce Docker'ın yanlışlıkla dizin oluşturmasını engellemek için boş bir `.env` dosyası ve bir `data` klasörü oluşturduğunuzdan emin olun:
```bash
touch .env
mkdir data
```

Ardından sistemi başlatın ve panele erişimi sağlayın:

1. Konteyneri arka planda başlatmak için komutu çalıştırın:
   ```bash
   docker-compose up -d
   ```
2. Tarayıcınızdan **http://localhost:9124** adresine giderek panele erişin.

### Kalıcı Veri (Data Persistence)

ZeroStat-Go; Port, Yönetici Şifresi ve Telegram/Webhook kimlik bilgilerinizi web arayüzündeki Ayarlar (Settings) panelinden dinamik olarak yapılandırmanıza olanak tanır. Etkin kurallarınız ise Otomasyon (Automation) paneli üzerinden kontrol edilir.

`docker-compose` örneğinde gösterildiği gibi `.env` dosyasını (`- ./.env:/app/.env`) ve `data/` dizinini (`- ./data:/app/data`) dışarıya bağlayarak **Tam Veri Kalıcılığını** sağlarsınız:
1. **Uygulama Ayarları:** Ayarlar kaydedildiği anda anında `.env` dosyasına yazılır.
2. **Otomasyon Kuralları:** Herhangi bir kural eklendiğinde, silindiğinde veya aktifliği değiştirildiğinde anında `data/rules.json` dosyasına işlenir.

Bu sayede Docker konteyneriniz güncellenirse, yeniden oluşturulursa ya da silinirse **ayarlarınız ve tetikleyici kural yapılandırmalarınız kesinlikle kaybolmaz**. Sistem her yeniden başladığında güvenle tekrar diskten okunur.

### Yöntem 2: Doğrudan Cihaz Üzerinde Derleme (Native Build)

Cihazınızda Go `1.21+` kurulu olduğunu varsayarsak:

```bash
# Depoyu kopyalayın
git clone https://github.com/kullaniciadiniz/zerostat.git
cd zerostat

# Bağımlılıkları (paketleri) çekin
go mod tidy

# Çalıştırılabilir derlemeyi (exe/binary) oluşturun
go build -ldflags="-s -w" -o zerostat ./cmd/zerostat

# İşlemi başlatın (http://localhost:9124 üzerinden erişilebilir)
./zerostat
```

## Güvenlik

ZeroStat-Go paneli, güvenliği sıkılaştırılmış yalnızca HTTP'ye açık (`HttpOnly`) bir çerez oturumu (`SameSite=Lax`) yapısı arkasında korunmaktadır. Sistemin varsayılan şifresi `admin`'dir (veya `.env` dosyasında belirlediğiniz değer). **9124** portunu doğrudan genel internete açmadan önce /settings paneli altından ya da `.env` içerisinden bu şifreyi **derhal** değiştirmeniz, sistem güvenliği açısından son derece tavsiye edilir. Ek olarak, bozuk ya da son kullanma tarihi geçmiş bozuk çerezler, sistemi çökertmek (panic/error) yerine güvenlice temizlenerek otomatik bir şekilde giriş sayfasına (login) yönlendirilir.

## Lisans

Bu proje MIT Lisansı altında lisanslanmıştır.

---
<p align="center">
  <a href="https://erysngl.github.io">ERYSNGL | Github</a>
</p>

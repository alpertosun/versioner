# Versioner düzenleme planı

## 1. Varsayılan config ve çıktı sözleşmesi

Config dosyası bulunamadığında aşağıdaki kurallar kullanılacak. Repodaki
versioner.config.json aynı kuralları içerecek.

```json
{
  "rules": [
    { "pattern": "^main$", "strategy": "master", "prefix": "", "suffix": "" },
    { "pattern": "^master$", "strategy": "master", "prefix": "", "suffix": "" },
    { "pattern": "^release\\/(\\d+)\\.(\\d+)\\.0$", "strategy": "release", "prefix": "", "suffix": "" },
    { "pattern": "^develop$", "strategy": "develop", "prefix": "", "suffix": "" },
    { "pattern": "^feature\\/.*$", "strategy": "feature", "prefix": "", "suffix": "" },
    { "pattern": "^hotfix\\/.*$", "strategy": "hotfix", "prefix": "", "suffix": "" }
  ]
}
```

- rcN, betaN ve alphaN biçimleri korunacak; noktalı biçime geçilmeyecek.
- Varsayılan çıktılara SHA eklenmeyecek.
- Özel prefix, suffix ve placeholder desteği korunacak.
- Boş suffix metadata'yı kaldıracak. Eksik veya null suffix metadata'yı koruyacak.
- Genel bir katı SemVer zorunluluğu getirilmeyecek.

## 2. Ortak config ve etiket okuma hattı

- Config, etiketler okunmadan önce yüklenecek ve doğrulanacak.
- DefaultConfig() hem dosyasız kullanımın hem dal eşleşmesi fallback'inin kaynağı olacak.
- Config bulunamaması varsayılana geçirecek; bozuk JSON, okuma hatası, geçersiz
  regex ve bilinmeyen strateji başarısız çıkış üretecek.
- First ve longest eşleşmeleri korunacak. Kullanılan fallback kuralı etiket
  okuma hattına da dahil edilecek.
- Pattern dal adını eşleştirmeye devam edecek. Etiket formatı strateji,
  prefix, suffix ve placeholder bileşiminden çözümlenecek.
- Geçmiş etiketlerde etkin config'in bütün kuralları değerlendirilecek.
- ParsedTag, özgün adı, çekirdek sürümü, kararlı/prerelease türünü ve çıkarılan
  alanları taşıyacak.
- N sayaç, SHA commit kimliği olarak ele alınacak. SHA sıralaması sürüm
  yeniliğini belirlemeyecek.
- Eşleşme bütün etiketi kapsayacak. Uymayan etiketler atlanacak; başka bir
  genel parser'a düşülmeyecek. Config izin vermiyorsa v öneki de kabul edilmeyecek.
- Bir etikete farklı anlamlar veren kurallar ve belirsiz placeholder ayrımları
  açıklayıcı hata verecek. Üretim ve tekrar okuma tutarlılığı test edilecek.
- Release stratejisi sıfır mesafede de rc0 üretecek.
- Template değişimi tek geçişte yapılacak; yerleştirilen veriler tekrar
  placeholder olarak yorumlanmayacak. Tag placeholder'ı özgün etiketi kullanacak.
- Sayı dönüşümleri ve artırımlarda taşma kontrol edilecek. Büyük sayısal
  identifier karşılaştırmaları int dönüşümüyle yapılmayacak.

## 3. Git etiketi seçimi ve detached HEAD

- Yalnız HEAD'in geçmişindeki, config'e uygun etiketler aday olacak.
- En yakın aday tag..HEAD commit mesafesiyle seçilecek. Etiket metnini SemVer
  sırasına dizen eski fallback kaldırılacak.
- Eşit mesafedeki farklı anlamlar hata verecek; eşdeğer adaylarda etiket adına
  göre deterministik seçim yapılacak.
- Kararlı taban ile commit mesafesi referansı ayrı tutulacak. Main/master ve
  hotfix tabanı yalnızca config'in kararlı tanımladığı etiketlerden alınacak.
- Main/master patch artırımı kararlı tabana uzaklığa göre yapılacak.
- N referans etiketten sonraki commit sayısı olacak; etiketteki eski sayaca
  eklenmeyecek.
- Uygun etiket yoksa 0.1.0 başlangıç tabanı ve etiketsiz commit sayımı korunacak.
- Release ref'leri renkten bağımsız Git çıktısından okunacak. Git hataları
  etiket bulunamaması veya sıfır mesafe gibi yorumlanmayacak.
- VERSIONER_BRANCH dal override'ı eklenecek.
- Override yokken detached HEAD üzerindeki config'e uygun tek etiket özgün
  biçimiyle döndürülecek. Sıfır veya birden fazla adayda açıklayıcı hata verilecek.

## 4. Go ve CI güvenliği

- Minimum Go sürümü ve CI toolchain'i 1.27.3 olacak.
- Windows/amd64, Linux/amd64 ve macOS/arm64 hedefleri korunacak.
- Workflow varsayılan izni contents: read olacak. Write yalnızca yayın
  işine verilecek; build checkout'unda credential kalıcılığı kapatılacak.
- Actions mevcut ana sürümlerinin doğrulanmış tam commit SHA'larına sabitlenecek.
- PR ve release kontrollerinde test, go vet ve sabitlenmiş govulncheck çalışacak.
- Kontroller geçmeden yayın işi başlamayacak.
- Önceki Go 1.24.5 taramasındaki GO-2025-3956 (CVE-2025-47906) kaydı,
  sabit git komutunda istismar koşulları sağlanmadığından doğrulanmış
  uygulama açığı olarak sınıflandırılmayacak.
- Mevcut release dosyaları değiştirilmeyecek ve yeni yayın etiketi oluşturulmayacak.

## 5. Testler ve kabul ölçütleri

- Dosyasız kullanım ile varsayılan JSON altı dal kuralında aynı çıktıyı üretmeli.
- Özel prefix/suffix, sayaç, SHA, boş/eksik suffix ve tekrar eden placeholder'lar
  üretimden sonra doğru okunmalı.
- Config'e uymayan snapshot etiketi aynı commit'in sonucunu değiştirmemeli.
- Başka daldaki etiket ve prerelease referansı kararlı tabanı bozmamalı.
- rc0 ve 9/10/11 commit mesafeleri, alpha/beta üretimi, Git renk ayarları ve
  template deterministikliği doğrulanmalı.
- Bozuk config, belirsiz etiket, Git hatası ve sayı taşması doğru hata vermeli.
- Detached HEAD ve açık dal override senaryoları test edilmeli.
- Üç hedef platformda test, derleme ve güvenlik kontrolleri geçmeli.
- Dokümantasyon yeni varsayılanları, etiket okuma sözleşmesini, uyumsuz
  etiketlerin atlanmasını ve Go gereksinimini açıklamalı.
- Derleme örneği go build -o versioner ./cmd olarak düzeltilmeli.
- SHA'nın varsayılan çıktılardan kaldırılması ve eski etiketlerin config'e
  uymadığında dışlanması uyumluluk değişikliği olarak belirtilmeli.
- Kod, yorumlar, test adları ve commitler mevcut repo üslubunu izlemeli;
  gereksiz imza ve üretim notları eklenmemeli. Mevcut Git yazar ayarları korunmalı.

Tamamlanma ölçütü: ortak config hattı uygulanmış, özelleştirmeler korunmuş,
etiket seçimi düzeltilmiş ve hedef platform kontrolleri geçmiş olmalı.

## 6. Üretim ve parse için ortak kurallar

### Amaç ve kapsam

Stratejinin ürettiği biçim, placeholder tanımları ve suffix davranışı tek yerde
tanımlanacak. Üretim ve etiket okuma bu tanımları birlikte kullanacak. Bu bölüm,
yukarıdaki çıktı sözleşmesinin uygulamadaki tekrarlarını kaldırır; yeni config
alanı veya yeni etiket biçimi eklemez.

Başlangıç noktası: 9324d3a. Önceki bölümlerdeki davranışlar ve README.md mevcut
sözleşmedir. Go 1.27.3, standart kütüphane ve mevcut üç platformlu CI korunacak.
Yeni bağımlılık eklenmeyecek. Config şeması, varsayılan JSON, Git etiket seçimi,
commit mesafesi hesabı ve detached HEAD davranışı değişmeyecek.

### Mevcut tekrarlar

| Bilgi | Üretimdeki yeri | Okumadaki yeri | Ortak kaynak |
| --- | --- | --- | --- |
| rc/beta/alpha, metadata öneki ve patch=0 | core/strategy.go | core/tag.go: compileTagPattern | Stratejinin biçim tanımı |
| Suffix varsa metadata'yı kaldırma | core/format.go: RenderVersion | core/tag.go: compileTagPattern | Derlenmiş kuralın metadata kararı |
| Placeholder adları | core/format.go: values | core/tag.go: patterns | Tek alan tanım tablosu |
| Template parçalama | core/format.go: applyTemplate | core/tag.go: tokenize | Ortak template parçaları |
| Sayısal bileşen biçimi ve slug sınırı | core/version.go, core/format.go | core/tag.go | Mevcut yardımcılar ve ortak sınırlar |

### Hedef yapı

1. Her yerleşik strateji kendi format metodunda biçimini tanımlayacak: prerelease
   öneki, metadata kullanımı ve öneki, patch'in sıfırlanıp sıfırlanmadığı. main,
   master'a alias olacak. Strateji doğrulaması, StrategyByName ve parser aynı
   isim çözümlemesini kullanacak; ayrı izinli strateji listesi tutulmayacak.
2. Stratejiler sürümün tabanını ve artırılacak bileşenlerini hesaplamaya devam
   edecek. Prerelease, metadata ve patch normalizasyonunu ortak biçim uygulayıcısı
   yapacak. Parser'ın biçim kısıtları aynı tanımdan türetilecek. Stratejiler kendi
   biçimlerini kullanacak; içeriden isimle yeniden strateji aramayacak.
3. Config derlenirken prefix + zorunlu version alanı + suffix tek bir parça
   dizisine çevrilecek. RenderVersion bu diziyi dolduracak; ParseTag aynı diziyi
   eşleştirecek. Yerleştirilen değerler tekrar token olarak işlenmeyecek.
4. Her placeholder tek bir tabloda tanımlanacak. Tanım; değer okuyucusunu,
   kabul edilen metin biçimini ve eşleştirme uzunluk sınırını birlikte taşıyacak.
   Üretimdeki bağımsız values listesi ve parser'daki bağımsız patterns listesi
   kaldırılacak. Bilinmeyen placeholder'lar ortak tokenizasyon sırasında silinecek.
5. tag.go; ParsedTag, eşleştirme araması, alan tutarlılığı ve belirsizlik kontrolünü
   taşıyacak. İçinde stratejiye göre rc/beta/alpha/g seçen switch kalmayacak.
   Version.String, ParseVersion, sanitizeSlug ve mevcut placeholder regex'i
   kullanılacak; bunların yeni kopyaları yazılmayacak.
6. Config'in seçtiği strateji mevcut Engine.Register ile dala bağlanacak ve
   Engine.Resolve üzerinden çalıştırılacak. Engine ve Strategy.NextVersion
   arayüzleri korunacak; özel stratejilerin format metodu yazması gerekmeyecek.

### Dosyalar ve arayüzler

| Dosya | Sorumluluk |
| --- | --- |
| core/format_rules.go (yeni) | versionFormat, ortak alan tanımları, ruleFormat ve template derleme |
| core/strategy.go | İsim çözümleme, stratejilerin kendi biçim tanımları ve sürüm hesapları |
| cmd/main.go | Config'in seçtiği stratejiyi Engine.Register ve Resolve ile çalıştırma |
| core/engine_test.go (yeni) | Mevcut Engine API'siyle özel strateji, exact/prefix seçimi ve hata aktarımı |
| core/config.go | Ortak strateji doğrulaması ve CompiledRule içine derlenmiş formatı koyma |
| core/format.go | Ortak parçalar ve alan okuyucuları üzerinden render; mevcut slug yardımcıları |
| core/tag.go | Ortak format üzerinden eşleştirme, çözümleme ve belirsizlik denetimi |
| core/version.go | Ortak sayısal bileşen deseni; mevcut sürüm parse/karşılaştırma davranışı |
| core/format_rules_test.go (yeni) | Ortak tanımın iki yönde de kullanıldığını doğrulama |
| core/tag_test.go, core/config_test.go, cmd/main_test.go | Mevcut sözleşme ve uçtan uca regresyonlar |
| README.md | Ortak üretim/okuma akışını kısa biçimde açıklama |

İç arayüzler:

```go
type versionFormat struct {
    prereleasePrefix string
    metadataEnabled  bool
    metadataPrefix   string
    zeroPatch        bool
}

type formattedStrategy interface {
    Strategy
    format() versionFormat
}

type fieldDefinition struct {
    value       func(Version, GitContext, *ruleFormat) string
    pattern     func(*ruleFormat) string
    maxLength   func(*ruleFormat) int
    fromVersion bool
    numeric     bool
}

func lookupStrategy(name string) (formattedStrategy, error)
func (f versionFormat) apply(v Version, ctx GitContext) Version
func compileRuleFormat(rule CompiledRule, format versionFormat) *ruleFormat
func (f *ruleFormat) render(v Version, ctx GitContext) string
func parseReleaseBranch(branch string) (major, minor int, err error)
```

ruleFormat, ortak template parçalarını, alan tanımlarını, versionFormat değerini
ve keepMetadata kararını tutacak. Alan tablosunun değer okuyucuları özgün Version,
GitContext ve ruleFormat üzerinden çalışacak. Böylece version metadata'sız
gösterilirken buildMeta özgün değerini koruyabilecek. Regex ve uzunluk sınırları
biçim öneklerinin uzunluğundan ve alanın sınırlarından türetilecek.
fromVersion ve numeric işaretleri, alan türetimi ve sayısal doğrulamanın da aynı
tablodan yönetilmesini sağlayacak; parser'da ayrı alan listeleri tutulmayacak.
maxLength=0, branch/tag gibi sabit uzunluk sınırı olmayan alanlarda kalan girdi
uzunluğunu kullanacak; toplam eşleştirme adımı sınırı yine uygulanacak.

Mevcut dış arayüzler korunacak: Strategy.NextVersion, StrategyByName,
RenderVersion, ParseTag ve ParsedTag. CompiledRule.tag özel alanı ortak format
alanına dönüşecek. Derlenmemiş, geçerli bir CompiledRule ile doğrudan çağrılar da
aynı derleyiciyi kullanacak; ayrı bir fallback parser/renderer yazılmayacak.
RenderVersion'ın nil rule davranışı korunacak. Derlenmiş kurallar derleme anındaki
değerleri tutacak; değişen config yeniden Compile edilerek kullanılacak.

Özellikle incelenecek durumlar ve test sahipliği:

- Eksik/null/boş suffix ile version ve buildMeta'nın farklı görünmesi: 6.1 ve 6.3.
- Strateji öneki değiştiğinde eski regex/uzunluk sınırının kalması: 6.3.
- Derlemeden sonra kaynak config'in değiştirilmesiyle iki yönün ayrışması: 6.3.
- Bitişik/tekrarlı placeholder'ların yanlış ama başarılı parse edilmesi: 6.4.
- Dal/tag verisinin placeholder içermesi, Unicode veya boş değer taşıması: 6.4.

### Uygulama adımları

#### 6.1. Çıktı sözleşmesini sabitle

- [x] Mevcut TestDefaultVersionsRoundTrip, TestCustomTagRoundTrip,
  TestSuffixPresence, TestRepeatedFieldsMustAgree ve TestTemplateValuesAreNotExpanded
  testlerini başlangıç kontrolü olarak çalıştır: go test ./core -count=1.
- [x] TestFormatCompatibility tablosunu ekle: altı varsayılan dal, N=0/9/10/11,
  eksik/null/boş suffix, özel prefix ve suffix, main alias, özgün tag adı.
  Hem tam çıktı metnini hem parse sonucunun Version, Kind ve Fields değerlerini
  açık beklenen değerlerle doğrula; yalnızca iki fonksiyonu birbirine karşı test etme.
- [x] Metadata farkını sabitle: release/1.2.0, SHA=abc1234 ve N=0 için eksik/null
  suffix 1.2.0-rc0+gabc1234; boş suffix 1.2.0-rc0; +{sha} ise
  1.2.0-rc0+abc1234 üretmeli. {buildMeta}, suffix verilse de gabc1234 kalmalı.
- [x] Bu refactor testlerinin mevcut kodda geçtiğini doğrula; mevcut beklentileri
  yeni uygulamayı geçirmek için değiştirme.

#### 6.2. Strateji biçimlerini ortaklaştır

- [x] Biçimleri ilgili stratejinin format metodunda tanımla: master metadata'sız/kararlı; release rc/gSHA;
  develop beta/SHA; feature alpha/SHA; hotfix kararlı/gSHA. Sıfır patch yalnızca
  release, develop ve feature tanımlarında açıkça işaretlensin.
- [x] lookupStrategy isim çözümlemesini StrategyByName ve Config.Validate içinde kullan.
  main alias'ı master kaydına yönlendir. İzinli stratejilerin ikinci listesi kalksın.
- [x] NextVersion metotlarının çıktı biçimlendirmesini versionFormat.apply'a bağla.
  Sayısal artırımlar ve hata kontrolleri mevcut stratejilerde kalsın. Master'ın
  merge yolu dahil her çıkış ortak biçimlendirmeden geçsin.
- [x] CLI'de seçilen stratejiyi Engine.Register ve Engine.Resolve üzerinden
  çalıştır. Engine API'sini değiştirme; strategyDefinition veri katmanını ve
  applyStrategyFormat içindeki ikinci strateji aramasını kaldır.
- [x] LatestTagNameAndVersion'a eklenen kullanılmayan variadic config parametresini
  kaldır; mevcut parametresiz API'yi koru.
- [x] compileTagPattern/derive içindeki strateji biçimi bilgilerini aynı tanıma
  bağla. Release dalını çözümleyen mevcut kodu parseReleaseBranch'e çıkar;
  üretim ve yakalanmış branch doğrulaması bunu kullansın. Parse sırasında sürüm
  üretmek için ReleaseStrategy.NextVersion çağrılmasına gerek kalmasın.
- [x] go test ./core ./cmd -count=1 çalıştır; tümü geçmeli. Commit kapsamı yalnızca
  bu görev ve testleri olsun: "strateji biçimlerini ortak tanıma taşı".

#### 6.3. Renderer ve parser'ı aynı derlenmiş formata bağla

- [x] Ortak alan tablosunu ve compileRuleFormat'ı oluştur. Mevcut templatePart ve
  tokenize mantığını paylaş; genel amaçlı yeni bir template dili ekleme.
- [x] Sayısal bileşen desenini version.go'dan paylaş; taşma kontrolünü koru.
  Slug'ın 16 karakter sınırını sanitizeSlug ve alan tanımı için ortaklaştır.
  SHA sınırı 4..64 hex karakter olarak kalsın. Pre/buildMeta/version uzunlukları
  bağımsız sihirli sayılar yerine bu alanlardan ve biçim öneklerinden hesaplansın.
- [x] keepMetadata kararını yalnızca ortak format derlenirken suffix==nil ile
  belirle. Renderer ve parser bu kararı kullansın; kendi suffix kontrolünü yapmasın.
- [x] CompiledRule'ı ortak formata bağla. RenderVersion'daki ayrı values map'ini,
  applyTemplate yolunu ve tag.go'daki bağımsız alan regex tablosunu kaldır.
- [x] ParseTag ve alan türetimini ortak format/alan yardımcılarına bağla. Tekrarlı
  alanların eşitliği, branch/slug ilişkisi, SHA/metadata ilişkisi, tam eşleşme,
  100000 arama adımı sınırı ve SameMeaning davranışı korunsun. SHA sıralaması eklenmesin.
- [x] TestSharedFormatDefinition ekle: global kaydı değiştirmeden yerel bir biçimde
  prereleasePrefix=candidate, metadataPrefix=git-, metadataEnabled=true ve
  zeroPatch=false seç. 1.2.3, N=10, SHA=abc1234 ve eksik suffix için ortak üretim
  1.2.3-candidate10+git-abc1234 versin; aynı derlenmiş biçim bunu aynı alanlarla
  okusun. Parser'a candidate/git- özel durumu eklenmemeli.
- [x] TestSharedFormatFieldLengths ekle: aynı yerel biçimde prefix
  {pre}|{buildMeta}|, eksik suffix, N=9223372036854775807 ve SHA olarak 64 adet
  a karakteri kullan. Üç hedef platform da 64 bit olduğundan bu N, en büyük
  geçerli int değeridir. Beklenen pre=candidate9223372036854775807 (28 karakter),
  buildMeta=git- ardından 64 adet a (68 karakter) olmalı. Beklenen tam çıktı:
  candidate9223372036854775807|git-<64 adet a>|1.2.3-candidate9223372036854775807+git-<64 adet a>.
  Test kodunda <64 adet a> gösterimini gerçek metne aç; beklenen sonucu üretim
  veya ortak format yardımcılarıyla hesaplama. Name, çekirdek sürüm 1.2.3,
  PreRelease, BuildMeta, Kind=candidate ve Fields içindeki pre/buildMeta/N/sha
  değerlerini açık beklentilerle doğrula. Placeholder'lar zorunlu version alanından
  önce olmalı: suffix'te kullanıldıklarında alanlar version'dan türetilmiş olacağı
  için bağımsız eşleştirme uzunluk sınırları çalışmaz. Bu vaka eski pre=24 ve
  buildMeta=65 sınırlarından herhangi biri kalırsa başarısız olmalı.
  pre ve buildMeta'yı ayrı ayrı prefix'e yerleştiren alt vakalar da çalışmalı.
- [x] TestSharedFormatVersionLength ekle: ayrı bir yerel biçimde prereleasePrefix
  olarak 100 adet c, metadataPrefix=git-, metadataEnabled=true, zeroPatch=false,
  boş prefix, eksik suffix, sürüm 1.2.3, N=10 ve 64 adet a içeren SHA kullan.
  Beklenen tam çıktı "1.2.3-" + 100 adet c + "10+git-" + 64 adet a olmalı;
  uzunluğu 177 karakterdir. Çıktıyı açık beklenen metne karşı doğrula, ardından
  aynı derlenmiş biçimle Name, Version, Kind ve pre/buildMeta/N/sha alanlarını
  doğrula. Bu vaka eski version=160 sınırı kalırsa başarısız olmalı. Yerel biçimler
  yalnızca test verisi olacak; global strateji kaydı ve kullanıcı config şeması
  değiştirilmeyecek.
- [x] TestCompiledRuleSnapshot ekle: derleme sonrası kaynak prefix/suffix pointer'ını
  değiştirmek eski derlenmiş kuralın yalnızca bir yönünü değiştirmesin. Yeniden
  Compile edilmiş kural hem üretimde hem okumada yeni değeri kullansın.
- [x] go test ./core ./cmd -count=1 çalıştır; tümü geçmeli. Commit:
  "üretim ve etiket okumasını aynı derlenmiş formata bağla".

#### 6.4. Regresyonlar ve PR doğrulaması

- [x] Core testlerinde tekrarlı/bitişik placeholder, bilinmeyen placeholder,
  feature/{sha} dal adı, boş tag değeri, Unicode dal adı ve kısaltılan slug için
  kesin çıktı ve parse beklentilerini doğrula. Belirsiz girişler hata vermeye devam etsin.
  Slug örnekleri: feature/örnek -> feature-rnek;
  feature/abcdefghijklmnopq -> feature-abcdefgh. Boş tag için -from-{tag}
  suffix'i -from- olmalı; feature/{sha} içindeki ifade tekrar genişletilmemeli.
- [x] Mevcut Git entegrasyon testlerini çalıştır. snapshot, rc9/rc10 seçimi,
  kararlı taban, detached HEAD ve override sonuçları değişmemeli.
- [x] Engine.Register/Resolve ile harici özel strateji, exact/prefix önceliği,
  bilinmeyen dal ve strateji hatası aktarımını doğrula. CLI'de integration gibi
  farklı bir dala config üzerinden develop stratejisi atamasını test et.
- [x] TestUncompiledRuleAndNilRule ile derlenmemiş geçerli kuralın iki yönde
  kullanılmasını, nil rule çıktısını ve doğrudan renderer çağrısının yalnızca
  Version ile prefix/suffix verilerek kullanılabilmesini doğrula.
- [x] Fuzz testini varsayılan release biçimine ek olarak sabit ve açıkça ayrılmış
  özel prefix/suffix kombinasyonlarına genişlet. Belirsiz şablonlar ayrı hata
  testlerinde kalsın; fuzz sırasında belirsizliği sessizce başarı sayma.
- [x] go test ./... -count=1, go vet ./... ve
  go test ./core -run '^$' -fuzz '^FuzzDefaultTagRoundTrip$' -fuzztime=10s -parallel=2
  çalıştır. Hepsi geçmeli. govulncheck@v1.9.0 ve mevcut üç platformlu CI başarılı olmalı.
- [x] README'de üretim/okumanın aynı derlenmiş formatı kullandığını belirt.
  Git diff incelemesinde tag.go'da strateji öneki tablosu, ikinci placeholder
  listesi veya ikinci suffix politikası kalmadığını kontrol et.
- [x] Uygulama bitince PR #3'ün durumunu kontrol et. Açıksa mevcut dala takip
  commitlerini gönder ve PR açıklamasını güncelle; birleştirildiyse güncel master
  üzerinden takip PR'ı aç. Mevcut yayınlara veya etiketlere dokunma.

### Kabul ölçütü

Bir stratejinin biçimi veya placeholder tanımı tek yerde değiştirildiğinde üretim
ve parse birlikte değişmeli. Bu sonuç hem ortak tanımı değiştiren testle hem
bağımsız beklenen çıktı testleriyle doğrulanmalı. Uzunluk testleri pre/buildMeta
alanlarını version'dan önce eşleştirmeli ve 160 karakterden uzun canonical
sürümü ayrıca kapsamalı; türetilmiş alanlar yüzünden sınır kontrolünü atlamamalı.
Mevcut config ve çıktılar
korunmalı; etiket seçimi değişmemeli. Yalnızca kodu başka dosyaya taşımak yeterli
sayılmayacak; iki ayrı biçim tanımı kalmayacak.

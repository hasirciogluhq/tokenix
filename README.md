# tokenix

Tokenix, farkli LLM modelleri icin token sayisini tahmini olarak hesaplayan bir Go CLI aracidir.
Bu arac dosya, stdin ve direkt `--data` girdi yollarini destekler.

## Ozellikler

- Cobra tabanli CLI
- `--data` ile direkt metin uzerinden hesaplama
- `--file` ile dosyadan hesaplama
- `--dir` ile klasor ve altindaki tum dosyalari (ozyinelemeli) hesaplama
- `--include` / `--exclude` glob filtreleri
- stdin pipe ile hesaplama
- Model bazli min / max / average / net token tahmini
- Tum secili modeller icin genel ozet (spread dahil)
- JSON veya text cikti
- GitHub Actions CI + GoReleaser release akisi

## Desteklenen Modeller

- `gpt-4o-mini`
- `gpt-4o`
- `claude-3.5-sonnet`
- `claude-3-haiku`
- `gemini-1.5-flash`
- `gemini-1.5-pro`

## Kurulum

### 1) Homebrew ile (onerilen)

`homebrew-tap` reposu olustugunda:

```bash
brew tap hasirciogluhq/tap
brew install tokenix
```

Not: Eger formula ismi cakisirsa su sekilde de kurabilirsiniz:

```bash
brew install hasirciogluhq/tap/tokenix
```

### 2) Bash install scripti ile

Latest surumu kur:

```bash
curl -fsSL https://raw.githubusercontent.com/hasirciogluhq/tokenix/main/scripts/install.sh | bash
```

Belirli surum kur:

```bash
curl -fsSL https://raw.githubusercontent.com/hasirciogluhq/tokenix/main/scripts/install.sh | bash -s -- v0.1.0
```

### 3) Go ile

```bash
go install github.com/hasirciogluhq/tokenix@latest
```

## Kullanim

### 1) `--data` ile

```bash
tokenix --data "Merhaba dunya bu bir token tahmin denemesi"
```

### 2) Dosyadan

```bash
tokenix --file ./prompt.txt
```

### 3) stdin ile

```bash
echo "stdin ile gelen metin" | tokenix --model gpt-4o
```

### 4) Klasor (alt klasorler dahil) + filtre

```bash
tokenix --dir ./src
tokenix --dir . --include '*.go' --exclude '*_test.go,vendor'
tokenix --dir . -i 'internal/**' -x '*.md' --format json   # json cikti kullanilan dosyalari da listeler
```

Filtre kurallari:

- Desen goreli yola, dosya adina veya herhangi bir ust klasor adina uyarsa eslesir (`*.go`, `vendor`)
- `dir/**` o klasorun altindaki her seyi eslestirir
- `--include` bos ise tum dosyalar dahildir; `--exclude` her zaman onceliklidir
- `.git`, `node_modules` ve binary dosyalar otomatik atlanir

### 5) Birden fazla model secimi

```bash
tokenix --data "test" --model gpt-4o,claude-3.5-sonnet
```

### 6) JSON cikti

```bash
tokenix --data "test" --format json
```

## Branch ve release stratejisi

Profesyonel akista:

1. `main`: stabil branch
2. `development`: aktif gelistirme branchi
3. Ozellik branchleri `development` uzerinden acilir
4. `development` -> `main` merge sonrasi tag atilir
5. SemVer tagleri (`v0.1.0`, `v0.1.1`, ...)
6. Tag push ile GoReleaser otomatik binary release uretir
7. GoReleaser ayni anda Homebrew formula gunceller

Ornek:

```bash
git checkout -b development
git push -u origin development
```

Yeni versiyon:

```bash
git checkout main
git pull origin main
git tag v0.1.0
git push origin v0.1.0
```

### Homebrew otomasyonu icin gerekenler

- `hasirciogluhq/homebrew-tap` adinda bir repo olustur
- Bu repoya yazma izni olan `HOMEBREW_TAP_GITHUB_TOKEN` secret'ini `tokenix` reposuna ekle
- Tag release aninda formula otomatik push edilir

## Go Proxy yenileme scripti

`scripts/refresh-go-proxy.sh` scripti, yeni tag sonrasinda Go proxy cache'ini tetiklemek icin kullanilir:

```bash
./scripts/refresh-go-proxy.sh hasirciogluhq/tokenix v0.1.0
```

## Test

Bu repoda unit testler:

- `internal/token/estimator_test.go`
- `internal/input/read_test.go`
- `internal/input/dir_test.go`
- `cmd/root_test.go`

CI pipeline her push ve PR'da `go test ./...` calistirir.

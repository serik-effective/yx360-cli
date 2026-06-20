# yx360-cli

`yx360` — командная утилита для Яндекс 360. Сейчас она умеет входить через официальный Yandex OAuth, работать с почтой через IMAP/SMTP, читать и менять события календаря через CalDAV, а также создавать ссылки Telemost.

## Что уже работает

- `yx360 login` — вход через OAuth.
- `yx360 login --mail` — вход с правами на чтение почты.
- `yx360 login --mail --mail-send` — вход с правами на чтение и отправку.
- `yx360 login --calendar --telemost` — вход для календаря и создания Telemost-ссылок.
- `yx360 mail list` — список писем.
- `yx360 mail search` — поиск.
- `yx360 mail read` — чтение письма.
- `yx360 mail attachment` — скачивание вложения.
- `yx360 mail send` — отправка письма через SMTP.
- `yx360 mail unsubscribe` — предпросмотр и подтвержденная отписка по заголовкам `List-Unsubscribe`.
- `yx360 calendar list/read/create/update/delete` — работа с событиями календаря.
- `yx360 telemost create` — создание Telemost-ссылки.

Отправка письма по умолчанию останавливается на предпросмотре и спрашивает подтверждение. Флаг `--yes` нужен только для случаев, где человек уже явно согласовал адресатов, тему, текст и вложения.

## Сборка

Нужен Go 1.26, в модуле зафиксирован toolchain `go1.26.4`.

```bash
go build -o bin/yx360 ./cmd/yx360
```

Проверки:

```bash
go test ./...
go vet ./...
```

## Первый вход

Нужен OAuth client id от Яндекса:

```bash
export YX360_CLIENT_ID=<client-id>
./bin/yx360 login
```

Для почты:

```bash
./bin/yx360 login --mail --mail-send
```

В настройках Яндекс 360 Почты должен быть разрешен доступ почтовых клиентов по IMAP/SMTP и OAuth-токенам. Если это выключено, IMAP/SMTP-аутентификация не пройдет даже с валидным OAuth-токеном.

Для календаря и Telemost нужен отдельный OAuth app: Яндекс отклоняет смесь почтовых, календарных и Telemost-scope в одном приложении. Укажите второй client id отдельно:

```bash
export YX360_CALENDAR_CLIENT_ID=<calendar-telemost-client-id>
./bin/yx360 login --calendar --telemost
```

Нужные scopes второго приложения:

```text
login:info
calendar:all
telemost-api:conferences.create
```

## Примеры

```bash
./bin/yx360 mail list --limit 20
./bin/yx360 mail search --from user@example.com --since 2026-06-01 --json
./bin/yx360 mail read <uid> --json
./bin/yx360 mail attachment <uid> <attachment-id> --out ./downloads
./bin/yx360 mail send --to user@example.com --subject "Привет" --body "Текст письма"
./bin/yx360 mail unsubscribe <uid>
./bin/yx360 calendar list --from 2026-06-20 --to 2026-06-21
./bin/yx360 calendar create --title "Встреча" --starts-at 2026-06-22T09:00:00+06:00 --ends-at 2026-06-22T09:30:00+06:00 --telemost
./bin/yx360 calendar update <event-href> --title "Новое название"
./bin/yx360 calendar delete <event-href>
./bin/yx360 telemost create
```

`--json` включает машинно-читаемый вывод. Он нужен скриптам и агентам; человеку обычно проще читать обычный вывод.

## Хранение токена

По умолчанию токены лежат в системном keychain. Почта и Calendar/Telemost хранятся раздельно, потому что для них нужны разные OAuth-приложения. Для headless/CI есть флаг `--insecure-file-store`, но он пишет credential в plaintext-файл с правами `0600`.

Не включайте `--insecure-file-store` по привычке. С mail-scope токен дает доступ к почте, а с `mail:smtp` еще и к отправке писем.

## Ограничения

- Refresh токена не реализован. Зарегистрированное приложение требует `client_secret` для refresh, а CLI не должен тащить секрет внутри бинарника. При истечении токена нужно снова выполнить `yx360 login`.
- Сетевые вызовы к Яндексу идут по IPv4: в текущей среде IPv6-маршрут до Яндекса ломался.
- OpenClaw пока отмечен как docs-compatible, но отдельный executable smoke для адаптера `yx360` в OpenClaw не запускался.

## Документы для агентов

Основная документация для человека — этот README. Агентам нужны отдельные файлы:

- [AGENTS.md](AGENTS.md) — рабочее соглашение для Codex/Claude/OpenCode.
- [docs/agent-contract.md](docs/agent-contract.md) — контракт CLI для агентов: JSON, redaction, ошибки, подтверждение отправки.
- [docs/runtime-compatibility.md](docs/runtime-compatibility.md) — Codex, Claude Code, OpenCode, OpenClaw.

Эти файлы не являются пользовательским гайдом и не заменяют README.

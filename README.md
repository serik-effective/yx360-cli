# yx360-cli

`yx360` — командная утилита для Яндекс 360. Сейчас она умеет входить через официальный Yandex OAuth, работать с почтой через IMAP/SMTP, читать и менять события календаря через CalDAV, создавать ссылки Telemost, а также читать ответы и управлять формами через API Яндекс Форм.

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
- `yx360 calendar rooms list/add/discover` — локальный справочник переговорок для бронирования через Calendar.
- `yx360 telemost create` — создание Telemost-ссылки.
- `yx360 login --forms` — вход с правами на чтение и запись форм.
- `yx360 forms responses <survey-id>` — ответы на форму.
- `yx360 forms create` — создание формы (с подтверждением).
- `yx360 forms questions add <survey-id>` — добавление вопроса типа `rating` (оценка 1..N), `text` или `integer` (с подтверждением).
- `yx360 forms publish/unpublish <survey-id>` — публикация и снятие с публикации (с подтверждением).

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

Нужно два OAuth-приложения Яндекса.

Первое приложение — для почты:

```text
login:info
mail:imap_full
mail:smtp
```

Его client id передается через `YX360_CLIENT_ID`:

```bash
export YX360_CLIENT_ID=<client-id>
./bin/yx360 login --mail --mail-send
```

Второе приложение — для календаря и Telemost:

```text
login:info
calendar:all
telemost-api:conferences.create
```

Его client id передается через `YX360_CALENDAR_CLIENT_ID`:

```bash
export YX360_CALENDAR_CLIENT_ID=<calendar-telemost-client-id>
./bin/yx360 login --calendar --telemost
```

Третье приложение — для форм:

```text
login:info
forms:read
forms:write
```

Его client id передается через `YX360_FORMS_CLIENT_ID`, плюс нужен id организации в `YX360_FORMS_ORG_ID` (API Форм работает только для Яндекс 360 для бизнеса и требует заголовок `X-Org-Id`):

```bash
export YX360_FORMS_CLIENT_ID=<forms-client-id>
export YX360_FORMS_ORG_ID=<org-id>
./bin/yx360 login --forms
```

В настройках Яндекс 360 Почты должен быть разрешен доступ почтовых клиентов по IMAP/SMTP и OAuth-токенам. Если это выключено, IMAP/SMTP-аутентификация не пройдет даже с валидным OAuth-токеном.

Почта и Calendar/Telemost входят отдельно. Яндекс отклоняет смесь почтовых, календарных и Telemost-scope в одном OAuth-приложении, поэтому CLI хранит два токена в разных профилях.

## Вход без браузера (headless/VDS)

Если `yx360` запускается на удаленной машине без своего браузера (например VDS для агента), используйте вход в два шага `yx360 login --manual`. Браузер открывает человек у себя, а обратно вставляет только короткий код. PKCE тут secretless (публичный клиент), verifier наружу не печатается.

1. Начало — печатает только ссылку авторизации и сохраняет промежуточное состояние. Scope-флаги (`--mail`/`--mail-send`, `--calendar`/`--telemost`, `--forms`) задаются здесь и выбирают профиль; их не нужно повторять на шаге завершения.
2. Человек открывает ссылку в своем браузере и подтверждает доступ. Яндекс перенаправляет на страницу `https://oauth.yandex.ru/verification_code`, где код показан прямо в браузере.
3. Завершение — обменивает вставленный код на токен и сохраняет его в профиль. В `--code` можно вставить и сам код, и полный redirect-URL.

```bash
./bin/yx360 login --manual --begin --mail
# открыть ссылку в браузере, подтвердить, скопировать код со страницы verification_code
./bin/yx360 login --manual --complete --code <code> --insecure-file-store
```

На VDS без системного keychain шаг `--complete` нужно запускать с глобальным флагом `--insecure-file-store` (plaintext-файл с правами `0600`); без него keychain выдаст ошибку, тихого plaintext-фолбэка нет. `--manual` нельзя совмещать с `--device`, и нужен ровно один из `--begin`/`--complete`. Device-flow для вставки кода по-прежнему недоступен: обмен device-code на токен у Яндекса требует `client_secret`.

## Почта

```bash
./bin/yx360 mail list --limit 20
./bin/yx360 mail search --from user@example.com --since 2026-06-01 --json
./bin/yx360 mail read <uid> --json
./bin/yx360 mail attachment <uid> <attachment-id> --out ./downloads
./bin/yx360 mail send --to user@example.com --subject "Привет" --body "Текст письма"
./bin/yx360 mail unsubscribe <uid>
```

`mail send` и `mail unsubscribe --apply` по умолчанию показывают предпросмотр и ждут подтверждение. `--yes` используйте только когда действие уже проверено человеком.

## Календарь и Telemost

Календарь работает через CalDAV. Для CalDAV Яндекс принимает заголовок `Authorization: OAuth <token>`; обычный `Bearer` не подходит.

```bash
./bin/yx360 calendar list --from 2026-06-20 --to 2026-06-21
./bin/yx360 calendar read <event-href>
./bin/yx360 calendar rooms discover --from 2026-01-01 --to 2026-12-31
./bin/yx360 calendar rooms list
./bin/yx360 calendar rooms add Sun sun@effective.band
./bin/yx360 calendar create --title "Встреча" --starts-at 2026-06-22T09:00:00+06:00 --ends-at 2026-06-22T09:30:00+06:00
./bin/yx360 calendar create --title "Встреча" --starts-at 2026-06-22T09:00:00+06:00 --ends-at 2026-06-22T09:30:00+06:00 --room Sun
./bin/yx360 calendar create --title "Созвон" --starts-at 2026-06-22T10:00:00+06:00 --ends-at 2026-06-22T10:30:00+06:00 --telemost
./bin/yx360 calendar update <event-href> --title "Новое название" --room Sun
./bin/yx360 calendar delete <event-href>
./bin/yx360 telemost create
```

`event-href` — это `href` из JSON-ответа `calendar list` или `calendar create`.

Создание, изменение и удаление событий спрашивают подтверждение. `calendar create --telemost` сначала создает Telemost-ссылку, потом записывает ее в событие календаря.

`--room` бронирует переговорку через `ATTENDEE;CUTYPE=ROOM` из локального справочника. `--location` остается только отображаемым текстом места и сам по себе не бронирует комнату. Справочник хранится в пользовательском config-dir в JSON-файле с правами `0600`; путь можно переопределить через `YX360_CONFIG_HOME`. Это не секреты, но локальное состояние пользователя. `calendar rooms discover` наполняет справочник из уже существующих событий, где Яндекс вернул `CUTYPE=ROOM` или `CUTYPE=RESOURCE`; комнаты, которых не было в просмотренном диапазоне, нужно добавить вручную через `calendar rooms add`.

## Формы

Формы работают через API Яндекс Форм (`api.forms.yandex.net`). Доступ только у Яндекс 360 для бизнеса. Заголовок организации выбирается по формату `YX360_FORMS_ORG_ID`: числовой → `X-Org-Id`, нечисловой (Cloud, hex) → `X-Cloud-Org-Id`.

Полный сценарий — создать форму-опрос с пятью оценками 1–5 и опубликовать:

```bash
SID=$(./bin/yx360 --json forms create --title "Оценка мероприятия" --yes | jq -r .id)
for q in Контент Спикеры Организация Локация Нетворкинг; do
  ./bin/yx360 forms questions add "$SID" --label "$q" --rating 5 --yes
done
./bin/yx360 forms publish "$SID" --yes
./bin/yx360 forms responses "$SID" --json
```

`create` и `publish` печатают публичную ссылку `https://forms.yandex.ru/cloud/<survey-id>` и ссылку на ответы `https://forms.yandex.ru/cloud/admin/<survey-id>/answers?view=stats` (API их не возвращает — CLI выводит сам).

`forms create`, `forms questions add`, `forms publish` и `forms unpublish` по умолчанию показывают предпросмотр и ждут подтверждение; `--yes` — только когда человек уже согласовал. Опубликованная форма доступна по ссылке любому.

`survey-id` передается явно: команды «список всех форм» нет — у API Форм нет документированного эндпоинта перечисления. `forms create` задает только заголовок; вопросы добавляются отдельно через `forms questions add` (`--type rating|text|integer`, по умолчанию `rating` — оценка 1..N, radio).

## JSON

```bash
./bin/yx360 --json calendar list --from 2026-06-20 --to 2026-06-21
./bin/yx360 --json telemost create --yes
```

`--json` включает машинно-читаемый вывод. Он нужен скриптам и агентам; человеку обычно проще читать обычный вывод.

## Хранение токена

По умолчанию токены лежат в системном keychain. Почта и Calendar/Telemost хранятся раздельно, потому что для них нужны разные OAuth-приложения. Для headless/CI есть флаг `--insecure-file-store`, но он пишет credential в plaintext-файл с правами `0600`.

Не включайте `--insecure-file-store` по привычке. С mail-scope токен дает доступ к почте, а с `mail:smtp` еще и к отправке писем.

## Ограничения

- Refresh токена не реализован. Зарегистрированное приложение требует `client_secret` для refresh, а CLI не должен тащить секрет внутри бинарника. При истечении токена нужно снова выполнить `yx360 login`.
- Сетевые вызовы к Яндексу идут по IPv4: в текущей среде IPv6-маршрут до Яндекса ломался.
- `logout` пока очищает только старый default-профиль. Для почтового, calendar-telemost и forms профилей нужен отдельный follow-up.
- API Форм доступен только для Яндекс 360 для бизнеса и требует `YX360_FORMS_ORG_ID`. Личные аккаунты Форм не поддерживаются. Org id настраивается оператором разово; авто-определения по forms-токену нет (см. open-question по prompt-фолбэку).
- Команды «список всех форм» нет — у API Форм нет документированного эндпоинта перечисления, `survey-id` передается явно.
- `forms questions add` умеет типы `rating` (radio 1..N, live-verified), `text` и `integer`. Shape для `text`/`integer` собран по документации, но не подтвержден на живом API. Удаление вопросов не реализовано.
- Изменение события не умеет намеренно очищать строковое поле в пустое значение: пустой флаг трактуется как "не менять".
- Бронирование переговорок не делает Directory/org lookup и не ищет свободные слоты. CLI использует только локальный справочник комнат и статус, который вернет Calendar.
- Telemost-ссылку можно создать, но отмена/удаление конференции через официальный API пока не подтверждена.
- OpenClaw пока отмечен как docs-compatible, но отдельный executable smoke для адаптера `yx360` в OpenClaw не запускался.

## Переменные окружения

Для headless/CI/агентских запусков настройка идёт через env; флаги переопределяют там, где есть оба.

| Переменная | Нужна для | Назначение |
|---|---|---|
| `YX360_CLIENT_ID` | `login`, почта | client id почтового/дефолтного OAuth-приложения |
| `YX360_CALENDAR_CLIENT_ID` | `login --calendar`/`--telemost` | client id приложения Calendar+Telemost |
| `YX360_FORMS_CLIENT_ID` | `login --forms`, `forms *` | client id приложения форм |
| `YX360_FORMS_ORG_ID` | `forms *` | id организации; уходит в `X-Org-Id` (числовой) или `X-Cloud-Org-Id` (нечисловой) |
| `YX360_CONFIG_HOME` | опц. | переопределить config-root (файловое хранилище токена, справочник комнат) |
| `YX360_IMAP_HOST` / `YX360_SMTP_HOST` | опц. | хосты почты (по умолчанию `imap.yandex.ru` / `smtp.yandex.ru`) |
| `YX360_CALDAV_URL` | опц. | база CalDAV (по умолчанию `https://caldav.yandex.ru`) |
| `YX360_TELEMOST_API_URL` | опц. | база Telemost API |
| `YX360_FORMS_API_URL` | опц. | база Forms API (по умолчанию `https://api.forms.yandex.net`) |

`client_secret` нигде не используется — CLI это public-клиент OAuth (PKCE). Секреты, токены и внутренние URL не коммитим.

## Документы для агентов и skill-установщиков

Основная документация для человека — этот README. Если агент устанавливает `yx360` внутрь своего проекта или рантайма как skill/tool, ему нужны не рабочие инструкции этого репозитория, а контракт CLI:

- [docs/agent-contract.md](docs/agent-contract.md) — контракт CLI для агентов: JSON, redaction, ошибки, подтверждение отправки.
- [docs/agent-quickstart.md](docs/agent-quickstart.md) — короткий путь от нуля до первого `--json` вызова.
- [docs/agent-mode-roadmap.md](docs/agent-mode-roadmap.md) — предложенные agent-mode фичи (remote login, schema, dry-run, wrap-untrusted и т.д.).
- [docs/runtime-compatibility.md](docs/runtime-compatibility.md) — как подключать контракт к Codex, Claude Code, OpenCode, OpenClaw.

`AGENTS.md` в корне — это рабочее соглашение для разработки самого `yx360-cli` внутри этого репозитория. Не используйте его как install-doc для чужого проекта: он описывает Effective Harness-контекст этого checkout, а не контракт внешнего skill.

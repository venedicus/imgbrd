# imgbrd

Имиджборда на Go (handler → service → repository → SQLite). Репозиторий: [github.com/venedicus/imgbrd](https://github.com/venedicus/imgbrd).

## Возможности

- **Каталог:** сортировка тредов по бампу (`bumped_at`), закреп (`pinned`), лимит активных тредов на доску (`max_threads` → старые уходят в `archived`).
- **Посты:** sage (ответ без бампа), имя + трипкод, Markdown, ссылки `>>N` → якорь `#pN`, скрытие постов (модерация).
- **Файлы:** лимит размера (`IMGBRD_MAX_UPLOAD_MB`), проверка по magic bytes, **WebM/MP4**, превью JPEG для картинок, **дедуп по SHA-256** (повторный тот же файл не копируется).
- **Поиск:** FTS5 по тексту постов (с fallback на `LIKE`), страница `/search`.
- **Ленты и API:** `GET /rss/board/{slug}`, `GET /api/board/{slug}`, `GET /api/thread?id=`.
- **Экспорт треда:** `GET /thread/export?id=` — ZIP (`manifest.json`, `thread.html`, вложенные файлы).
- **Публичный журнал:** `/modlog` (HTML), дублируется в админке как JSON.
- **Баны по IP** (опционально привязка к доске), проверка при постинге.
- **Вебхук:** `POST` JSON при новом треде/посте (`IMGBRD_WEBHOOK_URL`, заголовок `X-Webhook-Secret`).
- **pprof:** отдельный listener на `IMGBRD_PPROF_ADDR` (или `127.0.0.1:6060` при `IMGBRD_ENABLE_PPROF=1`).

### Будущие возможности
- **Кэширование медиа для переиспользования:**

## Запуск

```bash
go run ./cmd/app
```

Нужны каталоги `templates/`, `static/` в рабочей директории. После обновления схемы при ошибках FTS удалите `data.db` и дайте приложению создать БД заново.

## Переменные окружения

| Переменная | По умолчанию | Назначение |
|------------|--------------|------------|
| `IMGBRD_SITE_TITLE` | `imgbrd` | Заголовок сайта |
| `IMGBRD_PUBLIC_ADDR` | `:8080` | Публичный HTTP |
| `IMGBRD_DEFAULT_THEME` | `futaba` | Тема без cookie |
| `IMGBRD_MAX_UPLOAD_MB` | `25` | Максимум размера файла |
| `IMGBRD_WEBHOOK_URL` | *(пусто)* | URL для событий постов/тредов |
| `IMGBRD_WEBHOOK_SECRET` | *(пусто)* | Секрет вебхука |
| `IMGBRD_ENABLE_PPROF` | off | `1` / `true` — включить pprof |
| `IMGBRD_PPROF_ADDR` | *(пусто)* | Адрес pprof; при пустом и включённом флаге — `127.0.0.1:6060` |
| `IMGBRD_ADMIN_ADDR` | *(пусто)* | Админ API |
| `IMGBRD_ADMIN_TOKEN` | *(пусто)* | Токен админ API |

## Админ API (отдельный порт)

Заголовок: `Authorization: Bearer <token>` или `X-Admin-Token`.

- `GET /health`, `GET /stats`, `GET /boards`, `POST /boards`
- `POST /board-config` — `{"slug":"b","max_threads":200,"nsfw":true}`
- `POST /ban` — `{"ip":"1.2.3.4","board_id":null,"reason":"spam","expires_hours":24}`
- `GET /bans`
- `POST /posts/hide`, `POST /posts/unhide` — `{"post_id":1}`
- `POST /posts/edit` — `{"post_id":1,"text":"..."}` (история в `post_edits`)
- `POST /threads/pin` — `{"thread_id":1,"pinned":true}`
- `GET /modlog` — JSON журнала

## Структура

- `cmd/app` — точка входа
- `internal/db` — SQL init + программный upgrade (колонки, FTS, триггеры)
- `templates/`, `static/`

## Лицензия

MIT

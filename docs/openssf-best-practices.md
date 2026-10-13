# OpenSSF Best Practices: ответы для плашки

Ответы на критерии уровня **passing** [OpenSSF Best Practices](https://www.bestpractices.dev/) для canvas-api — по строке на критерий, с доказательствами, которые будет искать проверяющий. Уровни silver и gold здесь не разбираются.

Файл держится в актуальном виде: когда ответ на критерий меняется, меняется и строка. Плашку подтверждают заново, а не получают один раз.

Статусы — на 2026-10-13, после T07. Критерий, который ещё не выполнен, назван так и ссылается на задачу, которая его закроет. Анкета на bestpractices.dev заполняется по этой таблице. Уровень passing пока не достигнут:
- `vulnerability_report_private` закрывает шаг автора в T07;
- `version_semver`, `version_tags`, `release_notes` и `release_notes_vulns` закрывает релиз (T08).

Идентификаторы критериев и ответы — как в анкете: Met, Unmet, N/A.

## Основы

| Критерий | Ответ | Доказательство |
|---|---|---|
| `description_good` | Met | [`README.md`](../README.md): что это и для кого — в первом абзаце |
| `interact` | Met | issues репозитория на GitHub |
| `contribution` | Met | [`CONTRIBUTING.md`](../CONTRIBUTING.md): как сообщить, что примут, что отклонят |
| `contribution_requirements` | Met | [`CONTRIBUTING.md`](../CONTRIBUTING.md), «What a change has to satisfy»; сами правила — [`CLAUDE.md`](../CLAUDE.md) |
| `floss_license`, `floss_license_osi`, `license_location` | Met | MIT в [`LICENSE`](../LICENSE), в корне |
| `documentation_basics` | Met | `README.md`, [`product.md`](product.md), [`spec.md`](spec.md), [`privacy.md`](privacy.md) |
| `documentation_interface` | Met | spec 8: каждый ресурс API, его запрос, ответ и ошибки; контракт `proto/` придёт в эпике MVP |
| `sites_https` | Met | сайт проекта — репозиторий на GitHub, только HTTPS. Адреса сервиса и SPA решает T09, раздают их Cloud Run и Firebase Hosting только по HTTPS |
| `discussion` | Met | issues на GitHub |
| `english` | Met | Сообщения об ошибках принимаются по-английски. По-английски написаны код, комментарии, `README.md`, `CONTRIBUTING.md`, `SECURITY.md` и `CLAUDE.md` (R02). Проектные документы `docs/` и план — по-русски, это рабочий язык автора (R25). Критерий — SHOULD |
| `maintained` | Met | идёт разработка; история коммитов |

## Управление изменениями

| Критерий | Ответ | Доказательство |
|---|---|---|
| `repo_public`, `repo_track`, `repo_interim`, `repo_distributed` | Met | публичный git-репозиторий на GitHub; каждое изменение — через PR |
| `version_unique` | Met | версия и коммит вшиваются в бинарник при сборке ([`internal/version`](../internal/version), `justfile`), и `/health` их называет |
| `version_semver` | Unmet | теги `vMAJOR.MINOR.PATCH` вводит релиз (T08, R45) |
| `version_tags` | Unmet | тег ставит workflow релиза на каждом зелёном `main` (T08) |
| `release_notes` | Unmet | `gh release create --generate-notes` (T08) |
| `release_notes_vulns` | Unmet | исправленная уязвимость описывается в GitHub advisory, на которую ссылается релиз (T08) |

## Сообщения об ошибках

| Критерий | Ответ | Доказательство |
|---|---|---|
| `report_process`, `report_tracker`, `report_archive` | Met | issues на GitHub, публичные и постоянные |
| `report_responses`, `enhancement_responses` | Met | критерий — о фактических ответах на issues, а сообщений об ошибках и предложений пока не было. Как их подавать, говорит [`CONTRIBUTING.md`](../CONTRIBUTING.md) |
| `vulnerability_report_process` | Met | [`SECURITY.md`](../SECURITY.md) |
| `vulnerability_report_private` | Unmet | станет Met, когда автор включит private vulnerability reporting на GitHub (шаг T07): до этого ссылка из `SECURITY.md` никуда не ведёт |
| `vulnerability_report_response` | Met | критерий — о фактическом первом ответе в пределах 14 дней на сообщения за последние полгода, а сообщений об уязвимостях не было. `SECURITY.md` срока не называет: ответ в эти 14 дней — то, что держит критерий, когда сообщение придёт |

## Качество

| Критерий | Ответ | Доказательство |
|---|---|---|
| `build`, `build_common_tools`, `build_floss` | Met | `just build` и многоэтапный `Dockerfile` для сервиса, `just web-build` для SPA. Тулчейн — Go, Node.js, `just` и Docker, все FLOSS |
| `test`, `test_invocation` | Met | `just test` и `just web-test`; проверки зовут `just ci-test` и `just ci-web` |
| `test_most` | Met | покрытие: `cmd/server` — 81,1 % инструкций, SPA — 95,7 % строк (T05) |
| `test_continuous_integration` | Met | [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) на каждом PR и каждом коммите `main` (R49) |
| `test_policy`, `tests_are_added`, `tests_documented_added` | Met | [`CLAUDE.md`](../CLAUDE.md), «Tests»: проверка считается покрытой, только когда её тест краснеет без неё. PR, у которого покрыто меньше 90 % изменённых строк, не мержится (`codecov.yml`, R50) |
| `warnings`, `warnings_fixed`, `warnings_strict` | Met | `golangci-lint` с конфигом в репозитории, Biome и проверка типов TypeScript: одно замечание роняет проверку. Находку анализатора закрывают правкой кода, а не пометкой (CLAUDE.md; standalone: R35) |

## Безопасность

| Критерий | Ответ | Доказательство |
|---|---|---|
| `know_secure_design` | Met | [`privacy.md`](privacy.md) и spec 12–13: границы доверия, что сервис хранит, а чего не хранит и не передаёт |
| `know_common_errors` | Met | spec 13; прочитанное с картинки — данные, а не указание: вердикт выносит программа по ключу (R38) |
| `crypto_published`, `crypto_call`, `crypto_floss`, `crypto_keylength`, `crypto_working`, `crypto_weaknesses`, `crypto_random` | N/A | криптографии в коде пока нет: сервер отвечает только на `/health`. Эпик MVP приносит её готовыми библиотеками, своей не будет: вход проверяет Firebase Admin SDK, PIN хешируется argon2id из `golang.org/x/crypto`, токены Centrifugo подписываются HS256 секретом из Secret Manager, случайное — из `crypto/rand` (spec 3.2, 12). Тогда строки становятся Met |
| `crypto_pfs` | N/A | сервис ещё не выкачен (T12). Тогда TLS завершает фронтенд Google, а до Neon, Upstash и Gemini тоже идёт TLS (spec 13) |
| `crypto_password_storage` | N/A | паролей нет: вход через Google или Apple. От PIN детского режима сервер будет хранить только хеш argon2id, а PIN — барьер интерфейса, а не граница безопасности (spec 12.2, R07) |
| `delivery_mitm`, `delivery_unsigned` | Met | всё по HTTPS. Образы закреплены по digest, actions — по коммиту, модули Go проверяет checksum database, пакеты npm — хеш реестра. Подпись релизов cosign приходит с T08 |
| `vulnerabilities_fixed_60_days`, `vulnerabilities_critical_fixed` | Met | `govulncheck`, `npm audit` по пакетам SPA и Trivy по образу сервиса роняют проверки (R48, R49); Dependabot предлагает обновления раз в неделю |
| `no_leaked_credentials` | Met | `gitleaks` по всей истории и по незакоммиченному на каждом PR (R49); секреты живут в Secret Manager |

## Анализ

| Критерий | Ответ | Доказательство |
|---|---|---|
| `static_analysis` | Met | `golangci-lint` для Go и Biome для SPA; CodeQL с набором `security-extended` для обоих; SonarCloud (R50) |
| `static_analysis_common_vulnerabilities` | Met | `gosec` в линтере и запросы CodeQL по потокам данных |
| `static_analysis_fixed` | Met | находку закрывают правкой кода (CLAUDE.md; standalone: R35); CodeQL держит мерж (R50) |
| `static_analysis_often` | Met | на каждом PR, CodeQL — ещё и раз в неделю |
| `dynamic_analysis` | Met | `go test -race` в каждом прогоне. Fuzz-тесты приходят вместе с кодом, который разбирает внешний ввод, — в эпике MVP (CLAUDE.md, «Tests») |
| `dynamic_analysis_unsafe` | N/A | Go и TypeScript безопасны по памяти; гонки ловит детектор |
| `dynamic_analysis_fixed` | Met | упавший тест роняет проверки |

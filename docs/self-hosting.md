# Развёртывание canvas-api

T10 — облако как код и разовый bootstrap, T11 — выкладка пайплайном, откат и возврат, T12 — чек-лист первой выкладки и расход сервиса (`just usage`). Полностью документ допишет T57. Решения, на которых всё держится, — R54–R58 в [decisions.md](decisions.md).

## Что делается руками

Один раз, и больше ничего:

1. **Bootstrap** — `just bootstrap`. Создаёт то, без чего пайплайну не войти в облако: проект, привязку к платёжному аккаунту, бакет состояния, федеративный пул с провайдером GitHub и аккаунт Terraform с его ролями.
2. **`infra/ci.env`** — две строки, которые печатает bootstrap, коммитятся в репозиторий.
3. **Секреты репозитория** (Settings → Secrets and variables → Actions):
   - `TF_VAR_billing_account` — ID платёжного аккаунта. Нужен каждому плану и apply пайплайна, без него выкладка не начнётся;
   - `TF_VAR_operator_email` — адрес для алерта нагрузки. Необязателен: без него алерта нет.
4. **Окружения GitHub** `prod` и `prod-hosting` (Settings → Environments → New environment), до первой выкладки. В каждом: «Deployment branches and tags» → «Selected branches and tags» → правило `main`. Ни ревьюеров, ни секретов в них нет. Окружение, которого нет, GitHub создаёт сам при первом задании — без правила, и тогда «только `main`» держал бы один текст workflow (R58).

Всё остальное — проект, сервис, Firebase, бюджет, алерт, журналы — описывает Terraform в `infra/terraform/`, а применяет пайплайн. Руками в консоли ничего не настраивается: следующий apply вернул бы настройку как было.

## Что нужно

- Платёжный аккаунт Google Cloud и право привязывать к нему проекты. Его ID — `gcloud billing accounts list`.
- Право создавать проекты.
- Devcontainer на x86_64: gcloud и Firebase CLI ставятся только туда (R28 standalone, R58). На arm64 те же команды запускаются из Cloud Shell, где есть оба.

## 1. Описать развёртывание

Три файла, все публичны. Платёжного аккаунта и почты оператора в них нет: они приходят из окружения.

| Файл | Что в нём |
|---|---|
| `infra/terraform/prod.auto.tfvars` | ID проекта, регион, репозиторий и числовой ID его владельца, пул, начало subject токенов GitHub, бюджет и его валюта |
| `infra/terraform/backend.hcl` | бакет состояния и префикс |
| `infra/ci.env` | две строки из вывода bootstrap |

Своя копия меняет здесь:
- ID проекта: он глобально уникален;
- бакет состояния;
- репозиторий и ID владельца — `https://api.github.com/users/<владелец>`;
- начало subject — `gh api repos/<владелец>/<репозиторий>/actions/oidc/customization/sub --jq .sub_claim_prefix`.

Адрес SPA выводится из ID проекта: `https://<ID проекта>.web.app` (R55).

## 2. Bootstrap

```bash
gcloud auth login --no-launch-browser
export TF_VAR_billing_account=…
just bootstrap
```

- Повтор безопасен: всё, что скрипт создаёт, он перед каждой попыткой ищет. Условие провайдера пула приводится к записанному в скрипте.
- Пул пускает один workflow — выкладку, `.github/workflows/deploy.yml`. Ни проверки, ни релиз не могут одолжить ни один аккаунт (R56).
- Аккаунт Terraform одалживают только задания окружения `prod` и план на PR — по subject их токенов (R58). Привязку «весь репозиторий», которую ставил bootstrap T10, повтор снимает.
- `roles/resourcemanager.projectIamAdmin` выдаётся с условием: выдавать и отзывать на проекте аккаунт Terraform может только две роли выкладки SPA.
- ID платёжного аккаунта скрипт не печатает и в ошибках gcloud заменяет его на `<billing account>`: вывод можно вставлять куда угодно.
- `just` не читает `.env`. Если ID лежит там строкой `TF_VAR_billing_account=…`, его экспортирует `set -a; . ./.env; set +a`.
- До всего остального скрипт сверяет валюту платёжного аккаунта с `budget_currency`: бюджет в чужой валюте отклоняется, и иначе это выяснилось бы только на первом apply.
- На только что созданном проекте права IAM доходят не сразу. Поэтому каждый шаг повторяется несколько раз, прежде чем показать ошибку.
- В конце скрипт печатает две строки — их в `infra/ci.env`.

Роли аккаунта Terraform перечислены в скрипте с причинами. Проверить, что выдано:

```bash
gcloud projects get-iam-policy <ID проекта> --flatten=bindings \
  --filter="bindings.members:serviceAccount:canvas-api-tf@<ID проекта>.iam.gserviceaccount.com" \
  --format='value(bindings.role, bindings.condition.title)'
```

Кто может его одолжить — два subject, `…:environment:prod` и `…:pull_request`:

```bash
gcloud iam service-accounts get-iam-policy canvas-api-tf@<ID проекта>.iam.gserviceaccount.com \
  --project=<ID проекта> --flatten='bindings[].members' --format='value(bindings.members)'
```

## 3. План вручную

Пайплайн строит план сам — на каждом PR, который трогает `infra/`, `deploy.yml`, actions или `justfile`, в summary прогона. Его можно построить и из контейнера, своими учётными данными:

```bash
gcloud auth application-default login --no-launch-browser
TF_VAR_billing_account=… just ci-tf-plan
```

- Если в пайплайне задан `TF_VAR_operator_email`, его нужно передать и здесь: без адреса план покажет удаление алерта и его канала.
- Весь вывод проходит через маскировку: ID платёжного аккаунта и адрес заменяются на `<billing account>` и `<operator email>`. ID бюджета содержит ID аккаунта, и без маскировки он попадал бы в строки `Refreshing state…`.
- План не берёт блокировку состояния, поэтому не мешает apply пайплайна. Apply ждёт чужую блокировку до 5 минут.

## Выкладка

Каждый мерж в `main` с зелёным CI выпускает подписанный релиз, и прогон «Release and deploy» тут же выкатывает его — если `main` всё ещё стоит на коммите релиза (R45, R58). Выкладка — `.github/workflows/deploy.yml`, задание за заданием:

1. **Which release** — какой релиз и выкатывать ли. Если `main` ушёл дальше, прогон зелёный с notice, а выкатывает новый релиз его собственный прогон.
2. **The cloud, as the release describes it** — окружение `prod`: `just ci-tf-apply` из коммита тега.
3. **The server of the release** — окружение `prod`:
   - сервер amd64 из релиза, подпись `SHA256SUMS` проверена (`just ci-release-fetch`);
   - вход аккаунтом деплоя, и ожидание, пока он даст токен (`ci-identity-ready`, до 8 минут): привязку нового аккаунта создаёт apply этой же выкладки, а IAM вводит её в силу обычно за 2 минуты, иногда за 7 и дольше (документация IAM, прочитана 2026-10-14);
   - образ вокруг него, проверка `/health` до push, push, ревизия по digest и трафик на неё (`ci-image-push`, `ci-deploy`);
   - smoke: `/health` по адресу сервиса называет релиз и коммит (`ci-smoke`).
4. **The SPA of the release** — окружение `prod-hosting`, свой аккаунт и то же ожидание его входа: архив SPA из релиза, `firebase deploy`, затем страница сайта сверяется с архивом (`ci-hosting-deploy`, `ci-hosting-check`).

Кнопка — Actions → Deploy → Run workflow на `main`: выкатывает релиз на вершине `main`, а если его нет — ничего. На другой ветке она падает в первом задании. Упавшую выкладку повторяют «Re-run failed jobs» в прогоне «Release and deploy»; повторная попытка сначала проверяет, что `main` всё ещё стоит на коммите релиза.

Первый apply создаёт:

- API проекта, Artifact Registry с автоочисткой;
- аккаунты сервиса, деплоя и выкладки SPA, свою узкую роль деплоя, две роли Hosting и привязки WIF по окружениям;
- Cloud Run `canvas-api` на начальном образе Google `hello` — он отвечает на любой путь, в том числе на `/health`, пока CD не выкатит свой образ;
- Firebase в проекте и сайт Hosting;
- бюджет и, если задан адрес, алерт;
- бакет журналов в регионе и перенаправление `_Default`, место хранения трасс.

Привязку, которая одалживает аккаунт, задания выкладки ждут сами. Роли, которые тот же apply выдаёт новому аккаунту, обычно успевают за это время; если какая-то не успела, шаг, которому она нужна, получает 403. Тогда, как и после отказа только что включённого API, выкладку повторяют «Re-run failed jobs». Первая выкладка этого развёртывания прошла со второй попытки, ещё без ожидания: аккаунт деплоя входил через 5 секунд после создания своей привязки и получил 403 (`docs/live/01-first-deploy.md`).

## Первая выкладка: чек-лист

Выкладка закончена, когда каждый пункт ниже ответил «да». Команды — из devcontainer на x86_64 или из Cloud Shell, с учётными данными человека, у которого есть права на проект. ID платёжного аккаунта берётся из переменной окружения и на экран не попадает.

1. **`/health` называет выкаченный релиз и коммит.** Рецепт сверяет, что короткий коммит в ответе — начало полного коммита тега:

   ```bash
   just ci-smoke https://canvas-api-796865465273.us-east4.run.app <версия> "$(git rev-parse '<версия>^{commit}')"
   ```

2. **Строка старта в журнале называет ту же версию и коммит.** Её пишет только сервер, выкаченный CD; начальный образ `hello` её не пишет:

   ```bash
   gcloud logging read 'resource.type="cloud_run_revision" AND jsonPayload.message="startup"' \
     --project=mathtrail-canvas-prod --bucket=default-us-east4 --location=us-east4 --view=_AllLogs \
     --freshness=1d --limit=5 \
     --format='value(timestamp,resource.labels.revision_name,jsonPayload.version,jsonPayload.commit)'
   ```

3. **Автоочистка Artifact Registry действует:** политики `keep-recent` (5 версий) и `delete-old` (30 дней), без пробного режима. Поля `cleanupPolicyDryRun` в ответе нет, пока оно не `true`:

   ```bash
   gcloud artifacts repositories describe canvas-api --location=us-east4 --project=mathtrail-canvas-prod \
     --format='yaml(cleanupPolicies,cleanupPolicyDryRun)'
   ```

4. **Бюджет есть и смотрит на этот проект:** `canvas-api spend`, 1 USD в месяц, `projects/<номер проекта>`, пороги 0.5, 0.9 и 1.0. На платёжном аккаунте могут быть бюджеты и других проектов — нужна строка с номером этого. Поле `name` в вывод не берётся: в нём ID платёжного аккаунта. Ошибка gcloud тоже называет аккаунт, поэтому её вывод маскируется:

   ```bash
   gcloud billing budgets list --billing-account="$TF_VAR_billing_account" \
     --billing-project=mathtrail-canvas-prod \
     --format='table(displayName,amount.specifiedAmount.units,amount.specifiedAmount.currencyCode,budgetFilter.projects.list(),thresholdRules.thresholdPercent.list())' \
     2>&1 | sed "s/${TF_VAR_billing_account}/<billing account>/g"
   ```

5. **Алерт нагрузки и его канал есть**, если задан `TF_VAR_operator_email`. Без адреса их нет, и это тоже ответ. Адрес канала не печатается: маска `fields` берёт только имя, вид и то, включён ли канал.

   ```bash
   gcloud monitoring policies list --project=mathtrail-canvas-prod \
     --format='value(displayName,enabled,notificationChannels.len())'
   curl -fsS -H @<(printf 'Authorization: Bearer %s\n' "$(gcloud auth print-access-token)") \
     "https://monitoring.googleapis.com/v3/projects/mathtrail-canvas-prod/notificationChannels?fields=notificationChannels(displayName,type,enabled)"
   ```

6. **SPA отдаёт страницу без кеша, и её скрипт называет релиз:**

   ```bash
   curl -fsS -D - -o /dev/null https://mathtrail-canvas-prod.web.app/ | grep -i '^cache-control'
   script=$(curl -fsS https://mathtrail-canvas-prod.web.app/ | grep -oE '/assets/index-[A-Za-z0-9_-]+\.js' | head -1)
   curl -fsS "https://mathtrail-canvas-prod.web.app${script}" | grep -c '<версия>'
   ```

7. **Конфигурация совпадает с облаком.** Выкладка, которой нечего менять, пишет `Apply complete! Resources: 0 added, 0 changed, 0 destroyed`. План вручную (выше) не печатает строки `Plan:`: меняться могут только выходы.

8. **Расход — в бесплатных лимитах:** `just usage 1d` — vCPU-секунды, ГиБ-секунды и запросы сервиса против бесплатного лимита месяца. Лимит — платёжного аккаунта, общий для всех его проектов, поэтому свободного остатка может быть меньше, чем показано.

## Откат и возврат

Откат — трафиком на прошлую ревизию и прошлым выпуском Hosting, а не `terraform apply` (spec 15.5). SPA и API откатываются вместе. Команды — из devcontainer на x86_64 или из Cloud Shell, с учётными данными человека, у которого есть права на проект (`gcloud auth login` и `gcloud auth application-default login`).

**Сервис.** Ревизии, новые сверху; закрепить трафик за прошлой и проверить, что отвечает её релиз:

```bash
gcloud run revisions list --service=canvas-api --region=us-east4 --project=mathtrail-canvas-prod
gcloud run services update-traffic canvas-api --region=us-east4 --project=mathtrail-canvas-prod \
  --to-revisions=<ревизия>=100
curl -fsS https://canvas-api-796865465273.us-east4.run.app/health
```

Пока трафик закреплён, новые ревизии его не получают (документация Cloud Run, прочитана 2026-10-14). Следующая выкладка возвращает его на новую ревизию сама (`ci-deploy`).

**SPA.** Выпуски сайта, новые сверху, с их версией, временем и видом: выкладка пишет выпуск `DEPLOY` с номером релиза в сообщении, а откат и возврат — `ROLLBACK` без сообщения, поэтому нужная версия ищется по строкам `DEPLOY`. `hosting:clone` берёт версию и снова делает её live:

```bash
curl -fsS -H @<(printf 'Authorization: Bearer %s\n' "$(gcloud auth print-access-token)") \
  -H "x-goog-user-project: mathtrail-canvas-prod" \
  "https://firebasehosting.googleapis.com/v1beta1/sites/mathtrail-canvas-prod/releases?pageSize=20" \
  | jq -r '.releases[] | "\((.version.name // "-") | split("/") | last)  \(.releaseTime)  \(.type)  \(.message // "")"'
GOOGLE_CLOUD_QUOTA_PROJECT=mathtrail-canvas-prod firebase hosting:clone \
  mathtrail-canvas-prod@<версия> mathtrail-canvas-prod:live --project mathtrail-canvas-prod
```

`firebase` берёт учётные данные `application-default`, а API Hosting с учётными данными человека требует проект квоты — его задаёт `GOOGLE_CLOUD_QUOTA_PROJECT` (документация Firebase, прочитана 2026-10-14).

**Возврат** — кнопкой выкладки: она выкатывает релиз на вершине `main` целиком, сервис и SPA, и возвращает трафик на новую ревизию. Вручную — то же в обратную сторону:

```bash
gcloud run services update-traffic canvas-api --region=us-east4 --project=mathtrail-canvas-prod --to-latest
GOOGLE_CLOUD_QUOTA_PROJECT=mathtrail-canvas-prod firebase hosting:clone \
  mathtrail-canvas-prod@<версия последнего релиза> mathtrail-canvas-prod:live --project mathtrail-canvas-prod
```

Откат и возврат проверены 2026-10-14 этими командами: на второй выкладке сервис и SPA вернулись к v0.1.3, а затем к v0.1.4, около полуминуты в каждую сторону (`docs/live/01-first-deploy.md`).

## Переменные

| Переменная | Откуда | По умолчанию | Что значит |
|---|---|---|---|
| `project_id`, `region` | `prod.auto.tfvars` | — | проект (он же проект Firebase) и регион всего контура (R54) |
| `github_repository`, `github_owner_id`, `workload_identity_pool_id` | `prod.auto.tfvars` | — | чьи workflow могут выкатывать и через какой пул |
| `github_oidc_subject_prefix` | `prod.auto.tfvars` | — | как GitHub начинает subject токенов репозитория: владелец и репозиторий с их числовыми ID; по нему и окружению аккаунты одалживаются (R58) |
| `budget_amount`, `budget_currency` | `prod.auto.tfvars` | — | месячный бюджет; валюта — платёжного аккаунта |
| `billing_account` | `TF_VAR_billing_account` | — | платёжный аккаунт; `sensitive` |
| `operator_email` | `TF_VAR_operator_email` | пусто | адрес алерта нагрузки; `sensitive`; пусто — алерта нет |
| `service_name` | — | `canvas-api` | имя сервиса и префикс его аккаунтов |
| `image` | — | `hello` по digest | начальный образ; дальше образом владеет CD |
| `max_instances`, `concurrency`, `cpu`, `memory`, `request_timeout` | — | 1, 40, `1`, `1Gi`, `30s` | размер сервиса (spec 15.3) |
| `load_alert_per_minute` | — | 300 | порог алерта нагрузки, запросов в минуту в среднем за 5 минут |
| `keep_images`, `image_max_age` | — | 5, `30d` | автоочистка образов |
| `log_retention_days` | — | 30 | срок хранения журналов (privacy 4, 8) |

## Журналы и трассы

Всё хранится в регионе развёртывания (R57):

- **Журналы** — в бакете `default-us-east4`, 30 дней. Бакет `_Default` проекта остаётся в `global` и больше не наполняется: место бакета не меняется, поэтому в новый бакет перенаправлен sink `_Default`.
- **Аудит изменений проекта** — бакет `_Required` — остаётся в `global`: ни его, ни его sink изменить нельзя.
- **Трассы** — в бакете `_Trace`, который платформа создаст в `us-east4` с первой трассой сервиса.

Журналы читаются из нового бакета:

```bash
gcloud logging read 'resource.type="cloud_run_revision"' --project=mathtrail-canvas-prod \
  --bucket=default-us-east4 --location=us-east4 --view=_AllLogs --freshness=1h --limit=20
```

В Logs Explorer для этого нужен «Refine scope» → «Log buckets» → `default-us-east4`.

## Что может стоить денег

Ожидается $0 в бесплатных лимитах. Бюджет $1 пишет на 50, 90 и 100 %, но трат не останавливает.

- **Бакет состояния** в `us-east4` — доли цента в месяц: регион вне бесплатных у Cloud Storage.
- **Cloud Run** — сверх бесплатных лимитов. Они считаются на платёжный аккаунт и делятся со всеми его проектами (spec 14.4). `us-east4` — Tier 1. Что сервис израсходовал против лимита месяца, показывает `just usage <окно>`.
- **Artifact Registry** — сверх 0,5 ГБ. Автоочистка держит 5 последних образов и удаляет старше 30 дней. Container Scanning API не включается: он стоит $0.26 за каждый образ.
- **Cloud Logging** — сверх 50 ГиБ в месяц на проект и при хранении дольше 30 дней.
- **Cloud Trace** — сверх 2,5 млн спанов в месяц.
- **Алерты Cloud Monitoring** — бесплатны по меньшей мере до 2027-09-01: плату начнут брать не раньше этой даты, около $0.35 в месяц за метрику в условии.
- **Firebase Hosting** — сверх бесплатной квоты Spark, которую тариф Blaze сохраняет.
- **Выкладка** — ничего сверх этого: GitHub Actions бесплатны для публичного репозитория, каждая выкладка добавляет один образ в реестр и одну ревизию. Ревизия без трафика при min 0 не стоит ничего.

## Когда меняется bootstrap

Признак: план на PR зелёный, а apply пайплайна получает 403. План на свежем проекте почти ничего не читает и прав аккаунта Terraform не проверяет. Apply же создаёт ресурс, на который у аккаунта ещё нет роли.

Порядок:

1. Роль добавляется в `infra/bootstrap.sh` тем же PR, что приносит ресурс, с объяснением в комментарии.
2. До мержа автор повторяет `just bootstrap` — повтор безопасен и выдаёт только новое.
3. Если мерж уже упал — повторить bootstrap, затем «Re-run failed jobs».

Убранную из списка роль скрипт не отзывает. Это делается руками: `gcloud projects remove-iam-policy-binding <ID проекта> --member=… --role=…`.

Особый случай — роль, которую конфигурация выдаёт на проект, как две роли выкладки SPA. Аккаунт Terraform может выдать только роли из условия своего `projectIamAdmin`, поэтому новая роль сначала попадает в выражение `hasOnly([...])` в bootstrap. Иначе apply получит 403 на её выдаче. Повтор заменяет прежнее условие новым, а не добавляет второе: IAM различает привязки по условию, и старое продолжало бы действовать.

Уже известные изменения:

- `roles/secretmanager.admin` — с первым секретом (эпик MVP);
- переименование `deploy.yml` — оно часть условия пула;
- переименование окружения `prod` — оно часть subject, по которому аккаунт Terraform одалживается.

## Удаление развёртывания

`terraform destroy` убирает сервис, реестр, аккаунты, роль, бюджет, алерт и бакет журналов. Firebase и сайт по умолчанию остаются в проекте: удалить их нельзя. Sink `_Default` продолжает указывать на удалённый бакет; вернуть его:

```bash
gcloud logging sinks update _Default \
  logging.googleapis.com/projects/<ID проекта>/locations/global/buckets/_Default --project=<ID проекта>
```

Проект, бакет состояния, пул и аккаунт Terraform созданы bootstrap и Terraform не принадлежат.

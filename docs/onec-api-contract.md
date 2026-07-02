# Контракт API 1С для MAX Bot

Все персональные endpoint должны получать `max_user_id` и `account_id` и проверять связку в 1С перед выдачей данных. Mock-ответы не являются реализацией production-интеграции.

| Go method | HTTP path | 1С handler | MAXBotИнтеграция | Читает 1С | Пишет 1С | Проверка `max_user_id/account_id` | TODO |
|---|---|---|---|---|---|---|---|
| `StartUser` | `POST /max/v1/users/start` | `UsersStartpost` | `ЗарегистрироватьПользователя` | `Справочник.MAXBotПользователи` | `Справочник.MAXBotПользователи` | нет ЛС | — |
| `SaveConsent` | `POST /max/v1/consents` | `Consentspost` | `ЗафиксироватьСогласие` | `MAXBotПользователи`, `MAXBotИсторияДанныхПользователей` | история согласия | нет ЛС | уточнить регламент хранения согласий |
| `StartAccountLink` | `POST /max/v1/account-link/start` | `AccountLinkStartpost` | `НачатьПривязкуЛицевогоСчета` | `Справочник.энргАбоненты` | `MAXBotИсторияДанныхПользователей` | по номеру ЛС | подключить штатную отправку SMS/e-mail |
| `ConfirmAccountLink` | `POST /max/v1/account-link/confirm` | `AccountLinkConfirmpost` | `ПодтвердитьПривязкуЛицевогоСчета`, `JSONЛицевойСчет` | `Справочник.энргАбоненты`, история кода | история привязки | подтверждает ЛС и пользователя | заменить TODO проверки попыток на объект с TTL/attempts, если в конфигурации есть регистр |
| `Accounts` | `GET /max/v1/accounts` | `Accountsget` | `ПолучитьЛицевыеСчетаПользователяJSON` | история привязок, `Справочник.энргАбоненты` | нет | по `max_user_id` | — |
| `Balance` | `GET /max/v1/accounts/{account_id}/balance` | `AccountBalanceget` | `GetBalance`, `ПолучитьПоследниеНачисленияТарифыУслуги` | `РегистрНакопления.энргВзаиморасчеты`, документы `энргНачисление*`, `энргУслугиТочекУчета`, `энргТарифы*` | нет | да | адаптировать имена измерений под рабочую ИБ |
| `Meters` | `GET /max/v1/accounts/{account_id}/meters` | `AccountMetersget` | `ПолучитьПриборыУчетаJSON`, `ПолучитьПриборыУчета`, `ПолучитьТочкиПередачиПоказаний` | основной источник услуг/тарифов: `РегистрНакопления.энргОбъемНачислений`; дополнительный источник приборов: `Справочник.энргПриборыУчетаАбонента`; последние показания: `РегистрСведений.энргПоказанияПриборовУчета` | нет | да | не возвращать все приборы без фильтра по ЛС; уточнить штатные правила связи точки учета/тарифа/прибора |
| `SendReading` | `POST /max/v1/accounts/{account_id}/meters/{meter_id}/readings` | `MeterReadingpost` | `СоздатьПоказаниеПрибора` | приборы текущего ЛС, последние показания | `Документ.энргРегистрацияПоказанийАбонента`, записанный без проведения | да | прямой записи в `РегистрСведений.энргПоказанияПриборовУчета` нет; проведение выполняет отдельный штатный процесс 1С |
| `CreateAppeal` | `POST /max/v1/accounts/{account_id}/appeals` | `AccountAppealpost` | `СоздатьОбращение` | ЛС/абонент, справочники обращений | штатный объект обращения | да | `NOT_IMPLEMENTED`: объект обращения/заявки надо сопоставить с бизнес-процессом |
| `Invoice` | `GET /max/v1/accounts/{account_id}/invoice` | TODO | TODO | `Документ.энргКвитанция` | нет | да | добавить handler в 1С |
| `PaymentLink` | `POST /max/v1/accounts/{account_id}/payment-link` | TODO | TODO | баланс/начисления | история операции/платежная ссылка | да | добавить handler и идемпотентность |
| `Outages` | `GET /max/v1/accounts/{account_id}/outages` | TODO | `GetOutages` | объект отключений не найден | нет | да | В конфигурации не найден объект отключений, endpoint возвращает пустой массив. |
| `CreateAppointment` | `POST /max/v1/accounts/{account_id}/appointments` | TODO | `CreateAppointment` | темы/расписание приема | объект записи на прием | да | `NOT_IMPLEMENTED`: объект записи на прием не найден |

## `GET /max/v1/accounts/{account_id}/meters`

Endpoint сохранен для обратной совместимости, но его смысл — получить **точки передачи показаний** по авторизованному лицевому счету, а не технические карточки всех приборов. 1С обязана проверить связку `max_user_id + account_id` и фильтровать данные по текущему ЛС/абоненту. Запрещено возвращать все записи `Справочник.энргПриборыУчетаАбонента` без фильтра по ЛС.

Основной источник расчетных строк — `РегистрНакопления.энргОбъемНачислений`. В метаданных регистра используются измерения `Абонент`, `Услуга`, `ТочкаУчета`, `ПриборУчета`, `Измеритель`, `Шкала`, `ТарифнаяЗона`, `ЗначениеТарифа`, `ПериодНачисления`, ресурсы `ОбъемУслуги` и `Сумма`, а также атрибуты `НачальныеПоказания`, `КонечныеПоказания`, `ДатаПоверки`, `СрокПоверкиИстек`. Дополнительный источник технического прибора — `Справочник.энргПриборыУчетаАбонента`, где найдены реквизиты `Код`, `КодАСКУЭ`, `КодТочкиУчета`, `Измеритель` и табличная часть `Измерители`.

Формат элемента массива:

```json
{
  "meter_id": "<GUID или код прибора/точки учета из 1С>",
  "display_name": "Тариф РЭК",
  "resource": "Электроэнергия",
  "service_id": "<GUID услуги>",
  "service_name": "Электроэнергия",
  "tariff_id": "<GUID тарифа>",
  "tariff_name": "Тариф РЭК",
  "unit": "кВт⋅ч",
  "serial_number": "<серийный номер, не показывается на основном экране>",
  "last_value": 10543.0,
  "last_reading_date": "2026-04-06",
  "verification_to": "",
  "can_submit": true,
  "reason": ""
}
```

`display_name` формирует 1С: публичное имя услуги, тариф для тарифных расчетных строк, комбинация услуги и тарифа при необходимости различения, либо ресурс как fallback. Поле не должно быть пустым и не должно содержать `MOCK` в реальной 1С. Если услуга/тариф есть в начислениях, но прибор не найден, строка может быть возвращена с `can_submit=false` и причиной; бот покажет ее, но не добавит в список доступных ID.

## `POST /max/v1/accounts/{account_id}/meters/{meter_id}/readings`

Прием показаний из MAX создает документ `Документ.энргРегистрацияПоказанийАбонента` и записывает его без проведения. Endpoint не пишет напрямую в `РегистрСведений.энргПоказанияПриборовУчета`; движения регистров при приеме показаний из MAX не формируются. Дальнейшее проведение документов регистрации показаний должно выполняться отдельным штатным процессом 1С.

Успешный ответ должен иметь `success=true`, `code=OK` и `data.posted=false`:

```json
{
  "success": true,
  "code": "OK",
  "message": "Показание зарегистрировано",
  "operation_id": "<operation_id>",
  "data": {
    "document_number": "<номер документа>",
    "document_date": "2026-05-06",
    "document_ref": "<GUID документа>",
    "posted": false,
    "meter_id": "<meter_id из запроса>",
    "value": 245.678,
    "status": "saved_unposted"
  }
}
```

`document_number` не должен быть пустым. `status=saved_unposted` означает, что документ сохранен в 1С и ожидает штатной обработки/проведения.

## Mock vs real 1С

`mock-onec-config.json` не является источником данных. Все значения внутри него нужны только для автономного smoke-теста Go-бота без 1С.

Для реальной 1С обязательны:

```dotenv
ONEC_BASE_URL=https://<server>/<base>/hs
ONEC_TOKEN=<token>
```

## Локальная contract inventory Go ↔ cf_billing

Статическая проверка выполняется тестом `TestOneCContractMatchesCfBilling` и не требует опубликованной 1С.

| Go method | HTTP method | Path | 1C HTTP handler | MAXBotИнтеграция function | Required body/query fields | Expected response data | Status |
|---|---:|---|---|---|---|---|---|
| StartUser | POST | `/max/v1/users/start` | `UsersStartPOST` | `ЗарегистрироватьПользователя` | body: `max_user_id`, `chat_id`, `source` | object + `operation_id` | OK |
| SaveConsent | POST | `/max/v1/consents` | `ConsentsPOST` | `ЗафиксироватьСогласие` | body: `max_user_id`, `consent_version`, `source` | object + `operation_id` | OK |
| StartAccountLink | POST | `/max/v1/account-link/start` | `AccountLinkStartPOST` | `НачатьПривязкуЛицевогоСчета` | body: `max_user_id`, `account_number`, `source` | object + `operation_id` | OK |
| ConfirmAccountLink | POST | `/max/v1/account-link/confirm` | `AccountLinkConfirmPOST` | `ПодтвердитьПривязкуЛицевогоСчета` | body: `max_user_id`, `account_number`, `code`, `source` | account | OK |
| Accounts | GET | `/max/v1/accounts?max_user_id={max_user_id}` | `AccountsGET` | `ПолучитьЛицевыеСчетаПользователяJSON` | query: `max_user_id` | accounts[] | OK |
| Balance | GET | `/max/v1/accounts/{account_id}/balance?max_user_id={max_user_id}` | `AccountBalanceGET` | `ПолучитьБалансJSON` | path: `account_id`; query: `max_user_id` | balance | OK |
| Meters | GET | `/max/v1/accounts/{account_id}/meters?max_user_id={max_user_id}` | `AccountMetersGET` | `ПолучитьПриборыУчетаJSON` | path: `account_id`; query: `max_user_id` | meters[] | OK |
| SendReading | POST | `/max/v1/accounts/{account_id}/meters/{meter_id}/readings` | `AccountMeterReadingsPOST` | `СоздатьПоказаниеПрибора` | body: `max_user_id`, `period`, `value`, `message_id`, `operation_id`, `source` | reading result | OK |
| CreateAppeal | POST | `/max/v1/accounts/{account_id}/appeals` | `AccountAppealsPOST` | `СоздатьОбращение` | body: `max_user_id`, `text`, `message_id`, `operation_id`, `source` | appeal result | OK |
| Help | GET | `/max/v1/reference/help` | `ReferenceHelpGET` | `ПолучитьСправкуJSON` | none | help text | OK |
| Organization | GET | `/max/v1/reference/organization` | `ReferenceOrganizationGET` | `ПолучитьОрганизациюJSON` | none | organization | OK |
| Emergency | GET | `/max/v1/reference/emergency` | `ReferenceEmergencyGET` | `ПолучитьАварийнуюСлужбуJSON` | none | emergency contacts | OK |
| Invoice | GET | `/max/v1/accounts/{account_id}/invoice?period=YYYY-MM&max_user_id={max_user_id}` | `AccountInvoiceGET` | `ПолучитьКвитанциюJSON` | path: `account_id`; query: `period`, `max_user_id` | invoice | OK |
| PaymentLink | POST | `/max/v1/accounts/{account_id}/payment-link` | `AccountPaymentLinkPOST` | `СоздатьСсылкуОплаты` | body: `max_user_id`, `operation_id`, `source` | payment link | OK |
| Outages | GET | `/max/v1/accounts/{account_id}/outages?max_user_id={max_user_id}` | `AccountOutagesGET` | `ПолучитьОтключенияJSON` | path: `account_id`; query: `max_user_id` | outages[] | OK |
| AppointmentTopics | GET | `/max/v1/reference/appointment-topics` | `ReferenceAppointmentTopicsGET` | `ПолучитьТемыЗаписиJSON` | none | topics[] | OK |
| CreateAppointment | POST | `/max/v1/accounts/{account_id}/appointments` | `AccountAppointmentsPOST` | `СоздатьЗаписьНаПрием` | body: `max_user_id`, `topic_id`, `operation_id`, `source` | appointment | OK |

`UNUSED_BY_BOT` endpoints in the checked `MAXBotHTTP` service were not found during this inventory.

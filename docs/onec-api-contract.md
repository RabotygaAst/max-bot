# Контракт Go-бота и HTTP API 1С MAX

Все методы 1С публикуются в HTTP-сервисе `MAXBotHTTP` с root URL `max_bot`; при публикации базы итоговый base URL для Go должен указывать на каталог HTTP-сервисов, например `https://<server>/<base>/hs`. Каждый обработчик проверяет `Authorization: Bearer <ONEC_TOKEN>` через `MAXBotИнтеграция.ПроверитьАвторизацию` и возвращает единый `APIResponse`.

| Go method | HTTP method | Path | 1С HTTP service handler | `MAXBotИнтеграция` function | Проверяет `max_user_id/account_id` | Какие объекты 1С читает/пишет |
|---|---:|---|---|---|---|---|
| `StartUser()` | POST | `/max/v1/users/start` | `MAXBotHTTP.UsersStartpost` | `StartUser`, `ЗарегистрироватьПользователя` | Нет, регистрация пользователя | Справочник `MAXBotПользователи` |
| `SaveConsent()` | POST | `/max/v1/consents` | `MAXBotHTTP.Consentspost` | `SaveConsent`, `ЗафиксироватьСогласие` | Нет | Регистр сведений `MAXBotСогласияПДн` |
| `StartAccountLink()` | POST | `/max/v1/account-link/start` | `MAXBotHTTP.AccountLinkStartpost` | `StartAccountLink`, `НачатьПривязкуЛицевогоСчета`, `НайтиЛицевойСчетПоНомеру` | Проверяет наличие ЛС по номеру | Справочник `энргАбоненты`, регистр `MAXBotКодыПривязкиЛС` |
| `ConfirmAccountLink()` | POST | `/max/v1/account-link/confirm` | `MAXBotHTTP.AccountLinkConfirmpost` | `ConfirmAccountLink`, `ПодтвердитьПривязкуЛицевогоСчета` | Создает связку `max_user_id → ЛС` | `MAXBotПользователи`, `MAXBotПривязкиЛицевыхСчетов`, `энргАбоненты` |
| `Accounts()` | GET | `/max/v1/accounts?max_user_id={max_user_id}` | `MAXBotHTTP.Accountsget` | `GetAccounts`, `ПолучитьЛицевыеСчетаПользователяJSON` | Да, фильтр по пользователю | `MAXBotПривязкиЛицевыхСчетов`, `энргАбоненты` |
| `Balance()` | GET | `/max/v1/accounts/{account_id}/balance?max_user_id={max_user_id}` | `MAXBotHTTP.AccountBalanceget` | `GetBalance`, `ПолучитьАвторизованныйЛицевойСчет`, `ПолучитьПоследниеНачисленияТарифыУслуги` | Да | `MAXBotПривязкиЛицевыхСчетов`, `энргАбоненты`, `энргВзаиморасчеты` |
| `Meters()` | GET | `/max/v1/accounts/{account_id}/meters?max_user_id={max_user_id}` | `MAXBotHTTP.AccountMetersget` | `GetMeters`, `ПолучитьПриборыУчета` | Да | `MAXBotПривязкиЛицевыхСчетов`, `энргПриборыУчетаАбонента` |
| `SendReading()` | POST | `/max/v1/accounts/{account_id}/meters/{meter_id}/readings` | `MAXBotHTTP.MeterReadingpost` | `SendReading`, `СоздатьПоказаниеПрибора`, `ПроверитьПоказание` | Да, `max_user_id` берется из body | `MAXBotПривязкиЛицевыхСчетов`; TODO: подключить штатный документ регистрации показаний при уточнении реквизитов |
| `CreateAppeal()` | POST | `/max/v1/accounts/{account_id}/appeals` | `MAXBotHTTP.AccountAppealpost` | `CreateAppeal`, `СоздатьОбращение` | Да | `MAXBotПривязкиЛицевыхСчетов`; TODO: подключить штатный объект обращений (`бестОбращения`) при уточнении реквизитов |
| `Help()` | GET | `/max/v1/reference/help` | `MAXBotHTTP.ReferenceHelpget` | `GetHelp` | Нет | Не читает персональные данные |
| `Organization()` | GET | `/max/v1/reference/organization` | `MAXBotHTTP.ReferenceOrganizationget` | `GetOrganization` | Нет | Справочник `Организации` |
| `Emergency()` | GET | `/max/v1/reference/emergency` | `MAXBotHTTP.ReferenceEmergencyget` | `GetEmergency` | Нет | TODO: вынести контакты в константы/регистр настроек |
| `Invoice()` | GET | `/max/v1/accounts/{account_id}/invoice?period=YYYY-MM&max_user_id={max_user_id}` | `MAXBotHTTP.AccountInvoiceget` | `GetInvoice` | Да | `MAXBotПривязкиЛицевыхСчетов`, `энргАбоненты`, `энргВзаиморасчеты`; TODO: подключить `бестЕдиныйПлатежныйДокумент`/печатную форму |
| `PaymentLink()` | POST | `/max/v1/accounts/{account_id}/payment-link` | `MAXBotHTTP.PaymentLinkpost` | `CreatePaymentLink` | Да, `max_user_id` берется из body | `MAXBotПривязкиЛицевыхСчетов`, `энргВзаиморасчеты`; TODO: подключить платежного провайдера |
| `Outages()` | GET | `/max/v1/accounts/{account_id}/outages?max_user_id={max_user_id}` | `MAXBotHTTP.AccountOutagesget` | `GetOutages` | Да | `MAXBotПривязкиЛицевыхСчетов`; TODO: подключить регистр отключений при наличии в конфигурации |
| `AppointmentTopics()` | GET | `/max/v1/reference/appointment-topics` | `MAXBotHTTP.AppointmentTopicsget` | `GetAppointmentTopics` | Нет | Встроенный справочник тем API; TODO: заменить на справочник тем, если будет добавлен |
| `CreateAppointment()` | POST | `/max/v1/accounts/{account_id}/appointments` | `MAXBotHTTP.AccountAppointmentspost` | `CreateAppointment` | Да | `MAXBotПривязкиЛицевыхСчетов`; TODO: подключить документ/регистр записи на прием при уточнении объекта |

## Проверка доступа

Персональные функции сначала вызывают `ПолучитьАвторизованныйЛицевойСчет(MaxUserID, AccountID)`. Helper ищет активную запись в `MAXBotПривязкиЛицевыхСчетов`, поддерживает стабильный GUID `account_id` и временную совместимость с `ACC-<номер ЛС>`, а при несоответствии выбрасывает `AUTH_REQUIRED` или `ACCESS_DENIED`.

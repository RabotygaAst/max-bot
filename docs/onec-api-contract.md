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
| `Meters` | `GET /max/v1/accounts/{account_id}/meters` | `AccountMetersget` | `ПолучитьПриборыУчетаJSON`, `ПолучитьПриборыУчета` | `Справочник.энргПриборыУчетаАбонента`, регистры показаний/услуг | нет | да | уточнить реквизит связи прибора с ЛС/объектом учета |
| `SendReading` | `POST /max/v1/accounts/{account_id}/meters/{meter_id}/readings` | `MeterReadingpost` | `СоздатьПоказаниеПрибора` | приборы текущего ЛС, последние показания | штатный объект показаний | да | `NOT_IMPLEMENTED`: штатный объект записи показаний не подтвержден в расширении |
| `CreateAppeal` | `POST /max/v1/accounts/{account_id}/appeals` | `AccountAppealpost` | `СоздатьОбращение` | ЛС/абонент, справочники обращений | штатный объект обращения | да | `NOT_IMPLEMENTED`: объект обращения/заявки надо сопоставить с бизнес-процессом |
| `Invoice` | `GET /max/v1/accounts/{account_id}/invoice` | TODO | TODO | `Документ.энргКвитанция` | нет | да | добавить handler в 1С |
| `PaymentLink` | `POST /max/v1/accounts/{account_id}/payment-link` | TODO | TODO | баланс/начисления | история операции/платежная ссылка | да | добавить handler и идемпотентность |
| `Outages` | `GET /max/v1/accounts/{account_id}/outages` | TODO | `GetOutages` | объект отключений не найден | нет | да | В конфигурации не найден объект отключений, endpoint возвращает пустой массив. |
| `CreateAppointment` | `POST /max/v1/accounts/{account_id}/appointments` | TODO | `CreateAppointment` | темы/расписание приема | объект записи на прием | да | `NOT_IMPLEMENTED`: объект записи на прием не найден |

## Mock vs real 1С

`mock-onec-config.json` не является источником данных. Все значения внутри него нужны только для автономного smoke-теста Go-бота без 1С.

Для реальной 1С обязательны:

```dotenv
ONEC_BASE_URL=https://<server>/<base>/hs
ONEC_TOKEN=<token>
```

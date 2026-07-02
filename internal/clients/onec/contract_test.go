package onec

import (
	"os"
	"strings"
	"testing"
)

type contractEndpoint struct {
	GoMethod    string
	HTTPMethod  string
	Path        string
	Handler     string
	Integration string
	Required    string
	Response    string
	Status      string
}

var oneCContractInventory = []contractEndpoint{
	{"StartUser", "POST", "/max/v1/users/start", "UsersStartPOST", "ЗарегистрироватьПользователя", "body: max_user_id, chat_id, source", "operation_id + object", "OK"},
	{"SaveConsent", "POST", "/max/v1/consents", "ConsentsPOST", "ЗафиксироватьСогласие", "body: max_user_id, consent_version, source", "operation_id + object", "OK"},
	{"StartAccountLink", "POST", "/max/v1/account-link/start", "AccountLinkStartPOST", "НачатьПривязкуЛицевогоСчета", "body: max_user_id, account_number, source", "operation_id", "OK"},
	{"ConfirmAccountLink", "POST", "/max/v1/account-link/confirm", "AccountLinkConfirmPOST", "ПодтвердитьПривязкуЛицевогоСчета", "body: max_user_id, account_number, code, source", "account", "OK"},
	{"Accounts", "GET", "/max/v1/accounts", "AccountsGET", "ПолучитьЛицевыеСчетаПользователяJSON", "query: max_user_id", "accounts[]", "OK"},
	{"Balance", "GET", "/max/v1/accounts/{account_id}/balance", "AccountBalanceGET", "ПолучитьБалансJSON", "path: account_id; query: max_user_id", "balance", "OK"},
	{"Meters", "GET", "/max/v1/accounts/{account_id}/meters", "AccountMetersGET", "ПолучитьПриборыУчетаJSON", "path: account_id; query: max_user_id", "meters[]", "OK"},
	{"SendReading", "POST", "/max/v1/accounts/{account_id}/meters/{meter_id}/readings", "AccountMeterReadingsPOST", "СоздатьПоказаниеПрибора", "path: account_id,meter_id; body: max_user_id, period, value, message_id, operation_id, source", "reading result", "OK"},
	{"CreateAppeal", "POST", "/max/v1/accounts/{account_id}/appeals", "AccountAppealsPOST", "СоздатьОбращение", "path: account_id; body: max_user_id, text, message_id, operation_id, source", "appeal result", "OK"},
	{"Help", "GET", "/max/v1/reference/help", "ReferenceHelpGET", "ПолучитьСправкуJSON", "none", "help", "OK"},
	{"Organization", "GET", "/max/v1/reference/organization", "ReferenceOrganizationGET", "ПолучитьОрганизациюJSON", "none", "organization", "OK"},
	{"Emergency", "GET", "/max/v1/reference/emergency", "ReferenceEmergencyGET", "ПолучитьАварийнуюСлужбуJSON", "none", "emergency", "OK"},
	{"Invoice", "GET", "/max/v1/accounts/{account_id}/invoice", "AccountInvoiceGET", "ПолучитьКвитанциюJSON", "path: account_id; query: period, max_user_id", "invoice", "OK"},
	{"PaymentLink", "POST", "/max/v1/accounts/{account_id}/payment-link", "AccountPaymentLinkPOST", "СоздатьСсылкуОплаты", "path: account_id; body: max_user_id, operation_id, source", "payment link", "OK"},
	{"Outages", "GET", "/max/v1/accounts/{account_id}/outages", "AccountOutagesGET", "ПолучитьОтключенияJSON", "path: account_id; query: max_user_id", "outages[]", "OK"},
	{"AppointmentTopics", "GET", "/max/v1/reference/appointment-topics", "ReferenceAppointmentTopicsGET", "ПолучитьТемыЗаписиJSON", "none", "topics[]", "OK"},
	{"CreateAppointment", "POST", "/max/v1/accounts/{account_id}/appointments", "AccountAppointmentsPOST", "СоздатьЗаписьНаПрием", "path: account_id; body: max_user_id, topic_id, operation_id, source", "appointment", "OK"},
}

func TestOneCContractMatchesCfBilling(t *testing.T) {
	client, err := os.ReadFile("client.go")
	if err != nil {
		t.Fatal(err)
	}
	xml, err := os.ReadFile("../../../billing_cf/HTTPServices/MAXBotHTTP.xml")
	if err != nil {
		t.Fatal(err)
	}
	httpModule, err := os.ReadFile("../../../billing_cf/HTTPServices/MAXBotHTTP/Ext/Module.bsl")
	if err != nil {
		t.Fatal(err)
	}
	integration, err := os.ReadFile("../../../billing_cf/CommonModules/MAXBotИнтеграция/Ext/Module.bsl")
	if err != nil {
		t.Fatal(err)
	}
	clientText, xmlText, httpText, integrationText := string(client), string(xml), string(httpModule), string(integration)
	missing := []string{}
	for _, e := range oneCContractInventory {
		if !strings.Contains(clientText, "func (c *Client) "+e.GoMethod+"(") {
			t.Fatalf("Go client method %s missing", e.GoMethod)
		}
		inXML := strings.Contains(xmlText, "<Template>"+e.Path+"</Template>")
		inHTTP := strings.Contains(httpText, e.Integration)
		inHandler := strings.Contains(httpText, "Функция "+e.Handler+"(")
		inIntegration := strings.Contains(integrationText, "Функция "+e.Integration) || strings.Contains(integrationText, "Процедура "+e.Integration)
		if e.Status == "MISSING_IN_1C_CONFIG" {
			if inXML && inHTTP && inIntegration {
				t.Fatalf("%s marked missing but is present in cf_billing", e.Path)
			}
			missing = append(missing, e.HTTPMethod+" "+e.Path)
			continue
		}
		if !inXML {
			t.Fatalf("endpoint missing in MAXBotHTTP.xml: %s", e.Path)
		}
		if !inHandler {
			t.Fatalf("handler function missing in HTTP module for %s: %s", e.Path, e.Handler)
		}
		if !inHTTP {
			t.Fatalf("handler for %s does not call MAXBotИнтеграция.%s", e.Path, e.Integration)
		}
		if !inIntegration {
			t.Fatalf("MAXBotИнтеграция export missing: %s", e.Integration)
		}
	}
	if !strings.Contains(httpText, "MAXUserID") {
		t.Fatal("HTTP module should use MAXUserID for personal methods")
	}
	if !strings.Contains(httpText, "СоздатьПоказаниеПрибора") {
		t.Fatal("readings handler should call СоздатьПоказаниеПрибора")
	}
	if !strings.Contains(httpText, "ПолучитьПриборыУчетаJSON") && !strings.Contains(integrationText, "ПолучитьТочкиПередачиПоказаний") {
		t.Fatal("meters contract should call meters integration")
	}
	t.Logf("MISSING_IN_1C_CONFIG: %s", strings.Join(missing, ", "))
}

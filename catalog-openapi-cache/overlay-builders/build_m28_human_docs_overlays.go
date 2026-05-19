//go:build ignore

package main

import (
	"encoding/json"
	"os"
	"sort"
)

type overlaySpec struct {
	ProviderID  string
	Title       string
	Description string
	ServerURL   string
	Sources     []string
	SourceNote  string
	Security    map[string]map[string]any
	Schemas     []string
	Paths       map[string]map[string]any
	OutputPath  string
}

func main() {
	for _, spec := range []overlaySpec{
		magentoOverlay(),
		gumroadOverlay(),
		invoiceNinjaOverlay(),
		erpnextOverlay(),
		woocommerceOverlay(),
		wiseOverlay(),
		dhlOverlay(),
		onfleetOverlay(),
		unleashedOverlay(),
		workableOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func magentoOverlay() overlaySpec {
	security := map[string]map[string]any{
		"magentoBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Adobe Commerce admin, customer, or integration token", "description": "Adobe Commerce REST API token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "magento",
		Title:       "Magento Adobe Commerce REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Adobe Commerce REST API human documentation. This is not an official Adobe OpenAPI document.",
		ServerURL:   "https://{commerce_host}/rest/{store_code}",
		Sources:     []string{"https://developer.adobe.com/commerce/webapi/rest/reference/", "https://developer.adobe.com/commerce/webapi/", "https://developer.adobe.com/commerce/webapi/rest/quick-reference/generate-local/", "https://developer.adobe.com/commerce/webapi/get-started/authentication/"},
		SourceNote:  "Adobe Commerce publishes human REST documentation and instance-local Swagger generation guidance, but no recorded stable public official OpenAPI document; this overlay covers selected catalog, customer, order, invoice, and shipment endpoints.",
		Security:    security,
		Schemas:     []string{"MagentoObject", "MagentoCollection", "MagentoError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/magento-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/V1/categories":                 {"get": op("getMagentoCategories", "Get category tree", nil, "", "#/components/schemas/MagentoObject", "magentoBearer")},
			"/V1/products":                   {"get": op("listMagentoProducts", "Search products", params(query("searchCriteria", "Adobe Commerce search criteria query parameters.")), "", "#/components/schemas/MagentoCollection", "magentoBearer"), "post": op("createMagentoProduct", "Create product", nil, "#/components/schemas/MagentoObject", "#/components/schemas/MagentoObject", "magentoBearer")},
			"/V1/products/{sku}":             {"get": op("getMagentoProduct", "Get product", params(path("sku", "Product SKU.")), "", "#/components/schemas/MagentoObject", "magentoBearer"), "put": op("updateMagentoProduct", "Update product", params(path("sku", "Product SKU.")), "#/components/schemas/MagentoObject", "#/components/schemas/MagentoObject", "magentoBearer")},
			"/V1/customers/search":           {"get": op("searchMagentoCustomers", "Search customers", params(query("searchCriteria", "Adobe Commerce search criteria query parameters.")), "", "#/components/schemas/MagentoCollection", "magentoBearer")},
			"/V1/orders":                     {"get": op("listMagentoOrders", "Search orders", params(query("searchCriteria", "Adobe Commerce search criteria query parameters.")), "", "#/components/schemas/MagentoCollection", "magentoBearer")},
			"/V1/invoices":                   {"get": op("listMagentoInvoices", "Search invoices", params(query("searchCriteria", "Adobe Commerce search criteria query parameters.")), "", "#/components/schemas/MagentoCollection", "magentoBearer")},
			"/V1/shipment":                   {"get": op("listMagentoShipments", "Search shipments", params(query("searchCriteria", "Adobe Commerce search criteria query parameters.")), "", "#/components/schemas/MagentoCollection", "magentoBearer")},
			"/V1/integration/admin/token":    {"post": op("createMagentoAdminToken", "Create admin access token", nil, "#/components/schemas/MagentoObject", "#/components/schemas/MagentoObject")},
			"/V1/integration/customer/token": {"post": op("createMagentoCustomerToken", "Create customer access token", nil, "#/components/schemas/MagentoObject", "#/components/schemas/MagentoObject")},
		},
	}
}

func gumroadOverlay() overlaySpec {
	security := map[string]map[string]any{
		"gumroadBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Gumroad access token", "description": "Gumroad OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "gumroad",
		Title:       "Gumroad API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Gumroad API human documentation. This is not an official Gumroad OpenAPI document.",
		ServerURL:   "https://api.gumroad.com",
		Sources:     []string{"https://gumroad.com/api", "https://gumroad.com/oauth/applications", "https://help.gumroad.com/article/280-create-application-api"},
		SourceNote:  "Gumroad publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected products, sales, offers, subscribers, licenses, and resource subscriptions.",
		Security:    security,
		Schemas:     []string{"GumroadObject", "GumroadCollection", "GumroadError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/gumroad-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v2/products":                        {"get": op("listGumroadProducts", "List products", nil, "", "#/components/schemas/GumroadCollection", "gumroadBearer")},
			"/v2/products/{product_id}":           {"get": op("getGumroadProduct", "Get product", params(path("product_id", "Gumroad product ID.")), "", "#/components/schemas/GumroadObject", "gumroadBearer")},
			"/v2/sales":                           {"get": op("listGumroadSales", "List sales", params(query("after", "Return sales after this date."), query("before", "Return sales before this date."), query("product_id", "Filter by product ID.")), "", "#/components/schemas/GumroadCollection", "gumroadBearer")},
			"/v2/sales/{sale_id}":                 {"get": op("getGumroadSale", "Get sale", params(path("sale_id", "Gumroad sale ID.")), "", "#/components/schemas/GumroadObject", "gumroadBearer")},
			"/v2/offers":                          {"get": op("listGumroadOffers", "List offers", nil, "", "#/components/schemas/GumroadCollection", "gumroadBearer"), "post": op("createGumroadOffer", "Create offer", nil, "#/components/schemas/GumroadObject", "#/components/schemas/GumroadObject", "gumroadBearer")},
			"/v2/licenses/verify":                 {"post": op("verifyGumroadLicense", "Verify license", nil, "#/components/schemas/GumroadObject", "#/components/schemas/GumroadObject", "gumroadBearer")},
			"/v2/resource_subscriptions":          {"get": op("listGumroadResourceSubscriptions", "List resource subscriptions", nil, "", "#/components/schemas/GumroadCollection", "gumroadBearer"), "post": op("createGumroadResourceSubscription", "Create resource subscription", nil, "#/components/schemas/GumroadObject", "#/components/schemas/GumroadObject", "gumroadBearer")},
			"/v2/resource_subscriptions/{sub_id}": {"delete": op("deleteGumroadResourceSubscription", "Delete resource subscription", params(path("sub_id", "Resource subscription ID.")), "", "", "gumroadBearer")},
		},
	}
}

func invoiceNinjaOverlay() overlaySpec {
	security := map[string]map[string]any{
		"invoiceNinjaAPIToken": {"type": "apiKey", "in": "header", "name": "X-API-TOKEN", "description": "Invoice Ninja v5 API token carried in the X-API-TOKEN header."},
	}
	return overlaySpec{
		ProviderID:  "invoice-ninja",
		Title:       "Invoice Ninja API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Invoice Ninja API human documentation. This is not an official Invoice Ninja OpenAPI document.",
		ServerURL:   "https://{invoice_ninja_host}",
		Sources:     []string{"https://api-docs.invoicing.co/", "https://invoiceninja.github.io/docs/developer-guide", "https://invoiceninja.github.io/docs/api-reference/invoice-ninja-api-reference"},
		SourceNote:  "Invoice Ninja publishes OpenAPI-rendered human API documentation but no recorded stable standalone downloadable OpenAPI document; this overlay covers selected clients, invoices, quotes, payments, products, projects, vendors, and webhooks.",
		Security:    security,
		Schemas:     []string{"InvoiceNinjaObject", "InvoiceNinjaCollection", "InvoiceNinjaError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/invoice-ninja-api-overlay.json",
		Paths: map[string]map[string]any{
			"/api/v1/clients":           {"get": op("listInvoiceNinjaClients", "List clients", params(query("per_page", "Page size.")), "", "#/components/schemas/InvoiceNinjaCollection", "invoiceNinjaAPIToken"), "post": op("createInvoiceNinjaClient", "Create client", nil, "#/components/schemas/InvoiceNinjaObject", "#/components/schemas/InvoiceNinjaObject", "invoiceNinjaAPIToken")},
			"/api/v1/clients/{id}":      {"get": op("getInvoiceNinjaClient", "Get client", params(path("id", "Client ID.")), "", "#/components/schemas/InvoiceNinjaObject", "invoiceNinjaAPIToken"), "put": op("updateInvoiceNinjaClient", "Update client", params(path("id", "Client ID.")), "#/components/schemas/InvoiceNinjaObject", "#/components/schemas/InvoiceNinjaObject", "invoiceNinjaAPIToken")},
			"/api/v1/invoices":          {"get": op("listInvoiceNinjaInvoices", "List invoices", params(query("per_page", "Page size.")), "", "#/components/schemas/InvoiceNinjaCollection", "invoiceNinjaAPIToken"), "post": op("createInvoiceNinjaInvoice", "Create invoice", nil, "#/components/schemas/InvoiceNinjaObject", "#/components/schemas/InvoiceNinjaObject", "invoiceNinjaAPIToken")},
			"/api/v1/quotes":            {"get": op("listInvoiceNinjaQuotes", "List quotes", params(query("per_page", "Page size.")), "", "#/components/schemas/InvoiceNinjaCollection", "invoiceNinjaAPIToken"), "post": op("createInvoiceNinjaQuote", "Create quote", nil, "#/components/schemas/InvoiceNinjaObject", "#/components/schemas/InvoiceNinjaObject", "invoiceNinjaAPIToken")},
			"/api/v1/payments":          {"get": op("listInvoiceNinjaPayments", "List payments", params(query("per_page", "Page size.")), "", "#/components/schemas/InvoiceNinjaCollection", "invoiceNinjaAPIToken"), "post": op("createInvoiceNinjaPayment", "Create payment", nil, "#/components/schemas/InvoiceNinjaObject", "#/components/schemas/InvoiceNinjaObject", "invoiceNinjaAPIToken")},
			"/api/v1/products":          {"get": op("listInvoiceNinjaProducts", "List products", params(query("per_page", "Page size.")), "", "#/components/schemas/InvoiceNinjaCollection", "invoiceNinjaAPIToken"), "post": op("createInvoiceNinjaProduct", "Create product", nil, "#/components/schemas/InvoiceNinjaObject", "#/components/schemas/InvoiceNinjaObject", "invoiceNinjaAPIToken")},
			"/api/v1/vendors":           {"get": op("listInvoiceNinjaVendors", "List vendors", params(query("per_page", "Page size.")), "", "#/components/schemas/InvoiceNinjaCollection", "invoiceNinjaAPIToken")},
			"/api/v1/projects":          {"get": op("listInvoiceNinjaProjects", "List projects", params(query("per_page", "Page size.")), "", "#/components/schemas/InvoiceNinjaCollection", "invoiceNinjaAPIToken")},
			"/api/v1/webhook_endpoints": {"get": op("listInvoiceNinjaWebhooks", "List webhook endpoints", params(query("per_page", "Page size.")), "", "#/components/schemas/InvoiceNinjaCollection", "invoiceNinjaAPIToken"), "post": op("createInvoiceNinjaWebhook", "Create webhook endpoint", nil, "#/components/schemas/InvoiceNinjaObject", "#/components/schemas/InvoiceNinjaObject", "invoiceNinjaAPIToken")},
		},
	}
}

func erpnextOverlay() overlaySpec {
	security := map[string]map[string]any{
		"frappeToken": {"type": "apiKey", "in": "header", "name": "Authorization", "description": "Frappe/ERPNext API key and secret carried in the Authorization header using the token scheme."},
	}
	return overlaySpec{
		ProviderID:  "erpnext",
		Title:       "ERPNext Frappe REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official ERPNext and Frappe REST API human documentation. This is not an official ERPNext OpenAPI document.",
		ServerURL:   "https://{site}",
		Sources:     []string{"https://docs.frappe.io/erpnext/rest-api", "https://docs.frappe.io/framework/user/en/api/rest", "https://docs.frappe.io/framework/user/en/api/rest#1-token-based-authentication"},
		SourceNote:  "ERPNext uses Frappe REST API human documentation but no recorded stable public official OpenAPI document; this overlay covers selected generic resource and common ERP DocType endpoints.",
		Security:    security,
		Schemas:     []string{"ERPNextObject", "ERPNextCollection", "ERPNextError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/erpnext-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/api/resource/{doctype}":        {"get": op("listERPNextResources", "List DocType records", params(path("doctype", "ERPNext DocType name."), query("fields", "JSON field list."), query("filters", "JSON filters.")), "", "#/components/schemas/ERPNextCollection", "frappeToken"), "post": op("createERPNextResource", "Create DocType record", params(path("doctype", "ERPNext DocType name.")), "#/components/schemas/ERPNextObject", "#/components/schemas/ERPNextObject", "frappeToken")},
			"/api/resource/{doctype}/{name}": {"get": op("getERPNextResource", "Get DocType record", params(path("doctype", "ERPNext DocType name."), path("name", "Record name.")), "", "#/components/schemas/ERPNextObject", "frappeToken"), "put": op("updateERPNextResource", "Update DocType record", params(path("doctype", "ERPNext DocType name."), path("name", "Record name.")), "#/components/schemas/ERPNextObject", "#/components/schemas/ERPNextObject", "frappeToken"), "delete": op("deleteERPNextResource", "Delete DocType record", params(path("doctype", "ERPNext DocType name."), path("name", "Record name.")), "", "", "frappeToken")},
			"/api/resource/Item":             {"get": op("listERPNextItems", "List items", nil, "", "#/components/schemas/ERPNextCollection", "frappeToken")},
			"/api/resource/Customer":         {"get": op("listERPNextCustomers", "List customers", nil, "", "#/components/schemas/ERPNextCollection", "frappeToken")},
			"/api/resource/Supplier":         {"get": op("listERPNextSuppliers", "List suppliers", nil, "", "#/components/schemas/ERPNextCollection", "frappeToken")},
			"/api/resource/Sales Order":      {"get": op("listERPNextSalesOrders", "List sales orders", nil, "", "#/components/schemas/ERPNextCollection", "frappeToken")},
			"/api/resource/Sales Invoice":    {"get": op("listERPNextSalesInvoices", "List sales invoices", nil, "", "#/components/schemas/ERPNextCollection", "frappeToken")},
			"/api/method/{method}":           {"post": op("callERPNextMethod", "Call whitelisted method", params(path("method", "Frappe whitelisted method path.")), "#/components/schemas/ERPNextObject", "#/components/schemas/ERPNextObject", "frappeToken")},
		},
	}
}

func woocommerceOverlay() overlaySpec {
	security := map[string]map[string]any{
		"wooCommerceBasic":          {"type": "http", "scheme": "basic", "description": "WooCommerce REST API consumer key and consumer secret carried using HTTP Basic authentication over HTTPS."},
		"wooCommerceConsumerKey":    {"type": "apiKey", "in": "query", "name": "consumer_key", "description": "WooCommerce REST API consumer key query parameter for environments where Basic auth is unavailable."},
		"wooCommerceConsumerSecret": {"type": "apiKey", "in": "query", "name": "consumer_secret", "description": "WooCommerce REST API consumer secret query parameter for environments where Basic auth is unavailable."},
	}
	return overlaySpec{
		ProviderID:  "woocommerce",
		Title:       "WooCommerce REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official WooCommerce REST API human documentation. This is not an official WooCommerce OpenAPI document.",
		ServerURL:   "https://{site}",
		Sources:     []string{"https://developer.woocommerce.com/docs/apis/rest-api/v3/", "https://woocommerce.github.io/woocommerce-rest-api-docs/", "https://woocommerce.github.io/woocommerce-rest-api-docs/#authentication"},
		SourceNote:  "WooCommerce publishes human REST API documentation but no recorded stable public official OpenAPI document; this overlay covers selected products, orders, customers, coupons, reports, refunds, and webhooks.",
		Security:    security,
		Schemas:     []string{"WooCommerceObject", "WooCommerceCollection", "WooCommerceError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/woocommerce-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/wp-json/wc/v3/products":                  {"get": op("listWooCommerceProducts", "List products", params(query("page", "Page number."), query("per_page", "Page size.")), "", "#/components/schemas/WooCommerceCollection", "wooCommerceBasic"), "post": op("createWooCommerceProduct", "Create product", nil, "#/components/schemas/WooCommerceObject", "#/components/schemas/WooCommerceObject", "wooCommerceBasic")},
			"/wp-json/wc/v3/products/{product_id}":     {"get": op("getWooCommerceProduct", "Get product", params(path("product_id", "Product ID.")), "", "#/components/schemas/WooCommerceObject", "wooCommerceBasic"), "put": op("updateWooCommerceProduct", "Update product", params(path("product_id", "Product ID.")), "#/components/schemas/WooCommerceObject", "#/components/schemas/WooCommerceObject", "wooCommerceBasic")},
			"/wp-json/wc/v3/orders":                    {"get": op("listWooCommerceOrders", "List orders", params(query("page", "Page number."), query("per_page", "Page size.")), "", "#/components/schemas/WooCommerceCollection", "wooCommerceBasic"), "post": op("createWooCommerceOrder", "Create order", nil, "#/components/schemas/WooCommerceObject", "#/components/schemas/WooCommerceObject", "wooCommerceBasic")},
			"/wp-json/wc/v3/orders/{order_id}":         {"get": op("getWooCommerceOrder", "Get order", params(path("order_id", "Order ID.")), "", "#/components/schemas/WooCommerceObject", "wooCommerceBasic"), "put": op("updateWooCommerceOrder", "Update order", params(path("order_id", "Order ID.")), "#/components/schemas/WooCommerceObject", "#/components/schemas/WooCommerceObject", "wooCommerceBasic")},
			"/wp-json/wc/v3/customers":                 {"get": op("listWooCommerceCustomers", "List customers", params(query("page", "Page number."), query("per_page", "Page size.")), "", "#/components/schemas/WooCommerceCollection", "wooCommerceBasic"), "post": op("createWooCommerceCustomer", "Create customer", nil, "#/components/schemas/WooCommerceObject", "#/components/schemas/WooCommerceObject", "wooCommerceBasic")},
			"/wp-json/wc/v3/coupons":                   {"get": op("listWooCommerceCoupons", "List coupons", nil, "", "#/components/schemas/WooCommerceCollection", "wooCommerceBasic"), "post": op("createWooCommerceCoupon", "Create coupon", nil, "#/components/schemas/WooCommerceObject", "#/components/schemas/WooCommerceObject", "wooCommerceBasic")},
			"/wp-json/wc/v3/orders/{order_id}/refunds": {"get": op("listWooCommerceRefunds", "List order refunds", params(path("order_id", "Order ID.")), "", "#/components/schemas/WooCommerceCollection", "wooCommerceBasic"), "post": op("createWooCommerceRefund", "Create order refund", params(path("order_id", "Order ID.")), "#/components/schemas/WooCommerceObject", "#/components/schemas/WooCommerceObject", "wooCommerceBasic")},
			"/wp-json/wc/v3/reports/sales":             {"get": op("getWooCommerceSalesReport", "Get sales report", params(query("date_min", "Start date."), query("date_max", "End date.")), "", "#/components/schemas/WooCommerceCollection", "wooCommerceBasic")},
			"/wp-json/wc/v3/webhooks":                  {"get": op("listWooCommerceWebhooks", "List webhooks", nil, "", "#/components/schemas/WooCommerceCollection", "wooCommerceBasic"), "post": op("createWooCommerceWebhook", "Create webhook", nil, "#/components/schemas/WooCommerceObject", "#/components/schemas/WooCommerceObject", "wooCommerceBasic")},
		},
	}
}

func wiseOverlay() overlaySpec {
	security := map[string]map[string]any{
		"wiseBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Wise OAuth 2.0 access token", "description": "Wise Platform API access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "wise",
		Title:       "Wise Platform API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Wise Platform API human documentation. This is not an official Wise OpenAPI document.",
		ServerURL:   "https://api.wise.com",
		Sources:     []string{"https://docs.wise.com/api-reference", "https://docs.wise.com/api-docs/features", "https://wise.com/us/business/api"},
		SourceNote:  "Wise publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected profiles, quotes, recipients, transfers, balances, statements, rates, and webhooks.",
		Security:    security,
		Schemas:     []string{"WiseObject", "WiseCollection", "WiseError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/wise-platform-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v1/profiles":                       {"get": op("listWiseProfiles", "List profiles", nil, "", "#/components/schemas/WiseCollection", "wiseBearer")},
			"/v3/profiles/{profile_id}/quotes":   {"post": op("createWiseQuote", "Create quote", params(path("profile_id", "Profile ID.")), "#/components/schemas/WiseObject", "#/components/schemas/WiseObject", "wiseBearer")},
			"/v1/accounts":                       {"get": op("listWiseRecipients", "List recipients", params(query("profileId", "Profile ID.")), "", "#/components/schemas/WiseCollection", "wiseBearer"), "post": op("createWiseRecipient", "Create recipient", nil, "#/components/schemas/WiseObject", "#/components/schemas/WiseObject", "wiseBearer")},
			"/v1/transfers":                      {"get": op("listWiseTransfers", "List transfers", params(query("profile", "Profile ID.")), "", "#/components/schemas/WiseCollection", "wiseBearer"), "post": op("createWiseTransfer", "Create transfer", nil, "#/components/schemas/WiseObject", "#/components/schemas/WiseObject", "wiseBearer")},
			"/v4/profiles/{profile_id}/balances": {"get": op("listWiseBalances", "List balances", params(path("profile_id", "Profile ID.")), "", "#/components/schemas/WiseCollection", "wiseBearer"), "post": op("createWiseBalance", "Create balance", params(path("profile_id", "Profile ID.")), "#/components/schemas/WiseObject", "#/components/schemas/WiseObject", "wiseBearer")},
			"/v1/rates":                          {"get": op("listWiseRates", "List exchange rates", params(query("source", "Source currency."), query("target", "Target currency.")), "", "#/components/schemas/WiseCollection", "wiseBearer")},
			"/v1/profiles/{profile_id}/balance-statements/{balance_id}/statement.json": {"get": op("getWiseBalanceStatement", "Get balance statement", params(path("profile_id", "Profile ID."), path("balance_id", "Balance ID."), query("intervalStart", "Statement start."), query("intervalEnd", "Statement end.")), "", "#/components/schemas/WiseObject", "wiseBearer")},
			"/v3/profiles/{profile_id}/subscriptions":                                  {"get": op("listWiseSubscriptions", "List webhook subscriptions", params(path("profile_id", "Profile ID.")), "", "#/components/schemas/WiseCollection", "wiseBearer"), "post": op("createWiseSubscription", "Create webhook subscription", params(path("profile_id", "Profile ID.")), "#/components/schemas/WiseObject", "#/components/schemas/WiseObject", "wiseBearer")},
		},
	}
}

func dhlOverlay() overlaySpec {
	security := map[string]map[string]any{
		"dhlAPIKey": {"type": "apiKey", "in": "header", "name": "DHL-API-Key", "description": "DHL Shipment Tracking API subscription key carried in the DHL-API-Key header."},
	}
	return overlaySpec{
		ProviderID:  "dhl",
		Title:       "DHL Shipment Tracking API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official DHL Shipment Tracking API human documentation. This is not an official DHL OpenAPI document.",
		ServerURL:   "https://api-eu.dhl.com",
		Sources:     []string{"https://developer.dhl.com/api-reference/shipment-tracking?language_content_entity=en", "https://developer.dhl.com/getting-started/find-the-right-api?language_content_entity=en"},
		SourceNote:  "DHL publishes Shipment Tracking - Unified human API documentation but no recorded stable public downloadable official OpenAPI document; this overlay covers the documented tracking endpoint.",
		Security:    security,
		Schemas:     []string{"DHLTrackingObject", "DHLTrackingCollection", "DHLTrackingError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/dhl-shipment-tracking-overlay.json",
		Paths: map[string]map[string]any{
			"/track/shipments": {"get": op("getDHLShipmentTracking", "Get shipment tracking details", params(query("trackingNumber", "DHL tracking number."), query("recipientPostalCode", "Recipient postal code for additional verification.")), "", "#/components/schemas/DHLTrackingCollection", "dhlAPIKey")},
		},
	}
}

func onfleetOverlay() overlaySpec {
	security := map[string]map[string]any{
		"onfleetBasic": {"type": "http", "scheme": "basic", "description": "Onfleet API key carried as the HTTP Basic username with an empty password."},
	}
	return overlaySpec{
		ProviderID:  "onfleet",
		Title:       "Onfleet API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Onfleet API human documentation. This is not an official Onfleet OpenAPI document.",
		ServerURL:   "https://onfleet.com/api/v2",
		Sources:     []string{"https://docs.onfleet.com/reference", "https://support.onfleet.com/hc/en-us/articles/360045763292-API"},
		SourceNote:  "Onfleet publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected auth, task, worker, team, hub, destination, recipient, admin, and webhook endpoints.",
		Security:    security,
		Schemas:     []string{"OnfleetObject", "OnfleetCollection", "OnfleetError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/onfleet-api-overlay.json",
		Paths: map[string]map[string]any{
			"/auth/test":       {"get": op("testOnfleetAuth", "Test authentication", nil, "", "#/components/schemas/OnfleetObject", "onfleetBasic")},
			"/tasks":           {"get": op("listOnfleetTasks", "List tasks", nil, "", "#/components/schemas/OnfleetCollection", "onfleetBasic"), "post": op("createOnfleetTask", "Create task", nil, "#/components/schemas/OnfleetObject", "#/components/schemas/OnfleetObject", "onfleetBasic")},
			"/tasks/{task_id}": {"get": op("getOnfleetTask", "Get task", params(path("task_id", "Task ID.")), "", "#/components/schemas/OnfleetObject", "onfleetBasic"), "put": op("updateOnfleetTask", "Update task", params(path("task_id", "Task ID.")), "#/components/schemas/OnfleetObject", "#/components/schemas/OnfleetObject", "onfleetBasic")},
			"/workers":         {"get": op("listOnfleetWorkers", "List workers", nil, "", "#/components/schemas/OnfleetCollection", "onfleetBasic"), "post": op("createOnfleetWorker", "Create worker", nil, "#/components/schemas/OnfleetObject", "#/components/schemas/OnfleetObject", "onfleetBasic")},
			"/teams":           {"get": op("listOnfleetTeams", "List teams", nil, "", "#/components/schemas/OnfleetCollection", "onfleetBasic")},
			"/hubs":            {"get": op("listOnfleetHubs", "List hubs", nil, "", "#/components/schemas/OnfleetCollection", "onfleetBasic")},
			"/destinations":    {"post": op("createOnfleetDestination", "Create destination", nil, "#/components/schemas/OnfleetObject", "#/components/schemas/OnfleetObject", "onfleetBasic")},
			"/recipients":      {"post": op("createOnfleetRecipient", "Create recipient", nil, "#/components/schemas/OnfleetObject", "#/components/schemas/OnfleetObject", "onfleetBasic")},
			"/admins":          {"get": op("listOnfleetAdmins", "List administrators", nil, "", "#/components/schemas/OnfleetCollection", "onfleetBasic")},
			"/webhooks":        {"get": op("listOnfleetWebhooks", "List webhooks", nil, "", "#/components/schemas/OnfleetCollection", "onfleetBasic"), "post": op("createOnfleetWebhook", "Create webhook", nil, "#/components/schemas/OnfleetObject", "#/components/schemas/OnfleetObject", "onfleetBasic")},
		},
	}
}

func unleashedOverlay() overlaySpec {
	security := map[string]map[string]any{
		"unleashedAPIID":      {"type": "apiKey", "in": "header", "name": "api-auth-id", "description": "Unleashed API ID carried in the api-auth-id header."},
		"unleashedSignature":  {"type": "apiKey", "in": "header", "name": "api-auth-signature", "description": "Unleashed request signature carried in the api-auth-signature header."},
		"unleashedClientType": {"type": "apiKey", "in": "header", "name": "client-type", "description": "Unleashed client-type header identifying the integration for API support and usage stats."},
	}
	return overlaySpec{
		ProviderID:  "unleashed-software",
		Title:       "Unleashed Software API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Unleashed API human documentation. This is not an official Unleashed OpenAPI document.",
		ServerURL:   "https://api.unleashedsoftware.com/v2",
		Sources:     []string{"https://apidocs.unleashedsoftware.com/", "https://support.unleashedsoftware.com/hc/en-us/articles/4402393233689", "https://apidocs.unleashedsoftware.com/Authentication"},
		SourceNote:  "Unleashed publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected sales orders, stock, products, customers, suppliers, purchase orders, and warehouses.",
		Security:    security,
		Schemas:     []string{"UnleashedObject", "UnleashedCollection", "UnleashedError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/unleashed-software-api-overlay.json",
		Paths: map[string]map[string]any{
			"/SalesOrders":        {"get": op("listUnleashedSalesOrders", "List sales orders", params(query("pageSize", "Page size."), query("modifiedSince", "Modified-since filter.")), "", "#/components/schemas/UnleashedCollection", "unleashedAPIID", "unleashedSignature", "unleashedClientType")},
			"/SalesOrders/{guid}": {"get": op("getUnleashedSalesOrder", "Get sales order", params(path("guid", "Sales order GUID.")), "", "#/components/schemas/UnleashedObject", "unleashedAPIID", "unleashedSignature", "unleashedClientType")},
			"/StockOnHand":        {"get": op("listUnleashedStockOnHand", "List stock on hand", params(query("pageSize", "Page size."), query("modifiedSince", "Modified-since filter.")), "", "#/components/schemas/UnleashedCollection", "unleashedAPIID", "unleashedSignature", "unleashedClientType")},
			"/Products":           {"get": op("listUnleashedProducts", "List products", params(query("pageSize", "Page size.")), "", "#/components/schemas/UnleashedCollection", "unleashedAPIID", "unleashedSignature", "unleashedClientType")},
			"/Customers":          {"get": op("listUnleashedCustomers", "List customers", params(query("pageSize", "Page size.")), "", "#/components/schemas/UnleashedCollection", "unleashedAPIID", "unleashedSignature", "unleashedClientType")},
			"/Suppliers":          {"get": op("listUnleashedSuppliers", "List suppliers", params(query("pageSize", "Page size.")), "", "#/components/schemas/UnleashedCollection", "unleashedAPIID", "unleashedSignature", "unleashedClientType")},
			"/PurchaseOrders":     {"get": op("listUnleashedPurchaseOrders", "List purchase orders", params(query("pageSize", "Page size.")), "", "#/components/schemas/UnleashedCollection", "unleashedAPIID", "unleashedSignature", "unleashedClientType")},
			"/Warehouses":         {"get": op("listUnleashedWarehouses", "List warehouses", nil, "", "#/components/schemas/UnleashedCollection", "unleashedAPIID", "unleashedSignature", "unleashedClientType")},
		},
	}
}

func workableOverlay() overlaySpec {
	security := map[string]map[string]any{
		"workableBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Workable access token", "description": "Workable API access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "workable",
		Title:       "Workable API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Workable API human documentation. This is not an official Workable OpenAPI document.",
		ServerURL:   "https://{subdomain}.workable.com/spi/v3",
		Sources:     []string{"https://workable.readme.io/reference", "https://help.workable.com/hc/en-us/articles/115013356548-Workable-API-Documentation"},
		SourceNote:  "Workable publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected jobs, candidates, members, recruiters, stages, scheduled events, and requisitions.",
		Security:    security,
		Schemas:     []string{"WorkableObject", "WorkableCollection", "WorkableError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/workable-api-overlay.json",
		Paths: map[string]map[string]any{
			"/jobs":                        {"get": op("listWorkableJobs", "List jobs", params(query("state", "Job state filter.")), "", "#/components/schemas/WorkableCollection", "workableBearer")},
			"/jobs/{shortcode}":            {"get": op("getWorkableJob", "Get job", params(path("shortcode", "Job shortcode.")), "", "#/components/schemas/WorkableObject", "workableBearer")},
			"/jobs/{shortcode}/candidates": {"get": op("listWorkableJobCandidates", "List job candidates", params(path("shortcode", "Job shortcode.")), "", "#/components/schemas/WorkableCollection", "workableBearer"), "post": op("createWorkableCandidate", "Create candidate", params(path("shortcode", "Job shortcode.")), "#/components/schemas/WorkableObject", "#/components/schemas/WorkableObject", "workableBearer")},
			"/candidates":                  {"get": op("listWorkableCandidates", "List candidates", params(query("updated_after", "Updated-after filter.")), "", "#/components/schemas/WorkableCollection", "workableBearer")},
			"/candidates/{candidate_id}":   {"get": op("getWorkableCandidate", "Get candidate", params(path("candidate_id", "Candidate ID.")), "", "#/components/schemas/WorkableObject", "workableBearer")},
			"/members":                     {"get": op("listWorkableMembers", "List members", nil, "", "#/components/schemas/WorkableCollection", "workableBearer")},
			"/recruiters":                  {"get": op("listWorkableRecruiters", "List recruiters", nil, "", "#/components/schemas/WorkableCollection", "workableBearer")},
			"/stages":                      {"get": op("listWorkableStages", "List pipeline stages", nil, "", "#/components/schemas/WorkableCollection", "workableBearer")},
			"/scheduled_events":            {"get": op("listWorkableScheduledEvents", "List scheduled events", nil, "", "#/components/schemas/WorkableCollection", "workableBearer")},
			"/requisitions":                {"get": op("listWorkableRequisitions", "List requisitions", nil, "", "#/components/schemas/WorkableCollection", "workableBearer")},
		},
	}
}

func build(spec overlaySpec) map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       spec.Title,
			"version":     "2026-05-19",
			"description": spec.Description,
		},
		"servers": []map[string]any{{"url": spec.ServerURL}},
		"paths":   orderedMap(spec.Paths),
		"components": map[string]any{
			"securitySchemes": orderedMap(spec.Security),
			"schemas":         schemas(spec.Schemas),
		},
		"x-apitools-overlay": map[string]any{
			"provider_id":       spec.ProviderID,
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs":       spec.Sources,
			"source_note":       spec.SourceNote,
		},
	}
}

func op(operationID, summary string, parameters []map[string]any, requestRef, responseRef string, securityNames ...string) map[string]any {
	out := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official human API documentation.",
		"responses": map[string]any{
			"200":     response(responseRef),
			"default": map[string]any{"description": "Provider error response."},
		},
	}
	if len(securityNames) > 0 {
		requirement := map[string][]string{}
		for _, name := range securityNames {
			if name != "" {
				requirement[name] = []string{}
			}
		}
		if len(requirement) > 0 {
			out["security"] = []map[string][]string{requirement}
		}
	}
	if len(parameters) > 0 {
		out["parameters"] = parameters
	}
	if requestRef != "" {
		out["requestBody"] = map[string]any{
			"required": true,
			"content":  map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": requestRef}}},
		}
	}
	return out
}

func response(ref string) map[string]any {
	out := map[string]any{"description": "Successful response."}
	if ref != "" {
		out["content"] = map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": ref}}}
	}
	return out
}

func params(items ...map[string]any) []map[string]any { return items }

func path(name, description string) map[string]any { return parameter(name, "path", description, true) }

func query(name, description string) map[string]any {
	return parameter(name, "query", description, false)
}

func parameter(name, in, description string, required bool) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          in,
		"required":    required,
		"description": description,
		"schema":      map[string]any{"type": "string"},
	}
}

func schemas(names []string) map[string]map[string]any {
	sort.Strings(names)
	out := map[string]map[string]any{}
	for _, name := range names {
		out[name] = map[string]any{"type": "object", "additionalProperties": true}
	}
	return out
}

func orderedMap[V any](in map[string]V) map[string]V {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := map[string]V{}
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}

func write(path string, doc map[string]any) {
	content, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o644); err != nil {
		panic(err)
	}
}

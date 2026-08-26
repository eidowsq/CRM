package controllers

import (
	"strings"
	"time"

	"crm/models"
	"github.com/beego/beego/v2/client/orm"
)

type DashboardController struct{ APIController }

func (c *DashboardController) Summary() {
	o := orm.NewOrm()
	user := strings.TrimSpace(currentUser(c.Ctx.Request))
	role := strings.TrimSpace(currentRole(c.Ctx.Request))
	now := time.Now()

	stats := map[string]any{
		"month_deal_amount":      0.0,
		"year_deal_amount":       0.0,
		"month_deal_count":       0,
		"year_deal_count":        0,
		"month_new_contacts":     0,
		"year_new_contacts":      0,
		"total_deals":            0,
		"total_contacts":         0,
		"total_deal_amount":      0.0,
		"month_payment_amount":   0.0,
		"year_payment_amount":    0.0,
		"total_payment_amount":   0.0,
		"month_pending_amount":   0.0,
		"year_pending_amount":    0.0,
		"total_pending_amount":   0.0,
	}

	var contacts []models.Contact
	_, _ = o.QueryTable(new(models.Contact)).RelatedSel().All(&contacts)
	for _, contact := range contacts {
		if !dashboardCanAccessContact(contact, user, role) {
			continue
		}
		stats["total_contacts"] = stats["total_contacts"].(int) + 1
		if sameMonth(contact.CreatedAt, now) {
			stats["month_new_contacts"] = stats["month_new_contacts"].(int) + 1
		}
		if sameYear(contact.CreatedAt, now) {
			stats["year_new_contacts"] = stats["year_new_contacts"].(int) + 1
		}
	}

	var contracts []models.Contract
	_, _ = o.QueryTable(new(models.Contract)).RelatedSel().All(&contracts)
	for _, contract := range contracts {
		if !dashboardCanAccessContract(contract, user, role) {
			continue
		}
		if strings.TrimSpace(contract.Status) != "approved" {
			continue
		}

		stats["total_deals"] = stats["total_deals"].(int) + 1
		stats["total_deal_amount"] = stats["total_deal_amount"].(float64) + contract.Amount

		dealTime := contract.ReviewedAt
		if dealTime.IsZero() {
			dealTime = contract.CreatedAt
		}
		if sameMonth(dealTime, now) {
			stats["month_deal_count"] = stats["month_deal_count"].(int) + 1
			stats["month_deal_amount"] = stats["month_deal_amount"].(float64) + contract.Amount
		}
		if sameYear(dealTime, now) {
			stats["year_deal_count"] = stats["year_deal_count"].(int) + 1
			stats["year_deal_amount"] = stats["year_deal_amount"].(float64) + contract.Amount
		}
	}

	var payments []models.Payment
	_, _ = o.QueryTable(new(models.Payment)).RelatedSel().All(&payments)
	for _, payment := range payments {
		if !dashboardCanAccessPayment(payment, user, role) {
			continue
		}
		paymentTime := payment.PaymentDate
		if paymentTime.IsZero() {
			paymentTime = payment.CreatedAt
		}

		switch strings.TrimSpace(payment.Status) {
		case "approved":
			stats["total_payment_amount"] = stats["total_payment_amount"].(float64) + payment.PaymentAmount
			if sameMonth(paymentTime, now) {
				stats["month_payment_amount"] = stats["month_payment_amount"].(float64) + payment.PaymentAmount
			}
			if sameYear(paymentTime, now) {
				stats["year_payment_amount"] = stats["year_payment_amount"].(float64) + payment.PaymentAmount
			}
		default:
			stats["total_pending_amount"] = stats["total_pending_amount"].(float64) + payment.PaymentAmount
			if sameMonth(paymentTime, now) {
				stats["month_pending_amount"] = stats["month_pending_amount"].(float64) + payment.PaymentAmount
			}
			if sameYear(paymentTime, now) {
				stats["year_pending_amount"] = stats["year_pending_amount"].(float64) + payment.PaymentAmount
			}
		}
	}

	c.respond(stats, 200)
}

func dashboardCanAccessContact(contact models.Contact, user, role string) bool {
	if role == "admin" {
		return true
	}
	if contact.Customer == nil {
		return false
	}
	return strings.TrimSpace(contact.Customer.Creator) == user || strings.TrimSpace(contact.Customer.Owner) == user
}

func dashboardCanAccessContract(contract models.Contract, user, role string) bool {
	if role == "admin" {
		return true
	}
	return strings.TrimSpace(contract.Submitter) == user
}

func dashboardCanAccessPayment(payment models.Payment, user, role string) bool {
	if role == "admin" {
		return true
	}
	if payment.Contract != nil && strings.TrimSpace(payment.Contract.Submitter) == user {
		return true
	}
	if payment.Customer != nil {
		return strings.TrimSpace(payment.Customer.Creator) == user || strings.TrimSpace(payment.Customer.Owner) == user
	}
	return false
}

func sameMonth(target, now time.Time) bool {
	return !target.IsZero() && target.Year() == now.Year() && target.Month() == now.Month()
}

func sameYear(target, now time.Time) bool {
	return !target.IsZero() && target.Year() == now.Year()
}

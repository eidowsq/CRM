package controllers

import (
	"strconv"
	"time"

	"crm/models"
	"github.com/beego/beego/v2/client/orm"
)

type ActivityController struct{ APIController }

func (c *ActivityController) List() {
	customerID, _ := strconv.Atoi(c.GetString("customer_id"))
	qs := orm.NewOrm().QueryTable(new(models.Activity)).RelatedSel()
	if customerID > 0 {
		qs = qs.Filter("customer_id", customerID)
	}
	var items []models.Activity
	_, err := qs.OrderBy("-created_at").All(&items)
	if err != nil {
		c.error(err.Error(), 500)
		return
	}
	c.respond(items, 200)
}

func (c *ActivityController) Create() {
	var input struct {
		CustomerID int    `json:"customer_id"`
		Type       string `json:"type"`
		Content    string `json:"content"`
		NextAction string `json:"next_action"`
	}
	if err := decodeBody(&c.APIController, &input); err != nil || input.CustomerID == 0 || input.Content == "" {
		c.error("客户和跟进内容不能为空", 400)
		return
	}
	activity := models.Activity{Customer: &models.Customer{Id: input.CustomerID}, Type: input.Type, Content: input.Content, NextAction: input.NextAction, CreatedAt: time.Now()}
	if _, err := orm.NewOrm().Insert(&activity); err != nil {
		c.error(err.Error(), 500)
		return
	}

	// Keep the parent customer's latest follow-up summary in sync with the feed.
	customer := models.Customer{Id: input.CustomerID}
	if err := orm.NewOrm().Read(&customer); err == nil {
		customer.FollowUpRecord = input.Content
		if input.NextAction != "" {
			if nextContact, err := time.ParseInLocation("2006-01-02", input.NextAction, time.Local); err == nil {
				customer.NextContact = nextContact
			}
		}
		_, _ = orm.NewOrm().Update(&customer)
	}

	c.respond(activity, 201)
}

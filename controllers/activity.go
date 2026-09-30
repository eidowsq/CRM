package controllers

import (
	"strconv"
	"strings"
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
	activity := models.Activity{
		Customer:   &models.Customer{Id: input.CustomerID},
		Type:       input.Type,
		Content:    input.Content,
		NextAction: input.NextAction,
		Publisher:  currentUser(c.Ctx.Request),
		CreatedAt:  time.Now(),
	}
	if _, err := orm.NewOrm().Insert(&activity); err != nil {
		c.error(err.Error(), 500)
		return
	}

	syncCustomerLatestFollowUp(input.CustomerID)

	c.respond(activity, 201)
}

func (c *ActivityController) Update() {
	id, _ := strconv.Atoi(c.Ctx.Input.Param(":id"))
	o := orm.NewOrm()
	var activities []models.Activity
	if _, err := o.QueryTable(new(models.Activity)).
		RelatedSel().
		Filter("id", id).
		All(&activities); err != nil || len(activities) == 0 {
		c.error("跟进记录不存在", 404)
		return
	}
	activity := activities[0]

	var input struct {
		Type       string `json:"type"`
		Content    string `json:"content"`
		NextAction string `json:"next_action"`
	}
	if err := decodeBody(&c.APIController, &input); err != nil {
		c.error("请求格式不正确", 400)
		return
	}
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" {
		c.error("跟进内容不能为空", 400)
		return
	}

	activity.Type = strings.TrimSpace(input.Type)
	activity.Content = input.Content
	activity.NextAction = strings.TrimSpace(input.NextAction)
	if _, err := o.Update(&activity, "Type", "Content", "NextAction"); err != nil {
		c.error(err.Error(), 500)
		return
	}
	if activity.Customer != nil {
		syncCustomerLatestFollowUp(activity.Customer.Id)
	}

	var updatedItems []models.Activity
	if _, err := o.QueryTable(new(models.Activity)).
		RelatedSel().
		Filter("id", id).
		All(&updatedItems); err == nil && len(updatedItems) > 0 {
		c.respond(updatedItems[0], 200)
		return
	}
	c.respond(activity, 200)
}

func syncCustomerLatestFollowUp(customerID int) {
	if customerID == 0 {
		return
	}
	o := orm.NewOrm()
	var latest []models.Activity
	if _, err := o.QueryTable(new(models.Activity)).
		Filter("customer_id", customerID).
		OrderBy("-created_at").
		All(&latest); err != nil || len(latest) == 0 {
		return
	}

	customer := models.Customer{Id: customerID}
	if err := o.Read(&customer); err != nil {
		return
	}
	customer.FollowUpRecord = latest[0].Content
	if nextAction := strings.TrimSpace(latest[0].NextAction); nextAction != "" {
		if nextContact, err := time.ParseInLocation("2006-01-02", nextAction, time.Local); err == nil {
			customer.NextContact = nextContact
		}
	}
	_, _ = o.Update(&customer, "FollowUpRecord", "NextContact")
}

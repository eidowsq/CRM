package controllers

import (
	"strconv"

	"crm/models"
	"github.com/beego/beego/v2/client/orm"
)

type ContactController struct{ APIController }

func (c *ContactController) List() {
	customerID, _ := strconv.Atoi(c.GetString("customer_id"))
	qs := orm.NewOrm().QueryTable(new(models.Contact)).RelatedSel()
	if customerID > 0 {
		qs = qs.Filter("customer_id", customerID)
	}
	var items []models.Contact
	_, err := qs.OrderBy("-created_at").All(&items)
	if err != nil {
		c.error(err.Error(), 500)
		return
	}
	c.respond(items, 200)
}

func (c *ContactController) Create() {
	var input struct {
		CustomerID int    `json:"customer_id"`
		Name       string `json:"name"`
		Role       string `json:"role"`
		Phone      string `json:"phone"`
		Email      string `json:"email"`
	}
	if err := decodeBody(&c.APIController, &input); err != nil || input.CustomerID == 0 || input.Name == "" {
		c.error("联系人姓名和所属客户不能为空", 400)
		return
	}
	item := models.Contact{
		Customer: &models.Customer{Id: input.CustomerID},
		Name:     input.Name,
		Role:     input.Role,
		Phone:    input.Phone,
		Email:    input.Email,
	}
	if _, err := orm.NewOrm().Insert(&item); err != nil {
		c.error(err.Error(), 500)
		return
	}
	c.respond(item, 201)
}

package controllers

import (
	"strconv"
	"strings"

	"crm/models"
	"github.com/beego/beego/v2/client/orm"
)

type CustomerController struct{ APIController }

func (c *CustomerController) List() {
	var items []models.Customer
	_, err := orm.NewOrm().QueryTable(new(models.Customer)).OrderBy("-updated_at").All(&items)
	if err != nil {
		c.error(err.Error(), 500)
		return
	}

	keyword := strings.TrimSpace(c.GetString("keyword"))
	stage := strings.TrimSpace(c.GetString("stage"))
	user := currentUser(c.Ctx.Request)
	role := currentRole(c.Ctx.Request)

	filtered := make([]models.Customer, 0, len(items))
	for _, item := range items {
		if role != "admin" {
			inPool := strings.TrimSpace(item.Owner) == "" || item.Owner == "未分配"
			createdByMe := strings.TrimSpace(item.Creator) == user
			if !inPool && !createdByMe {
				continue
			}
		}
		if stage != "" && item.Stage != stage {
			continue
		}
		if keyword != "" {
			text := strings.ToLower(strings.Join([]string{
				item.Name, item.Industry, item.Owner, item.Source, item.Phone, item.Address,
			}, " "))
			if !strings.Contains(text, strings.ToLower(keyword)) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	c.respond(filtered, 200)
}

func (c *CustomerController) Create() {
	var input models.Customer
	if err := decodeBody(&c.APIController, &input); err != nil {
		c.error("请求格式必须正确", 400)
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		c.error("客户名称不能为空", 400)
		return
	}
	if input.NextContact.IsZero() {
		c.error("下次联系时间不能为空", 400)
		return
	}
	if user := currentUser(c.Ctx.Request); user != "" {
		input.Creator = user
		if strings.TrimSpace(input.Owner) == "" {
			input.Owner = user
		}
	}
	if _, err := orm.NewOrm().Insert(&input); err != nil {
		c.error(err.Error(), 500)
		return
	}
	c.respond(input, 201)
}

func (c *CustomerController) Update() {
	id, _ := strconv.Atoi(c.Ctx.Input.Param(":id"))
	o := orm.NewOrm()
	item := models.Customer{Id: id}
	if err := o.Read(&item); err != nil {
		c.error("客户不存在", 404)
		return
	}
	var input models.Customer
	if err := decodeBody(&c.APIController, &input); err != nil {
		c.error("请求格式不正确", 400)
		return
	}
	item.Name = input.Name
	item.Industry = input.Industry
	item.Level = input.Level
	item.Stage = input.Stage
	item.Phone = input.Phone
	item.Mobile = input.Mobile
	item.Website = input.Website
	item.Source = input.Source
	item.Note = input.Note
	item.Creator = input.Creator
	item.Owner = input.Owner
	item.FollowUpRecord = input.FollowUpRecord
	item.Province = input.Province
	item.City = input.City
	item.District = input.District
	item.Address = input.Address
	item.Email = input.Email
	item.Amount = input.Amount
	if !input.NextContact.IsZero() {
		item.NextContact = input.NextContact
	}
	if _, err := o.Update(&item); err != nil {
		c.error(err.Error(), 500)
		return
	}
	c.respond(item, 200)
}

func (c *CustomerController) TransferToPool() {
	id, _ := strconv.Atoi(c.Ctx.Input.Param(":id"))
	o := orm.NewOrm()
	item := models.Customer{Id: id}
	if err := o.Read(&item); err != nil {
		c.error("客户不存在", 404)
		return
	}
	item.Owner = ""
	if _, err := o.Update(&item); err != nil {
		c.error(err.Error(), 500)
		return
	}
	c.respond(item, 200)
}

func (c *CustomerController) Claim() {
	id, _ := strconv.Atoi(c.Ctx.Input.Param(":id"))
	o := orm.NewOrm()
	item := models.Customer{Id: id}
	if err := o.Read(&item); err != nil {
		c.error("客户不存在", 404)
		return
	}
	user := strings.TrimSpace(currentUser(c.Ctx.Request))
	if user == "" {
		c.error("请先登录", 401)
		return
	}
	item.Owner = user
	if _, err := o.Update(&item); err != nil {
		c.error(err.Error(), 500)
		return
	}
	c.respond(item, 200)
}

func (c *CustomerController) Delete() {
	id, _ := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if _, err := orm.NewOrm().Delete(&models.Customer{Id: id}); err != nil {
		c.error(err.Error(), 500)
		return
	}
	c.respond(map[string]bool{"deleted": true}, 200)
}

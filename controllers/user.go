package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"crm/models"
	"crm/security"
	"github.com/beego/beego/v2/client/orm"
)

type UserController struct{ APIController }

func (c *UserController) List() {
	if currentRole(c.Ctx.Request) != "admin" {
		c.error("无权限查看账户列表", http.StatusForbidden)
		return
	}
	var users []models.User
	_, err := orm.NewOrm().QueryTable(new(models.User)).OrderBy("-created_at").All(&users)
	if err != nil {
		c.error(err.Error(), 500)
		return
	}
	c.respond(users, 200)
}

func (c *UserController) Create() {
	if currentRole(c.Ctx.Request) != "admin" {
		c.error("仅管理员可以创建账户", http.StatusForbidden)
		return
	}
	var input struct {
		Username string `json:"username"`
		Alias    string `json:"alias"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decodeBody(&c.APIController, &input); err != nil || input.Username == "" || input.Password == "" {
		c.error("用户名和密码不能为空", http.StatusBadRequest)
		return
	}
	if err := security.ValidatePassword(input.Password); err != nil {
		c.error(err.Error(), http.StatusBadRequest)
		return
	}
	if input.Role == "" {
		input.Role = "user"
	}

	var existing []models.User
	_, err := orm.NewOrm().QueryTable(new(models.User)).Filter("username", input.Username).All(&existing)
	if err != nil {
		c.error("创建账户失败", 500)
		return
	}
	if len(existing) > 0 {
		c.error("用户名已存在", http.StatusConflict)
		return
	}

	user := models.User{
		Username: input.Username,
		Alias:    input.Alias,
		Role:     input.Role,
		CreatedAt: time.Now(),
	}
	hashed, err := security.HashPassword(input.Password)
	if err != nil {
		c.error("密码加密失败", http.StatusInternalServerError)
		return
	}
	user.Password = hashed
	if _, err := orm.NewOrm().Insert(&user); err != nil {
		c.error(err.Error(), http.StatusInternalServerError)
		return
	}
	c.respond(map[string]any{
		"id":       user.Id,
		"username": user.Username,
		"alias":    user.Alias,
		"role":     user.Role,
	}, 201)
}

func (c *UserController) Delete() {
	if currentRole(c.Ctx.Request) != "admin" {
		c.error("仅管理员可以删除账户", http.StatusForbidden)
		return
	}
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id == 0 {
		c.error("账户不存在", http.StatusBadRequest)
		return
	}
	var users []models.User
	_, err = orm.NewOrm().QueryTable(new(models.User)).Filter("id", id).All(&users)
	if err != nil || len(users) == 0 {
		c.error("账户不存在", http.StatusNotFound)
		return
	}
	if users[0].Username == "admin" {
		c.error("默认管理员账户不可删除", http.StatusForbidden)
		return
	}
	if _, err := orm.NewOrm().Delete(&models.User{Id: id}); err != nil {
		c.error("删除账户失败", 500)
		return
	}
	c.respond(map[string]bool{"deleted": true}, 200)
}

func (c *UserController) Update() {
	if currentRole(c.Ctx.Request) != "admin" {
		c.error("仅管理员可以编辑账户", http.StatusForbidden)
		return
	}
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id == 0 {
		c.error("账户不存在", http.StatusBadRequest)
		return
	}

	var input struct {
		Alias    string `json:"alias"`
		Password string `json:"password"`
	}
	if err := decodeBody(&c.APIController, &input); err != nil {
		c.error("请求格式不正确", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Password) != "" {
		if err := security.ValidatePassword(input.Password); err != nil {
			c.error(err.Error(), http.StatusBadRequest)
			return
		}
	}

	var users []models.User
	_, err = orm.NewOrm().QueryTable(new(models.User)).Filter("id", id).All(&users)
	if err != nil || len(users) == 0 {
		c.error("账户不存在", http.StatusNotFound)
		return
	}

	user := users[0]
	user.Alias = input.Alias
	if strings.TrimSpace(input.Password) != "" {
		hashed, err := security.HashPassword(input.Password)
		if err != nil {
			c.error("更新账户失败", http.StatusInternalServerError)
			return
		}
		user.Password = hashed
	}

	if _, err := orm.NewOrm().Update(&user); err != nil {
		c.error(err.Error(), http.StatusInternalServerError)
		return
	}

	c.respond(map[string]any{
		"id":       user.Id,
		"username": user.Username,
		"alias":    user.Alias,
		"role":     user.Role,
	}, http.StatusOK)
}

func currentUser(req *http.Request) string {
	return req.Header.Get("X-CRM-USER")
}

func currentRole(req *http.Request) string {
	return req.Header.Get("X-CRM-ROLE")
}

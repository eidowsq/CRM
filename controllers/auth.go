package controllers

import (
	"net/http"
	"strings"

	"crm/models"
	"crm/security"
	"github.com/beego/beego/v2/client/orm"
)

type AuthController struct{ APIController }

func (c *AuthController) Login() {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeBody(&c.APIController, &input); err != nil {
		c.error("请求格式不正确", http.StatusBadRequest)
		return
	}

	var users []models.User
	_, err := orm.NewOrm().QueryTable(new(models.User)).
		Filter("username", input.Username).
		All(&users)
	if err != nil {
		c.error("登录失败", http.StatusInternalServerError)
		return
	}
	if len(users) == 0 {
		c.error("用户名或密码错误", http.StatusUnauthorized)
		return
	}

	user := users[0]
	ok, err := security.VerifyPassword(user.Password, input.Password)
	if err != nil {
		c.error("登录失败", http.StatusInternalServerError)
		return
	}
	if !ok {
		c.error("用户名或密码错误", http.StatusUnauthorized)
		return
	}
	if !security.IsPasswordHashed(user.Password) {
		if hashed, err := security.HashPassword(input.Password); err == nil {
			user.Password = hashed
			_, _ = orm.NewOrm().Update(&user, "Password")
		}
	}
	c.respond(map[string]any{
		"token": user.Username,
		"user":  user.Username,
		"alias": user.Alias,
		"role":  user.Role,
	}, http.StatusOK)
}

func (c *AuthController) ChangePassword() {
	userName := strings.TrimSpace(currentUser(c.Ctx.Request))
	if userName == "" {
		c.error("请先登录", http.StatusUnauthorized)
		return
	}

	var input struct {
		OldPassword     string `json:"old_password"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := decodeBody(&c.APIController, &input); err != nil {
		c.error("请求格式不正确", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.OldPassword) == "" || strings.TrimSpace(input.NewPassword) == "" {
		c.error("旧密码和新密码不能为空", http.StatusBadRequest)
		return
	}
	if input.NewPassword != input.ConfirmPassword {
		c.error("两次输入的新密码不一致", http.StatusBadRequest)
		return
	}
	if err := security.ValidatePassword(input.NewPassword); err != nil {
		c.error(err.Error(), http.StatusBadRequest)
		return
	}

	var users []models.User
	_, err := orm.NewOrm().QueryTable(new(models.User)).
		Filter("username", userName).
		All(&users)
	if err != nil || len(users) == 0 {
		c.error("账户不存在", http.StatusNotFound)
		return
	}

	user := users[0]
	ok, err := security.VerifyPassword(user.Password, input.OldPassword)
	if err != nil {
		c.error("密码校验失败", http.StatusInternalServerError)
		return
	}
	if !ok {
		c.error("旧密码不正确", http.StatusBadRequest)
		return
	}

	hashed, err := security.HashPassword(input.NewPassword)
	if err != nil {
		c.error("密码更新失败", http.StatusInternalServerError)
		return
	}
	user.Password = hashed
	if _, err := orm.NewOrm().Update(&user, "Password"); err != nil {
		c.error("密码更新失败", http.StatusInternalServerError)
		return
	}

	c.respond(map[string]bool{"updated": true}, http.StatusOK)
}

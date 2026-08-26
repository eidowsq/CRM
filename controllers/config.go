package controllers

import "github.com/beego/beego/v2/server/web"

type ConfigController struct{ APIController }

func (c *ConfigController) Get() {
	c.respond(map[string]any{
		"brand_name":    web.AppConfig.DefaultString("brand_name", web.AppConfig.DefaultString("appname", "northstar")),
		"workspace_name": web.AppConfig.DefaultString("workspace_name", "销售工作台"),
	}, 200)
}

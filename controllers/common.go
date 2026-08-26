package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/beego/beego/v2/server/web"
)

type APIController struct {
	web.Controller
}

func (c *APIController) respond(data any, status int) {
	c.Ctx.ResponseWriter.WriteHeader(status)
	c.Data["json"] = map[string]any{"data": data}
	c.ServeJSON()
}

func (c *APIController) error(message string, status int) {
	c.Data["json"] = map[string]any{"error": message}
	c.Ctx.ResponseWriter.WriteHeader(status)
	c.ServeJSON()
}

func decodeBody(c *APIController, target any) error {
	return json.Unmarshal(c.Ctx.Input.RequestBody, target)
}

func queryInt(c *APIController, key string) int {
	value, _ := c.GetInt(key)
	return value
}

var _ = http.StatusOK

package routers

import (
	_ "crm/controllers"
	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.SetStaticPath("/", "static")
}

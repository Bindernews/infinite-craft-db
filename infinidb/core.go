package infinidb

import (
	"log/slog"
	"net/http"

	"binder.fun/infinicraft/infinidb/rdb"
	"github.com/gin-gonic/gin"
)

type InfiniDb struct {
	Log *slog.Logger
	// The public hostname
	Hostname string
	// Database interface
	Db rdb.RecipeDb
	// Gin web server
	Web *gin.Engine
}

func (me *InfiniDb) Setup() (err error) {
	// Setup engine and routes
	r := gin.Default()
	r.LoadHTMLGlob("infinidb/template/*.html")
	r.GET("/", me.Home)
	r.GET("/item", me.ItemList)
	r.GET("/recipe/list/:item", me.ListRecipes)
	r.GET("/recipe/tree/:item", me.RecursiveRecipes)
	r.GET("/recipe/add", me.AddRecipeGET)
	r.POST("/recipe/add", me.AddRecipePOST)
	r.POST("/generate/start", me.GenerateStartPOST)
	r.POST("/generate/stop", me.GenerateStopPOST)
	me.Web = r

	return nil
}

func (me *InfiniDb) Home(c *gin.Context) {
	c.Redirect(http.StatusSeeOther, "/item")
}

func (me *InfiniDb) ItemList(c *gin.Context) {
	var param SearchParam
	var err error
	var items []string
	if err = c.Bind(&param); err != nil {
		me.ErrorPage(c, http.StatusBadRequest)
		return
	}
	if items, err = me.Db.ItemsLike(c.Request.Context(), param.Query, 500); err != nil {
		c.Error(err)
		me.ErrorPage(c, http.StatusBadRequest)
		return
	}
	c.HTML(http.StatusOK, "item.html", gin.H{
		"Items": items,
	})
}

func (me *InfiniDb) GenerateStartPOST(c *gin.Context) {

}

func (me *InfiniDb) GenerateStopPOST(c *gin.Context) {

}

func (me *InfiniDb) ErrorPage(c *gin.Context, code int) {
	c.HTML(code, "error.html", gin.H{
		"Code":   code,
		"Errors": c.Errors,
	})
}

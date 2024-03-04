package infinidb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"binder.fun/infinicraft/infinidb/rdb"
	"github.com/gin-gonic/gin"
)

type Recipe struct {
	// First recipe source
	SrcA string `form:"src_a" json:"src_a" binding:"required"`
	// Second recipe source
	SrcB string `form:"src_b" json:"src_b" binding:"required"`
	// Recipe output
	Result string `form:"dst" json:"dst" binding:"required"`
}

type SearchParam struct {
	// Query to search for
	Query string `form:"query" json:"query"`
}

func (me *InfiniDb) AddRecipeGET(c *gin.Context) {
	status := c.Query("status")
	if c.Query("thank-you") == "1" {
		status = "Thank you!"
	}
	c.HTML(http.StatusOK, "add_recipe.html", gin.H{
		"Status": status,
	})
}

func (me *InfiniDb) AddRecipePOST(c *gin.Context) {
	var params Recipe
	var err error

	if err = c.Bind(&params); err != nil {
		me.ErrorPage(c, http.StatusBadRequest)
		return
	}
	recipe := rdb.Recipe{SrcA: params.SrcA, SrcB: params.SrcB, Result: params.Result}
	if err = me.Db.AddRecipe(context.Background(), recipe); err != nil {
		var notFound rdb.ErrItemNotFound
		if errors.As(err, &notFound) {
			c.Redirect(http.StatusSeeOther, me.Hostname+"/recipe/add?status="+notFound.Error())
		} else {
			me.Log.Warn("add recipe", "error", err)
			me.ErrorPage(c, http.StatusBadRequest)
		}
		return
	}
	// Redirect to thank you page
	c.Redirect(http.StatusSeeOther, me.Hostname+"/recipe/add?thank-you=1")
}

func (me *InfiniDb) ListRecipes(c *gin.Context) {
	var err error
	var recipes []rdb.RecipeSources

	item := c.Param("item")
	if item == "" {
		me.ErrorPage(c, http.StatusNotFound)
		return
	}

	if recipes, err = me.Db.ListRecipes(c.Request.Context(), item); err != nil {
		me.Log.Warn("sql error", "item", item, "error", err)
		me.ErrorPage(c, http.StatusBadRequest)
		return
	}
	c.HTML(http.StatusOK, "recipe-list.html", gin.H{
		"Recipes": recipes,
	})
}

func (me *InfiniDb) RecursiveRecipes(c *gin.Context) {
	var err error
	var tree json.RawMessage

	item := c.Param("item")
	if item == "" {
		me.ErrorPage(c, http.StatusNotFound)
		return
	}

	// Try to get the tree
	if tree, err = me.Db.RecipeJsonTree(c.Request.Context(), item, true); err != nil {
		me.Log.Warn("sql error", "item", item, "error", err)
		me.ErrorPage(c, 404)
		return
	}
	// Print it
	c.HTML(http.StatusOK, "recipe-tree.html", gin.H{
		"Tree": string(tree),
	})
}
